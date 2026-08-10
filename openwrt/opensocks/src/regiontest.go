package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type regionTestResult struct {
	Address      string  `json:"address"`
	Country      string  `json:"country"`
	Province     string  `json:"province"`
	City         string  `json:"city"`
	ISP          string  `json:"isp"`
	CountryCode  int     `json:"country_code"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Source       string  `json:"source"`
	Health       string  `json:"health"`
	HealthScore  int     `json:"health_score"`
	RiskScore    int     `json:"risk_score"`
	Proxy        string  `json:"proxy"`
	IPType       string  `json:"ip_type"`
	ASN          string  `json:"asn"`
	Provider     string  `json:"provider"`
	Organization string  `json:"organization"`
	HealthError  string  `json:"health_error,omitempty"`
}

func testChinaExitRegion() (*regionTestResult, error) {
	speedMu.Lock()
	defer speedMu.Unlock()
	cmd, err := startSpeedSOCKS()
	if err != nil {
		return nil, err
	}
	defer func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() }()
	client := &http.Client{
		Transport: &http.Transport{DialContext: socksDial, DisableKeepAlives: true},
		Timeout:   15 * time.Second,
	}
	req, _ := http.NewRequest("GET", "https://api.bilibili.com/x/web-interface/zone?_="+fmt.Sprint(time.Now().UnixNano()), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "OpenSocks/1.0 OpenWrt")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("China exit request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("China exit service returned HTTP %d", resp.StatusCode)
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Address     string  `json:"addr"`
			Country     string  `json:"country"`
			Province    string  `json:"province"`
			City        string  `json:"city"`
			ISP         string  `json:"isp"`
			CountryCode int     `json:"country_code"`
			Latitude    float64 `json:"latitude"`
			Longitude   float64 `json:"longitude"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("invalid China exit response: %w", err)
	}
	if envelope.Code != 0 || envelope.Data.Country == "" {
		return nil, fmt.Errorf("China exit service returned code %d", envelope.Code)
	}
	result := &regionTestResult{
		Address: envelope.Data.Address, Country: envelope.Data.Country,
		Province: envelope.Data.Province, City: envelope.Data.City, ISP: envelope.Data.ISP,
		CountryCode: envelope.Data.CountryCode, Latitude: envelope.Data.Latitude,
		Longitude: envelope.Data.Longitude, Source: "bilibili", Health: "unknown",
	}
	if err := populateIPHealth(result); err != nil {
		result.HealthError = err.Error()
	}
	return result, nil
}

func populateIPHealth(result *regionTestResult) error {
	if net.ParseIP(result.Address) == nil {
		return fmt.Errorf("invalid exit IP")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://proxycheck.io/v2/" + result.Address + "?vpn=1&asn=1&risk=1")
	if err != nil {
		return fmt.Errorf("IP reputation request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("IP reputation service returned HTTP %d", resp.StatusCode)
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(io.LimitReader(resp.Body, 128<<10)).Decode(&raw); err != nil {
		return fmt.Errorf("invalid IP reputation response: %w", err)
	}
	var status string
	_ = json.Unmarshal(raw["status"], &status)
	if status != "ok" {
		return fmt.Errorf("IP reputation lookup failed")
	}
	entryRaw, ok := raw[result.Address]
	if !ok {
		return fmt.Errorf("IP reputation result is missing")
	}
	var entry struct {
		ASN          string          `json:"asn"`
		Provider     string          `json:"provider"`
		Organization string          `json:"organisation"`
		Proxy        string          `json:"proxy"`
		Type         string          `json:"type"`
		Risk         json.RawMessage `json:"risk"`
	}
	if json.Unmarshal(entryRaw, &entry) != nil {
		return fmt.Errorf("invalid IP reputation entry")
	}
	riskText := strings.Trim(string(entry.Risk), "\"")
	risk, err := strconv.ParseFloat(riskText, 64)
	if err != nil {
		risk = 0
	}
	if risk < 0 {
		risk = 0
	}
	if risk > 100 {
		risk = 100
	}
	result.RiskScore = int(risk + 0.5)
	result.HealthScore = 100 - result.RiskScore
	result.Proxy, result.IPType = entry.Proxy, entry.Type
	result.ASN, result.Provider, result.Organization = entry.ASN, entry.Provider, entry.Organization
	result.Health = "good"
	if result.RiskScore > 66 || strings.EqualFold(entry.Proxy, "yes") {
		result.Health = "poor"
	} else if result.RiskScore > 25 {
		result.Health = "warning"
	}
	return nil
}
