package main

import (
	"fmt"
	"sync"
	"time"
)

type speedJobState struct {
	Running      bool    `json:"running"`
	Provider     string  `json:"provider"`
	ServerID     string  `json:"server_id"`
	Stage        string  `json:"stage"`
	PingMS       float64 `json:"ping_ms"`
	CurrentMbps  float64 `json:"current_mbps"`
	DownloadMbps float64 `json:"download_mbps"`
	UploadMbps   float64 `json:"upload_mbps"`
	Bytes        int64   `json:"bytes"`
	Error        string  `json:"error,omitempty"`
	StartedUnix  int64   `json:"started_unix"`
	UpdatedUnix  int64   `json:"updated_unix"`
}

var speedJob = struct {
	sync.Mutex
	State      speedJobState
	stageStart time.Time
	stageBytes int64
}{}

func startSpeedJob(provider, id string) error {
	speedJob.Lock()
	if speedJob.State.Running {
		speedJob.Unlock()
		return fmt.Errorf("a speed test is already running")
	}
	now := time.Now()
	speedJob.State = speedJobState{Running: true, Provider: provider, ServerID: id, Stage: "preparing", StartedUnix: now.UnixMilli(), UpdatedUnix: now.UnixMilli()}
	speedJob.stageStart, speedJob.stageBytes = now, 0
	speedJob.Unlock()
	go runSpeedJob(provider, id)
	return nil
}

func runSpeedJob(provider, id string) {
	var result any
	var err error
	if provider == "speedtestcn" {
		var servers []speedTestCNServer
		servers, err = discoverSpeedTestCNServers()
		if err == nil {
			err = fmt.Errorf("SpeedTest.cn server %s not found", id)
			for _, server := range servers {
				if server.ID == id {
					result, err = runSpeedTestCN(server)
					break
				}
			}
		}
	} else {
		var servers []speedServer
		servers, err = discoverChinaSpeedServers()
		if err == nil {
			err = fmt.Errorf("Ookla server %s not found", id)
			for _, server := range servers {
				if server.ID == id {
					result, err = runChinaSpeedTest(server)
					break
				}
			}
		}
	}
	speedJob.Lock()
	speedJob.State.Running = false
	speedJob.State.UpdatedUnix = time.Now().UnixMilli()
	if err != nil {
		speedJob.State.Stage, speedJob.State.Error = "error", err.Error()
	} else {
		speedJob.State.Stage = "complete"
		switch r := result.(type) {
		case *speedResult:
			speedJob.State.PingMS, speedJob.State.DownloadMbps, speedJob.State.UploadMbps = r.PingMS, r.DownloadMbps, r.UploadMbps
		case *speedTestCNResult:
			speedJob.State.PingMS, speedJob.State.DownloadMbps, speedJob.State.UploadMbps = r.PingMS, r.DownloadMbps, r.UploadMbps
		}
	}
	speedJob.Unlock()
}

func setSpeedStage(stage string) {
	speedJob.Lock()
	if speedJob.State.Running {
		if stage == "upload" && speedJob.State.Stage == "download" {
			speedJob.State.DownloadMbps = speedJob.State.CurrentMbps
		}
		speedJob.State.Stage, speedJob.State.CurrentMbps, speedJob.State.Bytes = stage, 0, 0
		speedJob.stageStart, speedJob.stageBytes = time.Now(), 0
		speedJob.State.UpdatedUnix = time.Now().UnixMilli()
	}
	speedJob.Unlock()
}

func setSpeedPing(ms float64) {
	speedJob.Lock()
	if speedJob.State.Running {
		speedJob.State.PingMS, speedJob.State.UpdatedUnix = ms, time.Now().UnixMilli()
	}
	speedJob.Unlock()
}

func addSpeedBytes(n int64) {
	speedJob.Lock()
	if speedJob.State.Running && (speedJob.State.Stage == "download" || speedJob.State.Stage == "upload") {
		speedJob.stageBytes += n
		d := time.Since(speedJob.stageStart).Seconds()
		if d > 0 {
			speedJob.State.CurrentMbps = float64(speedJob.stageBytes) * 8 / d / 1e6
		}
		speedJob.State.Bytes, speedJob.State.UpdatedUnix = speedJob.stageBytes, time.Now().UnixMilli()
	}
	speedJob.Unlock()
}

func currentSpeedJob() speedJobState { speedJob.Lock(); defer speedJob.Unlock(); return speedJob.State }

type speedProgressWriter struct{}

func (speedProgressWriter) Write(p []byte) (int, error) {
	addSpeedBytes(int64(len(p)))
	return len(p), nil
}
