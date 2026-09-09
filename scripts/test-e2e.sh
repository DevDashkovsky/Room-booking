#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
task_tmp=$(mktemp -d "${TMPDIR:-/tmp}/room-booking-e2e.XXXXXX")
task_suffix=$(basename "$task_tmp" | tr '[:upper:].' '[:lower:]-')
task_project="${task_suffix}"
export API_PORT=0

if docker compose version > /dev/null 2>&1; then
    task_compose='docker compose'
else
    task_compose='docker-compose'
fi

compose() {
    $task_compose -p "$task_project" -f docker-compose.yaml "$@"
}

cleanup() {
    task_status=$?
    trap - EXIT
    if [ "$task_status" -ne 0 ]; then
        compose logs --no-color --tail=100 || true
    fi
    if ! compose down -v --remove-orphans; then
        task_status=1
    fi
    rmdir "$task_tmp"
    exit "$task_status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

compose up --build -d
task_address=$(compose port api 8080)
BASE_URL="http://$task_address"
export BASE_URL
task_attempt=0
until curl --fail --silent --connect-timeout 1 --max-time 2 "$BASE_URL/ready" > /dev/null; do
    task_attempt=$((task_attempt + 1))
    if [ "$task_attempt" -ge 60 ]; then
        echo 'API did not become ready after 60 attempts' >&2
        exit 1
    fi
    sleep 1
done

go test -tags=e2e -race -count=1 -timeout=2m -v .
