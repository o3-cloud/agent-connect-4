# syntax=docker/dockerfile:1
FROM golang:1.27.4-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY internal ./internal
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/arena ./cmd/arena \
 && CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/connect4 ./cmd/connect4

FROM scratch
WORKDIR /app
COPY --from=builder /out/arena /out/connect4 /app/
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=3s --retries=3 CMD ["/app/arena", "-healthcheck"]
ENTRYPOINT ["/app/arena"]
