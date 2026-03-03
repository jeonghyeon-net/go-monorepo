#!/bin/bash
# 서브디렉터리에서 실행된 make 명령을 repo root 기준으로 보정한다.
# svc/ 하위에 있으면 SVC를 자동 감지하여 추가한다.

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command' 2>/dev/null)

# make 명령이 아니면 그대로 통과
if [[ ! "$COMMAND" =~ ^make ]]; then
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}'
  exit 0
fi

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)

# 이미 repo root에 있으면 보정 불필요
if [[ -z "$REPO_ROOT" || "$PWD" == "$REPO_ROOT" ]]; then
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}'
  exit 0
fi

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
exit 0
