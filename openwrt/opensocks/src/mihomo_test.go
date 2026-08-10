package main

import (
	"encoding/json"
	"testing"
)

func ssConfig() *connectResponse {
	verify := true
	verifyPtr := &verify
	return &connectResponse{
		LineID:   7,
		LineName: "hk-01",
		Config: &connectConfig{
			Host:    "h.example.com",
			ProxyIP: "5.6.7.8",
			Proto:   0,
			SSConf: &bootsInfo{
				Proto:    "SS",
				Server:   "5.6.7.8",
				Port:     443,
				Password: "pw",
				Method:   "chacha20-ietf-poly1305",
			},
			TrojanConf: &bootsInfo{
				Proto:    "Trojan",
				Server:   "tr.example.com",
				Port:     443,
				Password: "tpw",
				Ssl: &struct {
					SNI    string `json:"sni"`
					Verify *bool  `json:"verify"`
				}{SNI: "tr.example.com", Verify: verifyPtr},
			},
		},
	}
}

func TestGenerateConfigSS(t *testing.T) {
	raw, err := generateConfig(ssConfig(), "smart", true)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	proxies := cfg["proxies"].([]any)
	if len(proxies) != 1 { // first usable (SS) wins
		t.Fatalf("expected 1 proxy, got %d", len(proxies))
	}
	p := proxies[0].(map[string]any)
	if p["type"] != "ss" || p["cipher"] != "chacha20-ietf-poly1305" {
		t.Fatalf("bad ss proxy: %v", p)
	}
	if cfg["tun"] == nil {
		t.Fatal("tun section missing")
	}
	rules := cfg["rules"].([]any)
	if rules[len(rules)-1] != "MATCH,DIRECT" {
		t.Fatalf("smart mode should end with MATCH,DIRECT: %v", rules)
	}
}

func TestGenerateConfigGlobal(t *testing.T) {
	raw, err := generateConfig(ssConfig(), "global", false)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	json.Unmarshal(raw, &cfg)
	rules := cfg["rules"].([]any)
	if rules[len(rules)-1] != "MATCH,PROXY" {
		t.Fatalf("global mode should end with MATCH,PROXY: %v", rules)
	}
	if cfg["tun"] != nil {
		t.Fatal("tun section should be absent in redirect mode")
	}
}

func TestGenerateConfigNoProxy(t *testing.T) {
	resp := &connectResponse{Config: &connectConfig{}}
	if _, err := generateConfig(resp, "smart", true); err == nil {
		t.Fatal("expected error for empty config")
	}
}
