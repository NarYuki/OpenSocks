package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"
)

type setupResult struct {
	OK        bool   `json:"ok"`
	LANDevice string `json:"lan_device"`
	Engine    string `json:"engine"`
	Mode      string `json:"mode"`
	Message   string `json:"message"`
}

// setupNetwork performs the complete low-memory router integration. It keeps
// the existing LAN/WAN topology intact and inserts/removes only our dedicated
// nftables table, making the operation safe to repeat.
func (c *controller) setupNetwork(mode string) (*setupResult, error) {
	if mode != "global" {
		mode = "smart"
	}
	if _, err := exec.LookPath("nft"); err != nil {
		return nil, fmt.Errorf("nft command is required: %w", err)
	}
	if _, err := exec.LookPath("ss-redir"); err != nil {
		logf("ss-redir is missing; installing lightweight engine")
		if out, installErr := exec.Command("opkg", "update").CombinedOutput(); installErr != nil {
			return nil, fmt.Errorf("opkg update failed: %w (%s)", installErr, truncate(string(out), 500))
		}
		if out, installErr := exec.Command("opkg", "install", "shadowsocks-libev-ss-redir", "nftables-json", "ca-bundle").CombinedOutput(); installErr != nil {
			return nil, fmt.Errorf("lightweight engine install failed: %w (%s)", installErr, truncate(string(out), 500))
		}
	}
	if _, err := exec.LookPath("ss-local"); err != nil {
		if out, installErr := exec.Command("opkg", "install", "shadowsocks-libev-ss-local").CombinedOutput(); installErr != nil {
			return nil, fmt.Errorf("speed-test helper install failed: %w (%s)", installErr, truncate(string(out), 500))
		}
	}
	saveSetting("engine_binary", "/usr/bin/ss-redir")
	saveSetting("tun", "0")
	saveSetting("mode", mode)
	saveSetting("auto_connect", "1")
	c.refreshSettings()
	if c.engine.isRunning() {
		if err := c.disconnect(); err != nil {
			logf("setup disconnect warning: %v", err)
		}
	}
	if err := c.connect(-1); err != nil {
		return nil, err
	}
	return &setupResult{OK: true, LANDevice: detectLANDevice(), Engine: "ss-redir", Mode: mode, Message: "network integration is active"}, nil
}

// controller wires the API client, engine and UCI settings together.
// It is invoked by the local control API (server.go).

type controller struct {
	api       *apiClient
	engine    *engine
	cfg       *settings
	authMu    sync.Mutex
	connectMu sync.Mutex
	cfgMu     sync.RWMutex
	linesMu   sync.RWMutex
	lines     []line
	lastAuth  time.Time
}

func newController(cfg *settings) *controller {
	return &controller{
		api:    newAPIClient(cfg.APIDomain),
		engine: newEngine(),
		cfg:    cfg,
	}
}

func (c *controller) refreshSettings() {
	config := readSettings()
	c.cfgMu.Lock()
	c.cfg = config
	c.cfgMu.Unlock()
	c.api.configure(config.APIDomain, config.Token)
}

func (c *controller) currentSettings() settings {
	c.cfgMu.RLock()
	defer c.cfgMu.RUnlock()
	return *c.cfg
}

func (c *controller) setSessionToken(token string) {
	c.cfgMu.Lock()
	c.cfg.Token = token
	c.cfgMu.Unlock()
}

// loggedIn reports whether we hold a usable session (cookie or legacy token).
func (c *controller) loggedIn() bool {
	return c.api.hasAuth() || c.currentSettings().Token != ""
}

// --- auth ------------------------------------------------------------------

// registerByDevice obtains a token for free lines without an account.
func (c *controller) registerByDevice() (string, error) {
	token, err := c.api.registerByDevice()
	if err != nil {
		return "", err
	}
	saveToken(token)
	c.setSessionToken(token)
	return token, nil
}

