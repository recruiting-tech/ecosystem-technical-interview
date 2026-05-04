#!/usr/bin/env bash
# Registers every Avro schema in schemas/ with the local Schema Registry.
# Subject naming: <Topic>-value (Confluent's TopicNameStrategy default).
# Compatibility policy is read from schemas/schemas.yaml.
set -euo pipefail

SR="${SCHEMA_REGISTRY_URL:-http://localhost:18081}"
SCHEMAS_DIR="$(cd "$(dirname "$0")/.." && pwd)/schemas"
YAML="${SCHEMAS_DIR}/schemas.yaml"

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required" >&2
  exit 1
fi

# Wait for SR
echo -n "Waiting for Schema Registry at ${SR} "
for _ in $(seq 1 30); do
  if curl -fsS "${SR}/subjects" >/dev/null 2>&1; then echo "✓"; break; fi
  echo -n "."; sleep 1
done

# Parse schemas.yaml — minimal, no yq dependency.
# Each schema entry: subject, file, compatibility.
python3 - <<'PY' > /tmp/schemas.json
import sys, json, re, pathlib
text = pathlib.Path("schemas/schemas.yaml").read_text()
# Very small YAML parser for the subset we use.
items, cur = [], None
in_schemas = False
for line in text.splitlines():
    if line.startswith("schemas:"):
        in_schemas = True; continue
    if in_schemas and line.startswith(("topics:", "base_schemas:")):
        in_schemas = False
    if not in_schemas:
        continue
    if line.startswith("  - "):
        if cur: items.append(cur)
        cur = {}
        line = line[4:]
    elif line.startswith("    "):
        line = line[4:]
    else:
        continue
    if ":" in line:
        k, v = line.split(":", 1)
        cur[k.strip()] = v.strip().strip('"')
if cur: items.append(cur)
print(json.dumps(items))
PY

cd "$(dirname "$0")/.."
COUNT=0
FAILED=0
for row in $(jq -c '.[]' /tmp/schemas.json); do
  subject=$(echo "$row" | jq -r .subject)
  file=$(echo "$row" | jq -r .file)
  compat=$(echo "$row" | jq -r '.compatibility // "BACKWARD"')
  path="${SCHEMAS_DIR}/${file}"
  if [ ! -f "$path" ]; then
    echo "✗ ${subject}: file not found at ${path}"
    FAILED=$((FAILED+1)); continue
  fi
  # Set compatibility on the subject (idempotent).
  curl -fsS -X PUT \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    --data "{\"compatibility\": \"${compat}\"}" \
    "${SR}/config/${subject}" > /dev/null

  body=$(jq -Rs --argjson schema_type '"AVRO"' \
    '{schemaType: "AVRO", schema: .}' < "$path")
  http_code=$(curl -s -o /tmp/sr_resp -w "%{http_code}" \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    --data "$body" \
    "${SR}/subjects/${subject}/versions")
  if [[ "$http_code" =~ ^2 ]]; then
    id=$(jq -r .id /tmp/sr_resp)
    printf "  ✓ %-40s compat=%-8s id=%s\n" "$subject" "$compat" "$id"
    COUNT=$((COUNT+1))
  else
    printf "  ✗ %-40s HTTP %s: %s\n" "$subject" "$http_code" "$(cat /tmp/sr_resp)"
    FAILED=$((FAILED+1))
  fi
done
echo
echo "Registered ${COUNT} schemas; ${FAILED} failures."
exit $((FAILED > 0))
