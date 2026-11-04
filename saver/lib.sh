restic() {
    command restic -o s3.bucket-lookup=path "$@"
}

snapshot_id() {
    printf '%s\n' "$1" | sed -n 's/.*"snapshot_id":"\([0-9a-f]*\)".*/\1/p' | tail -n 1
}

discard_pending() {
    stale=$(restic snapshots --host world --tag pending --json | grep -o '"id":"[0-9a-f]*"' | sed 's/"id":"\([0-9a-f]*\)"/\1/')

    if [ -n "$stale" ]; then
        restic forget $stale
    fi
}

promote() {
    restic tag --add ok --remove pending "$1" || return 1

    leftover=$(restic snapshots --host world --tag pending --json) || return 1
    case "$leftover" in
        *"$1"*) return 1 ;;
    esac
}

backup() {
    summary=$(restic backup /data --host world --tag pending --json --quiet "$@")
    backup_status=$?
    backup_id=$(snapshot_id "$summary")

    if [ "$backup_status" -ne 0 ]; then
        [ -z "$backup_id" ] || restic forget "$backup_id"
        return "$backup_status"
    fi

    promote "$backup_id"
}
