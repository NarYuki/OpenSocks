package main

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type latencyResult struct {
	Milliseconds float64
	Reachable    bool
	MeasuredAt   time.Time
}

var (
	latencyMu    sync.Mutex
	latencyCache = map[string]latencyResult{}
	pingTimeRE   = regexp.MustCompile(`time[=<]([0-9]+(?:\.[0-9]+)?)\s*ms`)
)

func lineTarget(ln line) string {
	if strings.TrimSpace(ln.IPAddr) != "" {
		return strings.TrimSpace(ln.IPAddr)
	}
	return strings.TrimSpace(ln.Domain)
}

func pingTarget(target string, port int) latencyResult {
	if target == "" {
		return latencyResult{MeasuredAt: time.Now()}
	}
	latencyMu.Lock()
	if cached, ok := latencyCache[target]; ok && time.Since(cached.MeasuredAt) < 60*time.Second {
		latencyMu.Unlock()
		return cached
	}
	latencyMu.Unlock()

	result := latencyResult{MeasuredAt: time.Now()}
	if port > 0 {
		started := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(target, strconv.Itoa(port)), 1500*time.Millisecond)
		if err == nil {
			conn.Close()
			result.Milliseconds = float64(time.Since(started).Microseconds()) / 1000
			result.Reachable = true
		}
		latencyMu.Lock()
		latencyCache[target] = result
		latencyMu.Unlock()
		return result
	}
	out, err := exec.Command("ping", "-c", "1", "-W", "1", target).CombinedOutput()
	if err == nil {
		if ms, parseErr := parsePingLatency(string(out)); parseErr == nil {
			result.Milliseconds, result.Reachable = ms, true
		}
	}
	latencyMu.Lock()
	latencyCache[target] = result
	latencyMu.Unlock()
	return result
}

func parsePingLatency(output string) (float64, error) {
	match := pingTimeRE.FindStringSubmatch(output)
	if len(match) != 2 {
		return 0, fmt.Errorf("ping latency not found")
	}
	return strconv.ParseFloat(match[1], 64)
}

func measureLineLatencies(lines []lineInfo) {
	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := 6
	if len(lines) < workers {
		workers = len(lines)
	}
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				result := pingTarget(lines[i].Target, lines[i].DetectPort)
				lines[i].Reachable = result.Reachable
				if result.Reachable {
					lines[i].LatencyMS = result.Milliseconds
				}
			}
		}()
	}
	for i := range lines {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}
