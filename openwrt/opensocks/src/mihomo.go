package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// mihomo (Clash Meta) engine supervision.
// Config is generated from the resolved server config and written as JSON
// (mihomo accepts both JSON and YAML), then the engine is started with the
// runtime config dir /tmp/opensocks (override with OPENSOCKS_DIR for testing).

var (
	// Runtime assets are intentionally kept in tmpfs. Typical OpenWrt targets
	// have too little writable flash for mihomo plus the GeoIP database.
	engineDir    = envOr("OPENSOCKS_DIR", "/tmp/opensocks")
	engineConf   = func() string { return engineDir + "/config.yaml" }()
	geoipFile    = func() string { return engineDir + "/country.mmdb" }()
	engineBinary = "/usr/bin/mihomo"
	engineAPI    = "127.0.0.1:9090"
	mixedPort    = 7890
	socksPort    = 7891
	dnsPort      = 1053
)

type engine struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	connected bool
	lineName  string
	lineID    int
	server    string
}

func newEngine() *engine {
	return &engine{}
}

// cleanupStaleEngine removes only ss-redir processes created from our config.
// This is needed when the kernel killed the Go parent before it could reap its
// child; otherwise a later start can fail with "address already in use".
func cleanupStaleEngine() {
	entries, _ := os.ReadDir("/proc")
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		cmdline, err := os.ReadFile("/proc/" + entry.Name() + "/cmdline")
		if err != nil {
			continue
		}
		text := strings.ReplaceAll(string(cmdline), "\x00", " ")
		if strings.Contains(text, "ss-redir") && strings.Contains(text, engineConf) {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}
}

