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
