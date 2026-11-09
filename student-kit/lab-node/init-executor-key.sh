#!/bin/sh
# genereaza o singura data perechea de chei ssh a executorului, pe volumul montat la /executor-key
set -eu
if [ ! -f /executor-key/id_ed25519 ]; then
  ssh-keygen -q -t ed25519 -N "" -C lab-executor -f /executor-key/id_ed25519
fi
