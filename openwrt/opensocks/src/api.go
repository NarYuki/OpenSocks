package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Transocks REST API client.
// The current server (as of 2026-08) authenticates via a session cookie
// (ABS_FOBWIFISS) issued by POST /api/1/login, and requires device + mac
// parameters on all account-scoped endpoints. Responses use either the
// legacy {"status","code"} or the newer {"state":{"Success",...}} envelope.

const (
	headerContentType = "application/json"
	headerAuth        = "Authorization"
	headerCookie      = "Cookie"
	sessionCookie     = "ABS_FOBWIFISS"
)

type apiClient struct {
	domain    string
	token     string
	cookie    string
	http      *http.Client
	sleep     func(time.Duration)
	stateMu   sync.RWMutex
	requestMu sync.Mutex
}

func newAPIClient(domain string) *apiClient {
	u, err := url.Parse(domain)
	if err != nil || u.Host == "" || (u.Scheme != "https" && !isLoopbackHost(u.Hostname())) {
		domain = "https://abscf2.fobwifi.com"
	}
	return &apiClient{
		domain: strings.TrimRight(domain, "/"),
		http: &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" && !isLoopbackHost(req.URL.Hostname()) {
				return fmt.Errorf("refusing insecure API redirect")
			}
			if len(via) >= 5 {
				return fmt.Errorf("too many API redirects")
			}
			return nil
		}},
		sleep: time.Sleep,
	}
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (c *apiClient) setToken(t string) {
	c.stateMu.Lock()
	c.token = t
	c.stateMu.Unlock()
}

func (c *apiClient) clearAuth() {
	c.stateMu.Lock()
	c.token, c.cookie = "", ""
	c.stateMu.Unlock()
}

func (c *apiClient) configure(domain, fallbackSession string) {
	u, err := url.Parse(domain)
	if err != nil || u.Host == "" || (u.Scheme != "https" && !isLoopbackHost(u.Hostname())) {
		domain = "https://abscf2.fobwifi.com"
	}
	c.stateMu.Lock()
	c.domain = strings.TrimRight(domain, "/")
	if fallbackSession != "" {
		c.token = fallbackSession
		if c.cookie == "" {
			c.cookie = fallbackSession
		}
	}
	c.stateMu.Unlock()
}

func (c *apiClient) hasAuth() bool {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.cookie != "" || c.token != ""
}

func (c *apiClient) authValues() (string, string) {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.token, c.cookie
}

// deviceParams are the BaseRequest fields the server validates.
type deviceParams struct {
	Device      string
	Mac         string
	UUID        string
	Langue      string
	PackageName string
	AppVersion  string
	Org         string
	Channel     string
	WidthHeight string
	SysVersion  string
	Model       string
}

func (c *apiClient) dev() deviceParams {
	return deviceParams{
		Device:      "phone",
		Mac:         macAddr(),
		UUID:        hashDeviceID(),
		Langue:      "en",
		PackageName: "com.fobwifi.normal",
		AppVersion:  "4.4.4",
		Org:         "transocks_mix",
		Channel:     "google",
		WidthHeight: "1080x2400",
		SysVersion:  "24",
		Model:       "OpenWrt",
	}
}

// toMap renders the device params as a JSON object (for request bodies).
func (d deviceParams) toMap() map[string]any {
	return map[string]any{
		"device":       d.Device,
		"mac":          d.Mac,
		"uuid":         d.UUID,
		"langue":       d.Langue,
		"package_name": d.PackageName,
		"app_version":  d.AppVersion,
		"org":          d.Org,
		"channel":      d.Channel,
		"width_height": d.WidthHeight,
		"sys_version":  d.SysVersion,
		"model":        d.Model,
	}
}

// toQuery renders the device params as query string values.
func (d deviceParams) toQuery() url.Values {
	v := url.Values{}
	v.Set("device", d.Device)
	v.Set("mac", d.Mac)
	v.Set("uuid", d.UUID)
	v.Set("langue", d.Langue)
	v.Set("package_name", d.PackageName)
	v.Set("app_version", d.AppVersion)
	v.Set("org", d.Org)
	v.Set("channel", d.Channel)
	v.Set("width_height", d.WidthHeight)
	v.Set("sys_version", d.SysVersion)
	v.Set("model", d.Model)
	return v
}

// --- request/response payloads ------------------------------------------

type tokenPayload struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Exp         int    `json:"exp"`
}

type registerByDeviceRequest struct {
	ID string `json:"id"`
	deviceParams
}

type registerByDeviceResponse struct {
	Token    *tokenPayload `json:"token"`
	Username string        `json:"username"`
	Password string        `json:"password"`
}

