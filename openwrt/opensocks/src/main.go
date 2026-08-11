package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "dev"

func main() {
	daemon := flag.Bool("daemon", false, "run as daemon")
	dualSessionProbe := flag.Bool("dual-session-probe", false, "experimentally verify two independent account sessions")
	cleanupExperimentSessions := flag.Bool("cleanup-experiment-sessions", false, "remove only sessions created by the legacy dual-session probe")
	recoverMainSession := flag.Bool("recover-main-session", false, "issue and persist the primary session using saved credentials")
	deleteExternalSessions := flag.Bool("delete-external-device-sessions", false, "delete Android/iOS sessions that do not belong to an OpenSocks slot")
	prepareTripleSpeed := flag.Bool("prepare-triple-speed-session", false, "issue an isolated third session for a temporary speed benchmark")
	flag.Parse()
	if *prepareTripleSpeed {
		if err := prepareTripleSpeedSession(); err != nil {
			fmt.Fprintf(os.Stderr, "third speed session failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *deleteExternalSessions {
		if err := deleteExternalDeviceSessions(); err != nil {
			fmt.Fprintf(os.Stderr, "external session cleanup failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *recoverMainSession {
		if err := recoverPrimarySession(); err != nil {
			fmt.Fprintf(os.Stderr, "primary session recovery failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *cleanupExperimentSessions {
		if err := cleanupLegacyExperimentSessions(); err != nil {
			fmt.Fprintf(os.Stderr, "experiment session cleanup failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *dualSessionProbe {
		if err := runDualSessionProbe(); err != nil {
			fmt.Fprintf(os.Stderr, "dual-session probe failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *daemon {
		runDaemon()
		return
	}
	// one-shot CLI mode: print status and exit
	ctl := newController(readSettings())
	ctl.refreshSettings()
	st := ctl.status()
	fmt.Printf("running=%v line=%s(%d) token=%v mode=%s tun=%v\n",
		st["running"], st["lineName"], st["lineID"], st["token"], st["mode"], st["tun"])
	os.Exit(0)
}

func runDaemon() {
	cfg := readSettings()
	ctl := newController(cfg)
	cleanupStaleEngine()
	cleanupStaleSpeedSOCKS()
	startTrafficSampler()
	go observeDNSAnswers()

	// supervisor: restart the engine if it dies while we are connected
	go ctl.engine.supervise(func() { ctl.restartEngine() })

	// auto-connect to a free line on boot
	go ctl.autoConnect()
	go ctl.autoRouteWatchdog()
	if cfg.MobileEnabled {
		token, err := loadOrCreateMobileToken()
		if err != nil {
			logf("mobile API disabled: %v", err)
		} else {
			go func() {
				logf("mobile api on 0.0.0.0:%d (token authentication enabled)", cfg.MobilePort)
				if err := newServer(ctl).listenAndServeMobile(cfg.MobilePort, token); err != nil {
					logf("mobile api stopped: %v", err)
				}
			}()
		}
	}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if currentSpeedJob().Running {
				continue
			}
			ctl.refreshSettings()
			if ctl.engine.isRunning() && ctl.currentSettings().Mode == "smart" {
				refreshDomainRoutes()
			}
		}
	}()

	// procd sends SIGTERM on stop/restart. Clean up the child redirection
	// process and firewall table so neither survives as an orphan.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signals
		logf("shutdown requested; cleaning up proxy and nftables")
		saveTrafficTotals()
		_ = ctl.disconnect()
		os.Exit(0)
	}()

	logf("opensocks %s: control api on 127.0.0.1:%d", version, cfg.ControlPort)
	if err := newServer(ctl).listenAndServe(cfg.ControlPort); err != nil {
		logf("fatal: %v", err)
		os.Exit(1)
	}
}
