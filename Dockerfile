FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
      -trimpath \
      -ldflags="-s -w -X mkvtea/internal/config.Version=${VERSION}" \
      -o /out/mkvtea .

FROM alpine:3.24 AS runner

LABEL org.opencontainers.image.title="mkvtea" \
      org.opencontainers.image.source="https://github.com/fraluc06/mkvtea"

# No pin: let apk resolve the newest build available on the pinned Alpine
# branch (3.24 → mkvtoolnix 99.x). A strict =99.0-r0-style pin broke v1.2.0
# the first time Alpine rebuilt the package (r0→r1). The base image version
# is what freezes the toolchain for a reproducible release.
RUN apk add --no-cache mkvtoolnix

# A dedicated non-root user owns /data, where the media library gets mounted.
RUN addgroup -S mkvtea && adduser -S -G mkvtea -h /data mkvtea && \
    mkdir -p /data && chown -R mkvtea:mkvtea /data

WORKDIR /data
USER mkvtea:mkvtea

ENV TERM=xterm-256color

COPY --from=builder /out/mkvtea /usr/local/bin/mkvtea

ENTRYPOINT ["/usr/local/bin/mkvtea"]
CMD ["--help"]
