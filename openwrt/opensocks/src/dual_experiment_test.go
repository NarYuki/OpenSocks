package main

import "testing"

func TestDistinctProbeLines(t *testing.T) {
	oldAccount := savedAcc
	savedAcc = &account{Expired: false, RemainingDays: 1}
	t.Cleanup(func() { savedAcc = oldAccount })
	lines := []line{{ID: 1, IsFree: true}, {ID: 2}, {ID: 3}}
	first, second, err := distinctProbeLines(lines)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != 2 || second.ID != 3 {
		t.Fatalf("selected lines = %d,%d", first.ID, second.ID)
	}
}

func TestSessionLineIDsFollowPrimaryByDefault(t *testing.T) {
	got := sessionLineIDs(1495, settings{}, 3)
	want := []int{1495, 1495, 1495}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slot %d line = %d, want %d", i+1, got[i], want[i])
		}
	}
}

func TestSessionLineIDsAllowIndependentServers(t *testing.T) {
	cfg := settings{Slot2LineID: 321, Slot3LineID: 654}
	got := sessionLineIDs(1495, cfg, 3)
	want := []int{1495, 321, 654}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slot %d line = %d, want %d", i+1, got[i], want[i])
		}
	}
	if dual := sessionLineIDs(1495, cfg, 2); len(dual) != 2 || dual[1] != 321 {
		t.Fatalf("dual lines = %v", dual)
	}
}
