ARG GO_VERSION=1.26

FROM golang:1.26-trixie@sha256:cff05dd6b81c901f2e8249eb6f91c69a60451d0f12c26bfef08e476a85b32d4d AS go1.26
FROM golang:1.27-trixie@sha256:f18c3d73a184fb0582828510722e723b63a93f7188d11dd223d9be6f6880e65d AS go1.27
FROM golangci/golangci-lint:v2.15.0@sha256:fbfcfd12a6f8d7f1b97b3362ca80fc2a244d46863148c73cb88e7793af96c481 AS golangci

# go1.26 and go1.27 above are the digest-pinned images; the build arg only selects one.
# hadolint ignore=DL3006
FROM go${GO_VERSION} AS base
RUN groupadd --gid 10001 dev && useradd --uid 10001 --gid 10001 --no-create-home --shell /usr/sbin/nologin dev
ENV HOME=/tmp \
    GOTOOLCHAIN=local \
    GOFLAGS=-mod=readonly \
    GOCACHE=/tmp/gocache \
    GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache
WORKDIR /work

# The module cache is filled from the lockfile only; containers that run without network read it from here.
FROM base AS deps
COPY go.mod go.sum ./
COPY tools/go.mod tools/go.sum tools/
RUN go mod download
WORKDIR /work/tools
RUN go mod download
WORKDIR /work

FROM base AS dev
COPY --from=deps /go/pkg/mod /go/pkg/mod
COPY --from=golangci /usr/bin/golangci-lint /usr/local/bin/golangci-lint
COPY --chown=10001:10001 . .
USER 10001:10001
