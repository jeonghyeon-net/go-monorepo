# ─────────────────────────────────────────────────────────────────────────────
# makefile — 모노레포 전체 서비스에 대한 오케스트레이션 명령을 제공한다.
# ─────────────────────────────────────────────────────────────────────────────
#
# 사용법:
#   make build [SVC=hello]  — 전체 빌드 (SVC 지정 시 해당 서비스만)
#   make test [SVC=hello]   — 전체 테스트 (SVC 지정 시 해당 서비스만)
#   make lint [SVC=hello]   — 전체 린트 (SVC 지정 시 해당 서비스만)
#   make fmt [SVC=hello]    — 전체 포맷 (SVC 지정 시 해당 서비스만)
#   make dev [SVC=hello]    — air로 서비스 개발 실행
#   make docker-build [SVC=hello] [PLATFORM=linux/arm64] — 서비스 이미지 빌드 (SVC 지정 시 해당 서비스만)
#   make setup              — 개발 환경 초기 설정
.PHONY: build test lint fmt dev setup docker-build

# mise가 관리하는 도구(golangci-lint, nilaway 등)의 PATH를 보장한다.
ifeq ($(NO_MISE),1)
SHELL := /bin/sh
else
SHELL := mise exec -- bash
endif
override IMAGE_PREFIX := go-monorepo
override IMAGE_TAG := latest

# ── 전체 대상 명령 ─────────────────────────────────────────────────────────────

# go.work에 등록된 모든 모듈(공유 패키지 + 서비스)을 빌드한다.
# go.work의 use 지시문에서 모듈 경로를 자동 추출한다.
build:
	@dirs="$(if $(SVC),./svc/$(SVC),$$(awk '/^[[:space:]]*\.\//{gsub(/^[[:space:]]+/,""); print}' go.work))"; \
	for dir in $$dirs; do \
			case "$$dir" in \
				./svc/*) \
					svc=$${dir#./svc/}; \
					if [ ! -d "$(CURDIR)/$$dir/cmd" ]; then echo "$$dir/cmd 디렉터리를 찾을 수 없습니다."; exit 1; fi; \
					mkdir -p $(CURDIR)/bin; \
					cd $(CURDIR)/$$dir && CGO_ENABLED=0 go build -trimpath -buildvcs=false -mod=readonly -tags timetzdata -ldflags="-s -w" -o $(CURDIR)/bin/$$svc ./cmd/ \
				;; \
				*) \
					cd $(CURDIR)/$$dir && go build ./... \
			;; \
		esac; \
	done

# go.work에 등록된 모든 모듈의 테스트를 실행한다.
test:
	@dirs="$(if $(SVC),./svc/$(SVC),$$(awk '/^[[:space:]]*\.\//{gsub(/^[[:space:]]+/,""); print}' go.work))"; \
	for dir in $$dirs; do \
			echo "=== $$dir ===" && cd $(CURDIR)/$$dir && go test ./... 2>&1 | grep -v '\[no test files\]'; test $${PIPESTATUS[0]} -eq 0; \
	done

# go.work에 등록된 모든 모듈에 golangci-lint와 nilaway를 실행한다.
lint:
	@dirs="$(if $(SVC),./svc/$(SVC),$$(awk '/^[[:space:]]*\.\//{gsub(/^[[:space:]]+/,""); print}' go.work))"; \
	for dir in $$dirs; do \
			echo "=== $$dir ===" && cd $(CURDIR)/$$dir && golangci-lint run --fix ./... && nilaway ./...; \
	done

# go.work에 등록된 모든 모듈의 코드를 포맷한다.
fmt:
	@dirs="$(if $(SVC),./svc/$(SVC),$$(awk '/^[[:space:]]*\.\//{gsub(/^[[:space:]]+/,""); print}' go.work))"; \
	for dir in $$dirs; do \
			cd $(CURDIR)/$$dir && gofmt -w . && golangci-lint fmt; \
	done

# air로 서비스를 핫리로드 실행한다.
dev:
	@svc=$(if $(SVC),$(SVC),hello); \
	if [ ! -d "$(CURDIR)/svc/$$svc/cmd" ]; then echo "svc/$$svc/cmd 디렉터리를 찾을 수 없습니다."; exit 1; fi; \
	SVC=$$svc air -c air.toml

# go.work에 등록된 모든 서비스 모듈의 컨테이너 이미지를 빌드한다.
# 사용법:
#   make docker-build [SVC=hello] PLATFORM=linux/arm64
docker-build:
	@dirs="$(if $(SVC),./svc/$(SVC),$$(awk '/^[[:space:]]*\.\/svc\//{gsub(/^[[:space:]]+\.\//,""); print}' go.work))"; \
	for dir in $$dirs; do \
		service=$${dir#svc/}; \
		service=$${service#./svc/}; \
		echo "=== $$service ==="; \
		docker buildx build \
			--platform $(if $(PLATFORM),$(PLATFORM),linux/arm64) \
			-f dockerfile \
			--build-arg SERVICE=$$service \
			-t $(IMAGE_PREFIX)/$$service:$(IMAGE_TAG) \
			.; \
	done

# ── 프로젝트 초기 설정 ────────────────────────────────────────────────────────

# 새로 프로젝트를 클론한 후 한 번 실행하는 초기 설정 명령이다.
# mise install: .mise.toml에 정의된 도구들을 설치한다.
setup:
	mise install
	git config core.hooksPath .githooks
