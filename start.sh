#!/bin/sh
set -e

# Check if database exists, if not try to restore from backup
if [ ! -f /pb/pb_data/data.db ]; then
  echo "Database not found, attempting to restore from backup..."
  litestream restore -config /etc/litestream.yml -if-replica-exists /pb/pb_data/data.db || echo "No backup found, will create new database"
fi

# Start litestream in background
litestream replicate -config /etc/litestream.yml &

# Start the main application
exec ./tybalt serve --http=0.0.0.0:8080 