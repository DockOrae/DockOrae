package cmd

// 构建信息,均由 ldflags 在构建时注入(发版只需打 Git tag,无需改源码):
//
//	go build -ldflags "\
//	  -X main.Version=v1.0.3 \
//	  -X main.Commit=abc1234 \
//	  -X main.BuildTime=2026-08-29T01:20:00Z \
//	  -X github.com/DockOrae/DockOrae/internal/service.AppVersion=v1.0.3"
//
// 未注入(本地开发/CI 检查)时为空字符串,运行时显示 unknown。
// 使用 Makefile 构建时自动从 git tag / commit 注入。
var (
	Version   string
	Commit    string
	BuildTime string
)
