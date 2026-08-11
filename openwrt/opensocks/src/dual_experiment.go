package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func prepareTripleSpeedSession() error {
	creds, err := loadCredentials()
	if err != nil {
		return err
	}
	cfg := readSettings()
	if cfg.SelectedLineID <= 0 {
		return fmt.Errorf("no selected line is available")
	}
	autokick := 0
	client := newAPIClientForSlot(cfg.APIDomain, 2)
	login := loginRequest{AuthBy: creds.AuthBy, AuthType: creds.AuthType, Username: creds.Username,
		Password: creds.Password, Email: creds.Email, Phone: creds.Phone, CC: creds.CC, Autokick: &autokick}
	var token string
	for attempt := 0; attempt < 3; attempt++ {
		token, err = client.login(login)
		if err == nil || !strings.Contains(err.Error(), "code\":20013") {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return err
	}
	if err := saveSlotSession(2, token); err != nil {
		return err
	}
	time.Sleep(8 * time.Second)
	recommend := false
	response, err := client.connect(cfg.SelectedLineID, connectRequest{
		Proto: "SS", RecommendLine: &recommend, AvailableProto: []string{"SS", "Trojan", "GTS"}, RegionID: parseRegionID(cfg.Region),
	})
	if err != nil {
		return err
	}
	boot := selectedBoot(response)
	if boot == nil || strings.EqualFold(boot.Proto, "Trojan") {
		return fmt.Errorf("third session returned no Shadowsocks configuration")
	}
	config, err := json.Marshal(map[string]any{
		"server": boot.Server, "server_port": boot.Port, "local_address": "127.0.0.1", "local_port": 7894,
		"password": boot.Password, "method": boot.Method, "timeout": 60, "mode": "tcp_only", "fast_open": false,
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(engineDir+"/config-3.yaml", config, 0600); err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(token))
	fmt.Printf("third session prepared line=%d token=%x\n", response.LineID, digest[:4])
	return nil
}

func legacyExperimentIdentity(slot int) (string, string) {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:opensocks-session:%d", hashDeviceID(), slot)))
	macBytes := append([]byte(nil), digest[:6]...)
	macBytes[0] = (macBytes[0] | 0x02) & 0xfe
	mac := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", macBytes[0], macBytes[1], macBytes[2], macBytes[3], macBytes[4], macBytes[5])
	return mac, hex.EncodeToString(digest[:16])
}

func recoverPrimarySession() error {
	creds, err := loadCredentials()
	if err != nil {
		return err
	}
	config := readSettings()
	client := newAPIClientForSlot(config.APIDomain, 0)
	session, err := client.login(loginRequest{
		AuthBy: creds.AuthBy, AuthType: creds.AuthType, Username: creds.Username,
		Password: creds.Password, Email: creds.Email, Phone: creds.Phone, CC: creds.CC,
	})
	if err != nil {
		return err
	}
	if err := saveSession(session); err != nil {
		return err
	}
	fmt.Println("primary session recovered")
	return nil
}

func deleteExternalDeviceSessions() error {
	config := readSettings()
	if config.Token == "" {
		return fmt.Errorf("primary daemon session is unavailable")
	}
	client := newAPIClientForSlot(config.APIDomain, 0)
	client.configure(config.APIDomain, config.Token)
	sessions, err := client.getSessions()
	if err != nil {
		return err
	}
	protectedMACs := map[string]bool{}
	for slot := 0; slot < 3; slot++ {
		mac, _ := deviceIdentityForSlot(slot)
		protectedMACs[mac] = true
	}
	removed := 0
	for _, session := range sessions {
		if session.Session == "" || protectedMACs[session.MAC] {
			continue
		}
		isExternalAndroid := session.Device == "Android" && session.Package == "com.fobwifi.normal"
		isExternalIOS := session.Device == "iOS" || session.Package == "com.fobwifi.fobwifi"
		if !isExternalAndroid && !isExternalIOS {
			continue
		}
		if err := client.deleteSession(session.Session); err != nil {
			return fmt.Errorf("delete %s session: %w", session.Device, err)
		}
		removed++
	}
	fmt.Printf("external sessions removed=%d\n", removed)
	return nil
}

func cleanupLegacyExperimentSessions() error {
	config := readSettings()
	if config.Token == "" {
		return fmt.Errorf("saved daemon session is unavailable")
	}
	client := newAPIClient(config.APIDomain)
	client.legacyDevice = true
	client.configure(config.APIDomain, config.Token)
	sessions, err := client.getSessions()
	if err != nil {
		creds, loadErr := loadCredentials()
		if loadErr != nil {
			return err
		}
		client = newAPIClientForSlot(config.APIDomain, 1)
		_, loginErr := client.login(loginRequest{
			AuthBy: creds.AuthBy, AuthType: creds.AuthType, Username: creds.Username,
			Password: creds.Password, Email: creds.Email, Phone: creds.Phone, CC: creds.CC,
		})
		if loginErr != nil {
			return fmt.Errorf("saved session invalid (%v); recovery login failed: %w", err, loginErr)
		}
		sessions, err = client.getSessions()
		if err != nil {
			return err
		}
	}
	legacyMACs := map[string]bool{}
	for slot := 1; slot <= 2; slot++ {
		mac, _ := legacyExperimentIdentity(slot)
		legacyMACs[mac] = true
	}
	removed := 0
	for _, session := range sessions {
		knownFormerOpenSocks := session.MAC == "" && session.Device == "Phone" && session.Model == "Pixel 8" && session.Package == "com.fobwifi.normal"
		if (!legacyMACs[session.MAC] && !knownFormerOpenSocks) || session.Session == "" || session.Session == config.Token {
			continue
		}
		if err := client.deleteSession(session.Session); err != nil {
			return fmt.Errorf("delete identified experimental session: %w", err)
		}
		removed++
	}
	fmt.Printf("identified=%d removed=%d preserved=%d\n", len(sessions), removed, len(sessions)-removed)
	if removed == 0 {
		currentMAC := macAddr()
		for _, session := range sessions {
			fingerprint := sha256.Sum256([]byte(session.Session + "\x00" + session.MAC))
			fmt.Printf("session=%x active=%d current=%t device=%q model=%q package=%q has_mac=%t\n",
				fingerprint[:4], session.ActiveAt, session.Session == config.Token || session.MAC == currentMAC,
				session.Device, session.Model, session.Package, session.MAC != "")
		}
	}
	return nil
}

func saveSlotSession(slot int, session string) error {
	dir := envOr("OPENSOCKS_STATE_DIR", "/etc/opensocks")
	if session == "" {
		return fmt.Errorf("slot %d has an empty issued session", slot+1)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, fmt.Sprintf("session_%d", slot+1))
	tmp, err := os.CreateTemp(dir, ".slot-session-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(session); err != nil {
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

func (c *controller) connectDualLine(ln *line, request connectRequest, switching, manualSelection bool, cfg settings) error {
	return c.connectMultiLine(ln, request, switching, manualSelection, cfg, 2)
}

func (c *controller) connectMultiLine(ln *line, request connectRequest, switching, manualSelection bool, cfg settings, count int) error {
	clients, responses, tokens, err := issueConnections(cfg.APIDomain, ln.ID, request, count)
	if err != nil {
		return err
	}
	cleanup := func() {
		for i := range clients {
			if responses[i] != nil {
				_ = clients[i].disconnect(responses[i].LineID)
			}
		}
	}
	if switching {
		c.engine.stop()
		teardownRedirect()
	}
	if err := c.engine.startConnections(responses...); err != nil {
		cleanup()
		return err
	}
	if err := setupRedirect(cfg.Mode, c.engine.server, count); err != nil {
		c.engine.stop()
		cleanup()
		return err
	}
	// The primary slot becomes the controller session; the secondary remains
	// isolated in its own client and root-only session file.
	c.api = clients[0]
	if err := saveSession(tokens[0]); err != nil {
		logf("could not persist primary dual session: %v", err)
	}
	c.setSessionToken(tokens[0])
	if err := appendHistory(ln, responses[0]); err != nil {
		logf("warning: could not save dual connection history: %v", err)
	}
	if manualSelection || cfg.SelectedLineID <= 0 {
		saveSetting("selected_line_id", fmt.Sprint(ln.ID))
		c.refreshSettings()
	}
	return nil
}

func issueDualConnections(domain string, lineID int, request connectRequest) ([]*apiClient, []*connectResponse, []string, error) {
	return issueConnections(domain, lineID, request, 2)
}

func issueConnections(domain string, lineID int, request connectRequest, count int) ([]*apiClient, []*connectResponse, []string, error) {
	if count < 2 || count > 3 {
		return nil, nil, nil, fmt.Errorf("session count must be 2 or 3")
	}
	creds, err := loadCredentials()
	if err != nil {
		return nil, nil, nil, err
	}
	authType := creds.AuthType
	if authType == "" {
		authType = "jwt"
	}
	autokick := 0
	login := loginRequest{AuthBy: creds.AuthBy, AuthType: authType, Username: creds.Username,
		Password: creds.Password, Email: creds.Email, Phone: creds.Phone, CC: creds.CC, Autokick: &autokick}
	clients := make([]*apiClient, count)
	for i := range clients {
		clients[i] = newAPIClientForSlot(domain, i)
	}
	tokens := make([]string, count)
	errs := make([]error, count)
	var wg sync.WaitGroup
	for i, client := range clients {
		wg.Add(1)
		go func(index int, api *apiClient) {
			defer wg.Done()
			for attempt := 0; attempt < 3; attempt++ {
				tokens[index], errs[index] = api.login(login)
				if errs[index] == nil || !strings.Contains(errs[index].Error(), "code\":20013") {
					break
				}
				time.Sleep(time.Duration(750+index*250) * time.Millisecond)
			}
			if errs[index] == nil {
				errs[index] = saveSlotSession(index, tokens[index])
			}
		}(i, client)
	}
	wg.Wait()
	for i, loginErr := range errs {
		if loginErr != nil {
			return nil, nil, nil, fmt.Errorf("session slot %d login: %w", i+1, loginErr)
		}
	}
	time.Sleep(8 * time.Second)
	responses := make([]*connectResponse, count)
	for i, client := range clients {
		wg.Add(1)
		go func(index int, api *apiClient) {
			defer wg.Done()
			responses[index], errs[index] = api.connect(lineID, request)
		}(i, client)
	}
	wg.Wait()
	for i, connectErr := range errs {
		if connectErr != nil {
			return nil, nil, nil, fmt.Errorf("session slot %d connect: %w", i+1, connectErr)
		}
		if responses[i] == nil || selectedBoot(responses[i]) == nil {
			return nil, nil, nil, fmt.Errorf("session slot %d returned no Shadowsocks config", i+1)
		}
	}
	return clients, responses, tokens, nil
}

// runDualSessionProbe uses the two already-issued account sessions matching
// the two persistent OpenWrt identities, with isolated concurrent clients.
func runDualSessionProbe() error {
	creds, err := loadCredentials()
	if err != nil {
		return fmt.Errorf("load saved credentials: %w", err)
	}
	config := readSettings()
	if config.SessionCount < 2 {
		return fmt.Errorf("multi-session mode is disabled")
	}
	authType := creds.AuthType
	if authType == "" {
		authType = "jwt"
	}
	login := loginRequest{
		AuthBy: creds.AuthBy, AuthType: authType, Username: creds.Username,
		Password: creds.Password, Email: creds.Email, Phone: creds.Phone, CC: creds.CC,
		Autokick: func() *int { value := 0; return &value }(),
	}
	clients := []*apiClient{newAPIClientForSlot(config.APIDomain, 0), newAPIClientForSlot(config.APIDomain, 1)}
	tokens := make([]string, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, client := range clients {
		wg.Add(1)
		go func(index int, api *apiClient) {
			defer wg.Done()
			for attempt := 0; attempt < 3; attempt++ {
				tokens[index], errs[index] = api.login(login)
				if errs[index] == nil || !strings.Contains(errs[index].Error(), "code\":20013") {
					break
				}
				time.Sleep(time.Duration(750+index*250) * time.Millisecond)
			}
			if errs[index] == nil {
				errs[index] = saveSlotSession(index, tokens[index])
			}
		}(i, client)
	}
	wg.Wait()
	for i, loginErr := range errs {
		if loginErr != nil {
			return fmt.Errorf("OpenWrt slot %d parallel login: %w", i+1, loginErr)
		}
	}
	time.Sleep(8 * time.Second)

	lineSets := make([][]line, 2)
	for i, client := range clients {
		wg.Add(1)
		go func(index int, api *apiClient) {
			defer wg.Done()
			lineSets[index], errs[index] = validateProbeSession(api)
		}(i, client)
	}
	wg.Wait()
	for i, validateErr := range errs {
		if validateErr != nil {
			return fmt.Errorf("issued session %d validation: %w", i+1, validateErr)
		}
	}
	first, _, err := distinctProbeLines(lineSets[0])
	if err != nil {
		return err
	}
	// Slot 1 is authoritative; slot 2 follows the exact same line.
	selected := []*line{first, first}
	request := connectRequest{Proto: "SS", RecommendLine: boolPtr(false), AvailableProto: []string{"SS"}}
	connections := make([]*connectResponse, 2)
	for i, client := range clients {
		wg.Add(1)
		go func(index int, api *apiClient) {
			defer wg.Done()
			connections[index], errs[index] = api.connect(selected[index].ID, request)
		}(i, client)
	}
	wg.Wait()
	for i, connectErr := range errs {
		if connectErr != nil {
			return fmt.Errorf("issued session %d connect line %d: %w", i+1, selected[i].ID, connectErr)
		}
		defer clients[i].disconnect(connections[i].LineID)
	}
	for i, connection := range connections {
		boot := selectedBoot(connection)
		if boot == nil {
			return fmt.Errorf("issued session %d returned no Shadowsocks configuration", i+1)
		}
		digest := sha256.Sum256([]byte(tokens[i]))
		fmt.Printf("slot=%d session=%x line=%d server=%s:%d\n", i+1, digest[:4], connection.LineID, boot.Server, boot.Port)
	}
	return nil
}

func validateProbeSession(client *apiClient) ([]line, error) {
	var lines []line
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		lines, err = client.getLines()
		if err == nil || !isAPIErrorCode(err, 20001) {
			return lines, err
		}
		time.Sleep(3 * time.Second)
	}
	return nil, err
}

func distinctProbeLines(lines []line) (*line, *line, error) {
	selected := make([]*line, 0, 2)
	for i := range lines {
		if activeVIP(loadAccount()) && lines[i].IsFree {
			continue
		}
		selected = append(selected, &lines[i])
		if len(selected) == 2 {
			return selected[0], selected[1], nil
		}
	}
	return nil, nil, fmt.Errorf("two distinct eligible lines are required")
}
