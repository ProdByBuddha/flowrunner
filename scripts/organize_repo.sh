#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "$0")/.." && pwd)
cd "$root_dir"

echo "[i] Organizing repository into clearer folders (non-destructive move)."

mkdir -p docs/archive examples/template_engine examples/mcp

move_if() {
  local src="$1" dst="$2"
  if [[ -e "$src" ]]; then
    echo "  - mv $src -> $dst"
    mkdir -p "$(dirname "$dst")"
    git mv -k "$src" "$dst" 2>/dev/null || mv -f "$src" "$dst"
  fi
}

# Archive narrative docs from root
for f in \
  COMPREHENSIVE_LLM_INTEGRATION_SUMMARY.md \
  DYNAMIC_INPUT_IMPLEMENTATION_SUMMARY.md \
  DYNAMODB_SECRET_VAULT_TESTING_COMPLETE.md \
  implementation_summary.md \
  INTERNAL_PROGRESS.md \
  PARALLEL_LLM_REGRESSION_TEST_RESULTS.md \
  POSTGRESQL_TESTING_GUIDE.md \
  POSTGRESQL_VERIFICATION.md \
  PRD.md \
  task_5_3_complete.md \
  task_5_3_debugging_complete.md \
  TASK_5_3_FINAL_STATUS.md \
  TASK_6_1_IMPLEMENTATION_PLAN.md \
  TEMPLATE_ENGINE_COMPLETE_SUMMARY.md; do
  move_if "$f" "docs/archive/$f"
done

# Move template engine examples
move_if template_engine_demo.go examples/template_engine/template_engine_demo.go
move_if test_secret_resolution.go examples/template_engine/test_secret_resolution.go
move_if strucouttemplate examples/template_engine/strucouttemplate

# MCP example (TypeScript snippet)
move_if mcpexample examples/mcp/mcpexample.ts

echo "[i] Done. Review changes with 'git status' and run your tests/build." 
