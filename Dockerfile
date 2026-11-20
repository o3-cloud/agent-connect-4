# syntax=docker/dockerfile:1
# The builder is Debian-based because the pinned Tailwind CLI (ADR-012) is
# linked against glibc. The runtime stage stays scratch (delivery/docker.md:4).
FROM golang:1.27.4-trixie@sha256:f18c3d73a184fb0582828510722e723b63a93f7188d11dd223d9be6f6880e65d AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
# The Makefile holds the Tailwind version and checksums; `make tailwind`
# downloads and verifies the CLI for the build platform.
COPY Makefile ./
RUN make tailwind
COPY internal ./internal
COPY cmd ./cmd
# `make build` runs `make css` first, so app.css is embedded in the binary.
RUN make build
# An empty directory for the journal (ADR-015). A named volume mounted on
# /data takes its owner, so the non-root user can write there.
RUN mkdir -p /out/data

FROM scratch
WORKDIR /app
COPY --from=builder /src/bin/arena /src/bin/connect4 /app/
COPY --from=builder --chown=65532:65532 /out/data /data
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=3s --retries=3 CMD ["/app/arena", "-healthcheck"]
ENTRYPOINT ["/app/arena"]
