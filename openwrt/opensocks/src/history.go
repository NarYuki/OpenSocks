package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var historyFile = envOr("OPENSOCKS_STATE_DIR", "/etc/opensocks") + "/history.json"

type connectionHistory struct {
	ID        string `json:"id"`
	Connected string `json:"connected_at"`
	LineID    int    `json:"line_id"`
	LineName  string `json:"line_name"`
	Location  string `json:"location,omitempty"`
	Category  string `json:"category,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Server    string `json:"server,omitempty"`
	Port      int    `json:"port,omitempty"`
}

var historyMu sync.Mutex

func loadHistory() ([]connectionHistory, error) {
	historyMu.Lock()
	defer historyMu.Unlock()
	return loadHistoryUnlocked()
}

func loadHistoryUnlocked() ([]connectionHistory, error) {
	b, err := os.ReadFile(historyFile)
	if os.IsNotExist(err) {
		return []connectionHistory{}, nil
	}
	if err != nil {
		return nil, err
	}
	var records []connectionHistory
	if err := json.Unmarshal(b, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func appendHistory(ln *line, resp *connectResponse) error {
	historyMu.Lock()
	defer historyMu.Unlock()
	records, err := loadHistoryUnlocked()
	if err != nil {
		return err
	}
	record := connectionHistory{
		ID: strconv.FormatInt(time.Now().UnixNano(), 10), Connected: time.Now().Format(time.RFC3339),
		LineID: ln.ID, LineName: firstNonempty(resp.LineName, ln.Name),
		Location: ln.Location, Category: ln.Category,
	}
	if boot := selectedBoot(resp); boot != nil {
		record.Protocol = firstNonempty(boot.Proto, strings.TrimPrefix(resp.ProtoName, "PROTO_"), "SS")
		record.Server, record.Port = boot.Server, boot.Port
	}
	records = append([]connectionHistory{record}, records...)
	if len(records) > 100 {
		records = records[:100]
	}
	if err := os.MkdirAll(filepath.Dir(historyFile), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	tmp := historyFile + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, historyFile)
}

func selectedBoot(resp *connectResponse) *bootsInfo {
	configs := append([]*connectConfig{resp.Config}, resp.Configs...)
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		for _, boot := range orderedBoots(cfg) {
			if boot != nil && boot.Server != "" && boot.Port > 0 {
				return boot
			}
		}
	}
	return nil
}

func findHistory(id string) (*connectionHistory, error) {
	records, err := loadHistory()
	if err != nil {
		return nil, err
	}
	if id == "" && len(records) > 0 {
		return &records[0], nil
	}
	for i := range records {
		if records[i].ID == id {
			return &records[i], nil
		}
	}
	return nil, fmt.Errorf("history entry %q not found", id)
}

func firstNonempty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func sortHistory(records []connectionHistory) {
	sort.SliceStable(records, func(i, j int) bool { return records[i].Connected > records[j].Connected })
}
