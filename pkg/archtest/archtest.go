// Package archtest는 Go DDD 프로젝트의 아키텍처 규칙을 자동 검증하는 공유 라이브러리다.
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
// ERROR 위반이 있으면 테스트 FAIL, WARNING만 있으면 PASS.
//
// pathPrefix가 주어지면 위반 파일 경로 앞에 붙여서 모노레포 내 위치를 명확히 한다.
// 예: RunAll(t, serviceRoot, "svc/hello") → file: svc/hello/internal/greeter
func RunAll(t *testing.T, projectRoot string, pathPrefix ...string) {
	t.Helper()

	prefix := ""
	if len(pathPrefix) > 0 {
		prefix = pathPrefix[0]
	}

	cfg, err := ruleset.NewConfig(projectRoot)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	files, err := analyzer.ParseDirectory(filepath.Join(projectRoot, "internal"))
	if err != nil {
		t.Fatalf("failed to parse directory: %v", err)
	}

	var allViolations []report.Violation

	t.Run("Dependencies", func(t *testing.T) {
		vs := ruleset.CheckDependencies(files, cfg)
		addPathContext(vs, projectRoot, prefix)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("Naming", func(t *testing.T) {
		vs := ruleset.CheckNaming(files, cfg)
		addPathContext(vs, projectRoot, prefix)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("InterfacePatterns", func(t *testing.T) {
		vs := ruleset.CheckInterfacePatterns(files, cfg)
		addPathContext(vs, projectRoot, prefix)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("Structure", func(t *testing.T) {
		vs := ruleset.CheckStructure(cfg)
		addPathContext(vs, projectRoot, prefix)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("Sqlc", func(t *testing.T) {
		vs := ruleset.CheckSqlc(cfg)
		addPathContext(vs, projectRoot, prefix)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Run("Testing", func(t *testing.T) {
		vs := ruleset.CheckTestingPatterns(cfg)
		addPathContext(vs, projectRoot, prefix)
		allViolations = append(allViolations, vs...)
		reportViolations(t, vs)
	})

	t.Log("\n" + report.Summary(allViolations))

	if report.HasErrors(allViolations) {
		t.Fatal("architecture violations with ERROR severity found")
	}
}

// addPathContext는 위반 파일 경로를 정규화하고 prefix를 붙인다.
// 절대 경로는 projectRoot 기준 상대 경로로 변환한 뒤, prefix를 앞에 붙인다.
func addPathContext(violations []report.Violation, projectRoot, prefix string) {
	for i := range violations {
		file := violations[i].File
		if filepath.IsAbs(file) {
			if rel, err := filepath.Rel(projectRoot, file); err == nil {
				file = rel
			}
		}
		if prefix != "" {
			file = filepath.Join(prefix, file)
		}
		violations[i].File = file
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
