package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Lightweight transparent routing for firewall4-based OpenWrt. ss-redir
// receives only TCP flows selected by this nftables chain. Smart mode uses a
// compact CIDR list instead of loading a MaxMind database into a large engine.

const chinaRoutesURL = "https://raw.githubusercontent.com/17mon/china_ip_list/master/china_ip_list.txt"
const chinaRoutesSHA256 = "1e855b2493221becffe6261c97efce005f2cea65bf506e3c6ca1687d7e7551ad"
const dynamicRouteTimeout = "30m"

// Startup recovery, explicit connect and the routing watchdog may converge at
// the same time. Serialize complete table replacement so nft never appends a
// second copy of the rules to a table another goroutine just created.
var redirectMu sync.Mutex

func setupRedirect(mode, proxyServer string, sessionCount int) error {
	redirectMu.Lock()
	defer redirectMu.Unlock()
	cfg := readSettings()
	serverIP := resolveIPv4(proxyServer)
	var cidrs []string
	var domainIPs []string
	serviceIPs := map[string][]string{}
	if mode != "global" {
		var err error
		cidrs, err = loadChinaRoutes()
		if err != nil {
			return err
		}
		domainIPs, serviceIPs = cachedRoutingIPs(cfg)
	}

	var script strings.Builder
	tcpRedirect := fmt.Sprintf("redirect to :%d", mixedPort)
	if sessionCount == 2 {
		tcpRedirect = "numgen random mod 2 vmap { 0 : jump tcp_slot1, 1 : jump tcp_slot2 }"
	} else if sessionCount >= 3 {
		tcpRedirect = "numgen random mod 3 vmap { 0 : jump tcp_slot1, 1 : jump tcp_slot2, 2 : jump tcp_slot3 }"
	}
	script.WriteString("table inet opensocks {\n")
	script.WriteString(" counter proxy_up {}\n counter proxy_down {}\n")
	for _, group := range chinaServiceGroups {
		script.WriteString(" counter svc_" + group.Name + "_up {}\n counter svc_" + group.Name + "_down {}\n")
	}
	script.WriteString(" counter svc_other_china_up {}\n counter svc_other_china_down {}\n")
	if sessionCount >= 2 {
		script.WriteString(fmt.Sprintf(" chain tcp_slot1 { meta l4proto tcp counter redirect to :%d; }\n", mixedPort))
		script.WriteString(fmt.Sprintf(" chain tcp_slot2 { meta l4proto tcp counter redirect to :%d; }\n", dualPort))
		if sessionCount >= 3 {
			script.WriteString(fmt.Sprintf(" chain tcp_slot3 { meta l4proto tcp counter redirect to :%d; }\n", triplePort))
		}
	}
	if mode != "global" {
		script.WriteString(" set cn4 { type ipv4_addr; flags interval; auto-merge; elements = {\n")
		script.WriteString(strings.Join(cidrs, ",\n"))
		script.WriteString("\n } }\n")
		script.WriteString(" set domain4 { type ipv4_addr; flags timeout; timeout " + dynamicRouteTimeout + ";")
		if len(domainIPs) > 0 {
			script.WriteString(" elements = { " + strings.Join(domainIPs, ", ") + " }")
		}
		script.WriteString(" }\n")
		writeIPv4Set(&script, "include4", validCIDRs(cfg.IncludeCIDRs))
		writeIPv4Set(&script, "exclude4", validCIDRs(cfg.ExcludeCIDRs))
		for _, group := range chinaServiceGroups {
			writeDynamicIPv4Set(&script, "svc_"+group.Name+"4", serviceIPs[group.Name])
		}
	}
	script.WriteString(" chain prerouting { type nat hook prerouting priority dstnat - 1; policy accept;\n")
	script.WriteString("  iifname != \"" + detectLANDevice() + "\" return\n")
	script.WriteString("  ip daddr { 0.0.0.0/8, 10.0.0.0/8, 100.64.0.0/10, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.168.0.0/16, 224.0.0.0/4, 240.0.0.0/4 } return\n")
	if serverIP != "" {
		script.WriteString("  ip daddr " + serverIP + " return\n")
	}
	if mode == "global" {
		script.WriteString("  meta l4proto tcp counter " + tcpRedirect + "\n")
	} else {
		script.WriteString("  ip daddr @exclude4 return\n")
		script.WriteString("  ip daddr @domain4 meta l4proto tcp counter " + tcpRedirect + "\n")
		script.WriteString("  ip daddr @include4 meta l4proto tcp counter " + tcpRedirect + "\n")
		script.WriteString("  ip daddr @cn4 meta l4proto tcp counter " + tcpRedirect + "\n")
	}
	script.WriteString(" }\n")
	// ss-redir receives UDP through TPROXY. 王者荣耀 uses UDP for PVP, so
	// redirecting TCP alone leaves the actual match outside the China route.
	script.WriteString(" chain prerouting_udp { type filter hook prerouting priority mangle; policy accept;\n")
	script.WriteString("  iifname != \"" + detectLANDevice() + "\" return\n")
	script.WriteString("  ip daddr { 0.0.0.0/8, 10.0.0.0/8, 100.64.0.0/10, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.168.0.0/16, 224.0.0.0/4, 240.0.0.0/4 } return\n")
	if serverIP != "" {
		script.WriteString("  ip daddr " + serverIP + " return\n")
	}
	if mode == "global" {
		script.WriteString(fmt.Sprintf("  meta l4proto udp counter meta mark set 0x51 tproxy ip to :%d accept\n", mixedPort))
	} else {
		script.WriteString("  ip daddr @exclude4 return\n")
		script.WriteString(fmt.Sprintf("  ip daddr @domain4 meta l4proto udp counter meta mark set 0x51 tproxy ip to :%d accept\n", mixedPort))
		script.WriteString(fmt.Sprintf("  ip daddr @include4 meta l4proto udp counter meta mark set 0x51 tproxy ip to :%d accept\n", mixedPort))
		script.WriteString(fmt.Sprintf("  ip daddr @cn4 meta l4proto udp counter meta mark set 0x51 tproxy ip to :%d accept\n", mixedPort))
	}
	script.WriteString(" }\n")
	if mode != "global" {
		script.WriteString(" chain force_china_tcp { type filter hook forward priority filter - 1; policy accept;\n")
		script.WriteString("  iifname \"" + detectLANDevice() + "\" ip daddr @domain4 udp dport 443 counter reject\n")
		script.WriteString("  iifname \"" + detectLANDevice() + "\" ip daddr @cn4 udp dport 443 counter reject\n }\n")
	}
	if mode != "global" {
		lan := detectLANDevice()
		script.WriteString(" chain service_up { type filter hook input priority filter - 3; policy accept;\n")
		for _, group := range chinaServiceGroups {
			script.WriteString("  iifname \"" + lan + "\" ct original ip daddr @svc_" + group.Name + "4 counter name svc_" + group.Name + "_up return\n")
		}
		script.WriteString("  iifname \"" + lan + "\" ct original ip daddr @domain4 counter name svc_other_china_up return\n")
		script.WriteString("  iifname \"" + lan + "\" ct original ip daddr @include4 counter name svc_other_china_up return\n")
		script.WriteString("  iifname \"" + lan + "\" ct original ip daddr @cn4 counter name svc_other_china_up\n")
		script.WriteString("}\n chain service_down { type filter hook output priority filter - 3; policy accept;\n")
		for _, group := range chinaServiceGroups {
			script.WriteString("  oifname \"" + lan + "\" ct original ip daddr @svc_" + group.Name + "4 counter name svc_" + group.Name + "_down return\n")
		}
		script.WriteString("  oifname \"" + lan + "\" ct original ip daddr @domain4 counter name svc_other_china_down return\n")
		script.WriteString("  oifname \"" + lan + "\" ct original ip daddr @include4 counter name svc_other_china_down return\n")
		script.WriteString("  oifname \"" + lan + "\" ct original ip daddr @cn4 counter name svc_other_china_down\n")
		script.WriteString("}\n")
	}
	if serverIP != "" {
		script.WriteString(" chain traffic_out { type filter hook output priority filter - 2; policy accept; ip daddr " + serverIP + " counter name proxy_up; }\n")
		script.WriteString(" chain traffic_in { type filter hook input priority filter - 2; policy accept; ip saddr " + serverIP + " counter name proxy_down; }\n")
	}
	script.WriteString("}\n")

	teardownRedirectLocked()
	if err := setupTPROXYRoute(); err != nil {
		return err
	}
	cmd := exec.Command("nft", "-f", "-")
	cmd.Stdin = strings.NewReader(script.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		teardownTPROXYRoute()
		return fmt.Errorf("nftables setup failed: %w (%s)", err, truncate(string(out), 500))
	}
	rememberDynamicElements("domain4", domainIPs)
	for _, group := range chinaServiceGroups {
		rememberDynamicElements("svc_"+group.Name+"4", serviceIPs[group.Name])
	}
	return nil
}

