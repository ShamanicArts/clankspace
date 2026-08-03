#!/bin/sh
set -eu

data_dir="${CLANKSPACE_DATA_DIR:-/data}"

# A Railway volume replaces the image's pre-owned mount directory with a
# root-owned filesystem. Repair only the directory and SQLite files that
# ClankSpace needs, then permanently drop privileges before starting.
chown clank:clank "$data_dir"
chmod 700 "$data_dir"
for sqlite_path in \
  "$data_dir/clankspace.db" \
  "$data_dir/clankspace.db-wal" \
  "$data_dir/clankspace.db-shm"
do
  if [ -e "$sqlite_path" ]; then
    chown clank:clank "$sqlite_path"
    chmod 600 "$sqlite_path"
  fi
done

umask 077
exec su-exec clank "$@"
