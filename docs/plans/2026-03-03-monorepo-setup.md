# Go 모노레포 환경 구성 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Go Workspace 기반 멀티모듈 모노레포의 환경(디렉터리 구조, 빌드 도구, 린트 설정)을 구성한다. 서비스 코드는 작성하지 않는다.

**Architecture:** `go.work`로 여러 독립 Go 모듈을 하나의 workspace로 연결한다. 루트에 공유 설정(린트, 도구 버전, Makefile)을 두고, `services/`와 `pkg/` 디렉터리에 각각 서비스와 공유 패키지를 배치한다.

**Tech Stack:** Go 1.26, Go Workspace (`go.work`), golangci-lint v2, mise, Make

---

### Task 1: 디렉터리 구조 생성

**Files:**
- Create: `services/.gitkeep`
- Create: `pkg/.gitkeep`

**Step 1: 디렉터리 생성**

```bash
mkdir -p services pkg
touch services/.gitkeep pkg/.gitkeep
```

**Step 2: 커밋**

```bash
git add services/.gitkeep pkg/.gitkeep
git commit -m "services/, pkg/ 디렉터리 추가"
```

---

### Task 2: go.work 생성

**Files:**
- Create: `go.work`

**Step 1: go.work 파일 생성**

```bash
go work init
```

이 명령은 현재 Go 버전(1.26.0)으로 `go.work`를 생성한다. 아직 서비스가 없으므로 `use` 디렉티브는 비어있다.

생성 결과:
```go
go 1.26.0
```

**Step 2: 검증**

```bash
cat go.work
```

Expected: `go 1.26.0`이 포함된 파일.

**Step 3: 커밋**

```bash
git add go.work
git commit -m "go.work 추가 — Go Workspace 초기화"
```

---

### Task 3: .mise.toml 생성

**Files:**
- Create: `.mise.toml`

rest-api의 `.mise.toml`을 기반으로, 모든 서비스에서 공통으로 사용할 도구 버전을 정의한다.

**Step 1: .mise.toml 작성**

```toml
[tools]
go = "1.26.0"
golangci-lint = "2.10.1"
"go:github.com/sqlc-dev/sqlc/cmd/sqlc" = "latest"
"go:github.com/pressly/goose/v3/cmd/goose" = "latest"
"go:go.uber.org/nilaway/cmd/nilaway" = "latest"
```

rest-api와 동일한 도구/버전을 사용한다.

**Step 2: 커밋**

```bash
git add .mise.toml
git commit -m ".mise.toml 추가 — 도구 버전 통일"
```

---

### Task 4: .gitignore 생성

**Files:**
- Create: `.gitignore`

**Step 1: .gitignore 작성**

rest-api의 `.gitignore`를 기반으로 모노레포에 맞게 확장한다.

```gitignore
# 빌드 산출물
tmp/

# 환경 변수
.env

# 테스트 커버리지
coverage.out
coverage.html

# SQLite DB
*.db
data/

# OS
.DS_Store

# Air (핫 리로드)
.air.toml
```

**Step 2: 커밋**

```bash
git add .gitignore
git commit -m ".gitignore 추가"
```

---

### Task 5: .golangci.yml 생성

**Files:**
- Create: `.golangci.yml`

rest-api의 `.golangci.yml`을 복사하되, 프로젝트별 설정(모듈 경로, import prefix 등)을 모노레포에 맞게 일반화한다.

**Step 1: rest-api의 .golangci.yml 복사 후 수정**

변경이 필요한 부분만 명시한다. 나머지는 rest-api와 동일:

1. **gofumpt module-path** 제거:
   - 변경 전: `module-path: rest-api`
   - 변경 후: 이 설정을 삭제한다. Go Workspace에서 golangci-lint가 각 모듈의 `go.mod`를 자동 감지한다.

2. **gci sections prefix** 변경:
   - 변경 전: `- prefix(rest-api)`
   - 변경 후: `- prefix(go-monorepo)` (모노레포 모듈명의 공통 prefix)

3. **wrapcheck ignore-package-globs** 변경:
   - 변경 전: `- "rest-api/*"`
   - 변경 후: `- "go-monorepo/*"` (모노레포 내 모든 모듈의 내부 에러는 래핑 불필요)

4. **ireturn allow 항목 유지**: huma, fiber 항목은 라이브러리 기반 설정이므로 그대로 유지. 서비스가 해당 라이브러리를 쓰지 않아도 해가 없다.

5. **exclusions rules의 path** 유지: `internal/domain/`, `cmd/`, `_test\.go$` 등은 패턴이 범용적이므로 그대로 유지.

6. **주석의 rest-api 레퍼런스**: 주석에서 `rest-api`를 `프로젝트`나 `모노레포`로 변경.

**Step 2: 커밋**

```bash
git add .golangci.yml
git commit -m ".golangci.yml 추가 — 공유 린트 설정"
```

---

### Task 6: 루트 Makefile 생성

**Files:**
- Create: `Makefile`

**Step 1: Makefile 작성**

```makefile
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
```

**Step 2: 검증**

```bash
make setup
```

Expected: mise가 `.mise.toml`의 도구들을 설치한다 (이미 설치된 경우 스킵).

**Step 3: 커밋**

```bash
git add Makefile
git commit -m "루트 Makefile 추가 — 전체 서비스 오케스트레이션"
```

---

### Task 7: 최종 검증

**Step 1: 전체 구조 확인**

```bash
find . -not -path './.git/*' -not -path './.claude/*' | sort
```

Expected:
```
.
./.gitignore
./.golangci.yml
./.mise.toml
./docs
./docs/plans
./docs/plans/2026-03-03-monorepo-setup-design.md
./docs/plans/2026-03-03-monorepo-setup.md
./go.work
./Makefile
./pkg
./pkg/.gitkeep
./services
./services/.gitkeep
```

**Step 2: go.work 동작 확인**

```bash
go env GOWORK
```

Expected: `/Users/me/Desktop/go-monorepo/go.work` 경로가 출력된다.

**Step 3: make 명령 확인**

```bash
make build
make test
make lint
make fmt
```

Expected: 서비스가 없으므로 build/test는 아무 일 없이 종료. lint/fmt도 검사할 Go 파일이 없으므로 성공.
