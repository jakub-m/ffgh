#!/bin/bash

export PATH=${PATH}:$PWD/bin/

set -eux
set -o pipefail
make
command=$(./bin/ffgh  show-command)
./bin/ffgh -v fzf | eval "$command"
