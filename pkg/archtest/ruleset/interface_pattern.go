package ruleset

// 이 파일은 인터페이스 패턴 규칙을 검사한다.
//
// "3종 세트" 패턴: exported interface + unexported impl + constructor
//
// 4가지 규칙:
//  1. 공개 인터페이스에 비공개 구현체가 같은 파일에 있어야 한다
//  2. 생성자(New*) 함수는 구체 타입이 아닌 인터페이스를 반환해야 한다
//  3. 구현체 struct는 unexported여야 한다
//  4. 인터페이스 + 구현체가 있으면 생성자도 있어야 한다

import (
	"fmt"
	"strings"
	"unicode"

	"go-monorepo/pkg/archtest/analyzer"
	"go-monorepo/pkg/archtest/report"
)

// CheckInterfacePatterns은 모든 파일의 인터페이스 패턴을 검사한다.
func CheckInterfacePatterns(files []*analyzer.FileInfo, cfg *Config) []report.Violation {
	//nolint:prealloc // 각 파일에서 반환되는 위반 수가 가변적이므로 사전 할당이 불가능하다.
	var violations []report.Violation

	for _, file := range files {
		violations = append(violations, checkFilePatterns(file)...)
	}

	return violations
}

// checkFilePatterns는 단일 파일의 인터페이스 패턴 규칙을 모두 검사한다.
func checkFilePatterns(file *analyzer.FileInfo) []report.Violation {
	var violations []report.Violation

	var interfaces []analyzer.TypeInfo
	var structs []analyzer.TypeInfo

	for _, t := range file.Types {
		if t.IsInterface {
			interfaces = append(interfaces, t)
		} else {
			structs = append(structs, t)
		}
	}

	// 규칙 1 + 4: 공개 인터페이스에 대응하는 비공개 구현체와 생성자 검사
	for _, iface := range interfaces {
		if !iface.IsExported {
			continue
		}
		if v := checkMissingImpl(file, iface, structs); v != nil {
			violations = append(violations, *v)
			continue
		}
		if v := checkMissingConstructor(file, iface); v != nil {
			violations = append(violations, *v)
		}
	}

	// 규칙 2: 생성자가 인터페이스를 반환하는지 검사
	for _, fn := range file.Functions {
		if v := checkConstructorReturn(file, fn, interfaces); v != nil {
			violations = append(violations, *v)
		}
	}

	// 규칙 3: 구현체 struct가 exported되어 있지 않은지 검사
	for _, s := range structs {
		if s.IsExported {
			if v := checkExportedImpl(file, s, interfaces); v != nil {
				violations = append(violations, *v)
			}
		}
	}

	return violations
}

// ──────────────────────────────────────────────
// 규칙 1: 공개 인터페이스에 비공개 구현체가 있어야 한다
// ──────────────────────────────────────────────

// checkMissingImpl은 공개 인터페이스에 대응하는 비공개 구현체가 있는지 확인한다.
func checkMissingImpl(file *analyzer.FileInfo, iface analyzer.TypeInfo, structs []analyzer.TypeInfo) *report.Violation {
	for _, s := range structs {
		if !s.IsExported {
			return nil
		}
	}

	return &report.Violation{
		Rule:     "interface/missing-impl",
		Severity: report.Warning,
		Message:  fmt.Sprintf("exported interface %q has no unexported implementation in the same file", iface.Name),
		File:     file.Path,
		Line:     iface.Line,
		Fix:      fmt.Sprintf("add unexported struct %q implementing %s in the same file", toLowerFirst(iface.Name), iface.Name),
	}
}

// ──────────────────────────────────────────────
// 규칙 4: 3종 세트 완전성 — 생성자가 존재해야 한다
// ──────────────────────────────────────────────

// checkMissingConstructor는 공개 인터페이스 + 비공개 구현체가 있을 때 생성자도 있는지 확인한다.
func checkMissingConstructor(file *analyzer.FileInfo, iface analyzer.TypeInfo) *report.Violation {
	for _, fn := range file.Functions {
		if !fn.IsExported || !strings.HasPrefix(fn.Name, "New") || fn.Receiver != "" {
			continue
		}
		if len(fn.ReturnTypes) == 0 {
			continue
		}

		firstReturn := strings.TrimPrefix(fn.ReturnTypes[0], "*")
		if firstReturn == iface.Name {
			return nil
		}
	}

	return &report.Violation{
		Rule:     "interface/missing-constructor",
		Severity: report.Warning,
		Message:  fmt.Sprintf("exported interface %q has implementation but no constructor (New*) returning it", iface.Name),
		File:     file.Path,
		Line:     iface.Line,
		Fix:      fmt.Sprintf("add constructor: func New() %s { return &%s{} }", iface.Name, toLowerFirst(iface.Name)),
	}
}

// ──────────────────────────────────────────────
// 규칙 2: 생성자는 인터페이스를 반환해야 한다
// ──────────────────────────────────────────────

// checkConstructorReturn은 New* 함수가 인터페이스를 반환하는지 검사한다.
func checkConstructorReturn(file *analyzer.FileInfo, fn analyzer.FuncInfo, interfaces []analyzer.TypeInfo) *report.Violation {
	if !fn.IsExported || !strings.HasPrefix(fn.Name, "New") || fn.Receiver != "" {
		return nil
	}
	if len(fn.ReturnTypes) == 0 || len(interfaces) == 0 {
		return nil
	}

	firstReturn := strings.TrimPrefix(fn.ReturnTypes[0], "*")

	for _, iface := range interfaces {
		if iface.Name == firstReturn {
			return nil
		}
	}

	for _, iface := range interfaces {
		if !iface.IsExported {
			continue
		}
		if strings.EqualFold(toLowerFirst(iface.Name), firstReturn) {
			return &report.Violation{
				Rule:     "interface/constructor-return",
				Severity: report.Error,
				Message:  fmt.Sprintf("constructor %q returns concrete type %q instead of interface %q", fn.Name, firstReturn, iface.Name),
				File:     file.Path,
				Line:     fn.Line,
				Fix:      "change return type to " + iface.Name,
			}
		}
	}

	return nil
}

// ──────────────────────────────────────────────
// 규칙 3: 구현체 struct는 unexported여야 한다
// ──────────────────────────────────────────────

// checkExportedImpl은 공개된 struct가 인터페이스의 구현체처럼 보이는지 감지한다.
// {인터페이스명}Impl 또는 Default{인터페이스명} 패턴을 감지.
func checkExportedImpl(file *analyzer.FileInfo, structInfo analyzer.TypeInfo, interfaces []analyzer.TypeInfo) *report.Violation {
	for _, iface := range interfaces {
		if !iface.IsExported {
			continue
		}
		if structInfo.Name == iface.Name+"Impl" || structInfo.Name == "Default"+iface.Name {
			return &report.Violation{
				Rule:     "interface/exported-impl",
				Severity: report.Warning,
				Message:  fmt.Sprintf("struct %q appears to implement %q but is exported", structInfo.Name, iface.Name),
				File:     file.Path,
				Line:     structInfo.Line,
				Fix:      fmt.Sprintf("make the implementation struct unexported: %q", toLowerFirst(structInfo.Name)),
			}
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// 유틸리티 함수
// ──────────────────────────────────────────────

// toLowerFirst는 문자열의 첫 글자를 소문자로 바꾼다.
func toLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
