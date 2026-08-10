package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionStoreIsRootOnly(t *testing.T) {
	old := sessionFile
	sessionFile = filepath.Join(t.TempDir(), "state", "session")
	t.Cleanup(func() { sessionFile = old })
	if err := saveSession("session-value"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(sessionFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("session mode = %o, want 600", info.Mode().Perm())
	}
	if got := loadSession(); got != "session-value" {
		t.Fatalf("loadSession() = %q", got)
	}
	if err := saveSession(""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Fatalf("session still exists after removal: %v", err)
	}
}

func TestAPIClientRejectsPlainHTTPRemote(t *testing.T) {
	c := newAPIClient("http://example.com")
	if c.domain != "https://abscf2.fobwifi.com" {
		t.Fatalf("insecure remote domain was accepted: %s", c.domain)
	}
	loopback := newAPIClient("http://127.0.0.1:1234")
	if loopback.domain != "http://127.0.0.1:1234" {
		t.Fatalf("loopback test endpoint rejected: %s", loopback.domain)
	}
}
