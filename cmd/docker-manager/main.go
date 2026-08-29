package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/api"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/auth"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/config"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/logger"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// 构建信息,均由 ldflags 在构建时注入(发版只需打 Git tag,无需改源码):
//
//	go build -ldflags "\
//	  -X main.Version=v1.0.3 \
//	  -X main.Commit=abc1234 \
//	  -X main.BuildTime=2026-08-29T01:20:00Z \
//	  -X github.com/MinimaxFlora/Docker_Manager_Go/internal/service.AppVersion=v1.0.3"
//
// 未注入(本地开发/CI 检查)时为空字符串,运行时显示 unknown。
// 使用 Makefile 构建时自动从 git tag / commit 注入。
var (
	Version   string
	Commit    string
	BuildTime string
)

func main() {
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

	s := st.Settings.Get()
	basePath := s.WebBasePath
	listenAddr := s.WebListen
	if listenAddr == "" {
		listenAddr = "0.0.0.0"
	}
	addr := fmt.Sprintf("%s:%d", listenAddr, s.WebPort)

	engine := api.Router(st, basePath, serveStatic(basePath))

	srv := &http.Server{
		Addr:              addr,
		Handler:           engine,
		ReadHeaderTimeout: 30 * time.Second,
		ErrorLog:          log.New(tlsErrFilter{}, "", 0),
	}

	go func() {
		log.Printf("Docker Manager 已启动: http://%s%s", addr, basePath)
		// 仅在默认密码仍有效时提示(避免每次启动泄露默认密码,且改密后不再误导)
		if u := st.FindUser("admin"); u != nil && auth.VerifyPassword("123456", u.PasswordHash) {
			log.Printf("Default account: admin / 123456. Please change the password in Settings after first login.")
		}
		var serveErr error
		if s.WebCertFile != "" && s.WebKeyFile != "" {
			if _, err := tls.LoadX509KeyPair(s.WebCertFile, s.WebKeyFile); err != nil {
				if s.WebForceSSL {
					// 强制 HTTPS:证书无效直接退出,绝不降级 HTTP(勾选后仅允许 HTTPS 访问)
					log.Fatalf("强制 HTTPS 已开启但证书加载失败: %v", err)
				}
				// 未开启强制 HTTPS:降级 HTTP 监听而不是崩溃退出(否则容器 restart 循环,面板彻底失联)。
				// 日志会明确提示,用户可进面板修正证书路径后重启生效。
				log.Printf("TLS 证书加载失败,已降级为 HTTP 监听: %v", err)
				serveErr = srv.ListenAndServe()
			} else {
				log.Printf("TLS enabled: %s / %s (SNI 白名单: %s)", s.WebCertFile, s.WebKeyFile, s.WebDomain)
				// 手动构建 TLSConfig:GetCertificate 按 SNI 白名单放行,
				// IP 直连(空 SNI)与陌生域名在握手阶段即被拒绝,扫描噪音大幅减少。
				srv.TLSConfig = &tls.Config{
					GetCertificate: makeGetCertificate(s.WebDomain, s.WebCertFile, s.WebKeyFile),
					MinVersion:     tls.VersionTLS12,
				}
				serveErr = srv.ListenAndServeTLS("", "")
			}
		} else if s.WebForceSSL {
			log.Fatalf("强制 HTTPS 已开启但未配置证书路径(webCertFile/webKeyFile)")
		} else {
			serveErr = srv.ListenAndServe()
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Fatalf("listen: %v", serveErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
