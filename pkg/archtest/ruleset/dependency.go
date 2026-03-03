package ruleset

// 이 파일은 의존성 규칙을 검사한다.
//
// 8가지 규칙:
//  1. 서브도메인 간 직접 의존 금지 (예외: core는 허용)
//  2. 서브도메인에서 도메인 레이어(svc/, handler/, infra/) import 금지
//  3. 도메인 간 직접 의존 금지 (alias.go만 허용)
//  4. 서브도메인에서 다른 도메인 직접 import 금지
//  5. 레이어 역방향 의존 금지 (model ← repo ← svc)
//  6. Saga가 도메인 내부 import 금지 (alias.go만 허용)
//  7. Saga 간 직접 의존 금지
//  8. 서브도메인에서 Saga import 금지

import (
	"fmt"
	"path/filepath"

	"go-monorepo/pkg/archtest/analyzer"
	"go-monorepo/pkg/archtest/report"
)

const layerRoot = "root"

// CheckDependencies는 모든 파일의 import문을 검사해서 의존성 규칙 위반을 찾는다.
func CheckDependencies(files []*analyzer.FileInfo, cfg *Config) []report.Violation {
	var violations []report.Violation

	for _, file := range files {
		relPath, err := filepath.Rel(cfg.ProjectRoot, file.Path)
		if err != nil {
			continue
		}

		src := analyzer.ParseDomainPath(relPath)
		if src == nil {
			continue
		}

		for _, imp := range file.Imports {
			target := analyzer.ImportToDomainPath(imp.Path, cfg.ModuleName)
			if target == nil {
				continue
			}

			if v := checkSagaDependency(src, target, file.Path, imp); v != nil {
				violations = append(violations, *v)
				continue
			}
			if v := checkSubdomainImportsSaga(src, target, file.Path, imp); v != nil {
				violations = append(violations, *v)
				continue
			}

			if v := checkCrossSubdomain(src, target, file.Path, imp); v != nil {
				violations = append(violations, *v)
			}
			if v := checkSubdomainImportsDomainLayer(src, target, file.Path, imp); v != nil {
				violations = append(violations, *v)
			}
			if v := checkCrossDomain(src, target, file.Path, imp); v != nil {
				violations = append(violations, *v)
			}
			if v := checkLayerDirection(src, target, file.Path, imp); v != nil {
				violations = append(violations, *v)
			}
		}
	}

	return violations
}

// ──────────────────────────────────────────────
// 규칙 6,7: Saga 의존성 규칙
// ──────────────────────────────────────────────

// checkSagaDependency는 Saga 파일의 import 대상을 검사한다.
// Saga는 도메인 root(alias.go)만 import 가능. Saga 간 의존도 금지.
func checkSagaDependency(src, target *analyzer.DomainPath, file string, imp analyzer.ImportInfo) *report.Violation {
	if !src.IsSaga {
		return nil
	}

	if target.IsSaga {
		return &report.Violation{
			Rule:     "dependency/saga-cross-saga",
			Severity: report.Error,
			Message:  fmt.Sprintf("saga %q imports saga %q directly", src.SagaName, target.SagaName),
			File:     file,
			Line:     imp.Line,
			Fix:      "sagas should be independent; extract shared logic into a domain service",
		}
	}

	if target.Domain != "" {
		if target.Layer == layerRoot {
			return nil
		}

		return &report.Violation{
			Rule:     "dependency/saga-internal-import",
			Severity: report.Error,
			Message:  fmt.Sprintf("saga %q imports internal package of domain %q (layer: %s)", src.SagaName, target.Domain, target.Layer),
			File:     file,
			Line:     imp.Line,
			Fix:      fmt.Sprintf("import %q (alias.go) instead of accessing internal packages directly", target.Domain),
		}
	}

	return nil
}

// ──────────────────────────────────────────────
// 규칙 8: 서브도메인에서 Saga import 금지
// ──────────────────────────────────────────────

// checkSubdomainImportsSaga는 서브도메인이 Saga를 import하는 것을 감지한다.
// Saga 호출은 Public Service(svc/) 레이어에서만 가능하다.
func checkSubdomainImportsSaga(src, target *analyzer.DomainPath, file string, imp analyzer.ImportInfo) *report.Violation {
	if src.Subdomain == "" || !target.IsSaga {
		return nil
	}

	return &report.Violation{
		Rule:     "dependency/subdomain-imports-saga",
		Severity: report.Error,
		Message:  fmt.Sprintf("subdomain %q in domain %q imports saga %q", src.Subdomain, src.Domain, target.SagaName),
		File:     file,
		Line:     imp.Line,
		Fix:      "subdomains cannot depend on sagas; move saga invocation to Public Service layer",
	}
}

// ──────────────────────────────────────────────
// 규칙 1: 서브도메인 간 직접 의존 금지
// ──────────────────────────────────────────────

