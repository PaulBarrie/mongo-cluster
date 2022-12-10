#!/bin/bash

set -e

(crontab -l ; echo "* * * * * bash /scripts/logs.sh") | crontab

./scripts/run.sh

exec "$@"