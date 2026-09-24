FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
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

# ~= pins the 99.x series but tolerates Alpine repo rebuilds (r0 → r1 → ...),
# which a strict =99.0-r0 pin would break on the first rebuild.
RUN apk add --no-cache 'mkvtoolnix~=99.0'

# A dedicated non-root user owns /data, where the media library gets mounted.
RUN addgroup -S mkvtea && adduser -S -G mkvtea -h /data mkvtea && \
    mkdir -p /data && chown -R mkvtea:mkvtea /data

WORKDIR /data
USER mkvtea:mkvtea

ENV TERM=xterm-256color

COPY --from=builder /out/mkvtea /usr/local/bin/mkvtea

ENTRYPOINT ["/usr/local/bin/mkvtea"]
CMD ["--help"]
