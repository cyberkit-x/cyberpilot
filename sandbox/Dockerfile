# syntax=docker/dockerfile:1.7
FROM alpine:3.22.1@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1

RUN apk add --no-cache ca-certificates curl python3 tini \
    && addgroup -g 65532 cyberpilot \
    && adduser -D -H -u 65532 -G cyberpilot cyberpilot \
    && mkdir -p /workspace \
    && chown 65532:65532 /workspace

COPY --chmod=0555 helper.sh /usr/local/bin/cyberpilot-runner

USER 65532:65532
WORKDIR /workspace
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/cyberpilot-runner"]
CMD ["hold"]
