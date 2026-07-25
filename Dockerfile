# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

# Cache module downloads separately from the source.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Static, stripped binary — CGO off so it runs on a scratch base.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags='-s -w' -o /out/github-exporter ./cmd

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/github-exporter /usr/local/bin/github-exporter

EXPOSE 9101
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/github-exporter"]
