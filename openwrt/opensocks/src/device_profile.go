package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
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
	deviceProfileDir               = envOr("OPENSOCKS_STATE_DIR", "/etc/opensocks")
	deviceProfileRandom  io.Reader = rand.Reader
	deviceProfileMu      sync.Mutex
	cachedDeviceProfiles = map[int]deviceProfile{}
)

func persistentDeviceProfileForSlot(slot int) deviceProfile {
	if slot < 0 || slot > 2 {
		slot = 0
	}
	deviceProfileMu.Lock()
	defer deviceProfileMu.Unlock()
	if profile, ok := cachedDeviceProfiles[slot]; ok {
		return profile
	}
	path := filepath.Join(deviceProfileDir, fmt.Sprintf("device_profile_%d.json", slot+1))
	if profile, ok := readDeviceProfile(path); ok {
		cachedDeviceProfiles[slot] = profile
		return profile
	}

	used := map[deviceProfile]bool{}
	for other := 0; other < 3; other++ {
		if other == slot {
			continue
		}
		if profile, ok := cachedDeviceProfiles[other]; ok {
			used[profile] = true
			continue
		}
		otherPath := filepath.Join(deviceProfileDir, fmt.Sprintf("device_profile_%d.json", other+1))
		if profile, ok := readDeviceProfile(otherPath); ok {
			used[profile] = true
		}
	}
	candidates := make([]deviceProfile, 0, len(deviceProfiles))
	for _, profile := range deviceProfiles {
		if !used[profile] {
			candidates = append(candidates, profile)
		}
	}
	if len(candidates) == 0 {
		candidates = deviceProfiles
	}
	var random [1]byte
	index := 0
	if _, err := io.ReadFull(deviceProfileRandom, random[:]); err == nil {
		index = int(random[0]) % len(candidates)
	}
	profile := candidates[index]
	_ = writeDeviceProfile(path, profile)
	cachedDeviceProfiles[slot] = profile
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
