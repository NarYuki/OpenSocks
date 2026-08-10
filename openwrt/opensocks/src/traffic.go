package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const trafficStorePath = "/etc/opensocks/traffic.json"

var allCountersRE = regexp.MustCompile(`(?s)counter ([a-zA-Z0-9_]+) \{.*?bytes ([0-9]+)`)

type serviceTraffic struct {
	Up    uint64 `json:"up_bytes"`
	Down  uint64 `json:"down_bytes"`
	Total uint64 `json:"total_bytes"`
}
type persistedTraffic struct {
	Up        uint64                     `json:"up_bytes"`
	Down      uint64                     `json:"down_bytes"`
	Total     uint64                     `json:"total_bytes"`
	Services  map[string]*serviceTraffic `json:"services"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

var trafficState struct {
	sync.Mutex
	Data             persistedTraffic
	UpRate, DownRate float64
}

func readNFTCounters() map[string]uint64 {
	var commands strings.Builder
	commands.WriteString("list counter inet opensocks proxy_up\nlist counter inet opensocks proxy_down\n")
	for _, g := range chinaServiceGroups {
		commands.WriteString("list counter inet opensocks svc_" + g.Name + "_up\nlist counter inet opensocks svc_" + g.Name + "_down\n")
	}
	cmd := exec.Command("nft", "-f", "-")
	cmd.Stdin = strings.NewReader(commands.String())
	out, err := cmd.Output()
	if err != nil {
		return map[string]uint64{}
	}
	result := map[string]uint64{}
	for _, m := range allCountersRE.FindAllSubmatch(out, -1) {
		v, _ := strconv.ParseUint(string(m[2]), 10, 64)
		result[string(m[1])] = v
	}
	return result
}

func loadTrafficTotals() persistedTraffic {
	d := persistedTraffic{Services: map[string]*serviceTraffic{}}
	if b, err := os.ReadFile(trafficStorePath); err == nil {
		_ = json.Unmarshal(b, &d)
	}
	if d.Services == nil {
		d.Services = map[string]*serviceTraffic{}
	}
	if d.Up+d.Down == 0 && d.Total > 0 {
		d.Down = d.Total
	}
	for _, s := range d.Services {
		if s.Up+s.Down == 0 && s.Total > 0 {
			s.Down = s.Total
		}
	}
	return d
}

func saveTrafficTotals() {
	trafficState.Lock()
	trafficState.Data.UpdatedAt = time.Now()
	trafficState.Data.Total = trafficState.Data.Up + trafficState.Data.Down
	b, err := json.Marshal(trafficState.Data)
	trafficState.Unlock()
	if err != nil {
		return
	}
	_ = os.MkdirAll("/etc/opensocks", 0700)
	tmp := trafficStorePath + ".tmp"
	if os.WriteFile(tmp, b, 0600) == nil {
		_ = os.Rename(tmp, trafficStorePath)
	}
}

func delta(current, previous uint64) uint64 {
	if current >= previous {
		return current - previous
	}
	return current
}

func startTrafficSampler() {
	trafficState.Data = loadTrafficTotals()
	go func() {
		previous := map[string]uint64{}
		previousAt := time.Time{}
		saveTicker := time.NewTicker(5 * time.Minute)
		defer saveTicker.Stop()
		for {
			now, current := time.Now(), readNFTCounters()
			trafficState.Lock()
			if !previousAt.IsZero() {
				dt := now.Sub(previousAt).Seconds()
				du, dd := delta(current["proxy_up"], previous["proxy_up"]), delta(current["proxy_down"], previous["proxy_down"])
				trafficState.Data.Up += du
				trafficState.Data.Down += dd
				upNow, downNow := float64(du)/dt, float64(dd)/dt
				if upNow > 0 {
					trafficState.UpRate = upNow
				} else {
					trafficState.UpRate *= 0.6
				}
				if downNow > 0 {
					trafficState.DownRate = downNow
				} else {
					trafficState.DownRate *= 0.6
				}
				for _, g := range chinaServiceGroups {
					s := trafficState.Data.Services[g.Name]
					if s == nil {
						s = &serviceTraffic{}
						trafficState.Data.Services[g.Name] = s
					}
					s.Up += delta(current["svc_"+g.Name+"_up"], previous["svc_"+g.Name+"_up"])
					s.Down += delta(current["svc_"+g.Name+"_down"], previous["svc_"+g.Name+"_down"])
					s.Total = s.Up + s.Down
				}
			}
			trafficState.Unlock()
			previous, current, previousAt = current, nil, now
			select {
			case <-saveTicker.C:
				saveTrafficTotals()
			default:
			}
			time.Sleep(time.Second)
		}
	}()
}

func trafficStats() map[string]any {
	trafficState.Lock()
	defer trafficState.Unlock()
	d := trafficState.Data
	return map[string]any{"up_bytes": d.Up, "down_bytes": d.Down, "total_bytes": d.Up + d.Down, "up_bps": trafficState.UpRate, "down_bps": trafficState.DownRate, "total_bps": trafficState.UpRate + trafficState.DownRate, "services": d.Services, "updated_at": d.UpdatedAt}
}
