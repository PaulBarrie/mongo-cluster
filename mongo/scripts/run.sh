#!/bin/bash
set -e

# Initialize first run
if [[ -e /.firstrun ]]; then
    /scripts/first_run.sh
fi

# Startup cron for log rotation.
cron

echo "Starting MongoDB..."

/usr/bin/mongod --replSet $MONGODB_REPLICA_ID --dbpath /data/db --bind_ip 0.0.0.0 --clusterAuthMode keyFile --keyFile /etc/secrets-volume/password --setParameter authenticationMechanisms=SCRAM-SHA-256 --auth --logpath /data/mongodb.log;

exec "$@"