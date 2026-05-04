#!/usr/bin/env bash
# Run all four apps in parallel with prefixed, color-coded logs.
# Fallback for `make dev` when overmind/honcho aren't installed.
# ^C (or SIGTERM) stops all four cleanly via per-pgid signal.

set -uo pipefail
set -m  # job control: each backgrounded pipeline gets its own pgid

cd "$(dirname "$0")/.."

apps=(api-svc athlete-svc indexer web)
colors=(36 35 33 32)  # cyan magenta yellow green
pgids=()

cleanup() {
  trap - INT TERM EXIT
  printf '\n\033[36m→\033[0m stopping stack...\n'
  for pgid in "${pgids[@]}"; do
    kill -TERM -- "-$pgid" 2>/dev/null || true
  done
  sleep 1
  for pgid in "${pgids[@]}"; do
    kill -KILL -- "-$pgid" 2>/dev/null || true
  done
  # Some dev servers (Django StatReloader, Vite) reparent their workers into
  # a fresh pgid, so the kills above miss them. Sweep the listen ports too.
  for port in 8000 8001 8002 8003; do
    if command -v fuser >/dev/null 2>&1; then
      fuser -k -KILL "$port/tcp" 2>/dev/null || true
    elif command -v lsof >/dev/null 2>&1; then
      lsof -ti ":$port" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
    fi
  done
  wait 2>/dev/null || true
  exit 0
}
trap cleanup INT TERM EXIT

i=0
for app in "${apps[@]}"; do
  color="${colors[$i]}"
  prefix=$(printf '\033[%sm[%-11s]\033[0m' "$color" "$app")
  (
    make -C "apps/$app" run 2>&1 \
      | awk -v p="$prefix" '{ print p, $0; fflush() }'
  ) &
  pgids+=($!)
  i=$((i+1))
done

printf '\033[36m→\033[0m stack running:\n'
printf '    api-svc      :8001\n'
printf '    athlete-svc  :8002\n'
printf '    indexer      :8003\n'
printf '    web          :8000\n'
printf '  ^C to stop all four\n\n'

wait
