package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/api"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/config"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/logger"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

const Version = "1.0.0"

func main() {
	cfg := config.Load()

	// 日志同时写 stdout + 环形缓冲(供面板日志弹窗)
	log.SetOutput(logger.NewMultiWriter(api.LogRing))

	st, err := state.New(cfg)
	if err != nil {
		log.Fatalf("init state: %v", err)
	}

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
	}

	go func() {
		log.Printf("Docker Manager 已启动: http://%s%s", addr, basePath)
		log.Printf("Default account: admin / 123456. Please change the password in Settings after first login.")
		var serveErr error
		if s.WebCertFile != "" && s.WebKeyFile != "" {
			log.Printf("TLS enabled: %s / %s", s.WebCertFile, s.WebKeyFile)
			serveErr = srv.ListenAndServeTLS(s.WebCertFile, s.WebKeyFile)
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
