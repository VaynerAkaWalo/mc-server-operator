restic() {
    command restic -o s3.bucket-lookup=path "$@"
}

discard() {
    id=$(printf '%s\n' "$1" | sed -n 's/.*"snapshot_id":"\([0-9a-f]*\)".*/\1/p' | tail -n 1)
    if [ -n "$id" ]; then
        restic forget "$id"
    fi
}

backup() {
    summary=$(restic backup /data --host world --tag world --json --quiet "$@")
    status=$?

    if [ "$status" -ne 0 ]; then
        discard "$summary"
    fi

    return "$status"
}
