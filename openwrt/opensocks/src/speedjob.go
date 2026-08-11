package main

import (
	"fmt"
	"sync"
	"sync/atomic"
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
	DualSession  bool    `json:"-"`
	Streams      int     `json:"-"`
	Sessions     int     `json:"sessions"`
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

var speedStreamLimit atomic.Int32
var speedProgressBytes atomic.Int64
var speedProgressUpdated atomic.Int64

func startSpeedJob(provider, id string, streams int) error {
	speedJob.Lock()
	if speedJob.State.Running {
		speedJob.Unlock()
		return fmt.Errorf("a speed test is already running")
	}
	if streams == 0 {
		if readSettings().SessionCount == 3 {
			streams = 6
		} else {
			streams = 4
		}
	}
	if streams < 1 || streams > 8 {
		speedJob.Unlock()
		return fmt.Errorf("speed test streams must be between 1 and 8")
	}
	speedStreamLimit.Store(int32(streams))
	now := time.Now()
	speedJob.State = speedJobState{Running: true, Provider: provider, ServerID: id, Stage: "preparing", Streams: streams, StartedUnix: now.UnixMilli(), UpdatedUnix: now.UnixMilli()}
	speedJob.stageStart, speedJob.stageBytes = now, 0
	speedJob.Unlock()
	go runSpeedJob(provider, id)
	return nil
}

func activeSpeedStreams() int {
	n := int(speedStreamLimit.Load())
	if n < 1 {
		return 4
	}
	return n
}

func runSpeedJob(provider, id string) {
	var result any
	var err error
	if readSettings().SessionCount == 3 && !fileExists(engineDir+"/config-3.yaml") {
		err = prepareTripleSpeedSession()
	}
	if err != nil {
		speedJob.Lock()
		speedJob.State.Running = false
		speedJob.State.Stage, speedJob.State.Error = "error", err.Error()
		speedJob.State.UpdatedUnix = time.Now().UnixMilli()
		speedJob.Unlock()
		return
	}
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
	} else if provider == "ookla-external" {
		server, ok := externalBenchmarkServers[id]
		if !ok {
			err = fmt.Errorf("external Ookla server %s not found", id)
		} else {
			result, err = runChinaSpeedTest(server)
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
	speedProgressBytes.Store(0)
	speedProgressUpdated.Store(0)
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

func setSpeedDual(active bool) {
	speedJob.Lock()
	if speedJob.State.Running {
		speedJob.State.DualSession = active
		speedJob.State.UpdatedUnix = time.Now().UnixMilli()
	}
	speedJob.Unlock()
}

func setSpeedSessions(count int) {
	speedJob.Lock()
	if speedJob.State.Running {
		speedJob.State.Sessions = count
		speedJob.State.DualSession = count > 1
		speedJob.State.UpdatedUnix = time.Now().UnixMilli()
	}
	speedJob.Unlock()
}

func addSpeedBytes(n int64) {
	total := speedProgressBytes.Add(n)
	nowMillis := time.Now().UnixMilli()
	last := speedProgressUpdated.Load()
	// Network readers can call this hundreds of times per second. Updating the
	// UI faster than 20 Hz only adds mutex/time syscall overhead on MT7621.
	if nowMillis-last < 50 || !speedProgressUpdated.CompareAndSwap(last, nowMillis) {
		return
	}
	speedJob.Lock()
	if speedJob.State.Running && (speedJob.State.Stage == "download" || speedJob.State.Stage == "upload") {
		speedJob.stageBytes = total
		d := time.Since(speedJob.stageStart).Seconds()
		if d > 0 {
			speedJob.State.CurrentMbps = float64(speedJob.stageBytes) * 8 / d / 1e6
		}
		speedJob.State.Bytes, speedJob.State.UpdatedUnix = speedJob.stageBytes, nowMillis
	}
	speedJob.Unlock()
}

func currentSpeedJob() speedJobState { speedJob.Lock(); defer speedJob.Unlock(); return speedJob.State }

type speedProgressWriter struct{}

func (speedProgressWriter) Write(p []byte) (int, error) {
	addSpeedBytes(int64(len(p)))
	return len(p), nil
}