// checkCrossSubdomain은 같은 도메인 내에서 서브도메인 간 직접 import를 감지한다.
// 같은 서브도메인 내 import와 core 서브도메인은 허용.
func checkCrossSubdomain(src, target *analyzer.DomainPath, file string, imp analyzer.ImportInfo) *report.Violation {
	if src.IsSaga || target.IsSaga {
		return nil
	}
	if src.Domain != target.Domain {
		return nil
	}
	if src.Subdomain == "" || target.Subdomain == "" {
		return nil
	}
	if src.Subdomain == target.Subdomain {
		return nil
	}
	if target.Subdomain == "core" {
		return nil
	}

	return &report.Violation{
		Rule:     "dependency/cross-subdomain",
		Severity: report.Error,
		Message:  fmt.Sprintf("subdomain %q imports subdomain %q directly", src.Subdomain, target.Subdomain),
		File:     file,
		Line:     imp.Line,
		Fix:      "use Public Service or move shared logic to core/",
	}
}

// ──────────────────────────────────────────────
// 규칙 2: 서브도메인에서 도메인 레이어 import 금지
// ──────────────────────────────────────────────

// checkSubdomainImportsDomainLayer는 서브도메인이 같은 도메인의 상위 레이어를 import하는 것을 감지한다.
// domain root(alias.go)만 허용.
func checkSubdomainImportsDomainLayer(src, target *analyzer.DomainPath, file string, imp analyzer.ImportInfo) *report.Violation {
	if src.IsSaga || target.IsSaga {
		return nil
	}
	if src.Subdomain == "" {
		return nil
	}
	if src.Domain != target.Domain {
		return nil
	}
	if target.Subdomain != "" {
		return nil
	}
	if target.Layer == layerRoot {
		return nil
	}

	return &report.Violation{
		Rule:     "dependency/subdomain-imports-domain-layer",
		Severity: report.Error,
		Message:  fmt.Sprintf("subdomain %q in domain %q imports domain-level %s/", src.Subdomain, src.Domain, target.Layer),
		File:     file,
		Line:     imp.Line,
		Fix:      "subdomains cannot depend on domain-level layers (svc/, handler/, infra/)",
	}
}

// ──────────────────────────────────────────────
// 규칙 3,4: 도메인 간 직접 의존 금지
// ──────────────────────────────────────────────

// checkCrossDomain은 다른 도메인의 내부 패키지를 직접 import하는 것을 감지한다.
// 다른 도메인은 반드시 alias.go(root)를 통해서만 접근해야 한다.
func checkCrossDomain(src, target *analyzer.DomainPath, file string, imp analyzer.ImportInfo) *report.Violation {
	if src.IsSaga || target.IsSaga {
		return nil
	}
	if src.Domain == target.Domain {
		return nil
	}
	if target.Layer == layerRoot {
		return nil
	}

	if src.Subdomain != "" {
		return &report.Violation{
			Rule:     "dependency/cross-domain-from-subdomain",
			Severity: report.Error,
			Message:  fmt.Sprintf("subdomain %q in domain %q imports domain %q directly", src.Subdomain, src.Domain, target.Domain),
			File:     file,
			Line:     imp.Line,
			Fix:      "subdomains cannot depend on other domains; move dependency to Public Service layer",
		}
	}

	return &report.Violation{
		Rule:     "dependency/cross-domain",
		Severity: report.Error,
		Message:  fmt.Sprintf("domain %q imports internal package of domain %q (layer: %s)", src.Domain, target.Domain, target.Layer),
		File:     file,
		Line:     imp.Line,
		Fix:      fmt.Sprintf("import %q (alias.go) instead of accessing internal packages directly", target.Domain),
	}
}

// ──────────────────────────────────────────────
// 규칙 5: 레이어 역방향 의존 금지
// ──────────────────────────────────────────────

// checkLayerDirection은 같은 서브도메인 내에서 레이어 의존 방향을 검사한다.
// model(0) ← repo(1) ← svc(2) 방향만 허용.
func checkLayerDirection(src, target *analyzer.DomainPath, file string, imp analyzer.ImportInfo) *report.Violation {
	if src.IsSaga || target.IsSaga {
		return nil
	}
	if src.Domain != target.Domain || src.Subdomain != target.Subdomain {
		return nil
	}
	if src.Subdomain == "" {
		return nil
	}

	srcIdx := layerIndex(src.Layer)
	targetIdx := layerIndex(target.Layer)
	if srcIdx < 0 || targetIdx < 0 {
		return nil
	}

	if srcIdx < targetIdx {
		return &report.Violation{
			Rule:     "dependency/layer-direction",
			Severity: report.Error,
			Message:  fmt.Sprintf("layer %q cannot depend on layer %q (reverse dependency)", src.Layer, target.Layer),
			File:     file,
			Line:     imp.Line,
			Fix:      "dependency direction must be: model <- repo <- svc",
		}
	}

	return nil
}

// layerIndex는 레이어 이름을 LayerOrder에서의 인덱스로 변환한다.
func layerIndex(layer string) int {
	for i, l := range LayerOrder {
		if l == layer {
			return i
		}
	}
	return -1
}
