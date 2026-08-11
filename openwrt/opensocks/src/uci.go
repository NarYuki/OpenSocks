package main

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

// uci access to /etc/config/opensocks. The daemon runs as root on OpenWrt,
// so shelling out to `uci` is the idiomatic and robust way to read/write config.

const uciRoot = "opensocks.settings"

type settings struct {
	Token          string
	Mode           string
	Tun            bool
	FreeOnly       bool
	AutoConnect    bool
	AutoRoute      bool
	SessionCount   int
	Region         string
	ExcludeRegions string
	IncludeDomains string
	ExcludeDomains string
	IncludeCIDRs   string
	ExcludeCIDRs   string
	SelectedLineID int
	APIDomain      string
	ControlPort    int
	MobileEnabled  bool
	MobilePort     int
	GeoIPURL       string
	EngineBinary   string
}

func readSettings() *settings {
	values := uciSnapshot()
	s := &settings{
		Mode:          "smart",
		Tun:           true,
		FreeOnly:      true,
		AutoConnect:   true,
		AutoRoute:     true,
		APIDomain:     "https://abscf2.fobwifi.com",
		ControlPort:   9091,
		MobileEnabled: true,
		MobilePort:    9092,
		GeoIPURL:      "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/country.mmdb",
		EngineBinary:  "/usr/bin/ss-redir",
		SessionCount:  1,
	}
	s.Token = loadSession()
	// One-time migration from releases that stored the session in UCI.
	if legacy := values["token"]; legacy != "" {
		if s.Token != "" {
			uciDelete("token")
		} else if err := saveSession(legacy); err == nil {
			s.Token = legacy
			uciDelete("token")
		}
	}
	if v := values["mode"]; v != "" {
		s.Mode = v
	}
	s.Tun = valueBool(values["tun"], s.Tun)
	s.FreeOnly = valueBool(values["free_only"], s.FreeOnly)
	s.AutoConnect = valueBool(values["auto_connect"], s.AutoConnect)
	s.AutoRoute = valueBool(values["auto_route"], s.AutoRoute)
	if n, err := strconv.Atoi(values["session_count"]); err == nil && n >= 1 && n <= 3 {
		s.SessionCount = n
	} else if valueBool(values["dual_session_experimental"], false) {
		// Compatibility with installations that enabled the former dual mode.
		s.SessionCount = 2
	}
	s.Region = values["region"]
	s.ExcludeRegions = values["exclude_regions"]
	s.IncludeDomains = values["include_domains"]
	s.ExcludeDomains = values["exclude_domains"]
	s.IncludeCIDRs = values["include_cidrs"]
	s.ExcludeCIDRs = values["exclude_cidrs"]
	if v := values["selected_line_id"]; v != "" {
		s.SelectedLineID, _ = strconv.Atoi(v)
	}
	if v := values["api_domain"]; v != "" {
		s.APIDomain = v
	}
	if v := values["control_port"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.ControlPort = n
		}
	}
	s.MobileEnabled = valueBool(values["mobile_enabled"], s.MobileEnabled)
	if v := values["mobile_port"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < 65536 && n != s.ControlPort {
			s.MobilePort = n
		}
	}
	if v := values["geoip_url"]; v != "" {
		s.GeoIPURL = v
	}
	if v := values["engine_binary"]; v != "" {
		s.EngineBinary = v
	}
	return s
}

// engine settings are read on demand so changes apply without daemon restart.
func cfgEngineBinary() string { return readSettings().EngineBinary }

func saveToken(token string) {
	if err := saveSession(token); err != nil {
		logf("could not update root-only session file: %v", err)
	}
}

func uciDelete(option string) {
	exec.Command("uci", "-q", "delete", uciRoot+"."+option).Run()
	exec.Command("uci", "-q", "commit", "opensocks").Run()
}

func saveSetting(key, value string) {
	uciSet(key, value)
}

func saveSettings(values map[string]string) {
	for key, value := range values {
		exec.Command("uci", "-q", "set", uciRoot+"."+key+"="+value).Run()
	}
	exec.Command("uci", "-q", "commit", "opensocks").Run()
}

func uciSnapshot() map[string]string {
	out, err := exec.Command("uci", "-q", "show", uciRoot).Output()
	if err != nil {
		return map[string]string{}
	}
	values := map[string]string{}
	prefix := uciRoot + "."
	for _, line := range bytes.Split(out, []byte{'\n'}) {
		parts := bytes.SplitN(line, []byte{'='}, 2)
		if len(parts) != 2 || !bytes.HasPrefix(parts[0], []byte(prefix)) {
			continue
		}
		key := strings.TrimPrefix(string(parts[0]), prefix)
		value := string(parts[1])
		if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			value = value[1 : len(value)-1]
			value = strings.ReplaceAll(value, "'\\''", "'")
		}
		values[key] = value
	}
	return values
}

func uciGet(option string) string {
	out, err := exec.Command("uci", "-q", "get", uciRoot+"."+option).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func uciSet(option, value string) {
	// NOTE: no shell is involved (exec.Command), so values must NOT be
	// wrapped in quotes — uci would store them literally.
	exec.Command("uci", "-q", "set", uciRoot+"."+option+"="+value).Run()
	exec.Command("uci", "-q", "commit", "opensocks").Run()
}

func uciBool(option string, def bool) bool {
	return valueBool(uciGet(option), def)
}

func valueBool(v string, def bool) bool {
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes"
}
