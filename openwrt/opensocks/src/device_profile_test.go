package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestDeviceProfileIsRandomlySelectedOnceAndPersisted(t *testing.T) {
	oldPath, oldRandom, oldCache := deviceProfileFile, deviceProfileRandom, cachedDeviceProfile
	t.Cleanup(func() {
		deviceProfileFile, deviceProfileRandom, cachedDeviceProfile = oldPath, oldRandom, oldCache
	})
	deviceProfileFile = filepath.Join(t.TempDir(), "device_profile.json")
	deviceProfileRandom = bytes.NewReader([]byte{4})
	cachedDeviceProfile = nil

	first := persistentDeviceProfile()
	if first != deviceProfiles[4] {
		t.Fatalf("selected profile = %#v, want %#v", first, deviceProfiles[4])
	}

	// Simulate a daemon restart with a different random source. The saved
	// profile must win and keep the API identity stable.
	deviceProfileRandom = bytes.NewReader([]byte{1})
	cachedDeviceProfile = nil
	second := persistentDeviceProfile()
	if second != first {
		t.Fatalf("profile changed after restart: %#v -> %#v", first, second)
	}

	device := newAPIClient("https://example.com").dev()
	if device.Device != "Android" || device.Model != first.Model || device.SysVersion != "16" || device.WidthHeight != first.WidthHeight {
		t.Fatalf("unexpected API device identity: %#v", device)
	}
}
