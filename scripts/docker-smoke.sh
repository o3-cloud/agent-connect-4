#!/bin/sh
# Smoke-test the built container image (delivery/docker.md, ADR-012).
# Starts the image the way compose.yaml does (read-only, no capabilities),
# waits for /healthz and checks that the browser UI's stylesheet is served.
# Then it starts the image again with a different PORT and checks that the
# server and the image's health check follow it, with no rebuild, and that an
# invalid PORT stops the container with exit 2, and that games survive a
# restart with the journal on a volume (BDR-018, BDR-033).
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

docker stop "$container" >/dev/null
trap - EXIT

# An invalid setting stops the container at startup, naming the variable.
for bad in abc 70000; do
	status=0
	out=$(docker run --rm --env "PORT=$bad" "$image" 2>&1) || status=$?
	case $status:$out in
	2:*"PORT must be a port number"*) ;;
	*)
		echo "docker-smoke: with PORT=$bad, want exit 2 naming PORT, got exit $status: $out" >&2
		exit 1
		;;
	esac
done

# BDR-018, BDR-033 scenario 1: with the journal on a volume, a game and an
# API key survive a container restart.
volume="connect4arena-smoke-$$"
docker volume create "$volume" >/dev/null
start_journaled() {
	container=$(docker run --detach --rm --read-only --cap-drop ALL \
		--env ARENA_JOURNAL_PATH=/data/arena.jsonl --volume "$volume:/data" \
		--publish "127.0.0.1:$port:8080" "$image")
	if ! curl -fs --retry 30 --retry-all-errors --retry-delay 1 -o /dev/null "$base/healthz"; then
		echo "docker-smoke: $image with a journal did not start" >&2
		docker logs "$container" >&2
		exit 1
	fi
}
trap 'docker stop "$container" >/dev/null 2>&1; docker volume rm "$volume" >/dev/null' EXIT
start_journaled
key=$(curl -fsS -X POST -H 'Content-Type: application/json' \
	-d '{"displayName":"Smoke","type":"agent"}' "$base/api/v1/agents" | sed -n 's/.*"apiKey":"\([^"]*\)".*/\1/p')
game=$(curl -fsS -X POST -H "Authorization: Bearer $key" -H 'Content-Type: application/json' \
	-d '{"mode":"agent_vs_agent"}' "$base/api/v1/games" | sed -n 's/.*"gameId":"\([^"]*\)".*/\1/p')
if [ -z "$key" ] || [ -z "$game" ]; then
	echo "docker-smoke: could not register an agent and create a game" >&2
	exit 1
fi
docker stop "$container" >/dev/null
start_journaled
if ! curl -fs -o /dev/null "$base/api/v1/games/$game"; then
	echo "docker-smoke: game $game did not survive a container restart" >&2
	docker logs "$container" >&2
	exit 1
fi
if ! curl -fs -o /dev/null -X POST -H "Authorization: Bearer $key" -H 'Content-Type: application/json' \
	-d '{"mode":"agent_vs_agent"}' "$base/api/v1/games"; then
	echo "docker-smoke: the API key did not survive a container restart" >&2
	exit 1
fi

echo "docker-smoke: $image serves /healthz and the stylesheet, follows PORT, rejects a bad PORT and keeps games on a volume."
