package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// server exposes the local control API used by the LuCI app.
// It binds to 127.0.0.1 only; the LuCI controller proxies requests server-side,
// so no CORS or authentication is required on this listener.

type server struct {
	ctl *controller
}

func newServer(ctl *controller) *server {
	return &server{ctl: ctl}
}

func (s *server) listenAndServe(port int) error {
	return serveHTTP(fmt.Sprintf("127.0.0.1:%d", port), s.routes())
}

func serveHTTP(address string, handler http.Handler) error {
	limit := make(chan struct{}, 32)
	guarded := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case limit <- struct{}{}:
			defer func() { <-limit }()
		default:
			http.Error(w, "server busy", http.StatusServiceUnavailable)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		handler.ServeHTTP(w, r)
	})
	server := &http.Server{
		Addr:              address,
		Handler:           guarded,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	return server.ListenAndServe()
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/register", s.handleRegister)
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/lines", s.handleLines)
	mux.HandleFunc("/connect", s.handleConnect)
	mux.HandleFunc("/disconnect", s.handleDisconnect)
	mux.HandleFunc("/history", s.handleHistory)
	mux.HandleFunc("/reconnect", s.handleReconnect)
	mux.HandleFunc("/settings", s.handleSettings)
	mux.HandleFunc("/setup", s.handleSetup)
	mux.HandleFunc("/logs", s.handleLogs)
	mux.HandleFunc("/traffic", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, trafficStats()) })
	mux.HandleFunc("/regiontest", s.handleRegionTest)
	mux.HandleFunc("/speedtest/servers", s.handleSpeedServers)
	mux.HandleFunc("/speedtest/run", s.handleSpeedRun)
	mux.HandleFunc("/speedtestcn/servers", s.handleSpeedTestCNServers)
	mux.HandleFunc("/speedtestcn/run", s.handleSpeedTestCNRun)
	mux.HandleFunc("/speedtest/job/start", s.handleSpeedJobStart)
	mux.HandleFunc("/speedtest/job/status", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, currentSpeedJob()) })
	mux.HandleFunc("/mobile/pairing", s.handleMobilePairing)
	return mux
}

func (s *server) handleSpeedJobStart(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var in struct {
		Provider string `json:"provider"`
		ID       string `json:"id"`
	}
	if decodeJSONBody(r.Body, &in) != nil || (in.Provider != "ookla" && in.Provider != "speedtestcn") || in.ID == "" {
		writeError(w, fmt.Errorf("invalid speed test job"))
		return
	}
	if err := startSpeedJob(in.Provider, in.ID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, currentSpeedJob())
}

func (s *server) listenAndServeMobile(port int, token string) error {
	routes := s.routes()
	return serveHTTP(fmt.Sprintf("0.0.0.0:%d", port), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get("Authorization")
		want := "Bearer " + token
		if len(provided) != len(want) || subtle.ConstantTimeCompare([]byte(provided), []byte(want)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "unauthorized"})
			return
		}
		routes.ServeHTTP(w, r)
	}))
}

func (s *server) handleRegionTest(w http.ResponseWriter, r *http.Request) {
	result, err := testChinaExitRegion()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, result)
}

func (s *server) handleSpeedTestCNServers(w http.ResponseWriter, r *http.Request) {
	servers, err := discoverSpeedTestCNServers()
	if err != nil {
		writeError(w, err)
		return
	}
	pointers := make([]*speedTestCNServer, len(servers))
	for i := range servers {
		pointers[i] = &servers[i]
	}
	measureSpeedHTTPPings(pointers, func(server *speedTestCNServer) string { return server.PingURL }, func(server *speedTestCNServer, ping float64) { server.PingMS = ping })
	sort.SliceStable(servers, func(i, j int) bool {
		return servers[i].PingMS > 0 && (servers[j].PingMS == 0 || servers[i].PingMS < servers[j].PingMS)
	})
	writeJSON(w, map[string]any{"servers": servers})
}

func (s *server) handleSpeedTestCNRun(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var in struct {
		ID string `json:"id"`
	}
	if decodeJSONBody(r.Body, &in) != nil {
		writeError(w, fmt.Errorf("invalid request"))
		return
	}
	servers, err := discoverSpeedTestCNServers()
	if err != nil {
		writeError(w, err)
		return
	}
	for _, sv := range servers {
		if sv.ID == in.ID {
			result, e := runSpeedTestCN(sv)
			if e != nil {
				writeError(w, e)
				return
			}
			writeJSON(w, result)
			return
		}
	}
	writeError(w, fmt.Errorf("SpeedTest.cn server %s not found", in.ID))
}

