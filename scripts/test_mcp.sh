#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
USER="${USER_NAME:-demo}"
PASS="${USER_PASS:-demo}"
TOOL_NAME="${TOOL_NAME:-}"
TOOL_PARAMS_JSON="${TOOL_PARAMS_JSON:-{}}"

# Transport selection
MCP_TRANSPORT="${MCP_TRANSPORT:-sse}" # sse|cmd|http

# SSE/HTTP endpoints (used when MCP_TRANSPORT=sse)
MCP_SSE_URL="${MCP_SSE_URL:-http://127.0.0.1:3006/mcp-sse}"
MCP_SSE_POST="${MCP_SSE_POST:-}"
MCP_HEADERS_JSON="${MCP_HEADERS_JSON:-}" # optional JSON object of headers
# Optional raw JSON bodies to post to messages endpoint (SSE/HTTP)
# Defaults tailored for openapi-mcp-server
MCP_RAW_BODY_LIST_JSON="${MCP_RAW_BODY_LIST_JSON:-{"method":"tools/list","params":{}}}"
MCP_RAW_BODY_EXEC_JSON="${MCP_RAW_BODY_EXEC_JSON:-}" # e.g., {"method":"tools/call","params":{"name":"tool","arguments":{}}}

# Dotenvx injection controls (applied to the YAML before upload when MCP_TRANSPORT=cmd)
DOTENVX="${DOTENVX:-true}"
DOTENVX_COMMAND="${DOTENVX_COMMAND:-npx}"
DOTENVX_USE_NPX="${DOTENVX_USE_NPX:-true}"
DOTENVX_PACKAGE="${DOTENVX_PACKAGE:-dotenvx@latest}"
# Comma-separated list -> YAML array
DOTENVX_FILES_CSV="${DOTENVX_FILES_CSV:-.env,.env.local,.env.mcp}"
# Optional extra args array for dotenvx (space-separated items become YAML list)
DOTENVX_ARGS="${DOTENVX_ARGS:-}"

inject_dotenvx() {
  # $1: path to YAML
  local f="$1"
  # If dotenvx already present, return file as-is
  if grep -qE '^\s*dotenvx:' "$f"; then
    cat "$f"
    return 0
  fi
  # Build YAML block (6-space indent)
  local files_block=""
  IFS=',' read -r -a files_arr <<< "$DOTENVX_FILES_CSV"
  for it in "${files_arr[@]}"; do
    it_trimmed="${it# }"; it_trimmed="${it_trimmed% }"
    [[ -n "$it_trimmed" ]] && files_block+=$'      - "'"$it_trimmed"$'"\n'
  done
  local args_block=""
  if [[ -n "$DOTENVX_ARGS" ]]; then
    # Build JSON-like inline array for YAML
    args_block+=$'      dotenvxArgs: ['
    local first=1
    for a in $DOTENVX_ARGS; do
      if [[ $first -eq 0 ]]; then args_block+=', '; fi
      args_block+=$'"'"$a"$'"'
      first=0
    done
    args_block+=']\n'
  fi
  awk -v dx="$DOTENVX" -v cmd="$DOTENVX_COMMAND" -v use_npx="$DOTENVX_USE_NPX" -v pkg="$DOTENVX_PACKAGE" \
      -v files_block="$files_block" -v args_block="$args_block" '
    BEGIN{ injected=0 }
    { print }
    $0 ~ /^[[:space:]]+params:[[:space:]]*$/ && injected==0 {
      injected=1
      print "      dotenvx: " dx
      print "      dotenvxCommand: \"" cmd "\""
      print "      dotenvxUseNpx: " use_npx
      print "      dotenvxPackage: \"" pkg "\""
      if (length(files_block) > 0) {
        print "      dotenvxFiles:"
        printf "%s", files_block
      }
      if (length(args_block) > 0) {
        printf "%s", args_block
      }
    }
  ' "$f"
}

