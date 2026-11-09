# o masina din lab - debian cu sshd, pe care executorul copiaza si ruleaza scripturile
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends openssh-server procps iproute2 && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /run/sshd && \
    rm -f /etc/ssh/ssh_host_*
# ^ apt genereaza host key-uri la instalare, deci ar fi aceleasi pe toate masinile

COPY entrypoint.sh init-executor-key.sh /
RUN chmod 0755 /entrypoint.sh /init-executor-key.sh

EXPOSE 22

# healthy = sshd ruleaza si host key-ul e publicat pt executor
HEALTHCHECK --interval=5s --timeout=3s --retries=20 \
    CMD pgrep -x sshd >/dev/null && test -s "/host-keys/$LAB_NODE_ID.known_hosts"

ENTRYPOINT ["/entrypoint.sh"]
