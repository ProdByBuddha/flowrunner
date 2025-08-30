#!/usr/bin/env bash
set -euo pipefail

PORT="${1:-}"
ACTION="${2:-}"
if [[ -z "$PORT" ]]; then
  echo "Usage: $0 <port> [--kill]" >&2
  exit 1
fi

echo "[i] Inspecting listeners on TCP port $PORT"

have_lsof=0; command -v lsof >/dev/null 2>&1 && have_lsof=1
have_ss=0; command -v ss >/dev/null 2>&1 && have_ss=1
have_fuser=0; command -v fuser >/dev/null 2>&1 && have_fuser=1

if [[ $have_lsof -eq 1 ]]; then
  echo "[lsof]"
  lsof -nP -iTCP:$PORT -sTCP:LISTEN || true
fi

if [[ $have_ss -eq 1 ]]; then
  echo "[ss]"
  # Try to show process with PID/command if permitted
  ss -lntp | awk 'NR==1 || $4 ~ /:'"$PORT"'$/ {print}' || true
fi

if [[ $have_fuser -eq 1 ]]; then
  echo "[fuser]"
  fuser -v ${PORT}/tcp || true
fi

# Best-effort: map inode to PIDs by scanning /proc
echo "[proc-scan]"
hexport=$(printf "%04X" "$PORT")
inode=""
while read -r _ _ local st _ _ _ _ _ inode _; do
  # header has 'local_address' skip
  [[ "$local" == local_address ]] && continue
  # state 0A is LISTEN
  if [[ "$st" == "0A" ]]; then
    phex=${local##*:}
    if [[ "$phex" == "$hexport" ]]; then
      echo "  found inode=$inode for :$PORT"
      break
    fi
  fi
done < /proc/net/tcp

if [[ -n "$inode" ]]; then
  for p in /proc/[0-9]*/fd/*; do
    if [[ -L "$p" ]]; then
      target=$(readlink "$p" || true)
      if [[ "$target" =~ socket:\[${inode}\] ]]; then
        pid=$(echo "$p" | awk -F/ '{print $3}')
        cmd=$(tr '\0' ' ' < /proc/$pid/cmdline 2>/dev/null || true)
        echo "  PID=$pid CMD=$cmd"
      fi
    fi
  done
else
  echo "  no inode found via /proc/net/tcp (might be IPv6 only)"
fi

if [[ "${ACTION:-}" == "--kill" ]]; then
  echo "[!] Killing listeners on :$PORT"
  killed=0
  if [[ $have_lsof -eq 1 ]]; then
    # Capture PIDs from lsof
    mapfile -t pids < <(lsof -nP -iTCP:$PORT -sTCP:LISTEN -t || true)
    for pid in "${pids[@]:-}"; do
      echo "  kill -9 $pid"
      kill -9 "$pid" || true
      killed=1
    done
  fi
  if [[ $killed -eq 0 && $have_fuser -eq 1 ]]; then
    echo "  fuser -k ${PORT}/tcp"
    fuser -k ${PORT}/tcp || true
  fi
  echo "[done]"
fi

