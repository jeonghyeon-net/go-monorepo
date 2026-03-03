#!/bin/bash

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command' 2>/dev/null)

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)

# make 명령은 항상 repo root에서 실행되도록 보정한다.
# 서브디렉터리(svc/*)에 있으면 SVC를 자동 감지하여 추가한다.
if [[ "$COMMAND" =~ ^make ]]; then
  if [[ -n "$REPO_ROOT" && "$PWD" != "$REPO_ROOT" ]]; then
    # svc/ 하위에 있고 SVC가 명시되지 않았으면 자동 감지
    SVC_ARG=""
    REL_PATH="${PWD#$REPO_ROOT/}"
    if [[ "$REL_PATH" =~ ^svc/([^/]+) ]]; then
      SVC_NAME="${BASH_REMATCH[1]}"
      if [[ ! "$COMMAND" =~ SVC= ]]; then
        SVC_ARG=" SVC=$SVC_NAME"
      fi
    fi
    UPDATED=$(echo "$COMMAND" | sed "s|^make|make -C $REPO_ROOT|")
    UPDATED="${UPDATED}${SVC_ARG}"
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
