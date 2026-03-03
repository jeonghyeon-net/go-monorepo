// Package ruleset은 아키텍처 규칙들을 정의하고 검사하는 패키지다.
//
// 6가지 규칙 카테고리:
//   - dependency: 의존성 방향 규칙
//   - naming: 네이밍 컨벤션
//   - interface: 인터페이스 패턴
//   - structure: 디렉토리 구조 규칙
//   - sqlc: 코드 생성 규칙
//   - testing: 테스트 품질 규칙
package ruleset

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config는 아키텍처 테스트 실행에 필요한 프로젝트 설정 정보를 담는다.
type Config struct {
	ModuleName  string // go.mod의 모듈 이름
	ProjectRoot string // 프로젝트 루트 절대 경로
}

// ──────────────────────────────────────────────
// 규칙에서 사용하는 상수 목록들
// ──────────────────────────────────────────────

var (
	// AllowedDomainDirs: 도메인 디렉토리 안에 허용되는 하위 디렉토리 목록.
	//nolint:gochecknoglobals // 아키텍처 규칙에서 사용하는 불변 설정값이다.
	AllowedDomainDirs = []string{"subdomain", "svc", "handler", "infra"}

	// AllowedSubdomainLayers: 서브도메인 안에 허용되는 레이어 디렉토리 목록.
	//nolint:gochecknoglobals // 아키텍처 규칙에서 사용하는 불변 설정값이다.
	AllowedSubdomainLayers = []string{"model", "repo", "svc"}

	// AllowedHandlerProtocols: handler/ 아래에 허용되는 프로토콜 디렉토리.
	//nolint:gochecknoglobals // 아키텍처 규칙에서 사용하는 불변 설정값이다.
	AllowedHandlerProtocols = []string{"http", "grpc", "jsonrpc"}

	// ForbiddenPackageNames: 사용이 금지된 패키지 이름 목록.
	// 책임이 불분명해서 비대해지기 쉬운 이름들이다.
	//nolint:gochecknoglobals // 아키텍처 규칙에서 사용하는 불변 설정값이다.
	ForbiddenPackageNames = []string{"util", "utils", "common", "misc", "helper", "helpers", "shared", "lib"}

	// LayerOrder는 레이어 간 의존성 방향을 인덱스로 정의한다.
	// 의존성 규칙: model(0) ← repo(1) ← svc(2). 안쪽이 바깥쪽을 import할 수 없다.
	//nolint:gochecknoglobals // 아키텍처 규칙에서 사용하는 불변 설정값이다.
	LayerOrder = []string{"model", "repo", "svc"}
)

// ──────────────────────────────────────────────
// Config 생성 함수
// ──────────────────────────────────────────────

// NewConfig는 프로젝트 루트 경로를 받아서 Config를 생성한다.
func NewConfig(projectRoot string) (*Config, error) {
	moduleName, err := readModuleName(projectRoot)
	if err != nil {
		return nil, err
	}
	return &Config{
		ModuleName:  moduleName,
		ProjectRoot: projectRoot,
	}, nil
}

// readModuleName은 go.mod 파일에서 모듈 이름을 추출한다.
func readModuleName(projectRoot string) (string, error) {
	file, err := os.Open(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("go.mod 파일 열기 실패: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if modName, found := strings.CutPrefix(line, "module "); found {
			return strings.TrimSpace(modName), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("go.mod 파일 읽기 실패: %w", err)
	}
	return "", nil
}
