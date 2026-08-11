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
