// 이 파일은 sqlc 코드 생성 규칙을 검사한다.
//
// 3가지 규칙:
//  1. sqlc/missing-sqlc-entry: repo/에 .sql 파일이 있지만 sqlc.yaml에 미등록
//  2. sqlc/manual-code-in-sqlc-repo: sqlc가 관리하는 repo/에 수기 .go 파일 존재
//  3. sqlc/orphan-sqlc-entry: sqlc.yaml에 있지만 실제 .sql 파일이 없는 항목
package ruleset

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"go-monorepo/pkg/archtest/report"
)

// ──────────────────────────────────────────────
// sqlc.yaml 파싱을 위한 구조체
// ──────────────────────────────────────────────

type sqlcConfig struct {
	SQL []sqlcEntry `yaml:"sql"`
}

type sqlcEntry struct {
	Queries string `yaml:"queries"`
}

// ──────────────────────────────────────────────
// 메인 검사 함수
// ──────────────────────────────────────────────

// CheckSqlc는 sqlc 관련 3가지 규칙을 모두 검사한다.
func CheckSqlc(cfg *Config) []report.Violation {
	var violations []report.Violation

	sqlcCfg, err := parseSqlcYaml(cfg.ProjectRoot)
	if err != nil {
		sqlcCfg = &sqlcConfig{}
	}

	registeredPaths := make(map[string]bool)
	for _, entry := range sqlcCfg.SQL {
		registeredPaths[filepath.Clean(entry.Queries)] = true
	}

	sqlRepoPaths := findSQLRepoPaths(cfg.ProjectRoot)

	// 규칙 1: .sql 파일이 있는 repo/가 sqlc.yaml에 등록되어 있는지 확인
	for _, repoPath := range sqlRepoPaths {
		relPath, err := filepath.Rel(cfg.ProjectRoot, repoPath)
		if err != nil {
			continue
		}
		cleaned := filepath.Clean(relPath)

		if !registeredPaths[cleaned] {
			violations = append(violations, report.Violation{
				Rule:     "sqlc/missing-sqlc-entry",
				Severity: report.Error,
				Message:  fmt.Sprintf("repo 디렉토리 %q에 .sql 파일이 있지만 sqlc.yaml에 등록되지 않음", cleaned),
				File:     repoPath,
				Fix:      "sqlc.yaml의 sql 배열에 해당 경로를 추가하고 'make sqlc-gen' 실행",
			})
		}

		// 규칙 2: sqlc가 관리하는 repo/에 수기 .go 파일이 있는지 확인
		violations = append(violations, checkManualGoFiles(repoPath)...)
	}

	// 규칙 3: sqlc.yaml에는 있지만 실제 .sql 파일이 없는 항목
	sqlRepoSet := make(map[string]bool)
	for _, p := range sqlRepoPaths {
		relPath, err := filepath.Rel(cfg.ProjectRoot, p)
		if err != nil {
			continue
		}
		sqlRepoSet[filepath.Clean(relPath)] = true
	}
	for registeredPath := range registeredPaths {
		if !sqlRepoSet[registeredPath] {
			violations = append(violations, report.Violation{
				Rule:     "sqlc/orphan-sqlc-entry",
				Severity: report.Warning,
				Message:  fmt.Sprintf("sqlc.yaml에 %q 항목이 있지만 해당 경로에 .sql 파일이 없음", registeredPath),
				File:     filepath.Join(cfg.ProjectRoot, "sqlc.yaml"),
				Fix:      "해당 도메인이 삭제되었다면 sqlc.yaml에서 항목을 제거",
			})
		}
	}

	return violations
}

// ──────────────────────────────────────────────
// sqlc.yaml 파싱 헬퍼
// ──────────────────────────────────────────────

func parseSqlcYaml(projectRoot string) (*sqlcConfig, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, "sqlc.yaml"))
	if err != nil {
		return nil, fmt.Errorf("sqlc.yaml 읽기 실패: %w", err)
	}

	var cfg sqlcConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("sqlc.yaml 파싱 실패: %w", err)
	}
	return &cfg, nil
}

// ──────────────────────────────────────────────
// .sql 파일이 있는 repo/ 디렉토리 탐색
// ──────────────────────────────────────────────

// findSQLRepoPaths는 internal/domain/ 아래의 repo/ 디렉토리 중
// .sql 파일이 있는 디렉토리의 절대 경로 목록을 반환한다.
func findSQLRepoPaths(projectRoot string) []string {
	var result []string

	domainRoot := filepath.Join(projectRoot, "internal", "domain")
	if _, err := os.Stat(domainRoot); os.IsNotExist(err) {
		return nil
	}

	//nolint:errcheck,gosec // WalkDir 콜백 내에서 에러를 개별 처리한다.
	filepath.WalkDir(domainRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // 접근 에러는 건너뛴다.
		}

		if !d.IsDir() || d.Name() != "repo" {
			return nil
		}

		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return nil //nolint:nilerr // 읽기 실패는 건너뛴다.
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
				result = append(result, path)
				return filepath.SkipDir
			}
		}
		return nil
	})

	return result
}

// ──────────────────────────────────────────────
// 수기 작성 Go 파일 검사
// ──────────────────────────────────────────────

// checkManualGoFiles는 sqlc가 관리하는 repo/ 디렉토리에서
// 자동 생성 헤더가 없는 수기 .go 파일을 찾아낸다.
func checkManualGoFiles(repoPath string) []report.Violation {
	var violations []report.Violation

	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(repoPath, entry.Name())
		if !hasSqlcGeneratedHeader(filePath) {
			violations = append(violations, report.Violation{
				Rule:     "sqlc/manual-code-in-sqlc-repo",
				Severity: report.Error,
				Message:  "sqlc를 사용하는 repo/ 디렉토리에 수기 작성된 Go 파일 발견: " + entry.Name(),
				File:     filePath,
				Fix:      "repo/의 Go 코드는 sqlc가 자동 생성해야 한다. 비즈니스 로직은 svc/ 레이어에 작성",
			})
		}
	}

	return violations
}

// hasSqlcGeneratedHeader는 파일 상단 5줄 안에 sqlc 자동 생성 헤더가 있는지 확인한다.
func hasSqlcGeneratedHeader(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	const sqlcHeaderMaxLines = 5

	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount > sqlcHeaderMaxLines {
			break
		}
		if strings.Contains(scanner.Text(), "Code generated by sqlc. DO NOT EDIT.") {
			return true
		}
	}
	return false
}
