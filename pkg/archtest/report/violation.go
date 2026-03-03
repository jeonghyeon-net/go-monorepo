// Package report는 아키텍처 규칙 위반 사항을 표현하고 보고하는 패키지다.
package report

import (
	"fmt"
	"strconv"
	"strings"
)

// Severity는 위반의 심각도를 나타내는 타입이다.
type Severity string

const (
	// Error는 반드시 수정해야 하는 위반이다. 테스트가 FAIL한다.
	Error Severity = "ERROR"
	// Warning은 권장 사항 위반이다. 로그에 표시되지만 테스트는 PASS한다.
	Warning Severity = "WARNING"
)

// Violation은 아키텍처 규칙 위반 하나를 나타내는 구조체다.
type Violation struct {
	Rule     string   // 위반한 규칙 이름
	Message  string   // 위반 내용 설명
	File     string   // 위반 파일 경로
	Fix      string   // 수정 안내
	Severity Severity // 심각도
	Line     int      // 위반 줄 번호 (0이면 없음)
}

// String은 Violation을 사람이 읽기 좋은 문자열로 변환한다.
func (v Violation) String() string {
	result := fmt.Sprintf("[%s] %s: %s\n  file: %s", v.Severity, v.Rule, v.Message, v.File)

	if v.Line > 0 {
		result += ":" + strconv.Itoa(v.Line)
	}

	if v.Fix != "" {
		result += "\n  fix: " + v.Fix
	}

	return result
}

// Summary는 모든 위반 사항의 요약 리포트를 생성한다.
func Summary(violations []Violation) string {
	if len(violations) == 0 {
		return "No architecture violations found."
	}

	errors := 0
	warnings := 0
	byRule := make(map[string]int)
	for _, v := range violations {
		if v.Severity == Error {
			errors++
		} else {
			warnings++
		}
		byRule[v.Rule]++
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Architecture violations: %d error(s), %d warning(s)\n\n", errors, warnings)

	for rule, count := range byRule {
		fmt.Fprintf(&builder, "  %s: %d violation(s)\n", rule, count)
	}
	fmt.Fprintln(&builder)

	for _, v := range violations {
		fmt.Fprintln(&builder, v.String())
		fmt.Fprintln(&builder)
	}

	return builder.String()
}

// HasErrors는 위반 목록에 Error 심각도가 하나라도 있으면 true를 반환한다.
func HasErrors(violations []Violation) bool {
	for _, v := range violations {
		if v.Severity == Error {
			return true
		}
	}
	return false
}
