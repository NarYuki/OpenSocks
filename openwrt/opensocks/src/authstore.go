package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Credentials are stored because the user explicitly opted into automatic
// session renewal. The state directory is root-only on OpenWrt.
var credentialsFile = envOr("OPENSOCKS_STATE_DIR", "/etc/opensocks") + "/credentials.json"

type savedCredentials struct {
	AuthBy   string `json:"auth_by"`
	AuthType string `json:"auth_type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	CC       string `json:"cc,omitempty"`
}

var credentialsMu sync.Mutex

func saveCredentials(creds *savedCredentials) error {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	if creds == nil {
		return os.Remove(credentialsFile)
	}
	if err := os.MkdirAll(filepath.Dir(credentialsFile), 0700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(credentialsFile), 0700); err != nil {
		return err
	}
	b, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	tmp := credentialsFile + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, credentialsFile)
}

func loadCredentials() (*savedCredentials, error) {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, err
	}
	var creds savedCredentials
	if err := json.Unmarshal(b, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func credentialsSaved() bool {
	_, err := os.Stat(credentialsFile)
	return err == nil
}
