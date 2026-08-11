package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type deviceIdentity struct {
	MAC  string `json:"mac"`
	UUID string `json:"uuid"`
}

var (
	deviceIdentityDir                = envOr("OPENSOCKS_STATE_DIR", "/etc/opensocks")
	deviceIdentityRandom   io.Reader = rand.Reader
	deviceIdentityMu       sync.Mutex
	cachedDeviceIdentities = map[int]deviceIdentity{}
)

func deviceIdentityForSlot(slot int) (string, string) {
	if slot < 0 || slot > 2 {
		slot = 0
	}
	deviceIdentityMu.Lock()
	defer deviceIdentityMu.Unlock()
	if identity, ok := cachedDeviceIdentities[slot]; ok {
		return identity.MAC, identity.UUID
	}
	path := filepath.Join(deviceIdentityDir, fmt.Sprintf("device_identity_%d.json", slot+1))
	if identity, ok := readDeviceIdentity(path); ok {
		cachedDeviceIdentities[slot] = identity
		return identity.MAC, identity.UUID
	}
	random := make([]byte, 22)
	if _, err := io.ReadFull(deviceIdentityRandom, random); err != nil {
		return macAddr(), hashDeviceID()
	}
	macBytes := random[:6]
	macBytes[0] = (macBytes[0] | 0x02) & 0xfe
	mac := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", macBytes[0], macBytes[1], macBytes[2], macBytes[3], macBytes[4], macBytes[5])
	uuid := hex.EncodeToString(random[6:22])
	identity := deviceIdentity{MAC: mac, UUID: uuid}
	_ = writeDeviceIdentity(path, identity)
	cachedDeviceIdentities[slot] = identity
	return mac, uuid
}

func readDeviceIdentity(path string) (deviceIdentity, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return deviceIdentity{}, false
	}
	var identity deviceIdentity
	if json.Unmarshal(data, &identity) != nil || identity.MAC == "" || len(identity.UUID) != 32 {
		return deviceIdentity{}, false
	}
	return identity, true
}

func writeDeviceIdentity(path string, identity deviceIdentity) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(identity)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".device-identity-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
