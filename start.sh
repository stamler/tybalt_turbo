#!/bin/sh
set -e

# Check if database exists, if not restore from backup
if [ ! -f /app/pb_data/data.db ]; then
  echo "Database not found, restoring from backup..."
  litestream restore -config /etc/litestream.yml /app/pb_data/data.db
  echo "Database restored successfully"
fi

# Start litestream in background
litestream replicate -config /etc/litestream.yml &

# Start the main application
exec ./tybalt serve --http=0.0.0.0:8080