type loginInput struct {
	AuthBy   string `json:"auth_by"`
	AuthType string `json:"auth_type"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	CC       string `json:"cc"`
	SmsCode  string `json:"sms_code"`
}

// account holds the login response summary for the web UI.
type account struct {
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Nick          string `json:"nick"`
	Expired       bool   `json:"expired"`
	ExpireAt      string `json:"expire_at"`
	RemainingDays int    `json:"remaining_days"`
}

func (c *controller) login(in loginInput) (string, error) {
	if in.AuthBy == "" {
		if in.Email != "" {
			in.AuthBy = "email"
		} else if in.Phone != "" {
			in.AuthBy = "phone"
		} else {
			in.AuthBy = "username"
		}
	}
	if in.AuthType == "" {
		in.AuthType = "password"
	}
	// force a fresh session: ignore any stale stored token
	c.api.clearAuth()
	session, err := c.api.login(loginRequest{
		AuthBy:   in.AuthBy,
		AuthType: in.AuthType,
		Username: in.Username,
		Password: in.Password,
		Email:    in.Email,
		Phone:    in.Phone,
		CC:       in.CC,
		SmsCode:  in.SmsCode,
	})
	if err != nil {
		return "", err
	}
	if err := saveCredentials(&savedCredentials{
		AuthBy: in.AuthBy, AuthType: in.AuthType, Username: in.Username,
		Password: in.Password, Email: in.Email, Phone: in.Phone, CC: in.CC,
	}); err != nil {
		logf("warning: could not save credentials for automatic login: %v", err)
	}
	saveToken(session)
	c.lastAuth = time.Now()
	c.setSessionToken(session)
	return session, nil
}

func (c *controller) logout() {
	c.api.clearAuth()
	saveToken("")
	c.setSessionToken("")
	saveAccount(nil)
	if err := saveCredentials(nil); err != nil && !errors.Is(err, os.ErrNotExist) {
		logf("warning: could not remove saved credentials: %v", err)
	}
}

// reauthenticate renews an expired session from the root-only credential
// store. authMu prevents concurrent line/status requests from issuing several
// sessions and invalidating one another.
func (c *controller) reauthenticate() error {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	creds, err := loadCredentials()
	if err != nil {
		return fmt.Errorf("session expired and no saved credentials are available: %w", err)
	}
	logf("session expired; automatically renewing the saved session")
	c.api.clearAuth()
	session, err := c.api.login(loginRequest{
		AuthBy: creds.AuthBy, AuthType: creds.AuthType, Username: creds.Username,
		Password: creds.Password, Email: creds.Email, Phone: creds.Phone, CC: creds.CC,
	})
	if err != nil {
		return fmt.Errorf("automatic login failed: %w", err)
	}
	saveToken(session)
	c.lastAuth = time.Now()
	c.setSessionToken(session)
	return nil
}

func (c *controller) getLinesAuthenticated() ([]line, error) {
	var lines []line
	var err error
	for cycle := 0; cycle < 3; cycle++ {
		lines, err = c.api.getLines()
		if err == nil || !isAPIErrorCode(err, 20001) {
			break
		}
		if authErr := c.reauthenticate(); authErr != nil {
			return nil, authErr
		}
	}
	if err == nil {
		c.linesMu.Lock()
		c.lines = append(c.lines[:0], lines...)
		c.linesMu.Unlock()
		return lines, err
	}
	return lines, err
}

func (c *controller) cachedLines() []line {
	c.linesMu.RLock()
	defer c.linesMu.RUnlock()
	return append([]line(nil), c.lines...)
}

// --- lines -----------------------------------------------------------------

type lineInfo struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Location   string  `json:"location"`
	Category   string  `json:"category"`
	IsFree     bool    `json:"isFree"`
	Target     string  `json:"target,omitempty"`
	LatencyMS  float64 `json:"latency_ms,omitempty"`
	Reachable  bool    `json:"reachable"`
	DetectPort int     `json:"detect_port,omitempty"`
}

func (c *controller) listLines(sortBy string) ([]lineInfo, error) {
	cfg := c.currentSettings()
	lines, err := c.getLinesAuthenticated()
	if err != nil {
		if isAPIErrorCode(err, 20001) {
			lines = c.cachedLines()
			if len(lines) == 0 {
				return nil, fmt.Errorf("server list is temporarily unavailable; please retry")
			}
			logf("using cached server list while authentication service recovers")
		} else {
			return nil, err
		}
	}
	out := make([]lineInfo, 0, len(lines))
	vip := activeVIP(loadAccount())
	for _, l := range lines {
		if l.ID <= 0 {
			continue // skip virtual auto lines in the picker
		}
		if vip && l.IsFree {
			continue
		}
		if !vip && cfg.FreeOnly && !l.IsFree {
			continue
		}
		if cfg.Region != "" && !lineMatchesRegion(l, cfg.Region) {
			continue
		}
		if matchesAnyRegion(l, cfg.ExcludeRegions) {
			continue
		}
		out = append(out, lineInfo{ID: l.ID, Name: l.Name, Location: l.Location, Category: l.Category, IsFree: l.IsFree, Target: lineTarget(l), DetectPort: l.DetectPort})
	}
	if sortBy == "ping" {
		measureLineLatencies(out)
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Reachable != out[j].Reachable {
				return out[i].Reachable
			}
			if out[i].Reachable && out[i].LatencyMS != out[j].LatencyMS {
				return out[i].LatencyMS < out[j].LatencyMS
			}
			return out[i].ID < out[j].ID
		})
		return out, nil
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsFree != out[j].IsFree {
			return out[i].IsFree
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func matchesAnyRegion(l line, values string) bool {
	for _, v := range splitConfigValues(values) {
		if lineMatchesRegion(l, v) {
			return true
		}
	}
	return false
}

func activeVIP(acc *account) bool {
	return acc != nil && !acc.Expired && acc.RemainingDays > 0
}

func lineMatchesRegion(l line, region string) bool {
	return containsFold(l.Location, region) || containsFold(l.Category, region) || containsFold(l.Name, region)
}

func containsFold(s, sub string) bool {
	return len(sub) == 0 || stringContainsFold(s, sub)
}

// --- connect / disconnect --------------------------------------------------

// pickLine resolves a line id; id <= 0 means "best free line".
func (c *controller) pickLine(lines []line, wantID int) (*line, error) {
	if wantID > 0 {
		for i := range lines {
			if lines[i].ID == wantID {
				return &lines[i], nil
			}
		}
		// The upstream line directory can temporarily return a partial list while
		// its authentication service is converging. A user-selected ID remains a
		// valid connect target; let the connect endpoint make the authoritative
		// decision instead of producing a false local "not found" error.
		return &line{ID: wantID, Name: fmt.Sprintf("Line %d", wantID)}, nil
	}
	preferVIP := activeVIP(loadAccount())
	for i := range lines {
		if (preferVIP && !lines[i].IsFree) || (!preferVIP && lines[i].IsFree) {
			return &lines[i], nil
		}
	}
	if len(lines) > 0 {
		return &lines[0], nil
	}
	return nil, fmt.Errorf("no server is currently available")
}

func (c *controller) connect(wantID int) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	c.refreshSettings()
	cfg := c.currentSettings()
	switching := false
	manualSelection := wantID > 0
	if wantID <= 0 && cfg.SelectedLineID > 0 {
		wantID = cfg.SelectedLineID
	}
	if !c.loggedIn() {
		return fmt.Errorf("not logged in: register (free) or login first")
	}
	if c.engine.isRunning() {
		if wantID <= 0 || wantID == c.engine.lineID {
			return nil
		}
		logf("switching server from line %d to line %d", c.engine.lineID, wantID)
		switching = true
	}

	var lines []line
	var err error
	if wantID > 0 {
		// A selected line ID is authoritative. Avoid the less reliable directory
		// request during switching; its metadata was cached when the picker loaded.
		lines = c.cachedLines()
	} else {
		lines, err = c.getLinesAuthenticated()
		if err != nil {
			return err
		}
	}
	ln, err := c.pickLine(lines, wantID)
	if err != nil {
		return err
	}
	regionID := parseRegionID(cfg.Region)
	recommend := wantID <= 0
	req := connectRequest{
		Proto:          "SS",
		RecommendLine:  boolPtr(recommend),
		AvailableProto: []string{"SS", "Trojan", "GTS"},
		RegionID:       regionID,
	}
	if cfg.SessionCount > 1 {
		return c.connectMultiLine(ln, req, switching, manualSelection, cfg, cfg.SessionCount)
	}
	var resp *connectResponse
	for cycle := 0; cycle < 3; cycle++ {
		resp, err = c.api.connect(ln.ID, req)
		if err == nil || !isAPIErrorCode(err, 20001) {
			break
		}
		if authErr := c.reauthenticate(); authErr != nil {
			return authErr
		}
	}
	if err != nil {
		return err
	}
	connectedLine := ln
	if resp.LineID > 0 && resp.LineID != ln.ID {
		for i := range lines {
			if lines[i].ID == resp.LineID {
				connectedLine = &lines[i]
				break
			}
		}
	}

	if switching {
		c.engine.stop()
		teardownRedirect()
	}
	if err := c.engine.start(resp, cfg.Mode, false, ""); err != nil {
		return err
	}
	if err := setupRedirect(cfg.Mode, c.engine.server, 1); err != nil {
		c.engine.stop()
		return err
	}
	if err := appendHistory(connectedLine, resp); err != nil {
		logf("warning: could not save connection history: %v", err)
	}
	if manualSelection || cfg.SelectedLineID <= 0 {
		saveSetting("selected_line_id", fmt.Sprint(connectedLine.ID))
		c.refreshSettings()
	}
	return nil
}

func (c *controller) reconnect(historyID string) error {
	record, err := findHistory(historyID)
	if err != nil {
		return err
	}
	return c.connect(record.LineID)
}

func (c *controller) disconnect() error {
	c.refreshSettings()
	c.engine.stop()
	teardownRedirect()
	return nil
}

// status returns the current state for the web UI.
func (c *controller) status() map[string]any {
	c.refreshSettings()
	cfg := c.currentSettings()
	acc := loadAccount()
	lanDevice, routingApplied := networkIntegrationState()
	return map[string]any{
		"running":            c.engine.isRunning(),
		"lineID":             c.engine.lineID,
		"lineName":           c.engine.lineName,
		"token":              cfg.Token != "",
		"mode":               cfg.Mode,
		"tun":                false,
		"engine":             "ss-redir",
		"region":             cfg.Region,
		"excludeRegions":     cfg.ExcludeRegions,
		"includeDomains":     cfg.IncludeDomains,
		"excludeDomains":     cfg.ExcludeDomains,
		"includeCIDRs":       cfg.IncludeCIDRs,
		"excludeCIDRs":       cfg.ExcludeCIDRs,
		"selectedLineID":     cfg.SelectedLineID,
		"freeOnly":           cfg.FreeOnly,
		"autoConnect":        cfg.AutoConnect,
		"autoRoute":          cfg.AutoRoute,
		"sessionCount":       cfg.SessionCount,
		"activeSessionCount": c.engine.activeSessionCount(),
		"account":            acc,
		"vip":                activeVIP(acc),
		"lanDevice":          lanDevice,
		"routingApplied":     routingApplied,
		"credentialsSaved":   credentialsSaved(),
	}
}

// autoConnect is invoked at daemon startup.
func (c *controller) autoConnect() {
	c.refreshSettings()
	cfg := c.currentSettings()
	if (!cfg.AutoConnect && !cfg.AutoRoute) || !c.loggedIn() {
		return
	}
	if err := c.connect(-1); err != nil {
		logf("auto-connect failed: %v", err)
	}
}

func (c *controller) autoRouteWatchdog() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	backoff, nextAttempt := 10*time.Second, time.Time{}
	for range ticker.C {
		if currentSpeedJob().Running {
			continue
		}
		c.refreshSettings()
		cfg := c.currentSettings()
		if !cfg.AutoRoute {
			continue
		}
		if !c.engine.isRunning() {
			if time.Now().Before(nextAttempt) {
				continue
			}
			if c.loggedIn() {
				if err := c.connect(-1); err != nil {
					logf("automatic connection recovery failed: %v", err)
					nextAttempt = time.Now().Add(backoff)
					backoff *= 2
					if backoff > 5*time.Minute {
						backoff = 5 * time.Minute
					}
				} else {
					backoff, nextAttempt = 10*time.Second, time.Time{}
				}
			}
			continue
		}
		_, applied := networkIntegrationState()
		if c.engine.isRunning() && !applied {
			if err := setupRedirect(cfg.Mode, c.engine.server, c.engine.activeSessionCount()); err != nil {
				logf("automatic routing recovery failed: %v", err)
			}
		}
	}
}

func (c *controller) restartEngine() {
	c.refreshSettings()
	if err := c.connect(-1); err != nil {
		logf("engine restart failed: %v", err)
	}
}
