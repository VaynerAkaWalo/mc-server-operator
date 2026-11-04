#!/bin/sh
set -u

. /usr/local/lib/saver/lib.sh

cycles=0

snapshot() {
    rcon-cli save-off || return 1

    status=0
    rcon-cli save-all flush || status=$?
    if [ "$status" -eq 0 ]; then
        backup || status=$?
    fi

    rcon-cli save-on || true
    return "$status"
}

retain() {
    cycles=$((cycles + 1))
    if [ $((cycles % 12)) -eq 0 ]; then
        restic forget --host world --keep-last "$KEEP_SNAPSHOTS" --prune
    else
        restic forget --host world --keep-last "$KEEP_SNAPSHOTS"
    fi
}

final() {
    backup --tag final
    exit $?
}

trap final TERM INT

while :; do
    sleep "$SNAPSHOT_INTERVAL_SECONDS" &
    wait $!

    snapshot && retain
done
