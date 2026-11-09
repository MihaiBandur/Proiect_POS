ARG BASE_IMAGE=python:3.12-slim
FROM ${BASE_IMAGE}

# ssh executor
# daca imaginea are deja python 3.11+ il folosim pe ala / daca nu il instalam
RUN python3 -m venv /opt/ssh-executor/venv 2>/dev/null || { \
        apt-get update && \
        apt-get install -y --no-install-recommends python3 python3-venv && \
        rm -rf /var/lib/apt/lists/* && \
        python3 -m venv --clear /opt/ssh-executor/venv; \
    } && \
    /opt/ssh-executor/venv/bin/pip install --no-cache-dir asyncssh==2.21.0 pymongo==4.15.1
COPY ssh-executor/ /opt/ssh-executor/
RUN chmod 0755 /opt/ssh-executor/run-with.sh
ENTRYPOINT ["/opt/ssh-executor/run-with.sh"]

# web server-ul pentru REST
WORKDIR /app
# COPY . /app
# RUN pip install --no-cache-dir -r requirements.txt
EXPOSE 8000
# CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
CMD ["python3", "-m", "http.server", "8000"]
