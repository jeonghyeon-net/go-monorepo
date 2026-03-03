package ruleset

// 이 파일은 네이밍 규칙을 검사한다.
//
// 5가지 규칙:
//  1. 금지된 패키지명 (util, common, misc, helper 등)
//  2. 패키지 스터터 (타입명에 패키지명 반복: repo.AppRepo → repo.App)
//  3. Impl 접미사 금지 (UserServiceImpl → userService)
//  4. 파일명-인터페이스명 일치 (svc/install.go → Install 인터페이스)
//  5. 레이어 접미사 파일명 금지 (install_svc.go → install.go)

import (
	"fmt"
	"path/filepath"
	"strings"

	"go-monorepo/pkg/archtest/analyzer"
	"go-monorepo/pkg/archtest/report"
)

// CheckNaming은 모든 파일의 패키지명과 타입명이 네이밍 규칙을 지키는지 검사한다.
func CheckNaming(files []*analyzer.FileInfo, cfg *Config) []report.Violation {
	var violations []report.Violation

	for _, file := range files {
		if v := checkForbiddenPackageName(file); v != nil {
			violations = append(violations, *v)
		}

		for _, t := range file.Types {
			if v := checkPackageStutter(file, t); v != nil {
				violations = append(violations, *v)
			}
			if v := checkImplSuffix(file, t); v != nil {
				violations = append(violations, *v)
			}
		}

		relPath, err := filepath.Rel(cfg.ProjectRoot, file.Path)
		if err != nil {
			continue
		}
		dp := analyzer.ParseDomainPath(relPath)
		if dp != nil {
			if v := checkFileInterfaceMatch(file, dp); v != nil {
				violations = append(violations, *v)
			}
			if v := checkLayerSuffixFilename(file, dp); v != nil {
				violations = append(violations, *v)
			}
		}
	}

	return violations
}

// ──────────────────────────────────────────────
// 규칙 1: 금지된 패키지명
// ──────────────────────────────────────────────

// checkForbiddenPackageName은 패키지명이 금지 목록에 있는지 검사한다.
func checkForbiddenPackageName(file *analyzer.FileInfo) *report.Violation {
	for _, forbidden := range ForbiddenPackageNames {
		if strings.EqualFold(file.Package, forbidden) {
			return &report.Violation{
				Rule:     "naming/forbidden-package",
				Severity: report.Error,
				Message:  fmt.Sprintf("package name %q is forbidden", file.Package),
				File:     file.Path,
				Fix:      "rename package to something more specific and descriptive",
			}
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// 규칙 2: 패키지 스터터
// ──────────────────────────────────────────────

// checkPackageStutter는 타입 이름에 패키지 이름이 반복되는 것을 감지한다.
// 예: repo.AppRepo → repo.App
func checkPackageStutter(file *analyzer.FileInfo, typeInfo analyzer.TypeInfo) *report.Violation {
	if !typeInfo.IsExported {
		return nil
	}

	pkg := strings.ToLower(file.Package)
	nameLower := strings.ToLower(typeInfo.Name)

	// 접두사 스터터: RepoManager → Manager
	if len(typeInfo.Name) > len(pkg) && strings.HasPrefix(nameLower, pkg) {
		suggested := typeInfo.Name[len(pkg):]
		return &report.Violation{
			Rule:     "naming/package-stutter",
			Severity: report.Warning,
			Message:  fmt.Sprintf("type %q stutters with package name %q (%s.%s)", typeInfo.Name, file.Package, file.Package, typeInfo.Name),
			File:     file.Path,
			Line:     typeInfo.Line,
			Fix:      fmt.Sprintf("rename to %q (callers use %s.%s)", suggested, file.Package, suggested),
		}
	}

	// 접미사 스터터: AppRepo → App
	if len(typeInfo.Name) > len(pkg) && strings.HasSuffix(nameLower, pkg) {
		suggested := typeInfo.Name[:len(typeInfo.Name)-len(pkg)]
		return &report.Violation{
			Rule:     "naming/package-stutter",
			Severity: report.Warning,
			Message:  fmt.Sprintf("type %q stutters with package name %q (%s.%s)", typeInfo.Name, file.Package, file.Package, typeInfo.Name),
			File:     file.Path,
			Line:     typeInfo.Line,
			Fix:      fmt.Sprintf("rename to %q (callers use %s.%s)", suggested, file.Package, suggested),
		}
	}

	return nil
}

// ──────────────────────────────────────────────
// 규칙 3: Impl 접미사 금지
// ──────────────────────────────────────────────

// checkImplSuffix는 타입 이름이 "Impl"로 끝나는지 검사한다.
func checkImplSuffix(file *analyzer.FileInfo, typeInfo analyzer.TypeInfo) *report.Violation {
	if strings.HasSuffix(typeInfo.Name, "Impl") {
		return &report.Violation{
			Rule:     "naming/impl-suffix",
			Severity: report.Error,
			Message:  fmt.Sprintf("type %q has forbidden 'Impl' suffix", typeInfo.Name),
			File:     file.Path,
			Line:     typeInfo.Line,
			Fix:      "use an unexported name for the implementation struct",
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// 규칙 4: 파일명-인터페이스명 일치
// ──────────────────────────────────────────────

// checkFileInterfaceMatch는 svc/, repo/ 파일에 공개 인터페이스가 있을 때
// 파일명과 인터페이스명이 일치하는지 검사한다 (install.go → Install).
func checkFileInterfaceMatch(file *analyzer.FileInfo, dp *analyzer.DomainPath) *report.Violation {
	if dp.Layer != "svc" && dp.Layer != "repo" {
		return nil
	}

	hasExportedInterface := false
	for _, t := range file.Types {
		if t.IsInterface && t.IsExported {
			hasExportedInterface = true
			break
		}
	}
	if !hasExportedInterface {
		return nil
	}

	baseName := strings.TrimSuffix(filepath.Base(file.Path), ".go")
	expected := snakeToPascal(baseName)

	for _, t := range file.Types {
		if t.IsInterface && t.IsExported && t.Name == expected {
			return nil
		}
	}

	return &report.Violation{
		Rule:     "naming/file-interface-match",
		Severity: report.Warning,
		Message:  fmt.Sprintf("file %q should contain interface %q", filepath.Base(file.Path), expected),
		File:     file.Path,
		Fix:      fmt.Sprintf("rename file or interface so they match (e.g., %s.go → %s interface)", baseName, expected),
	}
}

// snakeToPascal은 snake_case를 PascalCase로 변환한다.
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// ──────────────────────────────────────────────
// 규칙 5: 레이어 접미사 파일명 금지
// ──────────────────────────────────────────────

// checkLayerSuffixFilename은 파일명에 레이어 이름이 접미사로 포함되었는지 검사한다.
// 디렉토리가 이미 레이어를 표현하므로 파일명에 중복 불필요.
func checkLayerSuffixFilename(file *analyzer.FileInfo, _ *analyzer.DomainPath) *report.Violation {
	baseName := strings.TrimSuffix(filepath.Base(file.Path), ".go")

	for _, layer := range AllowedSubdomainLayers {
		suffix := "_" + layer
		if cleaned, found := strings.CutSuffix(baseName, suffix); found {
			return &report.Violation{
				Rule:     "naming/layer-suffix-filename",
				Severity: report.Error,
				Message:  fmt.Sprintf("filename %q contains layer suffix %q", baseName+".go", suffix),
				File:     file.Path,
				Fix:      fmt.Sprintf("rename to %q (directory already expresses the layer)", cleaned+".go"),
			}
		}
	}

	return nil
}