# Inject MCP server child env (e.g., override PORT to avoid conflicts)
# Adds under params: env: |\n  KEY=VAL\n  KEY2=VAL2
inject_env() {
  local f="$1"
  local env_lines="$2"   # newline-separated KEY=VAL
  if [[ -z "$env_lines" ]]; then
    cat "$f"; return 0
  fi
  # If env already present, do not duplicate; append after params: instead
  awk -v env_lines="$env_lines" '
    BEGIN{ injected=0 }
    { print }
    $0 ~ /^[[:space:]]+params:[[:space:]]*$/ && injected==0 {
      injected=1
      print "      env: |"
      n=split(env_lines, arr, "\n");
      for(i=1;i<=n;i++){ if(length(arr[i])>0) print "        " arr[i] }
    }
  ' "$f"
}

# Inject argsExtra list (e.g., ["--port","<port>"]) under params
inject_args_extra() {
  local f="$1"
  shift
  local arr=("$@")
  if [[ ${#arr[@]} -eq 0 ]]; then
    cat "$f"; return 0
  fi
  awk -v n=${#arr[@]} \
      $(for i in $(seq 1 99); do echo -n "-v a$i=\"\${arr[$((i-1))]}\" "; done) '
    BEGIN{ injected=0 }
    { print }
    $0 ~ /^[[:space:]]+params:[[:space:]]*$/ && injected==0 {
      injected=1
      print "      argsExtra:"
      for(i=1;i<=n;i++) if(length($("a"i))>0) printf("        - \"%s\"\n", $("a"i));
    }
  ' "$f"
}

# Inject SSE transport params (connectionType/url/messagesPostEndpoint/headers)
inject_sse_params() {
  local f="$1"
  local sse_url="$2"
  local sse_post="$3"
  local headers_json="$4"
  local raw_body_json="$5" # pass body per-flow
  local headers_block=""
  if [[ -n "$headers_json" ]]; then
    headers_block+=$'      headers:\n'
    python3 - "$headers_json" << 'PY'
import json,sys
obj=json.loads(sys.argv[1]) if len(sys.argv)>1 and sys.argv[1] else {}
for k,v in obj.items():
  s = str(v).replace('"','\\"')
  print(f'        {k}: "{s}"')
PY
  fi
  awk -v sse_url="$sse_url" -v sse_post="$sse_post" -v headers_block="$headers_block" '
    BEGIN{ injected=0 }
    { print }
    $0 ~ /^[[:space:]]+params:[[:space:]]*$/ && injected==0 {
      injected=1
      print "      connectionType: \"sse\""
      print "      url: \"" sse_url "\""
      print "      messagesPostEndpoint: \"" sse_post "\""
      if (length(headers_block) > 0) {
        printf "%s\n", headers_block
      }
      if (length(raw_body_json) > 0) {
        print "      rawBody: '" raw_body_json "'"
      }
    }
  ' "$f"
}

echo "[i] Using BASE_URL=$BASE_URL"

echo "[1] Health check"
curl -fsSL "$BASE_URL/health" | sed -n '1,200p' || true

echo "[2] Create account (idempotent)"
curl -fsSL -X POST "$BASE_URL/api/v1/accounts" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" || true
echo

echo "[3] Login"
TOKEN=$(curl -fsSL -X POST "$BASE_URL/api/v1/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" | jq -r .token)
if [[ -z "${TOKEN:-}" || "$TOKEN" == "null" ]]; then
  echo "[!] Failed to acquire token" >&2
  exit 1
fi
echo "[i] Token acquired (${#TOKEN} bytes)"

echo "[4] Create flows from demos (transport=$MCP_TRANSPORT)"

# Pick a free TCP port for the child MCP server unless explicitly provided
get_free_port() {
  if command -v python3 >/dev/null 2>&1; then
    python3 - << 'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.bind(('127.0.0.1', 0))
port = s.getsockname()[1]
s.close()
print(port)
PY
  else
    # Fallback if python3 not available
    echo 33007
  fi
}

# Auto-launch MCP server (SSE + POST) if requested
if [[ "$MCP_TRANSPORT" == "sse" && "${MCP_AUTO_LAUNCH:-false}" == "true" ]]; then
  MCP_SSE_PORT="${MCP_SSE_PORT:-}"
  MCP_POST_PORT="${MCP_POST_PORT:-}"
  if [[ -z "$MCP_SSE_PORT" ]]; then MCP_SSE_PORT=$(get_free_port); fi
  if [[ -z "$MCP_POST_PORT" ]]; then MCP_POST_PORT=$(get_free_port); fi

  echo "[i] Auto-launching MCP server: SSE=$MCP_SSE_PORT, POST=$MCP_POST_PORT"
  LAUNCH_ENV=""
  IFS=',' read -r -a sse_keys <<< "${MCP_SSE_ENV_KEYS_CSV:-SSE_PORT}"
  for k in "${sse_keys[@]}"; do
    k_trimmed="${k# }"; k_trimmed="${k_trimmed% }"
    [[ -n "$k_trimmed" ]] && LAUNCH_ENV+="$k_trimmed=$MCP_SSE_PORT "
  done
  IFS=',' read -r -a post_keys <<< "${MCP_POST_ENV_KEYS_CSV:-POST_PORT}"
  for k in "${post_keys[@]}"; do
    k_trimmed="${k# }"; k_trimmed="${k_trimmed% }"
    [[ -n "$k_trimmed" ]] && LAUNCH_ENV+="$k_trimmed=$MCP_POST_PORT "
  done

  EXPANDED_ARGS="${MCP_SERVER_ARGS//\{SSE_PORT\}/$MCP_SSE_PORT}"
  EXPANDED_ARGS="${EXPANDED_ARGS//\{POST_PORT\}/$MCP_POST_PORT}"

  mkdir -p testlogs
  bash -c "set -e; $LAUNCH_ENV ${MCP_LAUNCH_CMD:-npx} ${MCP_PACKAGE:-@prodbybuddha/openapi-mcp-server@latest} --config '${MCP_CONFIG:-$PWD/services.dynamic.json}' $EXPANDED_ARGS > testlogs/openapi-mcp-server.log 2>&1 & echo \$! > .mcp_server.pid"

  # Override SSE URL based on chosen port
  MCP_SSE_URL="http://127.0.0.1:$MCP_SSE_PORT/mcp-sse"
fi

MCP_PORT="${MCP_PORT:-}"
if [[ -z "$MCP_PORT" ]]; then
  MCP_PORT=$(get_free_port)
fi
echo "[i] Using MCP child PORT=$MCP_PORT"
# Additional KEY=VAL entries as CSV: e.g., "FOO=bar,BAZ=1"
MCP_EXTRA_ENV_CSV="${MCP_EXTRA_ENV_CSV:-}"
MCP_ENV_LINES="PORT=${MCP_PORT}"
if [[ -n "$MCP_EXTRA_ENV_CSV" ]]; then
  IFS=',' read -r -a extra_arr <<< "$MCP_EXTRA_ENV_CSV"
  for kv in "${extra_arr[@]}"; do
    kv_trimmed="${kv# }"; kv_trimmed="${kv_trimmed% }"
    [[ -n "$kv_trimmed" ]] && MCP_ENV_LINES+=$'\n'"$kv_trimmed"
  done
fi

MCP_PORT_FLAG="${MCP_PORT_FLAG:---port}"
MCP_PORT_ENV_KEYS_CSV="${MCP_PORT_ENV_KEYS_CSV:-}"
PORT_ENV_APPEND=""
if [[ -n "$MCP_PORT_ENV_KEYS_CSV" ]]; then
  IFS=',' read -r -a pkeys <<< "$MCP_PORT_ENV_KEYS_CSV"
  for k in "${pkeys[@]}"; do
    k_trimmed="${k# }"; k_trimmed="${k_trimmed% }"
    [[ -n "$k_trimmed" ]] && PORT_ENV_APPEND+=$'\n'"$k_trimmed=$MCP_PORT"
  done
fi
MCP_ENV_LINES="$MCP_ENV_LINES$PORT_ENV_APPEND"

inject_server_port_param() {
  local f="$1"; local port="$2"; local flag="$3"
  awk -v sp="$port" -v pf="$flag" '
    BEGIN{ injected=0 }
    { print }
    $0 ~ /^[[:space:]]+params:[[:space:]]*$/ && injected==0 {
      injected=1
      print "      serverPort: " sp
      print "      portFlag: \"" pf "\""
    }
  ' "$f"
}

discover_post_endpoint() {
  local base_sse="$1"; local headers_json="$2"; local body_json="$3"
  local hostport host port
  hostport=$(echo "$base_sse" | sed -E 's#^https?://([^/]+)/.*#\1#')
  host="${hostport%%:*}"; port="${hostport##*:}"

  # Build candidates
  local candidates=()
  if [[ -n "${MCP_POST_CANDIDATES_JSON:-}" ]]; then
    # JSON array of explicit URLs
    while IFS= read -r line; do
      [[ -n "$line" ]] && candidates+=("$line")
    done < <(python3 - << 'PY'
import json,sys
arr=json.loads(sys.stdin.read())
for u in arr:
  print(u)
PY
<<< "${MCP_POST_CANDIDATES_JSON}")
  else
    # Derive from port offsets and paths
    local offsets_csv="${MCP_POST_PORT_OFFSETS_CSV:--1,0,1}"
    local paths_csv="${MCP_POST_PATHS_CSV:-mcp,messages}"
    local offsets=()
    IFS=',' read -r -a toks <<< "$offsets_csv"
    for t in "${toks[@]}"; do
      t="${t# }"; t="${t% }"
      if [[ "$t" =~ ^-?[0-9]+$ ]]; then
        offsets+=($t)
      elif [[ "$t" =~ ^-?[0-9]+:-?[0-9]+$ ]]; then
        local a="${t%%:*}"; local b="${t##*:}"
        local i
        for ((i=a;i<=b;i++)); do offsets+=($i); done
      fi
    done
    IFS=',' read -r -a paths <<< "$paths_csv"
    local base_port=$port
    for off in "${offsets[@]}"; do
      local p=$((base_port+off))
      for path in "${paths[@]}"; do
        path="${path# /}"; path="${path% /}"
        candidates+=("http://$host:$p/$path")
      done
    done
  fi

  local hdrs=(-H 'Content-Type: application/json')
  if [[ -n "$headers_json" ]]; then
    mapfile -t H < <(python3 - "$headers_json" << 'PY'
import json,sys
obj=json.loads(sys.argv[1]) if len(sys.argv)>1 and sys.argv[1] else {}
for k,v in obj.items():
  print(f"{k}: {v}")
PY
)
    for h in "${H[@]:-}"; do hdrs+=(-H "$h"); done
  fi

  local regex="${MCP_POST_URL_REGEX:-}"
  for url in "${candidates[@]}"; do
    if [[ -n "$regex" ]] && ! printf '%s' "$url" | grep -Eq "$regex"; then
      [[ "${MCP_DEBUG:-}" == "1" ]] && echo "[debug] skip (regex): $url" >&2
      continue
    fi
    [[ "${MCP_DEBUG:-}" == "1" ]] && echo "[debug] try POST $url" >&2
    code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$url" "${hdrs[@]}" -d "$body_json" || true)
    if [[ "$code" =~ ^2[0-9][0-9]$ ]]; then echo "$url"; return 0; fi
  done
  echo ""; return 1
}

if [[ "$MCP_TRANSPORT" == "sse" ]]; then
  # If auto-launched, wait for SSE readiness
  if [[ "${MCP_AUTO_LAUNCH:-false}" == "true" ]]; then
    echo "[i] Waiting for SSE at $MCP_SSE_URL ..."
    for i in {1..30}; do
      code=$(curl -s -o /dev/null -w '%{http_code}' "$MCP_SSE_URL" || true)
      [[ "$code" == "200" ]] && break
      sleep 1
    done
    code=$(curl -s -o /dev/null -w '%{http_code}' "$MCP_SSE_URL" || true)
    if [[ "$code" != "200" ]]; then
      echo "[!] SSE endpoint did not become ready at $MCP_SSE_URL" >&2
      exit 1
    fi
  fi
  if [[ -z "${MCP_SSE_POST:-}" ]]; then
    echo "[i] Discovering messages POST endpoint based on $MCP_SSE_URL ..."
    MCP_SSE_POST=$(discover_post_endpoint "$MCP_SSE_URL" "$MCP_HEADERS_JSON" "$MCP_RAW_BODY_LIST_JSON") || true
    if [[ -z "$MCP_SSE_POST" ]]; then
      echo "[!] Could not discover messages POST endpoint automatically. Set MCP_SSE_POST explicitly." >&2
      exit 1
    fi
    echo "[i] Using messages POST: $MCP_SSE_POST"
  fi
  FLOW1_YAML=$(inject_sse_params demos/mcp_list_tools.yaml "$MCP_SSE_URL" "$MCP_SSE_POST" "$MCP_HEADERS_JSON" "$MCP_RAW_BODY_LIST_JSON")
else
  FLOW1_YAML=$(inject_dotenvx demos/mcp_list_tools.yaml | inject_env /dev/stdin "$MCP_ENV_LINES" | inject_args_extra /dev/stdin "$MCP_PORT_FLAG" "$MCP_PORT" | inject_server_port_param /dev/stdin "$MCP_PORT" "$MCP_PORT_FLAG")
fi
create_flow() {
  # $1: flow name, stdin: raw content string
  local name="$1"
  local payload
  payload=$(jq -Rs --arg name "$name" '{name:$name, content:.}')
  local out code body
  out=$(printf "%s" "$payload" | curl -sS -X POST "$BASE_URL/api/v1/flows" \
    -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d @- -w "\n%{http_code}")
  code=$(printf "%s" "$out" | tail -n1)
  body=$(printf "%s" "$out" | sed '$d')
  if [[ ! "$code" =~ ^2[0-9][0-9]$ ]]; then
    echo "[!] Flow create failed ($code). Body:" >&2
    printf "%s\n" "$body" >&2
    exit 1
  fi
  printf "%s" "$body" | jq -r .id
}

FLOW1_ID=$(printf "%s" "$FLOW1_YAML" | create_flow "MCP: List Tools")
echo "[i] Flow1 (list tools) ID: $FLOW1_ID"

if [[ "$MCP_TRANSPORT" == "sse" ]]; then
  FLOW2_YAML=$(inject_sse_params demos/mcp_execute_tool.yaml "$MCP_SSE_URL" "$MCP_SSE_POST" "$MCP_HEADERS_JSON" "$MCP_RAW_BODY_EXEC_JSON")
else
  FLOW2_YAML=$(inject_dotenvx demos/mcp_execute_tool.yaml | inject_env /dev/stdin "$MCP_ENV_LINES" | inject_args_extra /dev/stdin "$MCP_PORT_FLAG" "$MCP_PORT" | inject_server_port_param /dev/stdin "$MCP_PORT" "$MCP_PORT_FLAG")
fi
FLOW2_ID=$(printf "%s" "$FLOW2_YAML" | create_flow "MCP: Execute Tool")
echo "[i] Flow2 (execute tool) ID: $FLOW2_ID"

echo "[5] Run list tools"
EXEC1=$(curl -fsSL -X POST "$BASE_URL/api/v1/flows/$FLOW1_ID/run" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"input":{}}' | jq -r .execution_id)
echo "[i] Exec1: $EXEC1"

for i in {1..20}; do
  STATUS=$(curl -fsSL -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/executions/$EXEC1" | jq -r .status)
  echo "  status[$i]=$STATUS"
  [[ "$STATUS" == "completed" || "$STATUS" == "failed" ]] && break
  sleep 1
done
echo "[logs:list-tools]"
curl -fsSL -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/executions/$EXEC1/logs" | jq . | sed -n '1,200p'

if [[ -n "$TOOL_NAME" ]]; then
  echo "[6] Run execute tool ($TOOL_NAME)"
  EXEC2=$(jq -cn --arg name "$TOOL_NAME" --argjson params "$TOOL_PARAMS_JSON" '{input:{toolName:$name, toolParams:$params}}' \
    | curl -fsSL -X POST "$BASE_URL/api/v1/flows/$FLOW2_ID/run" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d @- | jq -r .execution_id)
  echo "[i] Exec2: $EXEC2"
  for i in {1..20}; do
    STATUS=$(curl -fsSL -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/executions/$EXEC2" | jq -r .status)
    echo "  status[$i]=$STATUS"
    [[ "$STATUS" == "completed" || "$STATUS" == "failed" ]] && break
    sleep 1
  done
  echo "[logs:execute-tool]"
  curl -fsSL -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/executions/$EXEC2/logs" | jq . | sed -n '1,200p'
fi

echo "[done]"

# Cleanup auto-launched server
if [[ "${MCP_AUTO_LAUNCH:-false}" == "true" && "${MCP_AUTO_KILL:-true}" == "true" && -f .mcp_server.pid ]]; then
  PID=$(cat .mcp_server.pid || true)
  if [[ -n "$PID" ]]; then
    echo "[i] Stopping MCP server PID=$PID"
    kill "$PID" >/dev/null 2>&1 || true
  fi
fi
