#!/usr/bin/env bash
# For every schema in schemas/schemas.yaml, ask Schema Registry whether the
# *current file on disk* would be a compatible next version against what is
# already registered. Prints a clean PASS/FAIL summary.
#
# Run this after editing any *.avsc file. Used by Ch2.
set -euo pipefail

SR="${SCHEMA_REGISTRY_URL:-http://localhost:18081}"
SCHEMAS_DIR="$(cd "$(dirname "$0")/.." && pwd)/schemas"

cd "$(dirname "$0")/.."
python3 - <<'PY' > /tmp/schemas.json
import json, pathlib
text = pathlib.Path("schemas/schemas.yaml").read_text()
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
        cur = {}; line = line[4:]
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

PASS=0; FAIL=0
for row in $(jq -c '.[]' /tmp/schemas.json); do
  subject=$(echo "$row" | jq -r .subject)
  file=$(echo "$row" | jq -r .file)
  path="${SCHEMAS_DIR}/${file}"
  if [ ! -f "$path" ]; then
    printf "  ⚠ %-40s file missing: %s\n" "$subject" "$file"; continue
  fi
  # If subject doesn't exist yet, treat as a fresh registration (always compatible).
  if ! curl -fsS "${SR}/subjects/${subject}/versions" >/dev/null 2>&1; then
    printf "  • %-40s (new subject — would be initial version)\n" "$subject"
    PASS=$((PASS+1)); continue
  fi
  body=$(jq -Rs '{schemaType: "AVRO", schema: .}' < "$path")
  resp=$(curl -fsS -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    --data "$body" "${SR}/compatibility/subjects/${subject}/versions/latest?verbose=true")
  is_compat=$(echo "$resp" | jq -r .is_compatible)
  if [ "$is_compat" = "true" ]; then
    printf "  ✓ %-40s compatible\n" "$subject"; PASS=$((PASS+1))
  else
    msgs=$(echo "$resp" | jq -r '.messages // [] | join("; ")')
    if [ -z "$msgs" ] || [ "$msgs" = "null" ]; then msgs="(see Schema Registry logs for detail)"; fi
    printf "  ✗ %-40s INCOMPATIBLE\n      %s\n" "$subject" "$msgs"
    FAIL=$((FAIL+1))
  fi
done
echo
if [ "$FAIL" -eq 0 ]; then
  printf "\033[32m✓\033[0m All %d schema(s) are backward-compatible.\n" "$PASS"
else
  printf "\033[31m✗\033[0m %d incompatible change(s); %d compatible.\n" "$FAIL" "$PASS"
  exit 1
fi
