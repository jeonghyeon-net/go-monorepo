# ─────────────────────────────────────────────────────────────────────────────
# Makefile — 모노레포 전체 서비스에 대한 오케스트레이션 명령을 제공한다.
# ─────────────────────────────────────────────────────────────────────────────
#
# 사용법:
#   make build              — 전체 빌드 (go.work 기준)
#   make test               — 전체 테스트
#   make lint               — 전체 린트 (golangci-lint + nilaway)
#   make fmt                — 전체 포맷
#   make svc-build SVC=rest-api  — 특정 서비스만 빌드
#   make setup              — 개발 환경 초기 설정
.PHONY: build test lint fmt setup

# mise가 관리하는 도구(golangci-lint, nilaway 등)의 PATH를 보장한다.
SHELL := mise exec -- bash

# ── 전체 대상 명령 ─────────────────────────────────────────────────────────────

# go.work에 등록된 모든 모듈(공유 패키지 + 서비스)을 빌드한다.
# go.work의 use 지시문에서 모듈 경로를 자동 추출한다.
build:
	@awk '/^[[:space:]]*\.\//{gsub(/^[[:space:]]+/,""); print}' go.work | while read dir; do \
		cd $(CURDIR)/$$dir && go build ./...; \
	done

# go.work에 등록된 모든 모듈의 테스트를 실행한다.
test:
	@awk '/^[[:space:]]*\.\//{gsub(/^[[:space:]]+/,""); print}' go.work | while read dir; do \
		echo "=== $$dir ===" && cd $(CURDIR)/$$dir && go test ./... 2>&1 | grep -v '\[no test files\]'; test $${PIPESTATUS[0]} -eq 0; \
	done

# go.work에 등록된 모든 모듈에 golangci-lint와 nilaway를 실행한다.
lint:
	@awk '/^[[:space:]]*\.\//{gsub(/^[[:space:]]+/,""); print}' go.work | while read dir; do \
		cd $(CURDIR)/$$dir && golangci-lint run --fix ./... && nilaway ./...; \
	done

# go.work에 등록된 모든 모듈의 코드를 포맷한다.
fmt:
	@awk '/^[[:space:]]*\.\//{gsub(/^[[:space:]]+/,""); print}' go.work | while read dir; do \
		cd $(CURDIR)/$$dir && gofmt -w . && golangci-lint fmt; \
	done

# ── 특정 서비스 대상 명령 ─────────────────────────────────────────────────────

# 사용법: make svc-build SVC=rest-api
# svc- 뒤의 이름이 서비스 Makefile의 타겟으로 전달된다.
svc-%:
	@if [ -z "$(SVC)" ]; then echo "SVC를 지정하세요. 예: make svc-build SVC=rest-api"; exit 1; fi
	$(MAKE) -C services/$(SVC) $*

# ── 프로젝트 초기 설정 ────────────────────────────────────────────────────────

# 새로 프로젝트를 클론한 후 한 번 실행하는 초기 설정 명령이다.
# mise install: .mise.toml에 정의된 도구들을 설치한다.
setup:
	mise install
