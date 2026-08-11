package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const ooklaServersURL = "https://www.speedtest.net/api/js/servers?engine=js&search=China&limit=30"
const speedTestCNNodesURL = "https://nodes-api.speedtest.cn?type=multi&https=1&browser=1&domainType=2&use_cdn=1"
const speedSOCKSPIDFile = "/tmp/opensocks/speedtest-ss-local.pid"
const speedDualSOCKSPIDFile = "/tmp/opensocks/speedtest-ss-local-2.pid"
const speedTripleSOCKSPIDFile = "/tmp/opensocks/speedtest-ss-local-3.pid"
const speedDualSOCKSPort = 7893
const speedTripleSOCKSPort = 7895

var speedDualPath atomic.Bool
var speedPathCount atomic.Int32

type speedServer struct {
	URL       string  `json:"url"`
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	CC        string  `json:"cc"`
	Sponsor   string  `json:"sponsor"`
	ID        string  `json:"id"`
	Host      string  `json:"host"`
	Latitude  float64 `json:"lat,string,omitempty"`
	Longitude float64 `json:"lon,string,omitempty"`
	PingMS    float64 `json:"ping_ms,omitempty"`
}
type speedResult struct {
	Server            speedServer `json:"server"`
	PingMS            float64     `json:"ping_ms"`
	DownloadMbps      float64     `json:"download_mbps"`
	UploadMbps        float64     `json:"upload_mbps"`
	BytesDownloaded   int64       `json:"bytes_downloaded"`
	BytesUploaded     int64       `json:"bytes_uploaded"`
	RequestedServerID string      `json:"requested_server_id,omitempty"`
	Fallback          bool        `json:"fallback"`
}

