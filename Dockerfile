# ================= Stage 1: 前端 dist =================
# 前端已独立到 DockOrae/DockOrae-Frontend 仓库,其 CI 发布 rolling release 资产;
# 这里直接下载 dist tar.gz(不 clone 源码、不 npm 构建,构建更快更稳定)。
FROM --platform=$BUILDPLATFORM alpine:3.20 AS web
RUN apk add --no-cache curl tar jq \
    && URL=$(curl -fsSL https://api.github.com/repos/DockOrae/DockOrae-Frontend/releases/tags/rolling | jq -r '.assets[] | select(.name=="dockorae-frontend-dist.tar.gz") | .browser_download_url') \
    && [ -n "$URL" ] \
    && curl -fsSL "$URL" -o /dist.tar.gz \
    && mkdir -p /fe/dist \
    && tar -xzf /dist.tar.gz -C /fe/dist

# ================= Stage 2: Go 后端编译(原生交叉编译,不需要 QEMU) =================
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS
ARG TARGETARCH
# 构建信息:版本/commit/构建时间,由 docker-publish 传 build-arg;默认 unknown
ARG VERSION=unknown
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN apk add --no-cache git ca-certificates
WORKDIR /app
# 国内代理加速;moby client 模块自带坏的 replace,用自身 replace 覆盖
RUN go env -w GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY public/embed.go ./public/
COPY --from=web /fe/dist ./public/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME} -X github.com/DockOrae/DockOrae/internal/service.AppVersion=${VERSION}" \
    -o docker-manager ./cmd/docker-manager

# ================= Stage 3: 运行镜像(目标平台) =================
FROM alpine:3.20
ARG TARGETARCH
ARG TARGETVARIANT
# OCI Labels(辅助信息):docker inspect 可查镜像版本/commit/构建时间
ARG VERSION=unknown
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
LABEL org.opencontainers.image.title="Docker Manager Go"
LABEL org.opencontainers.image.source="https://github.com/DockOrae/DockOrae"
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.revision=${COMMIT}
LABEL org.opencontainers.image.created=${BUILD_TIME}
RUN apk add --no-cache ca-certificates tini curl \
    && case "${TARGETARCH}${TARGETVARIANT}" in \
         amd64) A=x86_64 ;; \
         arm64) A=aarch64 ;; \
         armv7) A=armv7 ;; \
         armv6) A=armv6 ;; \
         s390x) A=s390x ;; \
         *) echo "unsupported arch: ${TARGETARCH}${TARGETVARIANT}" >&2; exit 1 ;; \
       esac \
    && curl -fsSL "https://github.com/docker/compose/releases/download/v5.5.0/docker-compose-linux-$A" -o /usr/local/bin/docker-compose \
    && chmod +x /usr/local/bin/docker-compose \
    && docker-compose version

COPY --from=build /app/docker-manager /usr/local/bin/docker-manager

ENV DATA_DIR=/data
VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["docker-manager"]
