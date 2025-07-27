#!/bin/bash
set -eu
set -o pipefail
while [ 1 ] ; do  ./bin/ffgh-bin -v sync || true ; echo "RESTART"; sleep 1; done
