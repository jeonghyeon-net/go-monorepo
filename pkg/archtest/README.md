# archtest

DDD 아키텍처 규칙을 AST 기반으로 자동 검증하는 공유 라이브러리.

## 사용법

서비스 루트에 `architecture_test.go`를 추가한다:

```go
package main_test

import (
    "path/filepath"
    "runtime"
    "testing"

    "go-monorepo/pkg/archtest"
)

func TestArchitecture(t *testing.T) {
    _, f, _, _ := runtime.Caller(0)
    archtest.RunAll(t, filepath.Dir(f))
}
```

`go test ./...`을 실행하면 6가지 아키텍처 규칙이 검증된다.

## 검증 항목

| 카테고리 | 검증 내용 |
|----------|-----------|
| Dependencies | import 방향이 DDD 레이어 규칙을 따르는지 |
| Naming | 패키지명, 타입명이 컨벤션을 따르는지 |
| InterfacePatterns | 인터페이스 + 구현체 + 생성자 3종 세트 |
| Structure | 디렉터리 구조가 정해진 형태인지 |
| Sqlc | sqlc 설정 완전성, 수기 코드 차단 |
| Testing | goleak 검출, 빌드 태그 강제 |

## 구조

```
archtest/
├── archtest.go           # RunAll(t, projectRoot) 진입점
├── analyzer/
│   ├── parser.go         # go/ast 기반 Go 파일 파서
│   └── domain.go         # 도메인 경로 파싱
├── ruleset/
│   ├── config.go         # Config 구조체 + 상수
│   ├── dependency.go     # 의존성 방향 규칙
│   ├── naming.go         # 네이밍 컨벤션 규칙
│   ├── interface_pattern.go  # 인터페이스 패턴 규칙
│   ├── structure.go      # 디렉터리 구조 규칙
│   ├── sqlc.go           # sqlc 코드 생성 규칙
│   └── testing.go        # 테스트 품질 규칙
└── report/
    └── violation.go      # 위반 보고 (ERROR/WARNING)
```

## 결과 처리

- **ERROR** 심각도 위반 → 테스트 FAIL
- **WARNING** 심각도 위반 → 테스트 PASS (로그에 경고 출력)
- 위반 없음 → 테스트 PASS
