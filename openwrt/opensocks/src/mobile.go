package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

const mobileTokenPath = "/etc/opensocks/mobile-token"

func loadOrCreateMobileToken() (string, error) {
	if data, err := os.ReadFile(mobileTokenPath); err == nil {
		token := strings.TrimSpace(string(data))
		if len(token) >= 64 {
			return token, nil
		}
	}
	if err := os.MkdirAll("/etc/opensocks", 0700); err != nil {
		return "", err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	if err := os.WriteFile(mobileTokenPath, []byte(token+"\n"), 0600); err != nil {
		return "", err
	}
	if err := os.Chmod(mobileTokenPath, 0600); err != nil {
		return "", err
	}
	return token, nil
}

func mobileLANAddress(port int) string {
	ifaces, _ := net.Interfaces()
	lan := detectLANDevice()
	// Prefer the actual LAN bridge. WAN addresses are often private too and
	// must not be suggested to a phone paired from the local network.
	for _, iface := range ifaces {
		if iface.Name != lan {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err == nil && ip.To4() != nil {
				return fmt.Sprintf("http://%s:%d", ip.String(), port)
			}
		}
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err == nil && ip.To4() != nil && ip.IsPrivate() {
				return fmt.Sprintf("http://%s:%d", ip.String(), port)
			}
		}
	}
	return fmt.Sprintf("http://192.168.1.1:%d", port)
}

// Pairing data is available only through the localhost-only LuCI proxy.
func (s *server) handleMobilePairing(w http.ResponseWriter, r *http.Request) {
	token, err := loadOrCreateMobileToken()
	if err != nil {
		writeError(w, err)
		return
	}
	cfg := readSettings()
	writeJSON(w, map[string]any{
		"enabled": cfg.MobileEnabled,
		"url":     mobileLANAddress(cfg.MobilePort),
		"token":   token,
		"version": 1,
	})
}
