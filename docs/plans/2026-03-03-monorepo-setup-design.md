# Go 모노레포 환경 설계

## 개요

다양한 유형의 Go 서비스(REST API, gRPC, CLI, worker 등)를 하나의 저장소에서 관리하기 위한 모노레포 환경을 구성한다. 공유 패키지를 여러 서비스에서 재사용할 수 있도록 한다.

## 결정 사항

- **방식**: Go Workspace (`go.work`) 기반 멀티모듈 모노레포
- **범위**: 환경 구성만 (기존 rest-api 프로젝트는 이동하지 않음)
- **CI/CD**: 이번 범위에 포함하지 않음

## 디렉터리 구조

```
go-monorepo/
├── go.work                  # Go workspace 정의
├── Makefile                 # 전체 서비스 오케스트레이션
├── .golangci.yml            # 공유 린트 설정
├── .mise.toml               # 도구 버전 통일
├── .gitignore               # 공통 무시 패턴
├── services/                # 서비스 디렉터리 (각 서비스가 독립 go.mod 보유)
└── pkg/                     # 공유 패키지 모듈 (독립 go.mod)
```

## Go Workspace

- `go.work`와 `go.work.sum`을 git에 커밋한다.
- 서비스나 공유 패키지를 추가할 때마다 `use` 디렉티브에 등록한다.
- 모듈 네이밍: `go-monorepo/services/<name>`, `go-monorepo/pkg`

## 루트 Makefile

전체 서비스를 순회하는 오케스트레이션 명령과, 특정 서비스만 지정하는 명령을 제공한다.

- `make build` / `make test` / `make lint` / `make fmt`: 전체 서비스 대상
- `make svc-<target> SVC=<name>`: 특정 서비스 대상

각 서비스는 자체 Makefile을 보유한다.

## 공유 설정

- `.golangci.yml`: rest-api의 설정을 기반으로 루트에 배치, 전체 서비스 공유
- `.mise.toml`: Go, golangci-lint, sqlc, goose, nilaway 등 도구 버전 통일
- `.gitignore`: tmp/, .env, coverage.*, *.db, .DS_Store 등

## 접근 방식 선택 근거

- **Go Workspace vs 단일 모듈**: 다양한 서비스 유형을 수용하려면 서비스별 독립 의존성 관리가 필수. 단일 모듈은 불필요한 의존성이 전체에 전파됨.
- **Go Workspace vs Bazel**: Bazel은 대규모(수백 서비스)에서 유리하나, 소~중규모에서는 학습 곡선과 설정 복잡도가 부담. Go 공식 도구 체인을 그대로 활용 가능한 Go Workspace가 적합.
