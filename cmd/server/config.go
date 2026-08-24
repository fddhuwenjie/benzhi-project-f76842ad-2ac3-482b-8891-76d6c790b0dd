package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const defaultAddress = "127.0.0.1:19081"

type config struct {
	address      string
	databasePath string
	selfcheck    bool
}

func parseConfig(args []string, getenv func(string) string) (config, error) {
	set := flag.NewFlagSet("sample-chain-server", flag.ContinueOnError)
	var cfg config
	set.StringVar(&cfg.address, "addr", "", "监听地址（仅允许回环地址）")
	set.StringVar(&cfg.databasePath, "db", "sample-chain.db", "SQLite 数据库路径")
	set.BoolVar(&cfg.selfcheck, "selfcheck", false, "运行完整 HTTP 自检后退出")
	if err := set.Parse(args); err != nil {
		return cfg, err
	}
	if set.NArg() != 0 {
		return cfg, fmt.Errorf("不支持的位置参数: %s", strings.Join(set.Args(), " "))
	}
	if cfg.address == "" {
		port := strings.TrimSpace(getenv("PORT"))
		if port != "" {
			number, err := strconv.Atoi(port)
			if err != nil || number < 1 || number > 65535 {
				return cfg, fmt.Errorf("PORT 必须是 1 到 65535 的端口号")
			}
			cfg.address = net.JoinHostPort("127.0.0.1", strconv.Itoa(number))
		} else {
			cfg.address = defaultAddress
		}
	}
	if err := validateAddress(cfg.address); err != nil {
		return cfg, err
	}
	if cfg.selfcheck {
		cfg.databasePath = ":memory:"
	}
	return cfg, nil
}

func validateAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("-addr 必须是 host:port: %w", err)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return fmt.Errorf("监听端口必须是 1 到 65535")
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("监听地址必须是回环地址，拒绝 %q", host)
	}
	return nil
}