// generateConfig renders the mihomo config (JSON) for the resolved server.
func generateConfig(conn *connectResponse, mode string, tun bool) ([]byte, error) {
	proxies := buildProxies(conn)
	if len(proxies) == 0 {
		return nil, fmt.Errorf("no usable proxy config from server")
	}
	proxyNames := make([]string, 0, len(proxies))
	for _, p := range proxies {
		if n, ok := p["name"].(string); ok {
			proxyNames = append(proxyNames, n)
		}
	}

	cfg := map[string]any{
		"mode":                "rule",
		"port":                mixedPort,
		"socks-port":          socksPort,
		"allow-lan":           true, // LAN clients reach the proxy through the router
		"external-controller": engineAPI,
		"log-level":           "info",
		"dns": map[string]any{
			"enable":             true,
			"ipv6":               false,
			"listen":             "0.0.0.0:" + fmt.Sprint(dnsPort),
			"enhanced-mode":      "fake-ip",
			"fake-ip-range":      "28.0.0.0/8",
			"nameserver":         []string{"223.5.5.5", "119.29.29.29", "8.8.4.4", "1.0.0.1"},
			"default-nameserver": []string{"223.5.5.5", "119.29.29.29", "8.8.4.4", "1.0.0.1"},
			"fallback-filter":    map[string]any{"geoip": false, "geoip-code": ""},
			"fake-ip-filter":     []string{"+.stun.*.*", "*.msftncsi.com", "*.mcdn.bilivideo.cn", "WORKGROUP"},
		},
		"proxies":      proxies,
		"proxy-groups": buildGroups(proxyNames),
		"rules":        buildRules(mode),
	}

	if tun {
		cfg["tun"] = map[string]any{
			"enable":                true,
			"stack":                 "system",
			"auto-route":            true,
			"auto-redirect":         true,
			"dns-hijack":            []string{"any:53"},
			"auto-detect-interface": true,
		}
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// buildProxies converts the resolved server config into clash proxy entries.
func buildProxies(conn *connectResponse) []map[string]any {
	var out []map[string]any
	add := func(c *connectConfig) {
		if c == nil {
			return
		}
		// prefer the line's declared protocol, fall back to any available
		for _, b := range orderedBoots(c) {
			if b == nil || b.Server == "" || b.Port <= 0 {
				continue
			}
			if b.Proto == "Trojan" {
				if p := trojanProxy(b); p != nil {
					out = append(out, p)
				}
			} else {
				if p := ssProxy(b); p != nil {
					out = append(out, p)
				}
			}
			return
		}
	}
	add(conn.Config)
	for _, c := range conn.Configs {
		add(c)
	}
	return out
}

// orderedBoots orders the protocol variants so the line's declared protocol
// (Config.proto) is tried first: 0=SS, 1=GTS, 3=SSW, 5=Trojan.
func orderedBoots(c *connectConfig) []*bootsInfo {
	first := map[int]*bootsInfo{0: c.SSConf, 1: c.GTSConf, 3: c.SSWConf, 5: c.TrojanConf}[c.Proto]
	rest := []*bootsInfo{c.SSConf, c.SSWConf, c.GTSConf, c.TrojanConf}
	out := make([]*bootsInfo, 0, len(rest))
	if first != nil {
		out = append(out, first)
	}
	for _, b := range rest {
		if b != nil && b != first {
			out = append(out, b)
		}
	}
	return out
}

func ssProxy(b *bootsInfo) map[string]any {
	return map[string]any{
		"type":     "ss",
		"name":     b.Server + ":" + fmt.Sprint(b.Port),
		"server":   b.Server,
		"port":     b.Port,
		"cipher":   defaultStr(b.Method, "chacha20-ietf-poly1305"),
		"password": b.Password,
		"udp":      true,
	}
}

func trojanProxy(b *bootsInfo) map[string]any {
	p := map[string]any{
		"type":     "trojan",
		"name":     b.Server + ":" + fmt.Sprint(b.Port),
		"server":   b.Server,
		"port":     b.Port,
		"password": b.Password,
		"udp":      true,
	}
	if b.Websocket != nil && b.Websocket.Enabled {
		ws := map[string]any{"path": b.Websocket.Path}
		if b.Websocket.Host != "" {
			ws["headers"] = map[string]any{"Host": b.Websocket.Host}
		}
		p["network"] = "ws"
		p["ws-opts"] = ws
	}
	if b.Ssl != nil {
		if b.Ssl.SNI != "" {
			p["sni"] = b.Ssl.SNI
		}
		if b.Ssl.Verify != nil && !*b.Ssl.Verify {
			p["skip-cert-verify"] = true
		}
	}
	return p
}

func buildGroups(proxyNames []string) []map[string]any {
	return []map[string]any{{
		"name":      "PROXY",
		"type":      "url-test",
		"url":       "https://connect.rom.miui.com/generate_204",
		"interval":  300,
		"tolerance": 120,
		"proxies":   proxyNames,
	}}
}

func buildRules(mode string) []string {
	var rules []string
	if mode != "global" {
		// smart / default routing: only Chinese sites & IPs go through the
		// proxy; everything else (Twitter, YouTube, ...) flows via the WAN
		// directly (MATCH,DIRECT).
		for _, d := range cnDomainSuffixes {
			rules = append(rules, "DOMAIN-SUFFIX,"+d+",PROXY")
		}
		rules = append(rules, "GEOIP,CN,PROXY")
	}
	if mode == "global" {
		rules = append(rules, "MATCH,PROXY")
	} else {
		rules = append(rules, "MATCH,DIRECT")
	}
	return rules
}

// cnDomainSuffixes covers the major Chinese streaming/media/social services
// so they are proxied even when the GeoIP database is not (yet) installed.
// The server's clash_rule endpoint was removed (404 on current servers),
// so this built-in list replaces it.
var cnDomainSuffixes = []string{
	// video / streaming
	"bilibili.com", "b23.tv", "bilivideo.com", "bilivideo.cn", "bilibili.cn", "hdslb.com",
	"iqiyi.com", "iqiyipic.com", "qiyi.com", "71.am",
	"youku.com", "youkucdn.com", "youku.net", "tudou.com",
	"qq.com", "gtimg.com", "qpic.cn", "qqvideo.com", "tencent.com", "tencentmusic.com",
	"y.qq.com", "music.qq.com", "qqmusic.com",
	"mgtv.com", "hunantv.com", "imgsfs.com",
	"sohu.com", "sohucs.com",
	"le.com", "letv.com", "lemall.com",
	"163.com", "netease.com", "music.163.com", "126.net",
	"kugou.com", "kuwo.cn", "kuaishou.com", "douyin.com", "douyincdn.com",
	"kuaishoucdn.com", "zhibo.tv",
	"cntv.cn", "cctv.com", "wasu.cn", "wasuband.cn",
	"pptv.com", "china.com", "fun.tv",
	// social / messaging
	"weibo.com", "weibo.cn", "sina.com.cn", "sinaimg.cn", "sinajs.cn", "weibo.com",
	"weixin.qq.com", "wechat.com", "wechatinc.com", "wx.qq.com",
	"zhihu.com", "zhimg.com",
	"douban.com", "doubanio.com",
	"xiaohongshu.com", "xhscdn.com",
	"tieba.baidu.com", "baidu.com", "bdstatic.com", "bdimg.com", "bcebos.com", "baidupcs.com",
	"iqiyi.com",
	// e-commerce
	"taobao.com", "tmall.com", "alicdn.com", "alibaba.com", "alibabacloud.com", "aliyuncs.com",
	"jd.com", "jdcdn.com", "360buyimg.com", "pinduoduo.com", "yangkeduo.com",
	"meituan.com", "dianping.com", "ele.me", "kaola.com", "suning.com",
	// payment / travel
	"alipay.com", "wechatpay.com", "unionpay.com", "12306.cn", "ctrip.com", "qunar.com",
	// maps / news
	"amap.com", "autonavi.com", "baidu.com",
	"gmw.cn", "people.com.cn", "xinhuanet.com", "chinanews.com", "thepaper.cn", "ifeng.com",
	// finance
	"eastmoney.com", "sinafinance.cn", "hexun.com", "xueqiu.com", "cs.com.cn", "10jqka.com.cn",
	// cloud / common CN CDN
	"myqcloud.com", "qcloud.com", "chinacache.com", "cdngslb.com",
	// domestic games / launchers / account and payment services
	"mihoyo.com", "hoyolab.com", "yuanshen.com", "bh3.com", "benghuai.com", "mhyurl.cn",
	"game.qq.com", "ieg.com", "wegame.com.cn", "dnf.qq.com", "lol.qq.com", "pvp.qq.com",
	"neteasegames.com", "163yun.com", "xyq.163.com", "mc.163.com", "my.163.com",
	"wanmei.com", "perfectworld.com.cn", "changyou.com", "cy.com", "shengqugames.com", "sdo.com",
	"biligame.com", "biligame.net", "4399.com", "7k7k.com", "9game.cn", "uc.cn",
	"kuaikanmanhua.com", "acfun.cn", "acfun.com", "douyu.com", "douyucdn.cn", "huya.com",
	// payment, identity, banking and mini-program infrastructure
	"alipayobjects.com", "alipaydev.com", "antfin.com", "蚂蚁.com", "tenpay.com", "95516.com",
	"weixinbridge.com", "qlogo.cn", "qcloudimg.com", "qqmail.com", "foxmail.com",
	"cmbchina.com", "icbc.com.cn", "ccb.com", "abchina.com", "bankcomm.com", "boc.cn",
}

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// ensureGeoIP downloads the GeoIP database if missing (needed by GEOIP rules).
func ensureGeoIP(url string) error {
	if fileExists(geoipFile) {
		return nil
	}
	if url == "" {
		return fmt.Errorf("geoip db missing and geoip_url is empty")
	}
	if err := download(url, geoipFile); err != nil {
		return fmt.Errorf("geoip download failed: %w", err)
	}
	return nil
}

// start launches the minimal Shadowsocks redirection engine. Current
// Transocks lines provide SS/aes-256-cfb, so a full Clash core is unnecessary.
func (e *engine) start(conn *connectResponse, mode string, tun bool, geoipURL string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunningLocked() {
		return fmt.Errorf("engine already running")
	}
	boot := selectedBoot(conn)
	if boot == nil || strings.EqualFold(boot.Proto, "Trojan") {
		return fmt.Errorf("lightweight engine requires a Shadowsocks line")
	}
	binary := cfgEngineBinary()
	if !fileExists(binary) {
		return fmt.Errorf("lightweight engine %s is not installed", binary)
	}
	conf, err := json.MarshalIndent(map[string]any{
		"server": boot.Server, "server_port": boot.Port,
		"local_address": "0.0.0.0", "local_port": mixedPort,
		"password": boot.Password, "method": boot.Method,
		"timeout": 60, "mode": "tcp_only", "fast_open": false,
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(engineDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(engineConf, conf, 0600); err != nil {
		return err
	}

	cmd := exec.Command(binary, "-c", engineConf)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ss-redir start failed: %w", err)
	}
	e.cmd = cmd
	e.connected = true
	e.lineName = conn.LineName
	e.lineID = conn.LineID
	e.server = boot.Server
	return nil
}

func (e *engine) stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cmd != nil && e.cmd.Process != nil {
		e.cmd.Process.Signal(syscall.SIGTERM)
		e.cmd.Wait()
		e.cmd = nil
	}
	e.connected = false
	e.lineName = ""
	e.lineID = 0
	e.server = ""
}

func (e *engine) isRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isRunningLocked()
}

func (e *engine) isRunningLocked() bool {
	if e.cmd == nil || e.cmd.Process == nil {
		return false
	}
	return e.cmd.Process.Signal(syscall.Signal(0)) == nil
}

// supervise restarts the engine if it died while we are still connected.
func (e *engine) supervise(onRestart func()) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		e.mu.Lock()
		connected := e.connected
		running := e.isRunningLocked()
		last := e.cmd
		e.mu.Unlock()
		if connected && !running && last != nil {
			logf("engine died, requesting restart")
			onRestart()
		}
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
