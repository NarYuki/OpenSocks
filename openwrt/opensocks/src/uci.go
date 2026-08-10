package main

import (
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
	Region         string
	ExcludeRegions string
	IncludeDomains string
	ExcludeDomains string
	IncludeCIDRs   string
	ExcludeCIDRs   string
	SelectedLineID int
	APIDomain      string
	ControlPort    int
	GeoIPURL       string
	EngineBinary   string
}

func readSettings() *settings {
	s := &settings{
		Mode:         "smart",
		Tun:          true,
		FreeOnly:     true,
		AutoConnect:  true,
		AutoRoute:    true,
		APIDomain:    "https://abscf2.fobwifi.com",
		ControlPort:  9091,
		GeoIPURL:     "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/country.mmdb",
		EngineBinary: "/usr/bin/ss-redir",
	}
	s.Token = loadSession()
	// One-time migration from releases that stored the session in UCI.
	if legacy := uciGet("token"); legacy != "" {
		if s.Token != "" {
			uciDelete("token")
		} else if err := saveSession(legacy); err == nil {
			s.Token = legacy
			uciDelete("token")
		}
	}
	if v := uciGet("mode"); v != "" {
		s.Mode = v
	}
	s.Tun = uciBool("tun", s.Tun)
	s.FreeOnly = uciBool("free_only", s.FreeOnly)
	s.AutoConnect = uciBool("auto_connect", s.AutoConnect)
	s.AutoRoute = uciBool("auto_route", s.AutoRoute)
	s.Region = uciGet("region")
	s.ExcludeRegions = uciGet("exclude_regions")
	s.IncludeDomains = uciGet("include_domains")
	s.ExcludeDomains = uciGet("exclude_domains")
	s.IncludeCIDRs = uciGet("include_cidrs")
	s.ExcludeCIDRs = uciGet("exclude_cidrs")
	if v := uciGet("selected_line_id"); v != "" {
		s.SelectedLineID, _ = strconv.Atoi(v)
	}
	if v := uciGet("api_domain"); v != "" {
		s.APIDomain = v
	}
	if v := uciGet("control_port"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.ControlPort = n
		}
	}
	if v := uciGet("geoip_url"); v != "" {
		s.GeoIPURL = v
	}
	if v := uciGet("engine_binary"); v != "" {
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
	v := uciGet(option)
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes"
}
