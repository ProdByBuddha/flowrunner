#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
USER="${USER_NAME:-demo}"
PASS="${USER_PASS:-demo}"

LIST_FLOW_FILE="${LIST_FLOW_FILE:-mcp_list_http.json}"
EXEC_FLOW_FILE="${EXEC_FLOW_FILE:-mcp_execute_http.json}"

echo "[i] Using BASE_URL=$BASE_URL"

echo "[1] Login"
TOKEN=$(curl -sS -X POST "$BASE_URL/api/v1/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" | jq -r .token)
if [[ -z "${TOKEN:-}" || "$TOKEN" == "null" ]]; then
  echo "[!] Failed to acquire token" >&2
  exit 1
fi

post_json_with_status() {
  # stdin: json body
  local out code body
  out=$(curl -sS -X POST "$1" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d @- -w "\n%{http_code}")
  code=$(printf "%s" "$out" | tail -n1)
  body=$(printf "%s" "$out" | sed '$d')
  if [[ ! "$code" =~ ^2[0-9][0-9]$ ]]; then
    echo "[!] Request failed ($code). Body:" >&2
    printf "%s\n" "$body" >&2
    exit 1
  fi
  printf "%s" "$body"
}

create_flow_from_file() {
  local name="$1"; local file="$2"
  jq -Rs --arg name "$name" '{name:$name, content:.}' < "$file" | post_json_with_status "$BASE_URL/api/v1/flows" | jq -r .id
}

echo "[2] Create list flow (HTTP-only)"
FLOW1_ID=$(create_flow_from_file "MCP: List Tools (HTTP)" "$LIST_FLOW_FILE")
echo "[i] Flow1 ID: $FLOW1_ID"

echo "[3] Run list flow"
EXEC1=$(printf '%s' '{"input":{}}' | post_json_with_status "$BASE_URL/api/v1/flows/$FLOW1_ID/run" | jq -r .execution_id)
echo "[i] Exec1: $EXEC1"

echo "[logs:list]"
# Poll briefly for logs to accumulate
for i in {1..10}; do
  curl -sS -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/executions/$EXEC1/logs" | jq . | sed -n '1,200p'
  STATUS=$(curl -sS -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/executions/$EXEC1" | jq -r .status)
  [[ "$STATUS" == "completed" || "$STATUS" == "failed" ]] && break
  sleep 1
done

if [[ -f "$EXEC_FLOW_FILE" ]]; then
  echo "[4] Create execute flow (HTTP-only)"
  FLOW2_ID=$(create_flow_from_file "MCP: Execute Tool (HTTP)" "$EXEC_FLOW_FILE")
  echo "[i] Flow2 ID: $FLOW2_ID"
fi

echo "[done]"

# Optional Agent->Router->MCP demo
if [[ "${RUN_AGENT_MCP:-false}" == "true" ]]; then
  echo "[agent] Create Agent->Router->MCP flow"
  AID=$(jq -Rs --arg name "Agent->Router->MCP (List)" '{name:$name, content:.}' < demos/agent_mcp_list.yaml | post_json_with_status "$BASE_URL/api/v1/flows" | jq -r .id)
  echo "[agent] Flow ID: $AID"
  echo "[agent] Run"
  AEXEC=$(printf '%s' '{"input":{}}' | post_json_with_status "$BASE_URL/api/v1/flows/$AID/run" | jq -r .execution_id)
  echo "[agent] Exec: $AEXEC"
  echo "[agent logs]"
  curl -sS -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/executions/$AEXEC/logs" | jq . | sed -n '1,200p'
fi
