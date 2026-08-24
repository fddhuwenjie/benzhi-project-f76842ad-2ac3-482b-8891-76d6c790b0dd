package main

import "testing"

func TestAddressPrecedenceAndSafety(t *testing.T) {
	env := func(key string) string {
		if key == "PORT" {
			return "19111"
		}
		return ""
	}
	cfg, err := parseConfig(nil, env)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.address != "127.0.0.1:19111" {
		t.Fatalf("PORT 未生效: %s", cfg.address)
	}
	cfg, err = parseConfig([]string{"-addr=127.0.0.1:19222"}, env)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.address != "127.0.0.1:19222" {
		t.Fatalf("-addr 应优先: %s", cfg.address)
	}
	if _, err = parseConfig([]string{"-addr=0.0.0.0:19081"}, env); err == nil {
		t.Fatal("非回环监听应被拒绝")
	}
}

func TestDefaultAddressIsHighLoopback(t *testing.T) {
	cfg, err := parseConfig(nil, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.address != defaultAddress {
		t.Fatalf("默认地址错误: %s", cfg.address)
	}
}
