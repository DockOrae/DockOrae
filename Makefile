# ============================================================
# Docker Manager Go — 构建 Makefile
#
# 常用目标:
#   make            完整构建(前端 web/dist + 后端二进制,当前平台)
#   make web        仅构建前端(web/dist,go:embed 嵌入用)
#   make backend    仅构建后端(需先 make web;前端未改时可跳过)
#   make dev        前端开发模式(vite :5173,API 代理到 :8080,需后端已启动)
#   make run        完整构建后直接运行
#   make cross      交叉编译全部 Linux 架构 → dist/*.tar.gz(发版用)
#   make test       质量检查:go vet + go test + go test -race(与 CI 一致)
#   make clean      清理全部构建产物
#   make help       查看帮助
# ============================================================

# ---------- 变量 ----------
GO        ?= go
NPM       ?= npm
BIN       := docker-manager-go
WEB_DIR   := web
DIST_DIR  := dist

# 版本号:优先取最近 Git tag(如 v1.0.3),无 tag 时 dev;可用 VERSION=xxx 覆盖(CI 传 tag)
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)

# ldflags:注入 main.Version + service.AppVersion,与 Git tag 同步(发版打 tag 即可,无需改源码)
LDFLAGS := -s -w -X main.Version=$(VERSION) -X github.com/MinimaxFlora/Docker_Manager_Go/internal/service.AppVersion=$(VERSION)

# Linux 交叉编译架构(与 README 支持平台一致)
LINUX_SPECS := amd64: arm64: arm:5 arm:6 arm:7 386: s390x:

.DEFAULT_GOAL := build

# ---------- 构建 ----------
.PHONY: build
build: web backend
	@echo ""
	@echo "✅ 构建完成: $(BIN) (v$(VERSION))"

.PHONY: web
web:
	@echo "==> 构建前端 (vite build)"
	@test -d $(WEB_DIR)/node_modules || (echo "    node_modules 缺失,先执行 npm install"; cd $(WEB_DIR) && $(NPM) install --no-audit --no-fund)
	cd $(WEB_DIR) && $(NPM) run build

.PHONY: backend
backend:
	@test -d $(WEB_DIR)/dist || { echo "❌ web/dist 不存在,请先执行 make web(前端未构建,go:embed 无法编译)"; exit 1; }
	@echo "==> 编译后端 $(BIN) (v$(VERSION), linux/$$(go env GOARCH))"
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/docker-manager

# ---------- 运行 / 开发 ----------
.PHONY: run
run: build
	./$(BIN)

.PHONY: dev
dev:
	@echo "==> 前端开发模式: http://localhost:5173 (API 代理到 :8080,请确保后端已启动)"
	cd $(WEB_DIR) && $(NPM) run dev

# ---------- 交叉编译(Linux 发版) ----------
.PHONY: cross
cross: web
	@mkdir -p $(DIST_DIR)
	@for spec in $(LINUX_SPECS); do \
	  arch=$${spec%%:*}; arm=$${spec##*:}; \
	  if [ "$$arch" = "arm" ]; then \
	    goarch=arm; goarm=$$arm; suffix=armv$$arm; \
	    echo "==> 交叉编译 linux/arm/$$goarm"; \
	  else \
	    goarch=$$arch; goarm=; suffix=$$arch; \
	    echo "==> 交叉编译 linux/$$goarch"; \
	  fi; \
	  rm -rf $(DIST_DIR)/docker-manager-go; \
	  mkdir -p $(DIST_DIR)/docker-manager-go; \
	  CGO_ENABLED=0 GOOS=linux GOARCH=$$goarch GOARM=$$goarm \
	    $(GO) build -trimpath -ldflags="$(LDFLAGS)" \
	    -o $(DIST_DIR)/docker-manager-go/$(BIN) ./cmd/docker-manager || exit 1; \
	  cp README.md $(DIST_DIR)/docker-manager-go/ 2>/dev/null || true; \
	  tar -czf $(DIST_DIR)/docker-manager-go-linux-$$suffix.tar.gz -C $(DIST_DIR) docker-manager-go || exit 1; \
	  rm -rf $(DIST_DIR)/docker-manager-go; \
	done
	@echo "✅ 交叉编译完成,产物:"
	@ls -lh $(DIST_DIR)/*.tar.gz

# ---------- 质量检查(与 .github/workflows/go-checks.yml 一致) ----------
.PHONY: test
test: vet
	$(GO) test ./...
	$(GO) test -race ./...

.PHONY: vet
vet:
	$(GO) vet ./...

# ---------- 清理 ----------
.PHONY: clean
clean:
	rm -rf $(DIST_DIR)
	rm -f $(BIN)
	rm -rf $(WEB_DIR)/dist
	@echo "✅ 已清理"

# ---------- 帮助 ----------
.PHONY: help
help:
	@echo "Docker Manager Go 构建命令:"
	@echo "  make            完整构建(前端 + 后端)"
	@echo "  make web        仅构建前端"
	@echo "  make backend    仅构建后端"
	@echo "  make dev        前端开发模式 (:5173)"
	@echo "  make run        构建并运行"
	@echo "  make cross      交叉编译全部 Linux 架构 (dist/)"
	@echo "  make test       质量检查 (vet + test + race)"
	@echo "  make clean      清理产物"
