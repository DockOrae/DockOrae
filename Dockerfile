# ================= Stage 1: 前端构建 =================
# --platform=$BUILDPLATFORM:始终在构建机原生平台执行(避免 QEMU 模拟 node 极慢)
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install --no-audit --no-fund --registry=https://registry.npmmirror.com
COPY web/ ./
RUN npm run build

# ================= Stage 2: Go 后端编译(原生交叉编译,不需要 QEMU) =================
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
RUN apk add --no-cache git ca-certificates
WORKDIR /app
# 国内代理加速;moby client 模块自带坏的 replace,用自身 replace 覆盖
RUN go env -w GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
COPY main.go web.go ./
COPY internal/ ./internal/
COPY --from=web /web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o docker-manager .

# ================= Stage 3: 运行镜像(目标平台) =================
FROM alpine:3.20
ARG TARGETARCH
RUN apk add --no-cache ca-certificates tini curl \
    && case "$TARGETARCH" in \
         amd64) A=x86_64 ;; \
         arm64) A=aarch64 ;; \
         *) echo "unsupported arch: $TARGETARCH" >&2; exit 1 ;; \
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
