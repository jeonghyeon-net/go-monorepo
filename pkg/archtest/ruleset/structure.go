package ruleset

// 이 파일은 디렉토리 구조 규칙을 검사한다.
//
// 7가지 규칙:
//  1. 모든 도메인에 alias.go 존재 여부
//  2. 도메인 하위 디렉토리: subdomain, svc, handler, infra만 허용
//  3. 서브도메인 레이어: model, repo, svc만 허용
//  4. 핸들러 프로토콜: http, grpc, jsonrpc만 허용
//  5. subdomain/ 루트에 파일 존재 금지
//  6. saga/ 루트에 파일 존재 금지
//  7. Saga 하위 디렉토리 금지 (단일 패키지 강제)

import (
	"fmt"
	"os"
	"path/filepath"

	"go-monorepo/pkg/archtest/report"
)

// CheckStructure는 internal/domain/과 internal/saga/의 디렉토리 구조를 검사한다.
func CheckStructure(cfg *Config) []report.Violation {
	var violations []report.Violation

	domainRoot := filepath.Join(cfg.ProjectRoot, "internal", "domain")
	if _, err := os.Stat(domainRoot); err == nil {
		entries, err := os.ReadDir(domainRoot)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				domainName := entry.Name()
				domainDir := filepath.Join(domainRoot, domainName)

				violations = append(violations, checkAliasFile(domainDir, domainName)...)
				violations = append(violations, checkDomainDirs(domainDir, domainName)...)
				violations = append(violations, checkSubdomains(domainDir, domainName)...)
				violations = append(violations, checkHandlerProtocols(domainDir, domainName)...)
			}
		}
	}

	violations = append(violations, checkSagaStructure(cfg)...)

	return violations
}

// ──────────────────────────────────────────────
// 규칙 1: alias.go 존재 여부
// ──────────────────────────────────────────────

// checkAliasFile은 도메인 루트에 alias.go가 있는지 검사한다.
// alias.go는 도메인의 공개 API를 정의하는 유일한 외부 진입점이다.
func checkAliasFile(domainDir, domainName string) []report.Violation {
	aliasPath := filepath.Join(domainDir, "alias.go")
	if _, err := os.Stat(aliasPath); os.IsNotExist(err) {
		return []report.Violation{{
			Rule:     "structure/missing-alias",
			Severity: report.Warning,
			Message:  fmt.Sprintf("domain %q is missing alias.go", domainName),
			File:     domainDir,
			Fix:      "create alias.go with type aliases for public interfaces",
		}}
	}
	return nil
}

// ──────────────────────────────────────────────
// 규칙 2: 허용된 도메인 하위 디렉토리
// ──────────────────────────────────────────────

// checkDomainDirs는 도메인 디렉토리 안에 허용된 디렉토리만 있는지 검사한다.
func checkDomainDirs(domainDir, domainName string) []report.Violation {
	var violations []report.Violation

	entries, err := os.ReadDir(domainDir)
	if err != nil {
		return nil
	}

	allowed := make(map[string]bool)
	for _, d := range AllowedDomainDirs {
		allowed[d] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !allowed[entry.Name()] {
			violations = append(violations, report.Violation{
				Rule:     "structure/invalid-domain-dir",
				Severity: report.Error,
				Message:  fmt.Sprintf("domain %q contains unexpected directory %q", domainName, entry.Name()),
				File:     filepath.Join(domainDir, entry.Name()),
				Fix:      fmt.Sprintf("allowed directories: %v", AllowedDomainDirs),
			})
		}
	}

	return violations
}

// ──────────────────────────────────────────────
// 규칙 3: 서브도메인 레이어 구조
// ──────────────────────────────────────────────

