package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Persistent login data is encrypted with a random device-local AES-256 key.
// Both files and their parent directory are restricted to root. Plain JSON is
// deliberately not accepted: clean installs only use this encrypted format.
var credentialsFile = envOr("OPENSOCKS_STATE_DIR", "/etc/opensocks") + "/credentials.enc"

const credentialAAD = "opensocks-credentials-v1"

type savedCredentials struct {
	AuthBy   string `json:"auth_by"`
	AuthType string `json:"auth_type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	CC       string `json:"cc,omitempty"`
}

type encryptedCredentials struct {
	Version    int    `json:"version"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

var credentialsMu sync.Mutex

func credentialKeyFile() string {
	return filepath.Join(filepath.Dir(credentialsFile), "credentials.key")
}

func credentialKey() ([]byte, error) {
	path := credentialKeyFile()
	if b, err := os.ReadFile(path); err == nil {
		if len(b) != 32 {
			return nil, fmt.Errorf("invalid credential key length")
		}
		return b, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, key, 0600); err != nil {
		return nil, err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	return key, nil
}

func saveCredentials(creds *savedCredentials) error {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	if creds == nil {
		err := os.Remove(credentialsFile)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	dir := filepath.Dir(credentialsFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	key, err := credentialKey()
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	plain, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	sealed := gcm.Seal(nil, nonce, plain, []byte(credentialAAD))
	for i := range plain {
		plain[i] = 0
	}
	envelope, err := json.Marshal(encryptedCredentials{
		Version: 1, Nonce: base64.RawStdEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawStdEncoding.EncodeToString(sealed),
	})
	if err != nil {
		return err
	}
	tmp := credentialsFile + ".tmp"
	if err := os.WriteFile(tmp, envelope, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		_ = os.Remove(tmp)
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
	var envelope encryptedCredentials
	if err := json.Unmarshal(b, &envelope); err != nil || envelope.Version != 1 || envelope.Nonce == "" || envelope.Ciphertext == "" {
		return nil, fmt.Errorf("unsupported credential store format")
	}
	key, err := credentialKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.Nonce)
	if err != nil || len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid credential nonce")
	}
	sealed, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("invalid credential ciphertext")
	}
	plain, err := gcm.Open(nil, nonce, sealed, []byte(credentialAAD))
	if err != nil {
		return nil, fmt.Errorf("credential decryption failed")
	}
	defer func() {
		for i := range plain {
			plain[i] = 0
		}
	}()
	var creds savedCredentials
	if err := json.Unmarshal(plain, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func credentialsSaved() bool {
	_, err := os.Stat(credentialsFile)
	return err == nil
}
