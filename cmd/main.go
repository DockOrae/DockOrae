package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/DockOrae/DockOrae/cmd/flags"
	"github.com/DockOrae/DockOrae/internal/config"
	"github.com/DockOrae/DockOrae/internal/logger"
	"github.com/DockOrae/DockOrae/internal/notify"
	"github.com/DockOrae/DockOrae/internal/service"
	"github.com/DockOrae/DockOrae/internal/state"
)

func main() {
	// 命令行 flag(参考 OpenList):显式传参时覆盖对应环境变量
	flag.StringVar(&flags.DataDir, "data", "", "data directory (default $DATA_DIR or /data)")
	flag.IntVar(&flags.Port, "port", 0, "listen port (default $PORT or 8080)")
	flag.Parse()
	if flags.DataDir != "" {
		_ = os.Setenv("DATA_DIR", flags.DataDir)
	}
	if flags.Port > 0 {
		_ = os.Setenv("PORT", strconv.Itoa(flags.Port))
	}

	cfg := config.Load()

	// 日志同时写 stdout + 环形缓冲(供面板日志弹窗)
	log.SetOutput(logger.NewMultiWriter(logger.LogRing))

	st, err := state.New(cfg)
	if err != nil {
		log.Fatalf("init state: %v", err)
	}
	// 应用商店自动同步(后台异步):全新部署无数据时自动拉取一次,幂等
	go func() {
		if err := service.NewAppStoreService(st).EnsureSynced(); err != nil {
			log.Printf("appstore auto-sync failed (可在面板手动同步): %v", err)
		} else {
			log.Printf("appstore auto-sync done")
		}
	}()

	// Telegram 周期报告调度(版本取 ldflags 注入的 DisplayVersion,不硬编码)
	notify.StartReporter(st.Settings, cfg.DataDir, service.DisplayVersion())

	// 许可证 V3 同步引擎(Event-Driven + SSE):启动 Verify → 建立 SSE → 事件驱动实时同步。
	// 无任何周期验证;Server 故障走 Grace(有限保护),SSE 重连成功即恢复。
	licenseSync := service.StartLicenseSync(st)
	defer licenseSync.Stop()

	runServer(st)
}
