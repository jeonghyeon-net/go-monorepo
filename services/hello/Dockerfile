FROM golang:1.26-alpine AS build
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd services/hello && CGO_ENABLED=0 go build -o /app ./cmd/

FROM gcr.io/distroless/static
COPY --from=build /app /app
ENTRYPOINT ["/app"]
