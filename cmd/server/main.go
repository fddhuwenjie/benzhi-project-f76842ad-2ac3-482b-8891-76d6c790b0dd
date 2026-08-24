package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(os.Args[1:], os.Getenv); err != nil {
		log.Printf("服务退出: %v", err)
		os.Exit(1)
	}
}

func run(args []string, getenv func(string) string) error {
	cfg, err := parseConfig(args, getenv)
	if err != nil {
		return err
	}
	rt, err := newRuntime(cfg)
	if err != nil {
		return err
	}
	rt.start()
	if cfg.selfcheck {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		checkErr := runSelfcheck(ctx, cfg.address)
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		shutdownErr := rt.shutdown(shutdownCtx)
		shutdownCancel()
		if checkErr != nil {
			return checkErr
		}
		if shutdownErr != nil {
			return shutdownErr
		}
		fmt.Println("自检通过：档案创建、异常交接、调查、证据复核和关闭归档流程完整")
		return nil
	}
	log.Printf("环境样品交接异常闭环服务监听 %s", cfg.address)
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-signalCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return rt.shutdown(shutdownCtx)
	case serveErr := <-rt.serveResult:
		_ = rt.store.Close()
		return serveErr
	}
}