// checkSubdomains는 각 서브도메인 안에 허용된 레이어 디렉토리만 있는지 검사한다.
func checkSubdomains(domainDir, domainName string) []report.Violation {
	var violations []report.Violation

	subdomainDir := filepath.Join(domainDir, "subdomain")
	if _, err := os.Stat(subdomainDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(subdomainDir)
	if err != nil {
		return nil
	}

	allowed := make(map[string]bool)
	for _, l := range AllowedSubdomainLayers {
		allowed[l] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			violations = append(violations, report.Violation{
				Rule:     "structure/file-in-subdomain-root",
				Severity: report.Error,
				Message:  fmt.Sprintf("unexpected file in subdomain directory of domain %q", domainName),
				File:     filepath.Join(subdomainDir, entry.Name()),
				Fix:      "files should be in subdomain layer directories (model/, repo/, svc/)",
			})
			continue
		}

		subDir := filepath.Join(subdomainDir, entry.Name())
		layerEntries, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}

		for _, le := range layerEntries {
			if !le.IsDir() {
				continue
			}
			if !allowed[le.Name()] {
				violations = append(violations, report.Violation{
					Rule:     "structure/invalid-subdomain-layer",
					Severity: report.Error,
					Message:  fmt.Sprintf("subdomain %q in domain %q contains unexpected directory %q", entry.Name(), domainName, le.Name()),
					File:     filepath.Join(subDir, le.Name()),
					Fix:      fmt.Sprintf("allowed layers: %v", AllowedSubdomainLayers),
				})
			}
		}
	}

	return violations
}

// ──────────────────────────────────────────────
// 규칙 5,6,7: Saga 디렉토리 구조
// ──────────────────────────────────────────────

// checkSagaStructure는 internal/saga/ 디렉토리의 구조를 검사한다.
// Saga는 단일 패키지로 유지해야 한다. 하위 디렉토리가 필요하면 도메인으로 분리.
func checkSagaStructure(cfg *Config) []report.Violation {
	var violations []report.Violation

	sagaRoot := filepath.Join(cfg.ProjectRoot, "internal", "saga")
	if _, err := os.Stat(sagaRoot); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(sagaRoot)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			violations = append(violations, report.Violation{
				Rule:     "structure/file-in-saga-root",
				Severity: report.Error,
				Message:  "unexpected file in saga root directory",
				File:     filepath.Join(sagaRoot, entry.Name()),
				Fix:      "files should be inside a saga directory (e.g., internal/saga/create_order/saga.go)",
			})
			continue
		}

		sagaDir := filepath.Join(sagaRoot, entry.Name())
		subEntries, err := os.ReadDir(sagaDir)
		if err != nil {
			continue
		}

		for _, se := range subEntries {
			if se.IsDir() {
				violations = append(violations, report.Violation{
					Rule:     "structure/saga-nested-dir",
					Severity: report.Error,
					Message:  fmt.Sprintf("saga %q contains nested directory %q", entry.Name(), se.Name()),
					File:     filepath.Join(sagaDir, se.Name()),
					Fix:      "sagas must be a single flat package; if complex, consider extracting into a domain",
				})
			}
		}
	}

	return violations
}

// ──────────────────────────────────────────────
// 규칙 4: 핸들러 프로토콜 디렉토리
// ──────────────────────────────────────────────

// checkHandlerProtocols는 handler/ 안에 허용된 프로토콜 디렉토리만 있는지 검사한다.
func checkHandlerProtocols(domainDir, domainName string) []report.Violation {
	var violations []report.Violation

	handlerDir := filepath.Join(domainDir, "handler")
	if _, err := os.Stat(handlerDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(handlerDir)
	if err != nil {
		return nil
	}

	allowed := make(map[string]bool)
	for _, p := range AllowedHandlerProtocols {
		allowed[p] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !allowed[entry.Name()] {
			violations = append(violations, report.Violation{
				Rule:     "structure/invalid-handler-protocol",
				Severity: report.Error,
				Message:  fmt.Sprintf("handler in domain %q contains unexpected protocol %q", domainName, entry.Name()),
				File:     filepath.Join(handlerDir, entry.Name()),
				Fix:      fmt.Sprintf("allowed protocols: %v", AllowedHandlerProtocols),
			})
		}
	}

	return violations
}
