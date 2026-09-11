package builtins

import (
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input    string
		expected [3]int
	}{
		{"2.1.2", [3]int{2, 1, 2}},
		{"v2.1.2", [3]int{2, 1, 2}},
		{"v3.0.0", [3]int{3, 0, 0}},
		{"1.0", [3]int{1, 0, 0}},
		{"v2.1.3-beta", [3]int{2, 1, 3}},
	}

	for _, tt := range tests {
		res := parseSemver(tt.input)
		if res != tt.expected {
			t.Errorf("parseSemver(%q) = %v; expected %v", tt.input, res, tt.expected)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current  string
		latest   string
		expected bool
	}{
		{"2.1.2", "v2.1.3", true},
		{"2.1.2", "v2.2.0", true},
		{"2.1.2", "v3.0.0", true},
		{"2.1.2", "v2.1.2", false},
		{"2.1.3", "v2.1.2", false},
		{"3.0.0", "v2.9.9", false},
	}

	for _, tt := range tests {
		res := isNewerVersion(tt.current, tt.latest)
		if res != tt.expected {
			t.Errorf("isNewerVersion(%q, %q) = %v; expected %v", tt.current, tt.latest, res, tt.expected)
		}
	}
}