type loginRequest struct {
	AuthBy   string `json:"auth_by,omitempty"`
	AuthType string `json:"auth_type,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	CC       string `json:"cc,omitempty"`
	SmsCode  string `json:"sms_code,omitempty"`
	Ticket   string `json:"ticket,omitempty"`
	Autokick *int   `json:"autokick,omitempty"`
}

type loginResponse struct {
	Token         *tokenPayload `json:"token"`
	Email         string        `json:"email"`
	Phone         string        `json:"phone"`
	CC            string        `json:"cc"`
	Nick          string        `json:"nick"`
	UserID        int           `json:"user_id"`
	Expired       bool          `json:"expired"`
	ExpireAt      string        `json:"expire_at"`
	RemainingDays int           `json:"remaining_days"`
	IsRealname    bool          `json:"is_realname"`
	EmailVerified bool          `json:"email_verified"`
	PasswordSetup bool          `json:"password_setup"`
}

type line struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	EnName     string `json:"enName"`
	IPAddr     string `json:"iPAddr"`
	Location   string `json:"location"`
	Category   string `json:"category"`
	IsFree     bool   `json:"isFree"`
	Domain     string `json:"domain"`
	DetectPort int    `json:"detectPort"`
}

type getLinesResponse struct {
	Lines []line `json:"lines"`
}

type connectRequest struct {
	Proto          string   `json:"proto,omitempty"`
	RecommendLine  *bool    `json:"recommendLine,omitempty"`
	AvailableProto []string `json:"available_proto,omitempty"`
	RegionID       *int     `json:"regionID,omitempty"`
}

// bootsInfo mirrors the server's per-protocol config block.
type bootsInfo struct {
	Proto     string `json:"proto"`
	Server    string `json:"server"`
	Port      int    `json:"port"`
	Password  string `json:"password"`
	Method    string `json:"method"`
	ProxyIP   string `json:"proxyIP,omitempty"`
	Websocket *struct {
		Enabled bool   `json:"enabled"`
		Path    string `json:"path"`
		Host    string `json:"host"`
	} `json:"websocket,omitempty"`
	Ssl *struct {
		SNI    string `json:"sni"`
		Verify *bool  `json:"verify"`
	} `json:"ssl,omitempty"`
}

type connectConfig struct {
	Host       string     `json:"host"`
	ProxyIP    string     `json:"proxyIP"`
	Proto      int        `json:"proto"`
	SSConf     *bootsInfo `json:"SSConf"`
	SSWConf    *bootsInfo `json:"SSWConf"`
	GTSConf    *bootsInfo `json:"GTSConf"`
	TrojanConf *bootsInfo `json:"trojanConf"`
}

type connectResponse struct {
	LineID       int              `json:"lineID"`
	LineName     string           `json:"lineName"`
	ProtoName    string           `json:"protoName"`
	BoostSession string           `json:"boostSession"`
	TestURL      string           `json:"testUrl"`
	Config       *connectConfig   `json:"config"`
	Configs      []*connectConfig `json:"configs"`
}

// --- helpers ------------------------------------------------------------

func (c *apiClient) do(method, path string, body any) ([]byte, error) {
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
		if os.Getenv("OPENSOCKS_DEBUG") != "" {
			debugBody := string(b)
			if strings.Contains(path, "login") {
				debugBody = "<redacted login payload>"
			}
			logf("debug: %s %s body=%s", method, path, debugBody)
		}
	}
	c.stateMu.RLock()
	domain, token, cookie := c.domain, c.token, c.cookie
	c.stateMu.RUnlock()
	req, err := http.NewRequest(method, domain+"/"+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", headerContentType)
	if token != "" {
		req.Header.Set(headerAuth, token)
	}
	if cookie != "" {
		req.Header.Set(headerCookie, sessionCookie+"="+cookie)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if sc := resp.Header.Get("Set-Cookie"); sc != "" {
		if v := extractCookie(sc, sessionCookie); v != "" {
			c.stateMu.Lock()
			c.cookie = v
			c.stateMu.Unlock()
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	if err := checkEnvelope(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// doAuthenticated retries the transient 20001 returned while a newly issued
// session is propagating between the authentication and account services.
// Other API and transport errors are returned immediately.
func (c *apiClient) doAuthenticated(method, path string, body any) ([]byte, error) {
	const attempts = 2
	for attempt := 0; ; attempt++ {
		raw, err := c.do(method, path, body)
		if err == nil || !isAPIErrorCode(err, 20001) || attempt == attempts-1 {
			return raw, err
		}
		delay := time.Second << attempt
		logf("authentication not ready; retrying %s %s in %s (%d/%d)", method, path, delay, attempt+2, attempts)
		c.sleep(delay)
	}
}

func extractCookie(setCookieHeader, name string) string {
	for _, part := range strings.Split(setCookieHeader, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, name+"=") {
			v := part[len(name)+1:]
			if i := strings.IndexByte(v, ';'); i >= 0 {
				v = v[:i]
			}
			return v
		}
	}
	return ""
}

// checkEnvelope rejects responses with an error envelope.
// Legacy: {"status":"fail","code":N,"error":...}
// Current: {"state":{"Success":false,"Code":N,"Error":...}}
type apiError struct {
	Code    int
	Message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("api error: code %d: %s", e.Code, e.Message)
}

func isAPIErrorCode(err error, code int) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.Code == code
}

func checkEnvelope(raw []byte) error {
	var env struct {
		Status string `json:"status"`
		Code   int    `json:"code"`
		Error  string `json:"error"`
		State  *struct {
			Success bool   `json:"Success"`
			Code    int    `json:"Code"`
			Error   string `json:"Error"`
		} `json:"state"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return nil
	}
	if env.State != nil && !env.State.Success {
		return &apiError{Code: env.State.Code, Message: env.State.Error}
	}
	if env.Status == "fail" {
		return &apiError{Code: env.Code, Message: env.Error}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func decode[T any](raw []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// --- API methods --------------------------------------------------------

// registerByDevice registers the router as an anonymous device and returns
// a token that grants access to free lines without an account.
// Note: the current server (2026-08) rejects this old endpoint with 400;
// login is the supported path.
func (c *apiClient) registerByDevice() (string, error) {
	req := map[string]any{"id": hashDeviceID()}
	for k, v := range c.dev().toMap() {
		req[k] = v
	}
	raw, err := c.do("POST", "api/1/app/user/register", req)
	if err != nil {
		return "", err
	}
	resp, err := decode[registerByDeviceResponse](raw)
	if err != nil {
		return "", err
	}
	if resp.Token != nil && resp.Token.AccessToken != "" {
		c.setToken(resp.Token.AccessToken)
		return resp.Token.AccessToken, nil
	}
	// device accounts: username+password, then login
	if resp.Username != "" {
		return c.login(loginRequest{AuthBy: "username", Username: resp.Username, Password: resp.Password})
	}
	return "", fmt.Errorf("register: no credentials in response")
}

func (c *apiClient) login(req loginRequest) (string, error) {
	raw, err := c.do("POST", "api/1/login", req)
	if err != nil {
		return "", err
	}
	resp, err := decode[loginResponse](raw)
	if err != nil {
		return "", err
	}
	_, cookie := c.authValues()
	if cookie == "" && resp.Token != nil && resp.Token.AccessToken != "" {
		c.setToken(resp.Token.AccessToken)
	}
	token, cookie := c.authValues()
	if cookie == "" && token == "" {
		return "", fmt.Errorf("login: no session cookie issued (check credentials)")
	}
	// persist account summary for the web UI
	saveAccount(&account{
		Email:         resp.Email,
		Phone:         resp.Phone,
		Nick:          resp.Nick,
		Expired:       resp.Expired,
		ExpireAt:      resp.ExpireAt,
		RemainingDays: resp.RemainingDays,
	})
	if cookie != "" {
		return cookie, nil
	}
	return token, nil
}

func (c *apiClient) getLines() ([]line, error) {
	q := c.dev().toQuery()
	q.Set("requestAll", "true")
	raw, err := c.doAuthenticated("GET", "api/2/line?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := decode[getLinesResponse](raw)
	if err != nil {
		return nil, err
	}
	return resp.Lines, nil
}

func (c *apiClient) connect(lineID int, req connectRequest) (*connectResponse, error) {
	body := map[string]any{}
	for k, v := range c.dev().toMap() {
		body[k] = v
	}
	body["proto"] = req.Proto
	if req.RecommendLine != nil {
		body["recommendLine"] = *req.RecommendLine
	}
	if len(req.AvailableProto) > 0 {
		body["available_proto"] = req.AvailableProto
	}
	if req.RegionID != nil {
		body["regionID"] = *req.RegionID
	}
	raw, err := c.doAuthenticated("POST", fmt.Sprintf("api/2/line/connect/%d", lineID), body)
	if err != nil {
		return nil, err
	}
	return decode[connectResponse](raw)
}

func (c *apiClient) disconnect(lineID int) error {
	body := map[string]any{}
	for k, v := range c.dev().toMap() {
		body[k] = v
	}
	_, err := c.doAuthenticated("POST", fmt.Sprintf("api/2/line/disconnect/%d", lineID), body)
	return err
}
