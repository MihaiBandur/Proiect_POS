# kit pos - ssh executor + masinile din lab

```
compose.yaml               - exemplu cu 3 masini de lab, rest + mongo
rest-service/
  Dockerfile               - containerul de rest: web server-ul vostru + executorul
  ssh-executor/
    executor.py            - executorul propriu-zis, nu trebuie modificat
    run-with.sh            - porneste executorul + serverul vostru si le opreste impreuna
lab-node/
  Dockerfile               - imaginea pt masinile din lab (debian + sshd) cu healthcheck
  entrypoint.sh            - creeaza userul, pune cheia executorului, publica host key-ul, porneste sshd
  init-executor-key.sh     - genereaza o data perechea de chei ssh a executorului
```

## Workflow-ul

1. Serviciul REST pune un job in mongo cu `state: "pending"`
2. Serviciul REST trimite id-ul job-ului pe socket - `{"type": "run", "job_id": "..."}`
3. Executorul face claim pe job (`state: "active"`), copiaza scriptul prin sftp pe fiecare calculator tinta, il ruleaza prin ssh
4. Executorul scrie rezultatul in acelasi document (`state: "finished"`, `status`, `results`, `notification: false`)
5. Job status service-ul vostru vede in mongo schimbarile cu `notification: false`, le trimite pe websocket si pune `notification: true`

- pe socket vine doar un id
- daca se pierde mesajul (rest picat, executor restartat etc) executorul gaseste oricum job-urile pending la un scan periodic (default la 10s)

## Containerul REST

- in `rest-service/Dockerfile` schimbati `BASE_IMAGE` pe imaginea limbajului vostru (debian/ubuntu) si completati partea de la final cu app-ul vostru
- lasati `ENTRYPOINT`-ul cum e si puneti comanda serverului in `CMD`
- containerul trebuie sa ruleze ca root (default) - cheia ssh e citibila doar de root
- daca moare executorul sau serverul, iese tot containerul - puneti `restart: unless-stopped` in compose

## Contractul IPC (socket)

- unix stream socket, la `IPC_SOCKET_PATH` (default `/run/lab-manager/ipc/executor.sock`)
- o conexiune = un request + un raspuns, fiecare e o linie de json terminata cu `\n`
- aveti 2s sa trimiteti linia dupa connect, max 16 KiB

### health

```json
{"type": "health"}
```
```json
{"status": "ready", "workers": 3, "queued": 0}
```

### run

```json
{"type": "run", "job_id": "6650f1c2a1b2c3d4e5f60718"}
```
```json
{"status": "accepted", "job_id": "6650f1c2a1b2c3d4e5f60718"}
```

`accepted` inseamna ca job-ul e in coada - rest-ul raspunde `202 Accepted` si clientul urmareste job-ul (polling pe resursa job-ului sau notificarile pe websocket)

### rejected

```json
{"status": "rejected", "error": "job not found"}
```

cazuri: json invalid sau care nu e obiect, `type` necunoscut, `job_id` lipsa / nu e string hex de 24, job inexistent, job care nu mai e pending (`job is already finished`), linie prea lunga, nimic trimis in 2s

## Campurile din mongo

default executorul foloseste `jobs`, `scripts`, `computers`, `clusters`, si le puteti schimba cu `JOBS_COLLECTION`, `SCRIPTS_COLLECTION`, `COMPUTERS_COLLECTION`, `CLUSTERS_COLLECTION`. id-urile pot fi `ObjectId` sau string-ul hex, le accepta pe amandoua

### ce citeste

| colectie (tabel spec) | camp | cum il foloseste |
| --- | --- | --- |
| scripturi (tabel 5) | `store_path` | fisierul de pe volumul de scripturi care se copiaza pe masini |
| | `store_comp` | calea absoluta pe masina unde se copiaza si de unde se ruleaza |
| | `is_active` | trebuie sa fie `true` - daca lipseste e considerat inactiv, ca in spec |
| | `name` | doar pt log |
| calculatoare (tabel 3) | `IP` | unde se conecteaza prin ssh |
| | `hostname` | doar copiat in rezultat |
| clustere (tabel 4) | `id_comp` | calculatoarele pe care ruleaza un job pe cluster |
| istoric rulari (tabel 6) | `id_script` | scriptul de rulat |
| | `state` | `pending` (sau lipsa) - doar astea se ruleaza |
| | `target` | **camp nou** - `{"tip": "calculator" \| "cluster", "id": ...}`, pe ce ruleaza. tabelul 6 nu are tinta, iar spec-ul zice ca puteti adauga campuri - formatul e acelasi ca in `id_comp_clust` din tabelul 2 |


