#!/bin/bash

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command' 2>/dev/null)

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)

# make 명령은 항상 repo root에서 실행되도록 보정한다.
if [[ "$COMMAND" =~ ^make ]]; then
  if [[ -n "$REPO_ROOT" && "$PWD" != "$REPO_ROOT" ]]; then
    UPDATED=$(echo "$COMMAND" | sed "s|^make|make -C $REPO_ROOT|")
    echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","updatedInput":{"command":"'"$UPDATED"'"}}}'
  else
    echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}'
  fi
  exit 0
fi

deny() {
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"'"$1"'을(를) 직접 실행하지 마세요. '"$2"'을(를) 사용하세요."}}'
  exit 0
}

case "$COMMAND" in
  *golangci-lint*) deny "golangci-lint" "make lint" ;;
  *gofumpt*)       deny "gofumpt" "make fmt" ;;
  *gofmt*)         deny "gofmt" "make fmt" ;;
  *nilaway*)       deny "nilaway" "make lint" ;;
  "go build"*|*" go build"*)       deny "go build" "make build" ;;
  "go test"*|*" go test"*)         deny "go test" "make test" ;;
  "go fmt"*|*" go fmt"*)           deny "go fmt" "make fmt" ;;
  "docker buildx"*|*" docker buildx"*) deny "docker buildx" "make docker-build" ;;
  "docker build"*|*" docker build"*)   deny "docker build" "make docker-build" ;;
esac

echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}'
exit 0
