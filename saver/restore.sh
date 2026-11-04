#!/bin/sh
set -eu

. /usr/local/lib/saver/lib.sh

set +e
restic cat config >/dev/null
status=$?
set -e

if [ "$status" -eq 10 ]; then
    restic init
elif [ "$status" -ne 0 ]; then
    exit "$status"
fi

restic unlock

snapshots=$(restic snapshots --host world --latest 1 --json)

case "$snapshots" in
    *'"id"'*) restic restore latest:/data --host world --target /data ;;
esac
