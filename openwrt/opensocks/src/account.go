package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Account state remains on persistent storage while large engine assets live
// in tmpfs. This lets the web UI retain account details across router restarts.

var accountFile = envOr("OPENSOCKS_STATE_DIR", "/etc/opensocks") + "/account.json"

var (
	accountMu sync.Mutex
	savedAcc  *account
)

func saveAccount(acc *account) {
	accountMu.Lock()
	defer accountMu.Unlock()
	savedAcc = acc
	if acc == nil {
		os.Remove(accountFile)
		return
	}
	b, err := json.Marshal(acc)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(accountFile), 0700); err != nil {
		return
	}
	os.WriteFile(accountFile, b, 0600)
}

func loadAccount() *account {
	accountMu.Lock()
	defer accountMu.Unlock()
	if savedAcc != nil {
		return savedAcc
	}
	b, err := os.ReadFile(accountFile)
	if err != nil {
		return nil
	}
	var acc account
	if json.Unmarshal(b, &acc) != nil {
		return nil
	}
	savedAcc = &acc
	return &acc
}

var _ = fmt.Sprintf