### ce scrie in job

| camp | cand | valoare |
| --- | --- | --- |
| `state` | claim / final | `active`, apoi `finished` |
| `date` | la fiecare schimbare de stare | data curenta (utc) |
| `notification` | la fiecare schimbare de stare | `false` - il faceti `true` dupa ce ati trimis notificarea |
| `status` | final | `succeeded` (toate masinile ok), `failed` (niciuna), `partial`, `error` (nici n-a ajuns pe masini) |
| `message` | final | camp nou - text scurt, de ex `2/2 computers succeeded` sau motivul erorii |
| `results` | final | camp nou - o intrare per calculator, vezi mai jos |

```json
{
  "id_comp": "ObjectId(...)",
  "IP": "10.89.0.11",
  "hostname": "node-1",
  "status": "succeeded",
  "exit_code": 0,
  "stdout": "...",
  "stderr": "",
  "output_truncated": false,
  "started": "2026-11-09T10:00:00Z",
  "finished": "2026-11-09T10:00:01Z"
}
```

`status` per calculator: `succeeded` (exit 0), `failed` (exit diferit de 0), `timed_out`, `output_limit_exceeded`, `error` (n-a mers conexiunea, copierea etc - detaliile in `stderr`)

daca executorul e restartat cat timp un job e `active`, la pornire il trece pe `finished` cu `status: "error"` - nu-l mai ruleaza inca o data, ca nu stim ce apucase sa faca pe masini

## cum ruleaza scripturile

- la fiecare rulare scriptul e copiat din `store_path` in `store_comp` (rename atomic), deci pe masini e mereu versiunea curenta - "instalarea" unui script pe un calculator (`id_script` din tabelul 3) e doar o inregistrare in mongo, nu trebuie copiat nimic separat
- daca scriptul incepe cu `#!` ruleaza cu interpretorul de acolo (`#!/bin/bash` etc)- daca nu atunci cu `/bin/sh`
- ruleaza ca userul `labuser`, deci `store_comp` ar trebui sa fie sub `/home/labuser/` (de ex `/home/labuser/scripts/x.sh`)
- pe un cluster ruleaza in paralel pe toate masinile

## env vars (toate au default)

| var | default | ce face |
| --- | --- | --- |
| `MONGODB_URI` | `mongodb://mongo:27017/lab_manager` | |
| `IPC_SOCKET_PATH` | `/run/lab-manager/ipc/executor.sock` | |
| `SCRIPT_VOLUME` | `/var/lib/lab-manager/scripts` | |
| `LAB_SSH_PRIVATE_KEY` | `/run/lab-manager/ssh/id_ed25519` | |
| `LAB_HOST_KEYS_DIR` | `/var/lib/lab-manager/host-keys` | |
| `LAB_SSH_USER` / `LAB_SSH_PORT` | `labuser` / `22` | |
| `EXECUTOR_WORKERS` | `3` | job-uri in paralel |
| `MAX_SSH_CONNECTIONS` | `6` | sesiuni ssh simultane in total |
| `SSH_CONNECT_TIMEOUT_SECONDS` | `5` | |
| `SCRIPT_COPY_TIMEOUT_SECONDS` | `30` | |
| `SCRIPT_TIMEOUT_SECONDS` | `30` | dupa atat scriptul e oprit (`timed_out`) |
| `MAX_OUTPUT_BYTES` | `262144` | per stdout / stderr, peste e oprit (`output_limit_exceeded`) |
| `RESCAN_SECONDS` | `10` | cat de des cauta job-uri pending ratate |
