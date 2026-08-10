package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// logf writes to stderr and to /var/log/opensocks.log (for the LuCI log view).
func logf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	line := time.Now().Format("2006-01-02 15:04:05") + " " + msg + "\n"
	os.Stderr.WriteString(line)
	if f, err := os.OpenFile("/var/log/opensocks.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
		_ = f.Chmod(0600)
		f.WriteString(line)
		f.Close()
	}
}

// tailLog returns the last n lines of the daemon log.
func tailLog(n int) string {
	data, err := os.ReadFile("/var/log/opensocks.log")
	if err != nil {
		return ""
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
