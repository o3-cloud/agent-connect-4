#!/bin/sh
# Smoke-test the built container image (delivery/docker.md, ADR-012).
# Starts the image the way compose.yaml does (read-only, no capabilities),
# waits for /healthz and checks that the browser UI's stylesheet is served.
# Then it starts the image again with a different PORT and checks that the
# server and the image's health check follow it, with no rebuild (BDR-033).
set -eu

image=${1:-connect4arena:local}
port=${SMOKE_PORT:-18080}
base="http://127.0.0.1:$port"

container=$(docker run --detach --rm --read-only --cap-drop ALL \
	--publish "127.0.0.1:$port:8080" "$image")
trap 'docker stop "$container" >/dev/null' EXIT

if ! curl -fs --retry 30 --retry-all-errors --retry-delay 1 -o /dev/null "$base/healthz"; then
	echo "docker-smoke: $image did not answer $base/healthz" >&2
	docker logs "$container" >&2
	exit 1
fi

css=$(curl -fsS "$base/static/app.css") || {
	echo "docker-smoke: $image does not serve /static/app.css (run make css in the image build)" >&2
	exit 1
}
for rule in .board .cell; do
	case $css in
	*"$rule"*) ;;
	*)
		echo "docker-smoke: /static/app.css lacks the $rule styles" >&2
		exit 1
		;;
	esac
done

docker stop "$container" >/dev/null
trap - EXIT

# BDR-033 AC-2: configuration from outside the image changes the listen port.
container=$(docker run --detach --rm --read-only --cap-drop ALL \
	--env PORT=9000 --publish "127.0.0.1:$port:9000" "$image")
trap 'docker stop "$container" >/dev/null' EXIT

if ! curl -fs --retry 30 --retry-all-errors --retry-delay 1 -o /dev/null "$base/api/v1/games"; then
	echo "docker-smoke: with PORT=9000, $image did not answer GET /api/v1/games on port 9000" >&2
	docker logs "$container" >&2
	exit 1
fi
if ! docker exec "$container" /app/arena -healthcheck; then
	echo "docker-smoke: with PORT=9000, the image's health check fails" >&2
	exit 1
fi

echo "docker-smoke: $image serves /healthz and the stylesheet, and follows PORT."
