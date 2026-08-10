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
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ooklaServersURL = "https://www.speedtest.net/api/js/servers?engine=js&search=China&limit=30"
const speedTestCNNodesURL = "https://nodes-api.speedtest.cn?type=multi&https=1&browser=1&domainType=2&use_cdn=1"

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
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
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
	cmd, err := startSpeedSOCKS()
	if err != nil {
		return nil, err
	}
	defer func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() }()
	client := &http.Client{Transport: &http.Transport{DialContext: socksDial, DisableKeepAlives: true, DisableCompression: true}, Timeout: 12 * time.Second}
	setSpeedStage("ping")
	var ping float64
	for i := 0; i < 3; i++ {
		st := time.Now()
		r, e := client.Get(server.PingURL + "?r=" + fmt.Sprint(time.Now().UnixNano()))
		if e != nil {
			return nil, e
		}
		io.Copy(io.Discard, io.LimitReader(r.Body, 4096))
		r.Body.Close()
		ping += float64(time.Since(st).Microseconds()) / 1000
		setSpeedPing(ping / float64(i+1))
	}
	ping /= 3
	setSpeedStage("download")
	st := time.Now()
	downloadDeadline := st.Add(8 * time.Second)
	var downloaded int64
	lastStatus := 0
	buffer := make([]byte, 32*1024)
	for time.Now().Before(downloadDeadline) && downloaded < 64<<20 {
		dctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
		req, _ := http.NewRequestWithContext(dctx, "GET", server.DownloadURL+"?size=25000000&r="+fmt.Sprint(time.Now().UnixNano()), nil)
		r, e := client.Do(req)
		if r != nil {
			lastStatus = r.StatusCode
			n, _ := io.CopyBuffer(speedProgressWriter{}, r.Body, buffer)
			downloaded += n
			r.Body.Close()
		}
		cancel()
		if e != nil && downloaded == 0 {
			return nil, e
		}
	}
	dd := time.Since(st).Seconds()
	if downloaded == 0 {
		return nil, fmt.Errorf("SpeedTest.cn download returned HTTP %d with no data", lastStatus)
	}
	setSpeedStage("upload")
	st = time.Now()
	deadline := st.Add(5 * time.Second)
	var uploaded int64
	for time.Now().Before(deadline) && uploaded < 32<<20 {
		body := &countingReader{R: io.LimitReader(zeroReader{}, 512<<10)}
		uctx, stop := context.WithTimeout(context.Background(), 7*time.Second)
		ureq, _ := http.NewRequestWithContext(uctx, "POST", server.UploadURL+"?r="+fmt.Sprint(time.Now().UnixNano()), body)
		ureq.ContentLength = 512 << 10
		ureq.Header.Set("Content-Type", "application/octet-stream")
		ur, e := client.Do(ureq)
		stop()
		uploaded += body.N
		if ur != nil {
			io.Copy(io.Discard, io.LimitReader(ur.Body, 4096))
			ur.Body.Close()
		}
		if e != nil && body.N == 0 {
			break
		}
	}
	ud := time.Since(st).Seconds()
	if uploaded == 0 {
		return nil, fmt.Errorf("SpeedTest.cn upload was rejected")
	}
	return &speedTestCNResult{Server: server, PingMS: ping, DownloadMbps: float64(downloaded) * 8 / dd / 1e6, UploadMbps: float64(uploaded) * 8 / ud / 1e6, BytesDownloaded: downloaded, BytesUploaded: uploaded}, nil
}

func startSpeedSOCKS() (*exec.Cmd, error) {
	if _, err := exec.LookPath("ss-local"); err != nil {
		return nil, fmt.Errorf("ss-local is required for China-route speed testing")
	}
	cmd := exec.Command("ss-local", "-c", engineConf, "-b", "127.0.0.1", "-l", fmt.Sprint(socksPort), "-u")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	time.Sleep(500 * time.Millisecond)
	return cmd, nil
}

func socksDial(ctx context.Context, network, address string) (net.Conn, error) {
	d := net.Dialer{Timeout: 8 * time.Second}
	c, err := d.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", socksPort))
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
	cmd, err := startSpeedSOCKS()
	if err != nil {
		return nil, err
	}
	defer func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() }()
	tr := &http.Transport{DialContext: socksDial, DisableKeepAlives: true}
	client := &http.Client{Transport: tr, Timeout: 12 * time.Second}
	base := strings.TrimSuffix(server.URL, "upload.php")
	setSpeedStage("ping")
	var ping float64
	for i := 0; i < 3; i++ {
		st := time.Now()
		r, e := client.Get(base + "latency.txt?x=" + fmt.Sprint(time.Now().UnixNano()))
		if e != nil {
			return nil, e
		}
		io.Copy(io.Discard, io.LimitReader(r.Body, 4096))
		r.Body.Close()
		ping += float64(time.Since(st).Microseconds()) / 1000
		setSpeedPing(ping / float64(i+1))
	}
	ping /= 3
	dctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	downloadURL := base + "random4000x4000.jpg?x=" + fmt.Sprint(time.Now().UnixNano())
	if parsed, e := url.Parse(server.URL); e == nil {
		parsed.Path = "/download"
		parsed.RawQuery = "size=25000000"
		downloadURL = parsed.String()
	}
	setSpeedStage("download")
	req, _ := http.NewRequestWithContext(dctx, "GET", downloadURL, nil)
	st := time.Now()
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	downloaded, _ := io.CopyBuffer(speedProgressWriter{}, r.Body, make([]byte, 32*1024))
	r.Body.Close()
	dd := time.Since(st).Seconds()
	if downloaded == 0 {
		return nil, fmt.Errorf("Ookla download test returned HTTP %d with no data; choose another China server", r.StatusCode)
	}
	setSpeedStage("upload")
	st = time.Now()
	deadline := st.Add(5 * time.Second)
	var uploaded int64
	for time.Now().Before(deadline) && uploaded < 32<<20 {
		body := &countingReader{R: io.MultiReader(strings.NewReader("content1="), io.LimitReader(zeroReader{}, 512<<10))}
		uctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		ureq, _ := http.NewRequestWithContext(uctx, "POST", server.URL, body)
		ureq.ContentLength = (512 << 10) + 9
		ureq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		ur, e := client.Do(ureq)
		cancel()
		uploaded += body.N
		if ur != nil {
			io.Copy(io.Discard, io.LimitReader(ur.Body, 4096))
			ur.Body.Close()
		}
		if e != nil && body.N == 0 {
			break
		}
	}
	ud := time.Since(st).Seconds()
	if uploaded == 0 {
		return nil, fmt.Errorf("Ookla upload test was rejected by server")
	}
	return &speedResult{Server: server, PingMS: ping, DownloadMbps: float64(downloaded) * 8 / dd / 1e6, UploadMbps: float64(uploaded) * 8 / ud / 1e6, BytesDownloaded: downloaded, BytesUploaded: uploaded}, nil
}

type countingReader struct {
	R io.Reader
	N int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, e := c.R.Read(p)
	c.N += int64(n)
	addSpeedBytes(int64(n))
	return n, e
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