func (s *server) handleSpeedServers(w http.ResponseWriter, r *http.Request) {
	servers, err := discoverChinaSpeedServers()
	if err != nil {
		writeError(w, err)
		return
	}
	pointers := make([]*speedServer, len(servers))
	for i := range servers {
		pointers[i] = &servers[i]
	}
	measureSpeedHTTPPings(pointers, func(server *speedServer) string {
		return strings.TrimSuffix(server.URL, "upload.php") + "latency.txt"
	}, func(server *speedServer, ping float64) { server.PingMS = ping })
	sort.SliceStable(servers, func(i, j int) bool {
		return servers[i].PingMS > 0 && (servers[j].PingMS == 0 || servers[i].PingMS < servers[j].PingMS)
	})
	writeJSON(w, map[string]any{"servers": servers})
}
func (s *server) handleSpeedRun(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var in struct {
		ID string `json:"id"`
	}
	if decodeJSONBody(r.Body, &in) != nil {
		writeError(w, fmt.Errorf("invalid request"))
		return
	}
	servers, err := discoverChinaSpeedServers()
	if err != nil {
		writeError(w, err)
		return
	}
	ordered := make([]speedServer, 0, len(servers))
	found := false
	for _, sv := range servers {
		if sv.ID == in.ID {
			ordered = append(ordered, sv)
			found = true
			break
		}
	}
	if !found {
		writeError(w, fmt.Errorf("China Ookla server %s not found", in.ID))
		return
	}
	for _, sv := range servers {
		if sv.ID != in.ID {
			ordered = append(ordered, sv)
		}
	}
	var lastErr error
	for _, sv := range ordered {
		result, e := runChinaSpeedTest(sv)
		if e != nil {
			lastErr = e
			logf("Ookla China server %s failed, trying fallback: %v", sv.ID, e)
			continue
		}
		result.RequestedServerID = in.ID
		result.Fallback = sv.ID != in.ID
		writeJSON(w, result)
		return
	}
	writeError(w, fmt.Errorf("all China Ookla servers failed; last error: %v", lastErr))
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.ctl.status())
}

func (s *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	_, err := s.ctl.registerByDevice()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var in loginInput
	if err := decodeJSONBody(r.Body, &in); err != nil {
		writeError(w, fmt.Errorf("bad request: %w", err))
		return
	}
	_, err := s.ctl.login(in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	s.ctl.disconnect()
	s.ctl.logout()
	writeJSON(w, map[string]any{"ok": true})
}

func (s *server) handleLines(w http.ResponseWriter, r *http.Request) {
	lines, err := s.ctl.listLines(r.URL.Query().Get("sort"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"lines": lines})
}

func (s *server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var in struct {
		LineID int `json:"line_id"`
	}
	_ = decodeJSONBody(r.Body, &in)
	if err := s.ctl.connect(in.LineID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "lineID": s.ctl.engine.lineID, "lineName": s.ctl.engine.lineName})
}

func (s *server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	if err := s.ctl.disconnect(); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *server) handleHistory(w http.ResponseWriter, r *http.Request) {
	records, err := loadHistory()
	if err != nil {
		writeError(w, err)
		return
	}
	sortHistory(records)
	writeJSON(w, map[string]any{"history": records})
}

func (s *server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var in struct {
		Mode string `json:"mode"`
	}
	_ = decodeJSONBody(r.Body, &in)
	result, err := s.ctl.setupNetwork(in.Mode)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, result)
}

func (s *server) handleReconnect(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var in struct {
		ID string `json:"id"`
	}
	_ = decodeJSONBody(r.Body, &in)
	if err := s.ctl.reconnect(in.ID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "lineID": s.ctl.engine.lineID, "lineName": s.ctl.engine.lineName})
}

func (s *server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.ctl.status())
	case http.MethodPost:
		wasRunning := s.ctl.engine.isRunning()
		var in struct {
			Mode           string `json:"mode"`
			Tun            *bool  `json:"tun"`
			FreeOnly       *bool  `json:"free_only"`
			AutoConnect    *bool  `json:"auto_connect"`
			AutoRoute      *bool  `json:"auto_route"`
			Region         string `json:"region"`
			ExcludeRegions string `json:"exclude_regions"`
			IncludeDomains string `json:"include_domains"`
			ExcludeDomains string `json:"exclude_domains"`
			IncludeCIDRs   string `json:"include_cidrs"`
			ExcludeCIDRs   string `json:"exclude_cidrs"`
		}
		if err := decodeJSONBody(r.Body, &in); err != nil {
			writeError(w, err)
			return
		}
		updates := map[string]string{}
		if in.Mode == "smart" || in.Mode == "global" {
			updates["mode"] = in.Mode
		}
		if in.Tun != nil {
			updates["tun"] = boolStr(*in.Tun)
		}
		if in.FreeOnly != nil {
			updates["free_only"] = boolStr(*in.FreeOnly)
		}
		if in.AutoConnect != nil {
			updates["auto_connect"] = boolStr(*in.AutoConnect)
		}
		if in.AutoRoute != nil {
			updates["auto_route"] = boolStr(*in.AutoRoute)
		}
		updates["region"] = in.Region
		updates["exclude_regions"] = in.ExcludeRegions
		updates["include_domains"] = in.IncludeDomains
		updates["exclude_domains"] = in.ExcludeDomains
		updates["include_cidrs"] = in.IncludeCIDRs
		updates["exclude_cidrs"] = in.ExcludeCIDRs
		saveSettings(updates)
		s.ctl.refreshSettings()
		if wasRunning {
			if err := s.ctl.disconnect(); err != nil {
				writeError(w, err)
				return
			}
			if err := s.ctl.connect(-1); err != nil {
				writeError(w, err)
				return
			}
		}
		writeJSON(w, map[string]any{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- helpers ---------------------------------------------------------------

func (s *server) handleLogs(w http.ResponseWriter, r *http.Request) {
	lines := 100
	if v := r.URL.Query().Get("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			lines = n
		}
	}
	writeJSON(w, map[string]any{"log": tailLog(lines)})
}

func requirePost(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
