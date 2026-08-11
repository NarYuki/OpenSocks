package main

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialStoreUsesRootOnlyPermissions(t *testing.T) {
	original := credentialsFile
	credentialsFile = filepath.Join(t.TempDir(), "credentials.json")
	t.Cleanup(func() { credentialsFile = original })
	want := &savedCredentials{AuthBy: "email", AuthType: "password", Email: "user@example.com", Password: "secret"}
	if err := saveCredentials(want); err != nil {
		t.Fatal(err)
	}
	got, err := loadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != want.Email || got.Password != want.Password {
		t.Fatalf("credentials = %#v", got)
	}
	info, err := os.Stat(credentialsFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions = %o, want 600", info.Mode().Perm())
	}
	raw, err := os.ReadFile(credentialsFile)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("secret")) || bytes.Contains(raw, []byte("user@example.com")) {
		t.Fatalf("credential file contains plaintext: %s", raw)
	}
	keyInfo, err := os.Stat(credentialKeyFile())
	if err != nil {
		t.Fatal(err)
	}
	if keyInfo.Mode().Perm() != 0600 {
		t.Fatalf("key permissions = %o, want 600", keyInfo.Mode().Perm())
	}
}

func TestCredentialStoreRejectsPlaintext(t *testing.T) {
	original := credentialsFile
	credentialsFile = filepath.Join(t.TempDir(), "credentials.enc")
	t.Cleanup(func() { credentialsFile = original })
	if err := os.WriteFile(credentialsFile, []byte(`{"email":"user@example.com","password":"secret"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCredentials(); err == nil {
		t.Fatal("plaintext credential store was accepted")
	}
}

func TestConnectionHistoryStoresServerAndRegion(t *testing.T) {
	original := historyFile
	historyFile = filepath.Join(t.TempDir(), "history.json")
	t.Cleanup(func() { historyFile = original })
	ln := &line{ID: 12, Name: "test-line", Location: "Shanghai", Category: "normal"}
	resp := ssConfig()
	resp.LineID, resp.LineName = ln.ID, ln.Name
	if err := appendHistory(ln, resp); err != nil {
		t.Fatal(err)
	}
	records, err := loadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("history length = %d", len(records))
	}
	got := records[0]
	if got.LineID != 12 || got.Location != "Shanghai" || got.Server != "5.6.7.8" || got.Port != 443 {
		t.Fatalf("history entry = %#v", got)
	}
}

func TestParsePingLatency(t *testing.T) {
	for _, test := range []struct {
		output string
		want   float64
	}{
		{"64 bytes: time=12.34 ms", 12.34},
		{"64 bytes: time<1 ms", 1},
	} {
		got, err := parsePingLatency(test.output)
		if err != nil || got != test.want {
			t.Fatalf("parsePingLatency(%q) = %v, %v", test.output, got, err)
		}
	}
}

func TestActiveVIP(t *testing.T) {
	if !activeVIP(&account{Expired: false, RemainingDays: 1}) {
		t.Fatal("active subscription was not recognized")
	}
	if activeVIP(&account{Expired: true, RemainingDays: 10}) {
		t.Fatal("expired subscription was recognized as active")
	}
	if activeVIP(nil) {
		t.Fatal("nil account was recognized as active")
	}
}

func TestHonorOfKingsDomainsIncludeSubdomains(t *testing.T) {
	for _, name := range []string{
		"pvp.qq.com",
		"update.pvp.qq.com",
		"cdn.game.gtimg.cn",
		"receiver.msdk.qq.com",
	} {
		if got := serviceForDNSNames([]string{name}); got != "honor_of_kings" {
			t.Fatalf("serviceForDNSNames(%q) = %q, want honor_of_kings", name, got)
		}
	}
}

func TestBilibiliCDNDomainsIncludeAllSubdomains(t *testing.T) {
	for _, name := range []string{
		"api.biliapi.net",
		"upos-sz-mirrorcos.bilivideo.com",
		"i0.hdslb.com",
		"static.bilicdn2.com",
	} {
		if got := serviceForDNSNames([]string{name}); got != "bilibili" {
			t.Fatalf("serviceForDNSNames(%q) = %q, want bilibili", name, got)
		}
	}
}

func TestDNSResponseKeepsEveryMatchingService(t *testing.T) {
	got := serviceGroupsForDNSNames([]string{"video.bilibili.com", "shared.alicdn.com"})
	want := map[string]bool{"bilibili": true, "alipay_alibaba": true}
	for _, group := range got {
		delete(want, group)
	}
	if len(want) != 0 {
		t.Fatalf("serviceGroupsForDNSNames() = %v, missing %v", got, want)
	}
}

func TestChinaRouteDNSNamesIncludeAllCNDomainsAndKnownServices(t *testing.T) {
	for _, name := range []string{"assets.example.cn", "quote.eastmoney.com", "cdn.game.gtimg.cn"} {
		if !chinaRouteForDNSNames([]string{name}) {
			t.Fatalf("China DNS name %q was not routed", name)
		}
	}
	if chinaRouteForDNSNames([]string{"www.google.com"}) {
		t.Fatal("non-China DNS name was routed")
	}
}

func TestHonorOfKingsDirectBattleServers(t *testing.T) {
	for _, address := range []string{"43.129.255.150", "43.129.111.193", "43.154.252.89"} {
		ip := net.ParseIP(address)
		matched := false
		for _, raw := range honorOfKingsDirectCIDRs {
			_, network, err := net.ParseCIDR(raw)
			if err != nil {
				t.Fatalf("invalid Honor of Kings CIDR %q: %v", raw, err)
			}
			if network.Contains(ip) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("direct battle server %s is not classified", address)
		}
	}
}
