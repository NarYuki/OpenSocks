package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"strings"
	"time"
)

// hashDeviceID derives a stable device id from machine identity files.
// Falls back to a random id persisted under /etc/opensocks.
func hashDeviceID() string {
	var inputs []string
	for _, p := range []string{
		"/etc/machine-id",
		"/proc/sys/kernel/random/boot_id",
		"/sys/class/dmi/id/product_uuid",
	} {
		if b, err := os.ReadFile(p); err == nil {
			inputs = append(inputs, strings.TrimSpace(string(b)))
		}
	}
	if len(inputs) == 0 {
		if persist, err := os.ReadFile("/etc/opensocks/device_id"); err == nil {
			if id := strings.TrimSpace(string(persist)); len(id) >= 16 {
				return id
			}
		}
		randBuf := make([]byte, 16)
		if _, err := rand.Read(randBuf); err == nil {
			id := hex.EncodeToString(randBuf)
			os.WriteFile("/etc/opensocks/device_id", []byte(id), 0644)
			return id
		}
		sum := sha256.Sum256([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
		return hex.EncodeToString(sum[:16])
	}
	sum := sha256.Sum256([]byte(strings.Join(inputs, "|")))
	return hex.EncodeToString(sum[:16])
}

// macAddr returns the router's WAN MAC address (required by the API),
// falling back to the Android-style random MAC.
func macAddr() string {
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, ifc := range ifaces {
			if ifc.Flags&net.FlagUp == 0 || len(ifc.HardwareAddr) != 6 {
				continue
			}
			if strings.HasPrefix(ifc.Name, "br-") || ifc.Name == "eth0" || ifc.Name == "wan" ||
				ifc.Name == "wan6" || ifc.Name == "br-lan" {
				return ifc.HardwareAddr.String()
			}
		}
		for _, ifc := range ifaces {
			if len(ifc.HardwareAddr) == 6 {
				return ifc.HardwareAddr.String()
			}
		}
	}
	return "02:00:00:00:00:00"
}
