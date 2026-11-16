# Job Management Service + executorul SSH din student-kit, in acelasi container.
# Se construieste din radacina proiectului (vezi docker-compose.yml: context ".").

# Etapa 1: compilarea serviciului Go (imagine mare, folosita doar la build).
FROM golang:1.27.3-alpine AS build
WORKDIR /src
# Dependintele intai: Docker refoloseste acest strat cat timp go.mod si go.sum nu se schimba.
COPY database_layer_service/go.mod database_layer_service/go.sum ./
RUN go mod download
COPY database_layer_service/ .
# CGO_ENABLED=0: executabil static. -p 1: compileaza un pachet o data; altfel, pe o masina
# virtuala cu putina memorie (Colima are implicit 2 GB), compilatorul poate fi oprit de sistem.
RUN CGO_ENABLED=0 go build -p 1 -trimpath -ldflags="-s -w" -o /out/job-management-service .

# Etapa 2: containerul REST din student-kit (rest-service/Dockerfile): executorul + serverul nostru.
FROM python:3.12-slim
# Executorul SSH, exact ca in kit (nu se modifica).
RUN python3 -m venv /opt/ssh-executor/venv && \
    /opt/ssh-executor/venv/bin/pip install --no-cache-dir asyncssh==2.21.0 pymongo==4.15.1
COPY student-kit/rest-service/ssh-executor/ /opt/ssh-executor/
RUN chmod 0755 /opt/ssh-executor/run-with.sh

COPY --from=build /out/job-management-service /usr/local/bin/job-management-service

ENV GIN_MODE=release \
    PORT=8080 \
    STORAGE=mongo \
    SCRIPTS_DIR=/var/lib/lab-manager/scripts
EXPOSE 8080

# Ruleaza ca root, cum cere kit-ul: cheia SSH a executorului e citibila doar de root.
# run-with.sh porneste executorul si serverul nostru (CMD) si le opreste impreuna.
ENTRYPOINT ["/opt/ssh-executor/run-with.sh"]
CMD ["/usr/local/bin/job-management-service"]