type speedTestCNServer struct {
	ID          string  `json:"id"`
	Host        string  `json:"host"`
	Province    string  `json:"province"`
	City        string  `json:"city"`
	Operator    string  `json:"operator"`
	Sponsor     string  `json:"sponsor"`
	PingURL     string  `json:"pingUrl"`
	DownloadURL string  `json:"downloadUrl"`
	UploadURL   string  `json:"uploadUrl"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	PingMS      float64 `json:"ping_ms,omitempty"`
}

func measureSpeedServerPings[T *speedServer | *speedTestCNServer](servers []T, hostOf func(T) string, setPing func(T, float64)) {
	jobs := make(chan T)
	var wg sync.WaitGroup
	for worker := 0; worker < 6; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for server := range jobs {
				host, portText, err := net.SplitHostPort(hostOf(server))
				if err != nil {
					continue
				}
				port, err := strconv.Atoi(portText)
				if err != nil {
					continue
				}
				if result := pingTarget(host, port); result.Reachable {
					setPing(server, result.Milliseconds)
				}
			}
		}()
	}
	for i := range servers {
		jobs <- servers[i]
	}
	close(jobs)
	wg.Wait()
}

// measureSpeedHTTPPings measures the same HTTP path used by the benchmark
// through the selected China proxy. A direct-WAN TCP connect can look fast even
// when that node is unusably slow from the active OpenSocks line.
func measureSpeedHTTPPings[T *speedServer | *speedTestCNServer](servers []T, urlOf func(T) string, setPing func(T, float64)) {
	speedMu.Lock()
	cmd, err := startSpeedSOCKS()
	if err != nil {
		speedMu.Unlock()
		return
	}
	defer func() {
		stopSpeedSOCKS(cmd)
		speedMu.Unlock()
	}()
	client := &http.Client{
		Transport: &http.Transport{DialContext: socksDial, DisableKeepAlives: true},
		Timeout:   3500 * time.Millisecond,
	}
	jobs := make(chan T)
	var wg sync.WaitGroup
	workers := 8
	if len(servers) < workers {
		workers = len(servers)
	}
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for server := range jobs {
				started := time.Now()
				separator := "?"
				if strings.Contains(urlOf(server), "?") {
					separator = "&"
				}
				resp, requestErr := client.Get(urlOf(server) + separator + "opensocks_ping=" + fmt.Sprint(time.Now().UnixNano()))
				if requestErr != nil {
					continue
				}
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
				resp.Body.Close()
				if resp.StatusCode < 500 {
					setPing(server, float64(time.Since(started).Microseconds())/1000)
				}
			}
		}()
	}
	for i := range servers {
		jobs <- servers[i]
	}
	close(jobs)
	wg.Wait()
}

type speedTestCNResult struct {
	Server          speedTestCNServer `json:"server"`
	PingMS          float64           `json:"ping_ms"`
	DownloadMbps    float64           `json:"download_mbps"`
	UploadMbps      float64           `json:"upload_mbps"`
	BytesDownloaded int64             `json:"bytes_downloaded"`
	BytesUploaded   int64             `json:"bytes_uploaded"`
}

var speedMu sync.Mutex
var speedTestCNCache struct {
	sync.Mutex
	Servers []speedTestCNServer
	At      time.Time
}
var speedServerCache struct {
	sync.Mutex
	Servers []speedServer
	At      time.Time
}

var builtInChinaSpeedServers = []speedServer{
	{URL: "http://beijing.unicomtest.com:8080/speedtest/upload.php", Name: "Beijing", Country: "China", CC: "CN", Sponsor: "BJ Unicom", ID: "43752", Host: "beijing.unicomtest.com:8080"},
	{URL: "http://speedtest.dukekunshan.edu.cn:8080/speedtest/upload.php", Name: "Kunshan", Country: "China", CC: "CN", Sponsor: "Duke Kunshan University", ID: "30852", Host: "speedtest.dukekunshan.edu.cn:8080"},
	{URL: "http://mobile.shunicomtest.com:8080/speedtest/upload.php", Name: "Shanghai", Country: "China", CC: "CN", Sponsor: "China Unicom 5G", ID: "24447", Host: "mobile.shunicomtest.com.prod.hosts.ooklaserver.net:8080"},
	{URL: "http://speedtest.jsqiuying.com:8080/speedtest/upload.php", Name: "Suzhou", Country: "China", CC: "CN", Sponsor: "JSQY", ID: "16204", Host: "speedtest.jsqiuying.com:8080"},
}

var externalBenchmarkServers = map[string]speedServer{
	"16176": {URL: "http://ookla-speedtest.hgconair.hgc.com.hk:8080/speedtest/upload.php", Name: "Sha Tin", Country: "Hong Kong", CC: "HK", Sponsor: "HGC", ID: "16176", Host: "ookla-speedtest.hgconair.hgc.com.hk:8080"},
	"18475": {URL: "http://klv3-1.speedtest.idv.tw:8080/speedtest/upload.php", Name: "Keelung", Country: "Taiwan", CC: "TW", Sponsor: "Chief Telecom", ID: "18475", Host: "klv3-1.speedtest.idv.tw:8080"},
	"48463": {URL: "http://speed.udx.icscoe.jp:8080/speedtest/upload.php", Name: "Tokyo", Country: "Japan", CC: "JP", Sponsor: "IPA CyberLab 400G", ID: "48463", Host: "speed.udx.icscoe.jp:8080"},
	"70133": {URL: "http://speed.sparcs.net:8080/speedtest/upload.php", Name: "Daejeon", Country: "South Korea", CC: "KR", Sponsor: "SPARCS", ID: "70133", Host: "speed.sparcs.net.prod.hosts.ooklaserver.net:8080"},
	"28910": {URL: "http://speedtest.hnd.fdcservers.net:8080/speedtest/upload.php", Name: "Tokyo", Country: "Japan", CC: "JP", Sponsor: "fdcservers.net", ID: "28910", Host: "speedtest.hnd.fdcservers.net:8080"},
	"32155": {URL: "http://speedtest.hk.chinamobile.com:8080/speedtest/upload.php", Name: "Hong Kong", Country: "Hong Kong", CC: "HK", Sponsor: "CMHK Mobile Service", ID: "32155", Host: "speedtest.hk.chinamobile.com:8080"},
}

func fallbackChinaSpeedServers() []speedServer {
	return append([]speedServer(nil), builtInChinaSpeedServers...)
}

func discoverChinaSpeedServers() ([]speedServer, error) {
	speedServerCache.Lock()
	if len(speedServerCache.Servers) > 0 && time.Since(speedServerCache.At) < 10*time.Minute {
		out := append([]speedServer(nil), speedServerCache.Servers...)
		speedServerCache.Unlock()
		return out, nil
	}
	speedServerCache.Unlock()
	r, err := (&http.Client{Timeout: 15 * time.Second}).Get(ooklaServersURL)
	if err != nil {
		return fallbackChinaSpeedServers(), nil
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fallbackChinaSpeedServers(), nil
	}
	var all []speedServer
	if err = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&all); err != nil {
		return fallbackChinaSpeedServers(), nil
	}
	out := all[:0]
	for _, s := range all {
		if s.CC == "CN" {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if len(out) == 0 {
		return fallbackChinaSpeedServers(), nil
	}
	speedServerCache.Lock()
	speedServerCache.Servers = append([]speedServer(nil), out...)
	speedServerCache.At = time.Now()
	speedServerCache.Unlock()
	return out, nil
}

func decryptSpeedTestCN(value string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(raw) == 0 || len(raw)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("invalid SpeedTest.cn node response")
	}
	block, _ := aes.NewCipher([]byte("5ECC5D62140EC099"))
	cipher.NewCBCDecrypter(block, []byte("E63EA892A702EEAA")).CryptBlocks(raw, raw)
	pad := int(raw[len(raw)-1])
	if pad < 1 || pad > aes.BlockSize || pad > len(raw) {
		return nil, fmt.Errorf("invalid SpeedTest.cn response padding")
	}
	return raw[:len(raw)-pad], nil
}

func discoverSpeedTestCNServers() ([]speedTestCNServer, error) {
	speedTestCNCache.Lock()
	if len(speedTestCNCache.Servers) > 0 && time.Since(speedTestCNCache.At) < 10*time.Minute {
		out := append([]speedTestCNServer(nil), speedTestCNCache.Servers...)
		speedTestCNCache.Unlock()
		return out, nil
	}
	speedTestCNCache.Unlock()
	// SpeedTest.cn's edge rejects the router's non-China WAN address. Fetch the
	// public node directory through the selected China line, just like its web
	// client does for a LAN browser.
	speedMu.Lock()
	cmd, err := startSpeedSOCKS()
	if err != nil {
		speedMu.Unlock()
		return nil, err
	}
	defer func() {
		stopSpeedSOCKS(cmd)
		speedMu.Unlock()
	}()
	req, _ := http.NewRequest("GET", speedTestCNNodesURL, nil)
	req.Header.Set("Referer", "https://www.speedtest.cn/")
	req.Header.Set("Origin", "https://www.speedtest.cn")
	req.Header.Set("User-Agent", "Mozilla/5.0 OpenSocks/SpeedTest.cn")
	r, err := (&http.Client{Transport: &http.Transport{DialContext: socksDial}, Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	var encrypted struct {
		Data string `json:"data"`
	}
	if r.StatusCode != 200 || json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&encrypted) != nil {
		return nil, fmt.Errorf("SpeedTest.cn node API returned HTTP %d", r.StatusCode)
	}
	plain, err := decryptSpeedTestCN(encrypted.Data)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data []speedTestCNServer `json:"data"`
	}
	if json.Unmarshal(plain, &envelope) != nil || len(envelope.Data) == 0 {
		return nil, fmt.Errorf("SpeedTest.cn returned no measurement nodes")
	}
	servers := envelope.Data
	// Mobile nodes are generally reachable through the consumer China lines;
	// keep the API order inside each class but choose one as the safe default.
	sort.SliceStable(servers, func(i, j int) bool {
		return strings.Contains(servers[i].Operator, "移动") && !strings.Contains(servers[j].Operator, "移动")
	})
	if len(servers) > 30 {
		servers = servers[:30]
	}
	speedTestCNCache.Lock()
	speedTestCNCache.Servers, speedTestCNCache.At = append([]speedTestCNServer(nil), servers...), time.Now()
	speedTestCNCache.Unlock()
	return servers, nil
}

func runSpeedTestCN(server speedTestCNServer) (*speedTestCNResult, error) {
	speedMu.Lock()
	defer speedMu.Unlock()
	cmds, dial, err := startSpeedSOCKSPool()
	if err != nil {
		return nil, err
	}
	defer stopSpeedSOCKSPool(cmds)
	tr := &http.Transport{
		DialContext:         dial,
		DisableCompression:  true,
		MaxIdleConns:        6,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     20 * time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: 12 * time.Second}
	setSpeedStage("ping")
	var ping float64
	successes := 0
	for attempt := 0; attempt < 6 && successes < 3; attempt++ {
		st := time.Now()
		pctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		req, _ := http.NewRequestWithContext(pctx, "GET", server.PingURL+"?r="+fmt.Sprint(time.Now().UnixNano()), nil)
		r, e := client.Do(req)
		cancel()
		if e != nil {
			tr.CloseIdleConnections()
			continue
		}
		io.Copy(io.Discard, io.LimitReader(r.Body, 4096))
		r.Body.Close()
		ping += float64(time.Since(st).Microseconds()) / 1000
		successes++
		setSpeedPing(ping / float64(successes))
		tr.CloseIdleConnections()
	}
	if successes == 0 {
		return nil, fmt.Errorf("SpeedTest.cn server did not answer through any active session")
	}
	ping /= float64(successes)
	setSpeedStage("download")
	st := time.Now()
	downloaded, lastStatus, downloadErr := parallelDownload(client, server.DownloadURL+"?size=25000000", 8*time.Second, 96<<20)
	dd := time.Since(st).Seconds()
	if downloaded == 0 {
		if downloadErr != nil {
			return nil, downloadErr
		}
		return nil, fmt.Errorf("SpeedTest.cn download returned HTTP %d with no data", lastStatus)
	}
	tr.CloseIdleConnections()
	setSpeedStage("upload")
	st = time.Now()
	uploaded := parallelUpload(client, server.UploadURL, "application/octet-stream", false, 5*time.Second, 32<<20)
	ud := time.Since(st).Seconds()
	if uploaded == 0 {
		return nil, fmt.Errorf("SpeedTest.cn upload was rejected")
	}
	return &speedTestCNResult{Server: server, PingMS: ping, DownloadMbps: float64(downloaded) * 8 / dd / 1e6, UploadMbps: float64(uploaded) * 8 / ud / 1e6, BytesDownloaded: downloaded, BytesUploaded: uploaded}, nil
}

func startSpeedSOCKS() (*exec.Cmd, error) {
	cmds, _, err := startSpeedSOCKSPoolWithDual(false)
	if err != nil {
		return nil, err
	}
	return cmds[0], nil
}

func startSpeedSOCKSPool() ([]*exec.Cmd, func(context.Context, string, string) (net.Conn, error), error) {
	count := readSettings().SessionCount
	dual := count >= 2 && fileExists(engineDir+"/config-2.yaml")
	return startSpeedSOCKSPoolWithPaths(dual, count >= 3 && fileExists(engineDir+"/config-3.yaml"))
}

func startSpeedSOCKSPoolWithDual(dual bool) ([]*exec.Cmd, func(context.Context, string, string) (net.Conn, error), error) {
	return startSpeedSOCKSPoolWithPaths(dual, false)
}

func startSpeedSOCKSPoolWithPaths(dual, triple bool) ([]*exec.Cmd, func(context.Context, string, string) (net.Conn, error), error) {
	if _, err := exec.LookPath("ss-local"); err != nil {
		return nil, nil, fmt.Errorf("ss-local is required for China-route speed testing")
	}
	// This target has no swap and only 128 MB RAM. Return idle Go pages before
	// starting two encrypted helpers and their HTTP socket buffers.
	debug.FreeOSMemory()
	cleanupStaleSpeedSOCKS()
	configs := []string{engineConf}
	ports := []int{socksPort}
	pidFiles := []string{speedSOCKSPIDFile}
	if dual {
		configs = append(configs, engineDir+"/config-2.yaml")
		ports = append(ports, speedDualSOCKSPort)
		pidFiles = append(pidFiles, speedDualSOCKSPIDFile)
	}
	if triple {
		configs = append(configs, engineDir+"/config-3.yaml")
		ports = append(ports, speedTripleSOCKSPort)
		pidFiles = append(pidFiles, speedTripleSOCKSPIDFile)
	}
	cmds := make([]*exec.Cmd, 0, len(configs))
	for i := range configs {
		// Speed tests are HTTP-only. Avoid allocating UDP relay state in both
		// helpers on low-memory routers.
		cmd := exec.Command("ss-local", "-c", configs[i], "-b", "127.0.0.1", "-l", fmt.Sprint(ports[i]))
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Start(); err != nil {
			stopSpeedSOCKSPool(cmds)
			return nil, nil, err
		}
		cmds = append(cmds, cmd)
		if err := os.WriteFile(pidFiles[i], []byte(strconv.Itoa(cmd.Process.Pid)), 0600); err != nil {
			stopSpeedSOCKSPool(cmds)
			return nil, nil, err
		}
	}
	time.Sleep(500 * time.Millisecond)
	for _, cmd := range cmds {
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			stopSpeedSOCKSPool(cmds)
			return nil, nil, fmt.Errorf("China-route test helper stopped during startup")
		}
	}
	setSpeedDual(len(ports) > 1)
	setSpeedSessions(len(ports))
	speedDualPath.Store(len(ports) > 1)
	speedPathCount.Store(int32(len(ports)))
	var next atomic.Uint32
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		port := ports[int(next.Add(1)-1)%len(ports)]
		return socksDialPort(ctx, address, port)
	}
	return cmds, dial, nil
}

func cleanupStaleSpeedSOCKS() {
	cleanupSpeedSOCKSPID(speedSOCKSPIDFile)
	cleanupSpeedSOCKSPID(speedDualSOCKSPIDFile)
	cleanupSpeedSOCKSPID(speedTripleSOCKSPIDFile)
}

func cleanupSpeedSOCKSPID(pidFile string) {
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err == nil && pid > 1 {
		cmdline, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if strings.Contains(strings.ReplaceAll(string(cmdline), "\x00", " "), "ss-local") {
			if process, findErr := os.FindProcess(pid); findErr == nil {
				_ = process.Kill()
			}
		}
	}
	_ = os.Remove(pidFile)
}

func stopSpeedSOCKS(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	raw, err := os.ReadFile(speedSOCKSPIDFile)
	if err == nil && strings.TrimSpace(string(raw)) == strconv.Itoa(cmd.Process.Pid) {
		_ = os.Remove(speedSOCKSPIDFile)
	}
}

func stopSpeedSOCKSPool(cmds []*exec.Cmd) {
	for _, cmd := range cmds {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}
	_ = os.Remove(speedSOCKSPIDFile)
	_ = os.Remove(speedDualSOCKSPIDFile)
	_ = os.Remove(speedTripleSOCKSPIDFile)
}

func socksDial(ctx context.Context, network, address string) (net.Conn, error) {
	return socksDialPort(ctx, address, socksPort)
}

func socksDialPort(ctx context.Context, address string, portNumber int) (net.Conn, error) {
	d := net.Dialer{Timeout: 8 * time.Second}
	c, err := d.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", portNumber))
	if err != nil {
		return nil, err
	}
	if _, err = c.Write([]byte{5, 1, 0}); err != nil {
		c.Close()
		return nil, err
	}
	b := make([]byte, 2)
	if _, err = io.ReadFull(c, b); err != nil || b[1] != 0 {
		c.Close()
		return nil, fmt.Errorf("SOCKS negotiation failed")
	}
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		c.Close()
		return nil, err
	}
	port := 0
	fmt.Sscan(portText, &port)
	req := []byte{5, 1, 0, 3, byte(len(host))}
	req = append(req, host...)
	p := make([]byte, 2)
	binary.BigEndian.PutUint16(p, uint16(port))
	req = append(req, p...)
	if _, err = c.Write(req); err != nil {
		c.Close()
		return nil, err
	}
	head := make([]byte, 4)
	if _, err = io.ReadFull(c, head); err != nil || head[1] != 0 {
		c.Close()
		return nil, fmt.Errorf("SOCKS connect failed")
	}
	n := 0
	switch head[3] {
	case 1:
		n = 4
	case 4:
		n = 16
	case 3:
		x := make([]byte, 1)
		io.ReadFull(c, x)
		n = int(x[0])
	}
	io.CopyN(io.Discard, c, int64(n+2))
	return c, nil
}

func runChinaSpeedTest(server speedServer) (*speedResult, error) {
	speedMu.Lock()
	defer speedMu.Unlock()
	cmds, dial, err := startSpeedSOCKSPool()
	if err != nil {
		return nil, err
	}
	defer stopSpeedSOCKSPool(cmds)
	tr := &http.Transport{
		DialContext:         dial,
		MaxIdleConns:        6,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     20 * time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: 12 * time.Second}
	base := strings.TrimSuffix(server.URL, "upload.php")
	setSpeedStage("ping")
	var ping float64
	successes := 0
	for attempt := 0; attempt < 6 && successes < 3; attempt++ {
		st := time.Now()
		pctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		req, _ := http.NewRequestWithContext(pctx, "GET", base+"latency.txt?x="+fmt.Sprint(time.Now().UnixNano()), nil)
		r, e := client.Do(req)
		cancel()
		if e != nil {
			tr.CloseIdleConnections()
			continue
		}
		io.Copy(io.Discard, io.LimitReader(r.Body, 4096))
		r.Body.Close()
		ping += float64(time.Since(st).Microseconds()) / 1000
		successes++
		setSpeedPing(ping / float64(successes))
		tr.CloseIdleConnections()
	}
	if successes == 0 {
		return nil, fmt.Errorf("Ookla server did not answer through any active session")
	}
	ping /= float64(successes)
	downloadURL := base + "random4000x4000.jpg?x=" + fmt.Sprint(time.Now().UnixNano())
	setSpeedStage("download")
	st := time.Now()
	downloaded, status, downloadErr := parallelDownload(client, downloadURL, 8*time.Second, 96<<20)
	dd := time.Since(st).Seconds()
	if downloaded == 0 {
		if downloadErr != nil {
			return nil, downloadErr
		}
		return nil, fmt.Errorf("Ookla download test returned HTTP %d with no data; choose another China server", status)
	}
	tr.CloseIdleConnections()
	setSpeedStage("upload")
	st = time.Now()
	uploaded := parallelUpload(client, server.URL, "application/x-www-form-urlencoded", true, 5*time.Second, 32<<20)
	ud := time.Since(st).Seconds()
	if uploaded == 0 {
		return nil, fmt.Errorf("Ookla upload test was rejected by server")
	}
	return &speedResult{Server: server, PingMS: ping, DownloadMbps: float64(downloaded) * 8 / dd / 1e6, UploadMbps: float64(uploaded) * 8 / ud / 1e6, BytesDownloaded: downloaded, BytesUploaded: uploaded}, nil
}

func parallelUpload(client *http.Client, rawURL, contentType string, form bool, duration time.Duration, limit int64) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	var total atomic.Int64
	var wg sync.WaitGroup
	for worker := 0; worker < activeSpeedStreams(); worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for ctx.Err() == nil && total.Load() < limit {
				var source io.Reader = io.LimitReader(zeroReader{}, 512<<10)
				length := int64(512 << 10)
				if form {
					source = io.MultiReader(strings.NewReader("content1="), source)
					length += 9
				}
				body := &countingReader{R: source}
				req, _ := http.NewRequestWithContext(ctx, "POST", rawURL+"?stream="+strconv.Itoa(worker)+"&r="+fmt.Sprint(time.Now().UnixNano()), body)
				req.ContentLength = length
				req.Header.Set("Content-Type", contentType)
				resp, _ := client.Do(req)
				total.Add(body.N)
				addSpeedBytes(body.N)
				if resp != nil {
					io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
					resp.Body.Close()
				}
			}
		}(worker)
	}
	wg.Wait()
	return total.Load()
}

type countingReader struct {
	R io.Reader
	N int64
}

// parallelDownload approximates the multi-connection behavior of desktop
// speed-test clients without large buffers. Dual mode uses one 32 KiB reader
// per session; single mode may use the configured number to fill a long path.
func parallelDownload(client *http.Client, rawURL string, duration time.Duration, limit int64) (int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	var total, status atomic.Int64
	var firstErr error
	var errMu sync.Mutex
	var wg sync.WaitGroup
	separator := "?"
	if strings.Contains(rawURL, "?") {
		separator = "&"
	}
	workers := activeSpeedStreams()
	// On dual paths BJ Unicom already saturates download with one stream per
	// session. Extra readers increase cipher/HTTP scheduling without throughput.
	if paths := int(speedPathCount.Load()); paths > 1 && workers > paths {
		workers = paths
	}
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			buffer := make([]byte, 32*1024)
			for ctx.Err() == nil && total.Load() < limit {
				req, _ := http.NewRequestWithContext(ctx, "GET", rawURL+separator+"r="+fmt.Sprint(time.Now().UnixNano())+"&stream="+fmt.Sprint(worker), nil)
				resp, err := client.Do(req)
				if err != nil {
					if ctx.Err() == nil {
						errMu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						errMu.Unlock()
					}
					return
				}
				status.Store(int64(resp.StatusCode))
				n, _ := io.CopyBuffer(speedProgressWriter{}, resp.Body, buffer)
				resp.Body.Close()
				total.Add(n)
			}
		}(worker)
	}
	wg.Wait()
	return total.Load(), int(status.Load()), firstErr
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, e := c.R.Read(p)
	c.N += int64(n)
	return n, e
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
