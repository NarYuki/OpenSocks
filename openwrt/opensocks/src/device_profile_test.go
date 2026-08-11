package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestDeviceProfileIsRandomlySelectedOnceAndPersisted(t *testing.T) {
	oldDir, oldRandom, oldCache := deviceProfileDir, deviceProfileRandom, cachedDeviceProfiles
	t.Cleanup(func() {
		deviceProfileDir, deviceProfileRandom, cachedDeviceProfiles = oldDir, oldRandom, oldCache
	})
	deviceProfileDir = filepath.Join(t.TempDir(), "profiles")
	deviceProfileRandom = bytes.NewReader([]byte{4, 1})
	cachedDeviceProfiles = map[int]deviceProfile{}

	first := persistentDeviceProfileForSlot(0)
	if first != deviceProfiles[4] {
		t.Fatalf("selected profile = %#v, want %#v", first, deviceProfiles[4])
	}

	// Simulate a daemon restart with a different random source. The saved
	// profile must win and keep the API identity stable.
	deviceProfileRandom = bytes.NewReader([]byte{1})
	cachedDeviceProfiles = map[int]deviceProfile{}
	second := persistentDeviceProfileForSlot(0)
	if second != first {
		t.Fatalf("profile changed after restart: %#v -> %#v", first, second)
	}

	device := newAPIClient("https://example.com").dev()
	if device.Device != "Android" || device.Model != first.Model || device.SysVersion != "16" || device.WidthHeight != first.WidthHeight {
		t.Fatalf("unexpected API device identity: %#v", device)
	}
	third := persistentDeviceProfileForSlot(1)
	if third != deviceProfiles[1] {
		t.Fatalf("slot 2 profile = %#v, want %#v", third, deviceProfiles[1])
	}
}
