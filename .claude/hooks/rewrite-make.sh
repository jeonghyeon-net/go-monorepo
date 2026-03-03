#!/bin/bash
# 서브디렉터리에서 실행된 make 명령을 repo root 기준으로 보정한다.
# svc/ 하위에 있으면 SVC를 자동 감지하여 추가한다.

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command' 2>/dev/null)

# make 명령이 포함되어 있지 않으면 그대로 통과
if [[ ! "$COMMAND" =~ make[[:space:]] && ! "$COMMAND" =~ make$ ]]; then
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}'
  exit 0
fi

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
if [[ -z "$REPO_ROOT" ]]; then
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}'
  exit 0
fi

# 복합 명령어에서 cd 대상 디렉터리 추출 (cd path && make ...)
WORK_DIR="$PWD"
if [[ "$COMMAND" =~ cd[[:space:]]+([^;&]+) ]]; then
  CD_TARGET="${BASH_REMATCH[1]}"
  CD_TARGET=$(echo "$CD_TARGET" | xargs)  # trim whitespace
  if [[ "$CD_TARGET" = /* ]]; then
    WORK_DIR="$CD_TARGET"
  else
    WORK_DIR="$PWD/$CD_TARGET"
  fi
fi

# 이미 repo root에 있으면 보정 불필요
if [[ "$WORK_DIR" == "$REPO_ROOT" ]]; then
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}'
  exit 0
fi

# svc/ 하위에 있고 SVC가 명시되지 않았으면 자동 감지
SVC_ARG=""
REL_PATH="${WORK_DIR#$REPO_ROOT/}"
if [[ "$REL_PATH" =~ ^svc/([^/]+) ]]; then
  SVC_NAME="${BASH_REMATCH[1]}"
  if [[ ! "$COMMAND" =~ SVC= ]]; then
    SVC_ARG=" SVC=$SVC_NAME"
  fi
fi

# 복합 명령어에서 make 부분만 추출하여 보정
# cd ... && make xxx → make -C root xxx SVC=...
MAKE_PART=$(echo "$COMMAND" | grep -oE 'make([[:space:]].*)?$')
UPDATED="$MAKE_PART"
if [[ ! "$MAKE_PART" =~ -C[[:space:]] ]]; then
  UPDATED=$(echo "$MAKE_PART" | sed "s|^make|make -C $REPO_ROOT|")
fi
UPDATED="${UPDATED}${SVC_ARG}"
echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","updatedInput":{"command":"'"$UPDATED"'"}}}'
exit 0
