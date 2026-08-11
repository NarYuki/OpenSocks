package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const maxLogSize = 512 << 10

var logMu sync.Mutex

// logf writes to stderr and to /var/log/opensocks.log (for the LuCI log view).
func logf(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	msg := fmt.Sprintf(format, args...)
	line := time.Now().Format("2006-01-02 15:04:05") + " " + msg + "\n"
	os.Stderr.WriteString(line)
	if info, err := os.Stat("/var/log/opensocks.log"); err == nil && info.Size() >= maxLogSize {
		_ = os.Remove("/var/log/opensocks.log.1")
		_ = os.Rename("/var/log/opensocks.log", "/var/log/opensocks.log.1")
	}
	if f, err := os.OpenFile("/var/log/opensocks.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
		_ = f.Chmod(0600)
		f.WriteString(line)
		f.Close()
	}
}

// tailLog returns the last n lines of the daemon log.
func tailLog(n int) string {
	logMu.Lock()
	defer logMu.Unlock()
	f, err := os.Open("/var/log/opensocks.log")
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return ""
	}
	const tailWindow = 128 << 10
	start := info.Size() - tailWindow
	if start < 0 {
		start = 0
	}
	_, _ = f.Seek(start, 0)
	data := make([]byte, info.Size()-start)
	read, _ := f.Read(data)
	data = data[:read]
	if start > 0 {
		if first := strings.IndexByte(string(data), '\n'); first >= 0 {
			data = data[first+1:]
		}
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func boolPtr(b bool) *bool { return &b }

func stringContainsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

func parseRegionID(region string) *int {
	// region is a free-form filter string; the server accepts regionID.
	// For named regions we let the server auto-select; only pass through
	// explicit numeric ids.
	n := 0
	if region == "" {
		return nil
	}
	for _, c := range region {
		if c < '0' || c > '9' {
			return nil
		}
		n = n*10 + int(c-'0')
	}
	return &n
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
