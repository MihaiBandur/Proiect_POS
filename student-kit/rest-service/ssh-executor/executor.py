"""ssh executor - ruleaza scripturile pe calculatoarele din lab

  1. rest-ul vostru pune job-ul in mongo, cu state "pending"
  2. trimite id-ul pe unix socket:  {"type": "run", "job_id": "..."}
  3. executorul face claim pe job (state "active"), copiaza scriptul prin sftp pe fiecare calculator tinta, il ruleaza prin ssh si strange output-ul
  4. scrie rezultatul inapoi in job (state "finished", notification false)
"""

import asyncio
import json
import logging
import os
import shlex
import signal
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath

import asyncssh
from bson import ObjectId
from pymongo import AsyncMongoClient, ReturnDocument

# config
MONGODB_URI = os.getenv("MONGODB_URI", "mongodb://mongo:27017/lab_manager")
IPC_SOCKET_PATH = os.getenv("IPC_SOCKET_PATH", "/run/lab-manager/ipc/executor.sock")
SCRIPT_VOLUME = Path(os.getenv("SCRIPT_VOLUME", "/var/lib/lab-manager/scripts")).resolve()
SSH_USER = os.getenv("LAB_SSH_USER", "labuser")
SSH_PORT = int(os.getenv("LAB_SSH_PORT", "22"))
SSH_PRIVATE_KEY = os.getenv("LAB_SSH_PRIVATE_KEY", "/run/lab-manager/ssh/id_ed25519")
HOST_KEYS_DIR = Path(os.getenv("LAB_HOST_KEYS_DIR", "/var/lib/lab-manager/host-keys"))
WORKERS = int(os.getenv("EXECUTOR_WORKERS", "3")) # cate job-uri ruleaza in paralel
MAX_SSH_CONNECTIONS = int(os.getenv("MAX_SSH_CONNECTIONS", "6")) # cate sesiuni ssh deschise simultan, in total
CONNECT_TIMEOUT = float(os.getenv("SSH_CONNECT_TIMEOUT_SECONDS", "5"))
COPY_TIMEOUT = float(os.getenv("SCRIPT_COPY_TIMEOUT_SECONDS", "30"))
SCRIPT_TIMEOUT = float(os.getenv("SCRIPT_TIMEOUT_SECONDS", "30"))
MAX_OUTPUT_BYTES = int(os.getenv("MAX_OUTPUT_BYTES", "262144")) # per stream (stdout / stderr)
RESCAN_SECONDS = float(os.getenv("RESCAN_SECONDS", "10"))

logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"), format="%(asctime)s %(levelname)s %(message)s")
logging.getLogger("asyncssh").setLevel(logging.WARNING)
log = logging.getLogger("ssh-executor")

mongo = AsyncMongoClient(MONGODB_URI, serverSelectionTimeoutMS=3000)
db = mongo.get_default_database()
jobs = db[os.getenv("JOBS_COLLECTION", "jobs")]
scripts = db[os.getenv("SCRIPTS_COLLECTION", "scripts")]
computers = db[os.getenv("COMPUTERS_COLLECTION", "computers")]
clusters = db[os.getenv("CLUSTERS_COLLECTION", "clusters")]

# daca lipseste campul "state" job-ul e tot pending (tabelul 6 din specs proiect)
PENDING = {"state": {"$in": [None, "pending"]}}

queue: asyncio.Queue[ObjectId] = asyncio.Queue()
queued: set[ObjectId] = set()
ssh_slots = asyncio.Semaphore(MAX_SSH_CONNECTIONS)


def now() -> datetime:
    return datetime.now(timezone.utc)


def to_object_id(value) -> ObjectId:
    if isinstance(value, ObjectId):
        return value
    if isinstance(value, str) and ObjectId.is_valid(value):
        return ObjectId(value)
    raise ValueError(f"not a valid ObjectId: {value!r}")


async def enqueue(job_id: ObjectId) -> None:
    if job_id not in queued:
        queued.add(job_id)
        await queue.put(job_id)


