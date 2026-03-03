# 공유 아키텍처 테스트 라이브러리 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** rest-api의 `test/architecture/` 코드를 모노레포의 `pkg/archtest/`로 이동하여 공유 라이브러리로 만든다.

**Architecture:** 기존 코드를 그대로 복사하되, import 경로를 `rest-api/test/architecture/*` → `go-monorepo/pkg/archtest/*`로 변경하고, `archtest.RunAll(t, root)` 진입점을 추가한다.

**Tech Stack:** Go 1.26, go/ast, gopkg.in/yaml.v3

---

### Task 1: pkg 모듈 초기화 및 디렉터리 생성

**Files:**
- Modify: `go.work`
- Create: `pkg/go.mod`
- Create: `pkg/archtest/analyzer/` (directory)
- Create: `pkg/archtest/ruleset/` (directory)
- Create: `pkg/archtest/report/` (directory)
- Delete: `pkg/.gitkeep`

**Step 1: pkg/go.mod 생성**

```bash
cd pkg && go mod init go-monorepo/pkg && cd ..
```

**Step 2: 디렉터리 생성**

```bash
mkdir -p pkg/archtest/analyzer pkg/archtest/ruleset pkg/archtest/report
rm pkg/.gitkeep
```

**Step 3: go.work에 pkg 등록**

```bash
go work use ./pkg
```

`go.work`가 다음과 같아야 한다:
```go
go 1.26.0

use ./pkg
```

**Step 4: yaml.v3 의존성 추가**

sqlc.go에서 `gopkg.in/yaml.v3`를 사용하므로:
```bash
cd pkg && go get gopkg.in/yaml.v3 && cd ..
```

**Step 5: 커밋**

```bash
git add pkg/ go.work
git commit -m "pkg 모듈 초기화 — archtest 디렉터리 구조 생성"
```

---

### Task 2: analyzer, report 패키지 복사

이 파일들은 내부 import가 없으므로 그대로 복사한다.

**Files:**
- Create: `pkg/archtest/analyzer/parser.go`
- Create: `pkg/archtest/analyzer/domain.go`
- Create: `pkg/archtest/report/violation.go`

**Step 1: 파일 복사**

소스 경로: `/Users/me/Desktop/rest-api/test/architecture/`

- `analyzer/parser.go` → `pkg/archtest/analyzer/parser.go` (변경 없음)
- `analyzer/domain.go` → `pkg/archtest/analyzer/domain.go` (변경 없음)
- `report/violation.go` → `pkg/archtest/report/violation.go` (변경 없음)

이 3개 파일은 표준 라이브러리만 import하므로 수정이 필요 없다.

**Step 2: 빌드 확인**

```bash
cd pkg && go build ./archtest/analyzer/ ./archtest/report/ && cd ..
```

Expected: 에러 없이 성공

**Step 3: 커밋**

```bash
git add pkg/archtest/analyzer/ pkg/archtest/report/
git commit -m "analyzer, report 패키지 추가 — AST 파서 및 위반 보고"
```

---

### Task 3: ruleset 패키지 복사 (import 경로 변경)

**Files:**
- Create: `pkg/archtest/ruleset/config.go`
- Create: `pkg/archtest/ruleset/dependency.go`
- Create: `pkg/archtest/ruleset/naming.go`
- Create: `pkg/archtest/ruleset/interface_pattern.go`
- Create: `pkg/archtest/ruleset/structure.go`
- Create: `pkg/archtest/ruleset/sqlc.go`
- Create: `pkg/archtest/ruleset/testing.go`

**Step 1: config.go 복사**

`ruleset/config.go`는 내부 import가 없으므로 그대로 복사한다.

**Step 2: 나머지 ruleset 파일 복사 + import 경로 변경**

다음 6개 파일에서 import 경로를 변경한다:

| 변경 전 | 변경 후 |
|---------|--------|
| `rest-api/test/architecture/analyzer` | `go-monorepo/pkg/archtest/analyzer` |
| `rest-api/test/architecture/report` | `go-monorepo/pkg/archtest/report` |

**변경 대상 파일과 해당 import:**

- `dependency.go`: analyzer + report import
- `naming.go`: analyzer + report import
- `interface_pattern.go`: analyzer + report import
- `structure.go`: report import만
- `sqlc.go`: report import만
- `testing.go`: report import만

**로직, 주석, 설정값은 일체 변경하지 않는다. import 경로만 변경한다.**

**Step 3: 빌드 확인**

```bash
cd pkg && go build ./archtest/ruleset/ && cd ..
```

Expected: 에러 없이 성공

**Step 4: 커밋**

```bash
git add pkg/archtest/ruleset/
git commit -m "ruleset 패키지 추가 — 6개 아키텍처 규칙 카테고리"
```

---

### Task 4: archtest.go 진입점 생성

**Files:**
- Create: `pkg/archtest/archtest.go`

rest-api의 `arch_test.go` 로직을 라이브러리 함수로 변환한다.

**Step 1: archtest.go 작성**

`pkg/archtest/archtest.go` 내용:

