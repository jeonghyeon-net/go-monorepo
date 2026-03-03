# 공유 아키텍처 테스트 라이브러리 설계

## 개요

rest-api의 `test/architecture/` 아키텍처 테스트 프레임워크를 모노레포의 공유 패키지(`pkg/archtest/`)로 이동하여, 모든 서비스가 동일한 아키텍처 규칙을 적용받도록 한다.

## 결정 사항

- **위치**: `pkg/archtest/` (공유 패키지 모듈 내)
- **규칙 적용**: 모든 서비스에 동일한 규칙 적용 (서비스별 선택 없음)
- **사용 방식**: 각 서비스의 `arch_test.go`에서 `archtest.RunAll(t, root)` 호출

## 구조

```
pkg/
├── go.mod                              # module go-monorepo/pkg
├── go.sum
└── archtest/
    ├── archtest.go                     # RunAll(t, projectRoot) 진입점
    ├── analyzer/
    │   ├── parser.go                   # AST 파서
    │   └── domain.go                   # 도메인 경로 파싱
    ├── ruleset/
    │   ├── config.go                   # Config + 상수 정의
    │   ├── dependency.go               # 의존성 방향 규칙
    │   ├── naming.go                   # 네이밍 컨벤션
    │   ├── interface_pattern.go        # 인터페이스 패턴
    │   ├── structure.go                # 디렉터리 구조
    │   ├── sqlc.go                     # sqlc 코드 생성
    │   └── testing.go                  # 테스트 품질
    └── report/
        └── violation.go                # 위반 보고
```

## 핵심 변경점

1. **`archtest.go` 진입점 추가**: rest-api의 `arch_test.go` 로직을 `RunAll(t, projectRoot)` 함수로 변환
2. **import 경로 변경**: `rest-api/test/architecture/*` → `go-monorepo/pkg/archtest/*`
3. **로직 변경 없음**: 모든 ruleset, analyzer, report 코드는 그대로 유지
4. **go.work 업데이트**: `use ./pkg` 추가
