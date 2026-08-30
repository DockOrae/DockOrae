# ============================================================
# DockOrae — 构建 Makefile
#
# 前端已独立到 DockOrae/DockOrae-Frontend 仓库:
#   make web  下载前端 dist(DockOrae-Frontend rolling release → public/dist)
#   make dev  本地前端开发(需先 clone 前端仓库到 ../DockOrae-Frontend)
#
# 常用目标:
#   make            完整构建(前端 public/dist + 后端二进制,当前平台)
#   make web        下载前端 dist(rolling release → public/dist)
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
BIN       := dockorae
PUBLIC_DIR := public
DIST_DIR  := dist
FRONTEND_REPO := DockOrae/DockOrae-Frontend

# 版本号:优先取最近 Git tag(如 v1.0.3),无 tag 时 unknown;可用 VERSION=xxx 覆盖(CI 传 tag)
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo unknown)
# 构建信息:commit + 构建时间(与版本一起经 ldflags 注入)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# ldflags:注入 main.Version/Commit/BuildTime + service.AppVersion,与 Git tag 同步(发版打 tag 即可,无需改源码)
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(GIT_COMMIT) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X github.com/DockOrae/DockOrae/internal/service.AppVersion=$(VERSION)

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
	@echo "==> 下载前端 dist (github.com/$(FRONTEND_REPO) rolling release)"
	@URL=$$(curl -fsSL $${GITHUB_TOKEN:+ -H "Authorization: Bearer $$GITHUB_TOKEN"} https://api.github.com/repos/$(FRONTEND_REPO)/releases/tags/rolling | jq -r '.assets[] | select(.name | startswith("dockorae-frontend-dist-") and endswith(".tar.gz")) | .browser_download_url' 2>/dev/null | head -1); \
	if [ -z "$$URL" ]; then echo "❌ 获取前端 dist 资产失败(rolling release 未发布?先推送 DockOrae-Frontend)"; exit 1; fi; \
	curl -fsSL $${GITHUB_TOKEN:+ -H "Authorization: Bearer $$GITHUB_TOKEN"} "$$URL" -o /tmp/dockorae-fe.tar.gz && \
	rm -rf $(PUBLIC_DIR)/dist && mkdir -p $(PUBLIC_DIR)/dist && \
	tar -xzf /tmp/dockorae-fe.tar.gz -C $(PUBLIC_DIR)/dist && \
	rm -f /tmp/dockorae-fe.tar.gz && \
	echo "✅ 前端 dist 下载完成 → $(PUBLIC_DIR)/dist"

.PHONY: backend
backend:
	@test -d $(PUBLIC_DIR)/dist || { echo "❌ public/dist 不存在,请先执行 make web(前端未构建,go:embed 无法编译)"; exit 1; }
	@echo "==> 编译后端 $(BIN) (v$(VERSION), linux/$$(go env GOARCH))"
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd

# ---------- 运行 / 开发 ----------
.PHONY: run
run: build
	./$(BIN)

.PHONY: dev
dev:
	@test -d ../DockOrae-Frontend || { echo "❌ 请先克隆前端仓库: git clone $(FRONTEND_REPO) ../DockOrae-Frontend"; exit 1; }
	@echo "==> 前端开发模式: http://localhost:5173 (API 代理到 :8080,请确保后端已启动)"
	cd ../DockOrae-Frontend && $(NPM) run dev

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
	  rm -rf $(DIST_DIR)/dockorae; \
	  mkdir -p $(DIST_DIR)/dockorae; \
	  CGO_ENABLED=0 GOOS=linux GOARCH=$$goarch GOARM=$$goarm \
	    $(GO) build -trimpath -ldflags="$(LDFLAGS)" \
	    -o $(DIST_DIR)/dockorae/$(BIN) ./cmd || exit 1; \
	  cp README.md $(DIST_DIR)/dockorae/ 2>/dev/null || true; \
	  tar -czf $(DIST_DIR)/dockorae-linux-$$suffix.tar.gz -C $(DIST_DIR) dockorae || exit 1; \
	  (cd $(DIST_DIR) && sha256sum dockorae-linux-$$suffix.tar.gz > dockorae-linux-$$suffix.tar.gz.sha256) || exit 1; \
	  rm -rf $(DIST_DIR)/dockorae; \
	done
	@echo "✅ 交叉编译完成,产物:"
	@ls -lh $(DIST_DIR)/*.tar.gz $(DIST_DIR)/*.sha256

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
	rm -rf $(PUBLIC_DIR)/dist
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