# ipc (unix socket)
async def handle_ipc(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
    try:
        line = await asyncio.wait_for(reader.readline(), timeout=2)
        reply = await answer(line)
    except Exception as error:
        reply = {"status": "rejected", "error": str(error) or type(error).__name__}
        log.warning("rejected IPC request: %s", reply["error"])
    try:
        writer.write(json.dumps(reply).encode() + b"\n")
        await writer.drain()
    finally:
        writer.close()


async def answer(line: bytes) -> dict:
    message = json.loads(line)
    if not isinstance(message, dict):
        raise ValueError("the message must be a JSON object")
    if message.get("type") == "health":
        return {"status": "ready", "workers": WORKERS, "queued": queue.qsize()}
    if message.get("type") != "run":
        raise ValueError("type must be 'health' or 'run'")

    # verificam id-ul si existenta job-ului in db
    job_id = message.get("job_id")
    if not isinstance(job_id, str) or not ObjectId.is_valid(job_id):
        raise ValueError("job_id must be a 24-character hex string")
    job = await jobs.find_one({"_id": ObjectId(job_id)}, {"state": 1})
    if job is None:
        raise ValueError("job not found")
    if job.get("state", "pending") != "pending":
        raise ValueError(f"job is already {job['state']}")
    await enqueue(ObjectId(job_id))
    return {"status": "accepted", "job_id": job_id}


# procesare job-uri
async def worker(number: int) -> None:
    while True:
        job_id = await queue.get()
        queued.discard(job_id)
        try:
            await run_job(job_id)
        except Exception as error:
            log.exception("worker %s: job %s crashed", number, job_id)
            await finish(job_id, "error", f"executor error: {error}", [])


async def run_job(job_id: ObjectId) -> None:
    # claim pe job: pending -> active
    # la fiecare schimbare de stare punem notification false, ca job status service-ul sa trimita notif
    job = await jobs.find_one_and_update(
        {"_id": job_id, **PENDING},
        {"$set": {"state": "active", "date": now(), "notification": False}},
        return_document=ReturnDocument.AFTER,
    )
    if job is None:
        return

    script = await scripts.find_one({"_id": job.get("id_script")})
    if script is None or not script.get("is_active", False):
        await finish(job_id, "error", "script does not exist or is not active", [])
        return
    try:
        local_path = local_script_path(script)
        remote_path = remote_script_path(script)
        targets = await target_computers(job)
    except ValueError as error:
        await finish(job_id, "error", str(error), [])
        return

    log.info("job %s: running %s on %s", job_id, script.get("name"), [c.get("IP") for c in targets])
    # run pe toate calculatoarele in paralel
    results = await asyncio.gather(*(run_on_computer(c, local_path, remote_path) for c in targets))

    succeeded = sum(result["status"] == "succeeded" for result in results)
    if succeeded == len(results):
        status = "succeeded"
    elif succeeded == 0:
        status = "failed"
    else:
        status = "partial"
    await finish(job_id, status, f"{succeeded}/{len(results)} computers succeeded", results)
    log.info("job %s: %s", job_id, status)


async def finish(job_id: ObjectId, status: str, message: str, results: list[dict]) -> None:
    await jobs.update_one(
        {"_id": job_id, "state": "active"},
        {"$set": {
            "state": "finished",
            "status": status, # succeeded / failed / partial / error
            "message": message,
            "results": results,
            "date": now(),
            "notification": False,
        }},
    )


def local_script_path(script: dict) -> Path:
    """store_path trebuie sa fie un fisier de pe volumul partajat de scripturi"""
    path = Path(script.get("store_path") or "/").resolve()
    if not path.is_relative_to(SCRIPT_VOLUME) or not path.is_file():
        raise ValueError(f"store_path must be an existing file inside {SCRIPT_VOLUME}")
    return path


def remote_script_path(script: dict) -> str:
    path = PurePosixPath(script.get("store_comp") or "")
    if not path.is_absolute() or ".." in path.parts:
        raise ValueError("store_comp must be an absolute path on the target computer")
    return str(path)


async def target_computers(job: dict) -> list[dict]:
    """job.target = {"tip": "calculator" | "cluster", "id": <ObjectId>} -- pt cluster luam calculatoarele din id_comp"""
    target = job.get("target") or {}
    if target.get("tip") == "calculator":
        ids = [to_object_id(target.get("id"))]
    elif target.get("tip") == "cluster":
        cluster = await clusters.find_one({"_id": to_object_id(target.get("id"))})
        if cluster is None:
            raise ValueError("target cluster not found")
        ids = [to_object_id(value) for value in cluster.get("id_comp", [])]
    else:
        raise ValueError("job.target.tip must be 'calculator' or 'cluster'")
    found = await computers.find({"_id": {"$in": ids}}).to_list(None)
    if not ids or len(found) != len(set(ids)):
        raise ValueError("the target has no computers, or one of them does not exist")
    return found


# ssh
def connect(address: str):
    # host key-urile vin de la masinile din lab (known_hosts pe volumul de host-keys)
    trusted = "".join(path.read_text() for path in sorted(HOST_KEYS_DIR.glob("*.known_hosts")))
    return asyncssh.connect(
        address,
        port=SSH_PORT,
        username=SSH_USER,
        client_keys=[SSH_PRIVATE_KEY],
        known_hosts=asyncssh.import_known_hosts(trusted),
        connect_timeout=CONNECT_TIMEOUT,
    )


async def run_on_computer(computer: dict, local_path: Path, remote_path: str) -> dict:
    result = {
        "id_comp": computer["_id"],
        "IP": computer.get("IP"),
        "hostname": computer.get("hostname"),
        "status": "error",
        "exit_code": None,
        "stdout": "",
        "stderr": "",
        "output_truncated": False,
        "started": now(),
        "finished": None,
    }
    try:
        if not computer.get("IP"):
            raise ValueError("computer has no IP")
        async with ssh_slots:
            async with connect(computer["IP"]) as connection:
                await asyncio.wait_for(copy_script(connection, local_path, remote_path), COPY_TIMEOUT)
                await execute(connection, local_path, remote_path, result)
    except Exception as error:
        result["stderr"] = f"{type(error).__name__}: {error}"[:2000]
    result["finished"] = now()
    return result


async def copy_script(connection, local_path: Path, remote_path: str) -> None:
    """copiem scriptul la fiecare rulare pentru a fi mereu the newest version pe masina"""
    temporary = f"{remote_path}.{ObjectId()}.tmp"
    directory = str(PurePosixPath(remote_path).parent)
    await connection.run(f"mkdir -p -- {shlex.quote(directory)}", check=True)
    async with connection.start_sftp_client() as sftp:
        await sftp.put(str(local_path), temporary)
        await sftp.chmod(temporary, 0o700)
        await sftp.posix_rename(temporary, remote_path)


async def execute(connection, local_path: Path, remote_path: str, result: dict) -> None:
    has_shebang = local_path.read_bytes()[:2] == b"#!"
    command = f"exec {shlex.quote(remote_path)}" if has_shebang else f"exec /bin/sh {shlex.quote(remote_path)}"
    process = await connection.create_process(command, encoding=None)
    readers = [asyncio.create_task(read_limited(stream, process)) for stream in (process.stdout, process.stderr)]
    timed_out = False
    try:
        await asyncio.wait_for(process.wait_closed(), SCRIPT_TIMEOUT)
    except TimeoutError:
        timed_out = True
        process.kill()
        process.close()
    (stdout, stdout_cut), (stderr, stderr_cut) = await asyncio.gather(*readers)

    result.update(stdout=stdout, stderr=stderr, output_truncated=stdout_cut or stderr_cut)
    result["exit_code"] = None if timed_out else process.returncode
    if result["output_truncated"]:
        result["status"] = "output_limit_exceeded"
    elif timed_out:
        result["status"] = "timed_out"
        result["stderr"] += f"\n[executor] stopped after {SCRIPT_TIMEOUT:g} seconds"
    else:
        result["status"] = "succeeded" if process.returncode == 0 else "failed"


async def read_limited(stream, process) -> tuple[str, bool]:
    """citim tot stream-ul, pastram din el MAX_OUTPUT_BYTES"""
    kept, truncated = bytearray(), False
    try:
        while chunk := await stream.read(65536):
            room = MAX_OUTPUT_BYTES - len(kept)
            kept += chunk[:room]
            if len(chunk) > room and not truncated:
                truncated = True
                process.terminate()
    except (asyncssh.Error, OSError):
        pass
    return kept.decode("utf-8", errors="replace"), truncated


# startup / background
async def rescan() -> None:
    while True:
        try:
            async for job in jobs.find(PENDING, {"_id": 1}).limit(100):
                await enqueue(job["_id"])
        except Exception:
            log.exception("pending-job scan failed")
        await asyncio.sleep(RESCAN_SECONDS)


async def wait_for_mongo() -> None:
    while True:
        try:
            await mongo.admin.command("ping")
            return
        except Exception as error:
            log.warning("waiting for MongoDB: %s", error)
            await asyncio.sleep(2)


async def main() -> None:
    await wait_for_mongo()

    interrupted = await jobs.update_many({"state": "active"}, {"$set": {
        "state": "finished", "status": "error", "message": "executor restarted while the job was running",
        "results": [], "date": now(), "notification": False,
    }})
    if interrupted.modified_count:
        log.warning("marked %s interrupted job(s) as error", interrupted.modified_count)

    socket_path = Path(IPC_SOCKET_PATH)
    socket_path.parent.mkdir(parents=True, exist_ok=True)
    socket_path.unlink(missing_ok=True)
    server = await asyncio.start_unix_server(handle_ipc, path=socket_path, limit=16384)
    socket_path.chmod(0o666)

    tasks = [asyncio.create_task(rescan())]
    tasks += [asyncio.create_task(worker(number)) for number in range(1, WORKERS + 1)]
    log.info("listening on %s with %s workers", socket_path, WORKERS)

    stop = asyncio.Event()
    for sig in (signal.SIGTERM, signal.SIGINT):
        asyncio.get_running_loop().add_signal_handler(sig, stop.set)
    async with server:
        await stop.wait()
    for task in tasks:
        task.cancel()
    socket_path.unlink(missing_ok=True)
    log.info("stopped")


if __name__ == "__main__":
    asyncio.run(main())