```go
// Package archtest는 Go DDD 프로젝트의 아키텍처 규칙을 자동 검증하는 공유 라이브러리다.
//
// 모노레포의 각 서비스에서 이 패키지를 import하여 아키텍처 테스트를 실행한다.
//
// 사용법:
//
//	func TestArchitecture(t *testing.T) {
//	    archtest.RunAll(t, projectRoot(t))
//	}
//
// 6가지 규칙 카테고리를 검증한다:
//   - Dependencies: import 방향이 DDD 레이어 규칙을 따르는지
//   - Naming: 패키지명, 타입명이 컨벤션을 따르는지
//   - InterfacePatterns: 인터페이스+구현체+생성자 3종 세트
//   - Structure: 디렉터리 구조가 정해진 형태인지
//   - Sqlc: sqlc 설정 완전성 + 수기 코드 차단
//   - Testing: goleak goroutine 누수 검출, 빌드 태그 강제
package archtest

import (
	"path/filepath"
	"testing"

	"go-monorepo/pkg/archtest/analyzer"
	"go-monorepo/pkg/archtest/report"
	"go-monorepo/pkg/archtest/ruleset"
)

// RunAll은 주어진 프로젝트 루트에 대해 6가지 아키텍처 규칙을 모두 실행하고 결과를 보고한다.
//
// projectRoot는 go.mod가 있는 디렉터리의 절대 경로다.
// 각 규칙은 서브테스트로 실행되어 개별적으로 결과를 확인할 수 있다.
//
// 결과 처리:
//   - ERROR 심각도 위반이 있으면 → 테스트 FAIL
//   - WARNING만 있으면 → 테스트 PASS (로그에 경고만 출력)
//   - 위반 없으면 → 테스트 PASS
func RunAll(t *testing.T, projectRoot string) { //nolint:paralleltest // 서브테스트가 allViolations 슬라이스를 공유하므로 병렬 실행 불가.
	t.Helper()

	cfg, err := ruleset.NewConfig(projectRoot)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	// internal/ 디렉터리 안의 모든 Go 파일을 파싱한다.
	files, err := analyzer.ParseDirectory(filepath.Join(projectRoot, "internal"))
	if err != nil {
		t.Fatalf("failed to parse directory: %v", err)
	}

	var allViolations []report.Violation

	t.Run("Dependencies", func(t *testing.T) { //nolint:paralleltest // allViolations 공유.
		vs := ruleset.CheckDependencies(files, cfg)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("Naming", func(t *testing.T) { //nolint:paralleltest // allViolations 공유.
		vs := ruleset.CheckNaming(files, cfg)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("InterfacePatterns", func(t *testing.T) { //nolint:paralleltest // allViolations 공유.
		vs := ruleset.CheckInterfacePatterns(files, cfg)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("Structure", func(t *testing.T) { //nolint:paralleltest // allViolations 공유.
		vs := ruleset.CheckStructure(cfg)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("Sqlc", func(t *testing.T) { //nolint:paralleltest // allViolations 공유.
		vs := ruleset.CheckSqlc(cfg)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("Testing", func(t *testing.T) { //nolint:paralleltest // allViolations 공유.
		vs := ruleset.CheckTestingPatterns(cfg)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Log("\n" + report.Summary(allViolations))

	if report.HasErrors(allViolations) {
		t.Fatal("architecture violations with ERROR severity found")
	}
}

// reportViolations는 위반 목록을 테스트 로그에 출력한다.
func reportViolations(t *testing.T, violations []report.Violation) {
	t.Helper()
	for _, v := range violations {
		if v.Severity == report.Error {
			t.Error(v.String())
		} else {
			t.Log(v.String())
		}
	}
}
```

**Step 2: 빌드 확인**

```bash
cd pkg && go build ./archtest/ && cd ..
```

Expected: 에러 없이 성공

**Step 3: 커밋**

```bash
git add pkg/archtest/archtest.go
git commit -m "archtest.RunAll 진입점 추가 — 서비스에서 한 줄로 아키텍처 테스트 실행"
```

---

### Task 5: 최종 검증

**Step 1: 전체 빌드 확인**

```bash
cd pkg && go build ./... && cd ..
```

Expected: 에러 없이 성공

**Step 2: go.work 확인**

```bash
go env GOWORK
cat go.work
```

Expected: `use ./pkg`가 포함되어 있어야 한다.

**Step 3: 디렉터리 구조 확인**

```bash
find pkg -type f | sort
```

Expected:
```
pkg/archtest/analyzer/domain.go
pkg/archtest/analyzer/parser.go
pkg/archtest/archtest.go
pkg/archtest/report/violation.go
pkg/archtest/ruleset/config.go
pkg/archtest/ruleset/dependency.go
pkg/archtest/ruleset/interface_pattern.go
pkg/archtest/ruleset/naming.go
pkg/archtest/ruleset/sqlc.go
pkg/archtest/ruleset/structure.go
pkg/archtest/ruleset/testing.go
pkg/go.mod
pkg/go.sum
```
