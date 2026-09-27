package main

import "testing"

func TestSplitPaths(t *testing.T) {
	got := splitPaths(" ./a.jpg, ,./b.jpg , ./c.jpg ")
	if len(got) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(got))
	}
	if got[0] != "./a.jpg" || got[1] != "./b.jpg" || got[2] != "./c.jpg" {
		t.Fatalf("unexpected paths: %#v", got)
	}
}