var proxyDomainHosts = cnDomainSuffixes

type serviceGroup struct {
	Name    string
	Domains []string
}

var chinaServiceGroups = []serviceGroup{
	{"honor_of_kings", []string{"pvp.qq.com", "game.gtimg.cn", "dlied5.qq.com", "ossweb-img.qq.com", "sqimg.qq.com", "msdk.qq.com", "gcloud.qq.com", "tpns.tencent.com", "bugly.qq.com"}},
	{"bilibili", []string{"bilibili.com", "biliapi.com", "biliapi.net", "bilivideo.com", "bilivideo.cn", "bilicdn1.com", "bilicdn2.com", "hdslb.com", "b23.tv", "biligame.com", "acgvideo.com", "upos-hz-mirrorakam.akamaized.net"}},
	{"baidu", []string{"baidu.com", "www.baidu.com", "m.baidu.com", "map.baidu.com", "hao123.com", "bdstatic.com", "bdimg.com", "bcebos.com", "baidubce.com", "baiducontent.com", "baidupcs.com"}},
	{"qq_wechat", []string{"qq.com", "gtimg.com", "gtimg.cn", "qpic.cn", "qlogo.cn", "qzone.com", "qqmail.com", "foxmail.com", "tencent.com", "myqcloud.com", "qcloud.com", "qcloudimg.com", "weixin.qq.com", "weixin.com", "wechat.com", "wechatinc.com", "weixinbridge.com", "wechatpay.com", "tenpay.com", "wxapp.tc.qq.com"}},
	{"alipay_alibaba", []string{"alipay.com", "alipayobjects.com", "alipaydev.com", "antfin.com", "antgroup.com", "taobao.com", "tmall.com", "1688.com", "etao.com", "alicdn.com", "alibaba.com", "aliyun.com", "aliyuncs.com", "alibabacloud.com", "mmstat.com", "tanx.com", "kunlunaq.com", "kunlungr.com"}},
	{"games", []string{"mihoyo.com", "mhyurl.cn", "hoyolab.com", "yuanshen.com", "bh3.com", "benghuai.com", "game.qq.com", "ieg.com", "wegame.com.cn", "dnf.qq.com", "lol.qq.com", "pvp.qq.com", "neteasegames.com", "163yun.com", "xyq.163.com", "mc.163.com", "wanmei.com", "perfectworld.com.cn", "changyou.com", "cy.com", "shengqugames.com", "sdo.com", "biligame.com", "4399.com", "7k7k.com", "9game.cn"}},
	{"video_music", []string{"iqiyi.com", "iqiyipic.com", "qiyi.com", "71.am", "youku.com", "youkucdn.com", "tudou.com", "qqvideo.com", "mgtv.com", "hunantv.com", "music.163.com", "126.net", "qqmusic.com", "tencentmusic.com", "kugou.com", "kuwo.cn", "douyu.com", "douyucdn.cn", "huya.com"}},
	{"social", []string{"weibo.com", "weibo.cn", "sina.com.cn", "sinaimg.cn", "zhihu.com", "zhimg.com", "xiaohongshu.com", "xhscdn.com", "douyin.com", "douyincdn.com", "kuaishou.com", "kuaishoucdn.com", "douban.com", "doubanio.com", "acfun.cn", "acfun.com"}},
	{"shopping", []string{"jd.com", "jdcdn.com", "360buyimg.com", "pinduoduo.com", "yangkeduo.com", "meituan.com", "dianping.com", "ele.me", "kaola.com", "suning.com"}},
	{"banking", []string{"unionpay.com", "95516.com", "cmbchina.com", "icbc.com.cn", "ccb.com", "abchina.com", "bankcomm.com", "boc.cn"}},
	{"travel", []string{"12306.cn", "ctrip.com", "qunar.com", "fliggy.com"}},
	{"maps_news", []string{"amap.com", "autonavi.com", "people.com.cn", "xinhuanet.com", "chinanews.com", "thepaper.cn", "ifeng.com"}},
	{"speedtest_cn", []string{"speedtest.cn", "www.speedtest.cn", "m.speedtest.cn", "bm.speedtest.cn", "b.speedtest.cn"}},
}

