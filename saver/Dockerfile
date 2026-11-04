FROM restic/restic:0.19.1 AS restic

FROM alpine:3.22 AS rcon
ARG TARGETARCH
ARG RCON_CLI_VERSION=1.7.9
ARG RCON_CLI_SHA256_AMD64=f2a10716ce42b6433ab2a870aad8d14f1d23a2dd261928f7ac4a92520cafc51c
ARG RCON_CLI_SHA256_ARM64=3c6c67c5401eed14df491815b10bf09e8b1d65fadef46e78be0ed6a0c7deaf3f
RUN apk add --no-cache curl tar \
    && case "$TARGETARCH" in \
        amd64) sha="$RCON_CLI_SHA256_AMD64" ;; \
        arm64) sha="$RCON_CLI_SHA256_ARM64" ;; \
    esac \
    && curl -fsSLo /tmp/rcon-cli.tar.gz "https://github.com/itzg/rcon-cli/releases/download/${RCON_CLI_VERSION}/rcon-cli_${RCON_CLI_VERSION}_linux_${TARGETARCH}.tar.gz" \
    && echo "$sha  /tmp/rcon-cli.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/rcon-cli.tar.gz -C /usr/local/bin rcon-cli

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=restic /usr/bin/restic /usr/local/bin/restic
COPY --from=rcon /usr/local/bin/rcon-cli /usr/local/bin/rcon-cli
COPY lib.sh /usr/local/lib/saver/lib.sh
COPY restore.sh save.sh /usr/local/bin/
USER 1000:1000
