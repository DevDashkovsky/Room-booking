#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
task_results=$(mktemp -d "${TMPDIR:-/tmp}/room-booking-load.XXXXXX")
task_project=$(basename "$task_results" | tr '[:upper:].' '[:lower:]-')
export API_PORT=0

if docker compose version > /dev/null 2>&1; then
    task_compose='docker compose'
else
    task_compose='docker-compose'
fi

compose() {
    $task_compose -p "$task_project" -f docker-compose.yaml -f docker-compose.load.yaml "$@"
}

cleanup() {
    task_status=$?
    trap - EXIT
    compose logs --no-color api postgres > "$task_results/services.log" 2>&1 || true
    if ! compose down -v --remove-orphans; then
        task_status=1
    fi
    printf '\nLoad test artifacts: %s\n' "$task_results"
    exit "$task_status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

{
    date -u '+UTC: %Y-%m-%dT%H:%M:%SZ'
    git rev-parse HEAD
    git status --short
    docker version
    docker info --format 'CPUs={{.NCPU}} Memory={{.MemTotal}} OS={{.OperatingSystem}} Arch={{.Architecture}}'
    printf 'LOAD_RATE=%s LOAD_SECONDS=%s LOAD_CONFLICT_SECONDS=%s DB_MAX_CONNS=%s\n' \
        "${LOAD_RATE:-100}" "${LOAD_SECONDS:-60}" "${LOAD_CONFLICT_SECONDS:-20}" "${DB_MAX_CONNS:-20}"
} > "$task_results/environment.txt"
# Preserve the actual test inputs even for a run with uncommitted changes.
cp scripts/load-test.js docker-compose.yaml docker-compose.load.yaml "$task_results/"
compose up --build -d api
task_address=$(compose port api 8080)
task_attempt=0
until curl --fail --silent --connect-timeout 1 --max-time 2 "http://$task_address/ready" > /dev/null; do
    task_attempt=$((task_attempt + 1))
    if [ "$task_attempt" -ge 60 ]; then
        echo 'API did not become ready after 60 attempts' >&2
        exit 1
    fi
    sleep 1
done
compose images > "$task_results/images.txt"
task_status=0
# Keep the exited container until its summary has been copied, including on failure.
compose run --name "$task_project-k6" -T --no-deps k6 > "$task_results/k6.log" 2>&1 || task_status=$?
cat "$task_results/k6.log"
if ! docker cp "$task_project-k6:/tmp/summary.json" "$task_results/summary.json"; then
    echo 'k6 did not produce a summary' >&2
    task_status=1
fi
exit "$task_status"