var legacyProxyDomainHosts = []string{
	"bilibili.com", "www.bilibili.com", "api.bilibili.com", "passport.bilibili.com",
	"space.bilibili.com", "live.bilibili.com", "api.live.bilibili.com", "bangumi.bilibili.com",
	"app.bilibili.com", "i.w.bilicdn1.com", "a.w.bilicdn1.com", "upos-sz-mirrorali.bilivideo.com",
	"iqiyi.com", "www.iqiyi.com", "api.iqiyi.com", "youku.com", "www.youku.com",
	"v.qq.com", "video.qq.com", "music.163.com", "y.qq.com", "mgtv.com", "douyin.com",
	"weibo.com", "www.weibo.com", "zhihu.com", "www.zhihu.com", "xiaohongshu.com",
	"taobao.com", "www.taobao.com", "tmall.com", "jd.com", "baidu.com", "www.baidu.com",
}

func resolveProxyDomains(cfg *settings) []string {
	hosts := append([]string{}, proxyDomainHosts...)
	hosts = append(hosts, legacyProxyDomainHosts...)
	hosts = append(hosts, splitConfigValues(cfg.IncludeDomains)...)
	excluded := map[string]bool{}
	for _, d := range splitConfigValues(cfg.ExcludeDomains) {
		excluded[strings.ToLower(d)] = true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	jobs := make(chan string)
	results := make(chan string, len(hosts)*4)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range jobs {
				ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
				if err != nil {
					continue
				}
				for _, ip := range ips {
					if ip.To4() != nil {
						results <- ip.String()
					}
				}
			}
		}()
	}
	go func() {
		for _, host := range hosts {
			if excluded[strings.ToLower(host)] {
				continue
			}
			jobs <- host
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	seen := map[string]bool{}
	for ip := range results {
		seen[ip] = true
	}
	out := make([]string, 0, len(seen))
	for ip := range seen {
		out = append(out, ip)
	}
	sort.Strings(out)
	return out
}

func resolveHosts(hosts []string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	seen := map[string]bool{}
	for _, host := range hosts {
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
		if err != nil {
			continue
		}
		for _, ip := range ips {
			if ip.To4() != nil {
				seen[ip.String()] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for ip := range seen {
		out = append(out, ip)
	}
	sort.Strings(out)
	return out
}

var routingIPCache struct {
	sync.Mutex
	at       time.Time
	key      string
	domains  []string
	services map[string][]string
}

// cachedRoutingIPs keeps server switching cheap. Domain classification does
// not depend on the selected proxy server, so resolving hundreds of service
// hosts again on every switch only creates latency and memory pressure.
func cachedRoutingIPs(cfg *settings) ([]string, map[string][]string) {
	key := cfg.IncludeDomains + "\x00" + cfg.ExcludeDomains
	routingIPCache.Lock()
	defer routingIPCache.Unlock()
	if routingIPCache.key == key && len(routingIPCache.domains) > 0 && time.Since(routingIPCache.at) < 10*time.Minute {
		return append([]string(nil), routingIPCache.domains...), cloneServiceIPs(routingIPCache.services)
	}

	services := make(map[string][]string, len(chinaServiceGroups))
	var domains []string
	var serviceMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1 + len(chinaServiceGroups))
	go func() { defer wg.Done(); domains = resolveProxyDomains(cfg) }()
	for _, group := range chinaServiceGroups {
		group := group
		go func() {
			defer wg.Done()
			ips := resolveHosts(group.Domains)
			serviceMu.Lock()
			services[group.Name] = ips
			serviceMu.Unlock()
		}()
	}
	wg.Wait()

	// Shared CDN addresses may legitimately serve multiple products. Preserve
	// every association; ordered nft rules select one group per connection.
	routingIPCache.key, routingIPCache.at = key, time.Now()
	routingIPCache.domains = append([]string(nil), domains...)
	routingIPCache.services = cloneServiceIPs(services)
	return domains, services
}

func cloneServiceIPs(source map[string][]string) map[string][]string {
	out := make(map[string][]string, len(source))
	for name, ips := range source {
		out[name] = append([]string(nil), ips...)
	}
	return out
}

func refreshDomainRoutes() {
	config := readSettings()
	ips, services := cachedRoutingIPs(config)
	if len(ips) == 0 {
		return
	}
	sets := map[string][]string{"domain4": ips}
	for _, group := range chinaServiceGroups {
		if values := services[group.Name]; len(values) > 0 {
			sets["svc_"+group.Name+"4"] = values
		}
	}
	addDynamicElements(sets)
}

var dynamicElementState = struct {
	sync.Mutex
	seen map[string]time.Time
}{seen: map[string]time.Time{}}

func rememberDynamicElements(set string, values []string) {
	dynamicElementState.Lock()
	defer dynamicElementState.Unlock()
	now := time.Now()
	for _, value := range values {
		dynamicElementState.seen[set+"\x00"+value] = now
	}
}

func addDynamicElements(sets map[string][]string) {
	dynamicElementState.Lock()
	now := time.Now()
	// DNS answers can churn indefinitely. Expire bookkeeping as well as nft
	// elements so the daemon's own map remains bounded on long-lived routers.
	for key, last := range dynamicElementState.seen {
		if now.Sub(last) > 2*time.Hour {
			delete(dynamicElementState.seen, key)
		}
	}
	pending := map[string][]string{}
	for set, values := range sets {
		for _, value := range values {
			last := dynamicElementState.seen[set+"\x00"+value]
			if last.IsZero() || now.Sub(last) > 31*time.Minute {
				pending[set] = append(pending[set], value)
			}
		}
	}
	dynamicElementState.Unlock()
	if len(pending) == 0 {
		return
	}
	var batch strings.Builder
	for set, values := range pending {
		batch.WriteString("add element inet opensocks " + set + " { ")
		writeTimedElements(&batch, values)
		batch.WriteString(" }\n")
	}
	cmd := exec.Command("nft", "-f", "-")
	cmd.Stdin = strings.NewReader(batch.String())
	if cmd.Run() == nil {
		for set, values := range pending {
			rememberDynamicElements(set, values)
		}
	}
}

func writeTimedElements(script *strings.Builder, values []string) {
	for i, value := range values {
		if i > 0 {
			script.WriteString(", ")
		}
		script.WriteString(value + " timeout " + dynamicRouteTimeout)
	}
}

func writeDynamicIPv4Set(script *strings.Builder, name string, values []string) {
	script.WriteString(" set " + name + " { type ipv4_addr; flags timeout; timeout " + dynamicRouteTimeout + ";")
	if len(values) > 0 {
		script.WriteString(" elements = { ")
		writeTimedElements(script, values)
		script.WriteString(" }")
	}
	script.WriteString(" }\n")
}

func splitConfigValues(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t' })
}

func validCIDRs(value string) []string {
	var out []string
	for _, v := range splitConfigValues(value) {
		if ip, n, err := net.ParseCIDR(v); err == nil && ip.To4() != nil {
			out = append(out, n.String())
		}
	}
	return out
}

func writeIPv4Set(script *strings.Builder, name string, cidrs []string) {
	script.WriteString(" set " + name + " { type ipv4_addr; flags interval; auto-merge;")
	if len(cidrs) > 0 {
		script.WriteString(" elements = { " + strings.Join(cidrs, ", ") + " }")
	}
	script.WriteString(" }\n")
}

func networkIntegrationState() (string, bool) {
	device := detectLANDevice()
	err := exec.Command("nft", "list", "chain", "inet", "opensocks", "prerouting").Run()
	return device, err == nil
}

func detectLANDevice() string {
	for _, key := range []string{"network.lan.device", "network.lan.ifname"} {
		if out, err := exec.Command("uci", "-q", "get", key).Output(); err == nil {
			if value := strings.TrimSpace(string(out)); value != "" && !strings.ContainsAny(value, " \t\n\";{}") {
				return value
			}
		}
	}
	return "br-lan"
}

func teardownRedirect() {
	redirectMu.Lock()
	defer redirectMu.Unlock()
	teardownRedirectLocked()
}

func teardownRedirectLocked() {
	exec.Command("nft", "delete", "table", "inet", "opensocks").Run()
	teardownTPROXYRoute()
}

func setupTPROXYRoute() error {
	teardownTPROXYRoute()
	if out, err := exec.Command("ip", "rule", "add", "pref", "100", "fwmark", "0x51/0xff", "lookup", "100").CombinedOutput(); err != nil {
		return fmt.Errorf("UDP policy rule setup failed: %w (%s)", err, truncate(string(out), 300))
	}
	if out, err := exec.Command("ip", "route", "add", "local", "0.0.0.0/0", "dev", "lo", "table", "100").CombinedOutput(); err != nil {
		teardownTPROXYRoute()
		return fmt.Errorf("UDP policy route setup failed: %w (%s)", err, truncate(string(out), 300))
	}
	return nil
}

func teardownTPROXYRoute() {
	_ = exec.Command("ip", "rule", "del", "pref", "100", "fwmark", "0x51/0xff", "lookup", "100").Run()
	_ = exec.Command("ip", "route", "flush", "table", "100").Run()
}

func resolveIPv4(host string) string {
	if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
		return ip.String()
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return ""
	}
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}

func loadChinaRoutes() ([]string, error) {
	path := filepath.Join(engineDir, "china_ip_list.txt")
	if !verifiedFile(path, chinaRoutesSHA256) {
		_ = os.Remove(path)
		if err := download(chinaRoutesURL, path); err != nil {
			return nil, fmt.Errorf("China route download failed: %w", err)
		}
	}
	if !verifiedFile(path, chinaRoutesSHA256) {
		_ = os.Remove(path)
		return nil, fmt.Errorf("China route signature hash mismatch")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Fields(string(b))
	cidrs := make([]string, 0, len(lines))
	for _, value := range lines {
		if _, _, err := net.ParseCIDR(value); err == nil {
			cidrs = append(cidrs, value)
		}
	}
	if len(cidrs) == 0 {
		return nil, fmt.Errorf("China route list is empty")
	}
	return cidrs, nil
}

func verifiedFile(path, expected string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)) == expected
}

// download atomically fetches a small runtime data file.
func download(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	client := http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	const maxDownloadSize = 16 << 20
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxDownloadSize+1))
	if err != nil || n > maxDownloadSize {
		f.Close()
		os.Remove(tmp)
		if err != nil {
			return err
		}
		return fmt.Errorf("download exceeds %d bytes", maxDownloadSize)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func decodeJSONBody(r io.Reader, v any) error {
	return json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(v)
}
