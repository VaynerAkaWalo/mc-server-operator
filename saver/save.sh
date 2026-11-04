#!/bin/sh
set -u

. /usr/local/lib/saver/lib.sh

cycles=0

saving_disabled() {
    case "$1" in
        *"saving is now disabled"*|*"already turned off"*) return 0 ;;
    esac

    return 1
}

flushed() {
    case "$1" in
        *"Saved the game"*) return 0 ;;
    esac

    return 1
}

snapshot() {
    status=1

    if reply=$(rcon-cli save-off) && saving_disabled "$reply"; then
        if reply=$(rcon-cli save-all flush) && flushed "$reply"; then
            backup && status=0
        fi
    fi

    rcon-cli save-on >/dev/null
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

    if snapshot; then
        retain
    else
        echo "snapshot skipped" >&2
    fi
done
