#!/usr/bin/env bash
# Posts a handful of camps to api-svc so the web's search page has results
# to render. Each POST publishes a Camp.V1 event, which the indexer (when
# running) consumes and writes to OpenSearch.
#
# Prerequisites:
#   make up                # infra
#   make register-schemas  # Camp.V1 + AthleteCreated.V1 in SR
#   make api-svc           # api-svc on :8001 (separate terminal)
#   make indexer           # indexer + /search on :8003 (separate terminal)
#                          # — only needed if you want results to show up
#                          #   in the web; api-svc creates rows regardless.
set -euo pipefail

API="${API_URL:-http://localhost:8001}"

# Wait briefly for api-svc to come up.
for _ in $(seq 1 15); do
  if curl -fsS -o /dev/null "${API}/camps?limit=1" 2>/dev/null; then break; fi
  sleep 1
done

post() {
  local body="$1"
  local name
  name=$(echo "$body" | jq -r .name)
  local code
  code=$(curl -s -o /tmp/seed_resp -w "%{http_code}" \
    -H "Content-Type: application/json" \
    -X POST "${API}/camps" -d "$body")
  if [[ "$code" =~ ^2 ]]; then
    printf "  \033[32m✓\033[0m %s\n" "$name"
  else
    printf "  \033[31m✗\033[0m %s — HTTP %s: %s\n" "$name" "$code" "$(cat /tmp/seed_resp)"
  fi
}

post '{"name":"Summer Football Camp","sport":"football","location":"Bradenton, FL","capacity":60,"start_date":"2026-06-15","end_date":"2026-06-22"}'
post '{"name":"Spring Tennis Academy","sport":"tennis","location":"Bradenton, FL","capacity":24,"start_date":"2026-04-10","end_date":"2026-04-14"}'
post '{"name":"Elite Basketball Skills","sport":"basketball","location":"Bradenton, FL","capacity":40,"start_date":"2026-07-08","end_date":"2026-07-12"}'
post '{"name":"Youth Soccer Showcase","sport":"soccer","location":"Tampa, FL","capacity":80,"start_date":"2026-05-20","end_date":"2026-05-24"}'
post '{"name":"Beach Volleyball Intensive","sport":"volleyball","location":"St. Petersburg, FL","capacity":32,"start_date":"2026-08-01","end_date":"2026-08-05"}'
post '{"name":"Pre-Season Football Combine","sport":"football","location":"Orlando, FL","capacity":120,"start_date":"2026-08-20","end_date":"2026-08-22"}'

printf "\n\033[36m→\033[0m If \`make indexer\` is running, give it ~2s for messages to be consumed,\n  then visit \033[4mhttp://localhost:8000\033[0m to see them in the web search.\n"
