package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDeviceIdentitySlotsAreStableAndDistinct(t *testing.T) {
	oldDir, oldRandom, oldCache := deviceIdentityDir, deviceIdentityRandom, cachedDeviceIdentities
	t.Cleanup(func() {
		deviceIdentityDir, deviceIdentityRandom, cachedDeviceIdentities = oldDir, oldRandom, oldCache
	})
	deviceIdentityDir = filepath.Join(t.TempDir(), "identities")
	deviceIdentityRandom = bytes.NewReader(append(bytes.Repeat([]byte{0x11}, 22), bytes.Repeat([]byte{0x22}, 22)...))
	cachedDeviceIdentities = map[int]deviceIdentity{}

	mac1, uuid1 := deviceIdentityForSlot(0)
	mac2, uuid2 := deviceIdentityForSlot(1)
	cachedDeviceIdentities = map[int]deviceIdentity{}
	mac1Again, uuid1Again := deviceIdentityForSlot(0)
	if mac1 != mac1Again || uuid1 != uuid1Again {
		t.Fatal("slot identity is not stable")
	}
	if mac1 == mac2 || uuid1 == uuid2 {
		t.Fatal("session slots share an identity")
	}
	if mac1[1] != '2' && mac1[1] != '6' && mac1[1] != 'a' && mac1[1] != 'e' {
		t.Fatalf("slot MAC is not locally administered: %s", mac1)
	}
	info, err := os.Stat(filepath.Join(deviceIdentityDir, "device_identity_1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("identity permissions = %o, want 600", info.Mode().Perm())
	}
}
