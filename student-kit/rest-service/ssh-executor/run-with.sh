#!/bin/bash
# porneste ssh executor-ul langa web server-ul vostru (comanda primita din CMD)
# daca unul din ele moare, il oprim si pe celalalt si iese containerul

/opt/ssh-executor/venv/bin/python /opt/ssh-executor/executor.py &
"$@" &

trap 'kill $(jobs -p)' TERM INT    # docker stop -> le oprim pe amandoua
wait -n                            # asteptam pana se termina oricare din ele
status=$?
kill $(jobs -p) 2>/dev/null        # si il oprim si pe celalalt
wait
exit $status
