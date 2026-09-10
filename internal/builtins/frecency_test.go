package builtins

import (
	"testing"
	"time"
)

func TestFrecencyScoreCalculation(t *testing.T) {
	now := time.Now().Unix()

	recent := &FrecencyEntry{
		Path:        "/tmp/recent",
		Count:       5,
		LastVisited: now - 300, // 5 min ago
	}
	if recent.Score() != 20.0 {
		t.Fatalf("expected recent score 20.0, got %f", recent.Score())
	}

	older := &FrecencyEntry{
		Path:        "/tmp/older",
		Count:       10,
		LastVisited: now - 100000, // > 24 hours ago
	}
	if older.Score() != 10.0 {
		t.Fatalf("expected older score 10.0, got %f", older.Score())
	}
}
