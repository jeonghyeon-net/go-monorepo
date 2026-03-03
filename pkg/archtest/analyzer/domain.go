package analyzer

// 이 파일은 Go 파일의 경로를 분석해서 도메인/서브도메인/레이어 또는 Saga 구조를 파악한다.
//
// 두 가지 경로 패턴을 처리한다:
//
//  1. 도메인 파일: "internal/domain/user/subdomain/core/model/user.go"
//     → Domain:"user", Subdomain:"core", Layer:"model"
//
//  2. Saga 파일: "internal/saga/create_order/saga.go"
//     → IsSaga:true, SagaName:"create_order"

import (
	"path/filepath"
	"strings"
)

// subdomainLayerIndex는 subdomain/{서브도메인명}/{레이어} 구조에서 레이어의 위치다.
const subdomainLayerIndex = 3

// DomainPath는 하나의 파일이 프로젝트 구조에서 어디에 위치하는지를 나타낸다.
//
// 프로젝트 디렉토리 구조:
//
//	internal/
//	├── domain/{도메인}/
//	│   ├── subdomain/{서브도메인}/{레이어}/
//	│   ├── svc/
//	│   ├── handler/{프로토콜}/
//	│   ├── infra/
//	│   └── alias.go
//	└── saga/{사가이름}/
//	    └── saga.go
type DomainPath struct {
	Domain    string // 도메인 이름
	Subdomain string // 서브도메인 이름 (없으면 빈 문자열)
	Layer     string // 레이어: "model", "repo", "svc", "handler", "infra", "root"
	Protocol  string // 핸들러 프로토콜: "http", "grpc" (handler 레이어에서만)
	File      string // 파일명

	SagaName string // Saga 이름
	IsSaga   bool   // true면 Saga 파일
}

// ParseDomainPath는 프로젝트 루트 기준 상대 경로를 DomainPath로 분해한다.
// 도메인/Saga 패턴에 해당하지 않으면 nil을 반환한다.
func ParseDomainPath(relPath string) *DomainPath {
	parts := strings.Split(filepath.ToSlash(relPath), "/")

	internalIdx := -1
	for i, p := range parts {
		if p == "internal" {
			internalIdx = i
			break
		}
	}
	if internalIdx < 0 || internalIdx+1 >= len(parts) {
		return nil
	}

	switch parts[internalIdx+1] {
	case "saga":
		return parseSagaPath(parts, internalIdx)
	case "domain":
		return parseDomainPathParts(parts, internalIdx)
	default:
		return nil
	}
}

// ──────────────────────────────────────────────
// Saga 경로 파싱
// ──────────────────────────────────────────────

// parseSagaPath는 "internal/saga/{사가이름}/..." 형태의 경로를 파싱한다.
func parseSagaPath(parts []string, internalIdx int) *DomainPath {
	sagaNameIdx := internalIdx + 2
	if sagaNameIdx >= len(parts) {
		return nil
	}

	dp := &DomainPath{
		IsSaga:   true,
		SagaName: parts[sagaNameIdx],
	}

	remaining := parts[sagaNameIdx+1:]
	if len(remaining) > 0 {
		last := remaining[len(remaining)-1]
		if strings.HasSuffix(last, ".go") {
			dp.File = last
		}
	}

	return dp
}

// ──────────────────────────────────────────────
// 도메인 경로 파싱
// ──────────────────────────────────────────────

// parseDomainPathParts는 "internal/domain/{도메인}/..." 형태의 경로를 파싱한다.
func parseDomainPathParts(parts []string, internalIdx int) *DomainPath {
	domainIdx := internalIdx + 2
	if domainIdx >= len(parts) {
		return nil
	}

	dp := &DomainPath{
		Domain: parts[domainIdx],
	}

	remaining := parts[domainIdx+1:]

	if len(remaining) > 0 {
		last := remaining[len(remaining)-1]
		if strings.HasSuffix(last, ".go") {
			dp.File = last
			remaining = remaining[:len(remaining)-1]
		}
	}

	if len(remaining) == 0 {
		dp.Layer = "root"
		return dp
	}

	switch remaining[0] {
	case "subdomain":
		if len(remaining) >= 2 {
			dp.Subdomain = remaining[1]
		}
		if len(remaining) >= subdomainLayerIndex {
			dp.Layer = remaining[2]
		}
	case "svc":
		dp.Layer = "svc"
	case "handler":
		dp.Layer = "handler"
		if len(remaining) >= 2 {
			dp.Protocol = remaining[1]
		}
	case "infra":
		dp.Layer = "infra"
	default:
		dp.Layer = remaining[0]
	}

	return dp
}

// ImportToDomainPath는 Go import 경로를 DomainPath로 변환한다.
// 외부 패키지는 nil을 반환한다.
func ImportToDomainPath(importPath, moduleName string) *DomainPath {
	if !strings.HasPrefix(importPath, moduleName+"/") {
		return nil
	}

	relPath := strings.TrimPrefix(importPath, moduleName+"/")
	return ParseDomainPath(relPath)
}
