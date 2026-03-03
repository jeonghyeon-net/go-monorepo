# syntax=docker/dockerfile:1.7
FROM golang:1.26-alpine AS build
ARG SERVICE
WORKDIR /src

RUN apk add --no-cache make bash

COPY svc/${SERVICE}/go.mod ./svc/${SERVICE}/go.mod
RUN --mount=type=cache,target=/go/pkg/mod \
    cd svc/${SERVICE} && GOWORK=off go mod download

COPY makefile go.work go.work.sum ./
COPY pkg/ ./pkg/
COPY svc/${SERVICE}/ ./svc/${SERVICE}/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    NO_MISE=1 make build SVC=${SERVICE} && cp /src/bin/${SERVICE} /app

FROM gcr.io/distroless/static-debian12:nonroot
ENV TZ=UTC
COPY --from=build /app /app
ENTRYPOINT ["/app"]
