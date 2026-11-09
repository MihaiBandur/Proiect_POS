#!/bin/sh
set -eu
LAB_USER="${LAB_SSH_USER:-labuser}"
NODE_ID="${LAB_NODE_ID:?LAB_NODE_ID lipseste}"

# 1. userul de lab - fara parola, se intra doar cu cheia publica a executorului
id "$LAB_USER" >/dev/null 2>&1 || useradd --create-home --shell /bin/bash "$LAB_USER"
passwd -d "$LAB_USER" >/dev/null
mkdir -p "/home/$LAB_USER/scripts" "/home/$LAB_USER/.ssh"
cp /executor-key/id_ed25519.pub "/home/$LAB_USER/.ssh/authorized_keys"
chmod 700 "/home/$LAB_USER/.ssh"
chown -R "$LAB_USER:$LAB_USER" "/home/$LAB_USER/scripts" "/home/$LAB_USER/.ssh"

# 2. host key-urile masinii
ssh-keygen -A

# 3. publicam host key-ul pe volumul comun, ca linie known_hosts: "<ip-urile masinii> <cheia>"
echo "$(hostname -I | xargs | tr ' ' ',') $(cut -d' ' -f1,2 /etc/ssh/ssh_host_ed25519_key.pub)" \
  > "/host-keys/$NODE_ID.known_hosts"

# 4. sshd doar cu chei, fara root/forwarding
cat > /etc/ssh/sshd_config.d/lab.conf <<EOF
PasswordAuthentication no
KbdInteractiveAuthentication no
PubkeyAuthentication yes
PermitRootLogin no
AllowTcpForwarding no
X11Forwarding no
AllowUsers $LAB_USER
EOF

exec /usr/sbin/sshd -D -e
