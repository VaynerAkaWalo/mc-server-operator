restic() {
    command restic -o s3.bucket-lookup=path "$@"
}

snapshot_id() {
    printf '%s\n' "$1" | sed -n 's/.*"snapshot_id":"\([0-9a-f]*\)".*/\1/p' | tail -n 1
}

backup() {
    summary=$(restic backup /data --host world --tag pending --json --quiet "$@")
    status=$?
    id=$(snapshot_id "$summary")

    if [ "$status" -ne 0 ]; then
        [ -z "$id" ] || restic forget "$id"
        return "$status"
    fi

    restic tag --add ok --remove pending "$id"
}
