package main

import (
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
