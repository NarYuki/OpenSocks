package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type deviceProfile struct {
	Model       string `json:"model"`
	SysVersion  string `json:"sys_version"`
	WidthHeight string `json:"width_height"`
}

// The original Android client sends Build.MODEL and Build.VERSION.RELEASE.
// Pixel devices expose these exact retail names through Build.MODEL, avoiding
// the regional SKU ambiguity found in many other manufacturers' devices.
var deviceProfiles = []deviceProfile{
	{Model: "Pixel 10", SysVersion: "16", WidthHeight: "1080x2424"},
	{Model: "Pixel 10 Pro", SysVersion: "16", WidthHeight: "1280x2856"},
	{Model: "Pixel 10 Pro XL", SysVersion: "16", WidthHeight: "1344x2992"},
	{Model: "Pixel 9a", SysVersion: "16", WidthHeight: "1080x2424"},
	{Model: "Pixel 9 Pro", SysVersion: "16", WidthHeight: "1280x2856"},
	{Model: "Pixel 9 Pro XL", SysVersion: "16", WidthHeight: "1344x2992"},
}

var (
	deviceProfileFile             = envOr("OPENSOCKS_STATE_DIR", "/etc/opensocks") + "/device_profile.json"
	deviceProfileRandom io.Reader = rand.Reader
	deviceProfileMu     sync.Mutex
	cachedDeviceProfile *deviceProfile
)

func persistentDeviceProfile() deviceProfile {
	deviceProfileMu.Lock()
	defer deviceProfileMu.Unlock()
	if cachedDeviceProfile != nil {
		return *cachedDeviceProfile
	}
	if profile, ok := readDeviceProfile(deviceProfileFile); ok {
		cachedDeviceProfile = &profile
		return profile
	}

	var random [1]byte
	index := 0
	if _, err := io.ReadFull(deviceProfileRandom, random[:]); err == nil {
		index = int(random[0]) % len(deviceProfiles)
	} else {
		fallback := sha256.Sum256([]byte(hashDeviceID()))
		index = int(fallback[0]) % len(deviceProfiles)
	}
	profile := deviceProfiles[index]
	_ = writeDeviceProfile(deviceProfileFile, profile)
	cachedDeviceProfile = &profile
	return profile
}

func readDeviceProfile(path string) (deviceProfile, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return deviceProfile{}, false
	}
	var saved deviceProfile
	if json.Unmarshal(data, &saved) != nil {
		return deviceProfile{}, false
	}
	for _, known := range deviceProfiles {
		if saved == known {
			return saved, true
		}
	}
	return deviceProfile{}, false
}

func writeDeviceProfile(path string, profile deviceProfile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".device-profile-*")
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
