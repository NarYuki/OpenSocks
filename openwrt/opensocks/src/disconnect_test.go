package main

import (
	"testing"
	"time"
)

// TestDisconnectSuspendsAutoRoute is the regression test for "disconnect does
// nothing". The disconnect tore the engine down, and autoRouteWatchdog - which
// reconnects whenever the engine is not running - put it back within one 10 s
// tick.
func TestDisconnectSuspendsAutoRoute(t *testing.T) {
	c := &controller{cfg: &settings{}}

	// Nothing disconnected yet: recovery is allowed to act.
	if c.autoRouteSuspended() {
		t.Fatal("auto route must not start out suspended")
	}

	c.suspendAutoRoute()
	if !c.autoRouteSuspended() {
		t.Fatal("a manual disconnect must suspend automatic recovery")
	}
}

// TestConnectResumesAutoRoute proves an explicit connect overrides an earlier
// disconnect, so reconnecting from the UI re-arms recovery immediately instead of
// waiting for the pause to expire.
func TestConnectResumesAutoRoute(t *testing.T) {
	c := &controller{cfg: &settings{}}

	c.suspendAutoRoute()
	if !c.autoRouteSuspended() {
		t.Fatal("suspendAutoRoute did not take effect")
	}

	c.resumeAutoRoute()
	if c.autoRouteSuspended() {
		t.Fatal("resumeAutoRoute must re-enable automatic recovery")
	}
}

// TestAutoRoutePauseExpires proves the suspension is bounded: a disconnect the
// user forgot about must not disable recovery forever. After the pause the
// watchdog is allowed to act again, and the flag is cleared so a stale deadline is
// never re-evaluated.
func TestAutoRoutePauseExpires(t *testing.T) {
	c := &controller{cfg: &settings{}}

	c.suspendAutoRoute()
	// Age the deadline past the pause instead of sleeping for two minutes.
	c.cfgMu.Lock()
	c.autoRouteOffUntil = time.Now().Add(-time.Second)
	c.cfgMu.Unlock()

	if c.autoRouteSuspended() {
		t.Fatal("an expired pause must not keep recovery suspended")
	}
	c.cfgMu.RLock()
	stillOff := c.autoRouteOff
	c.cfgMu.RUnlock()
	if stillOff {
		t.Fatal("autoRouteOff must be cleared once the pause expires")
	}
}
