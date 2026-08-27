package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/api"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/config"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

const Version = "1.0.0"

func main() {
	cfg := config.Load()
	log.Printf("data dir: %s", cfg.DataDir)
	log.Printf("panel port: %d", cfg.Port)
	log.Printf("Default account: admin / 123456. Please change the password in Settings after first login.")

	st, err := state.New(cfg)
	if err != nil {
		log.Fatalf("init state: %v", err)
	}

	engine := api.Router(st, serveStatic)

	srv := &http.Server{
		Addr:              "0.0.0.0:" + itoa(cfg.Port),
		Handler:           engine,
		ReadHeaderTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("Docker Manager 已启动: http://%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
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

func itoa(n uint16) string {
	if n == 0 {
		return "0"
	}
	var b [5]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
