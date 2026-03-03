#!/bin/bash
# 빌드 도구 직접 실행을 차단한다. 반드시 make를 통해 실행해야 한다.

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command' 2>/dev/null)

deny() {
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"'"$1"'을(를) 직접 실행하지 마세요. '"$2"'을(를) 사용하세요."}}'
  exit 0
}

case "$COMMAND" in
  make*) ;; # make는 rewrite-make.sh에서 처리
  *golangci-lint*) deny "golangci-lint" "make lint" ;;
  *gofumpt*)       deny "gofumpt" "make fmt" ;;
  *gofmt*)         deny "gofmt" "make fmt" ;;
  *nilaway*)       deny "nilaway" "make lint" ;;
  *"go build"*)  deny "go build" "make build" ;;
  *"go test"*)   deny "go test" "make test" ;;
  *"go run"*)    deny "go run" "make dev" ;;
  *"go fmt"*)    deny "go fmt" "make fmt" ;;
  *"docker buildx"*) deny "docker buildx" "make docker-build" ;;
  *"docker build"*)  deny "docker build" "make docker-build" ;;
esac

echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}'
exit 0
