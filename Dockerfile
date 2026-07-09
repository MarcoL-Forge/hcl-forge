# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/hcl-forge ./cmd/hcl-forge

FROM scratch

LABEL org.opencontainers.image.title="hcl-forge"
LABEL org.opencontainers.image.description="CLI for transforming HCL files"
LABEL org.opencontainers.image.source="https://github.com/MarcoL-Forge/hcl-forge"

COPY --from=builder /out/hcl-forge /usr/local/bin/hcl-forge
COPY third_party_licenses /licenses/hcl-forge/third_party
COPY LICENSE /licenses/hcl-forge/LICENSE

USER 10001:10001

ENTRYPOINT ["/usr/local/bin/hcl-forge"]