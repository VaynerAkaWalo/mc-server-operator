restic() {
    command restic -o s3.bucket-lookup=path "$@"
}

backup() {
    restic backup /data --host world --tag world "$@"
}
