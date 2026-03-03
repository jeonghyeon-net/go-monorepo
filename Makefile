# ─────────────────────────────────────────────────────────────────────────────
# Makefile — 모노레포 전체 서비스에 대한 오케스트레이션 명령을 제공한다.
# ─────────────────────────────────────────────────────────────────────────────
#
# 사용법:
#   make build          — 전체 서비스 빌드
#   make test           — 전체 서비스 테스트
#   make lint           — 전체 린트 (golangci-lint + nilaway)
#   make fmt            — 전체 포맷
#   make svc-build SVC=rest-api  — 특정 서비스만 빌드
#   make setup          — 개발 환경 초기 설정
.PHONY: build test lint fmt setup

# ── 전체 서비스 대상 명령 ─────────────────────────────────────────────────────

# 전체 서비스를 순회하며 빌드한다.
# services/ 아래 각 디렉터리의 Makefile을 호출한다.
# 서비스가 없으면 아무 일도 하지 않는다.
build:
	@for dir in services/*/; do \
		[ -f "$$dir/Makefile" ] && $(MAKE) -C "$$dir" build || true; \
	done

# 전체 서비스의 테스트를 실행한다.
test:
	@for dir in services/*/; do \
		[ -f "$$dir/Makefile" ] && $(MAKE) -C "$$dir" test || true; \
	done

# golangci-lint를 전체 workspace에 대해 실행한다.
# nilaway도 전체에 대해 실행한다.
lint:
	golangci-lint run ./...
	nilaway ./...

# 전체 서비스의 코드를 포맷한다.
fmt:
	gofmt -w .
	golangci-lint fmt

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
