# A&A Services (gRPC + MariaDB) intr-un container.
FROM python:3.13-slim

# Fara fisiere .pyc si cu log-urile print() afisate imediat in "docker logs".
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# Dependintele intai: Docker refoloseste acest strat cat timp requirements.txt nu se schimba.
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Serviciul nu ruleaza ca root; folderul logs/ trebuie sa fie al lui (vezi logs.py).
RUN useradd --create-home --uid 10001 auth && mkdir -p logs && chown -R auth /app
USER auth

EXPOSE 50051
CMD ["python", "auth_service.py"]
