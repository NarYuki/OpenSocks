package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var sessionFile = envOr("OPENSOCKS_STATE_DIR", "/etc/opensocks") + "/session"
var sessionMu sync.Mutex

func saveSession(session string) error {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	if session == "" {
		if err := os.Remove(sessionFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	dir := filepath.Dir(sessionFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	tmp := sessionFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(session+"\n"), 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, sessionFile)
}

func loadSession() string {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	b, err := os.ReadFile(sessionFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
