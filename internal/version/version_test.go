package version

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"equal exact", "1.0.0", "1.0.0", 0},
		{"a greater patch", "1.0.1", "1.0.0", 1},
		{"b greater patch", "1.0.0", "1.0.1", -1},
		{"a greater minor", "1.1.0", "1.0.0", 1},
		{"a greater major", "2.0.0", "1.0.0", 1},
		{"shorter b", "1.0.0", "1.0", 1},
		{"shorter a", "1.0", "1.0.0", -1},
		{"with v prefix a greater", "v1.0.0", "v0.9.0", 1},
		{"with v prefix b greater", "v0.9.0", "v1.0.0", -1},
		{"mixed prefix a greater", "v1.0.0", "0.9.0", 1},
		{"mixed prefix b greater", "0.9.0", "v1.0.0", -1},
		{"with pre-release", "1.0.0-beta", "1.0.0-alpha", 0}, // Same base version
		{"complex", "1.2.3", "1.2.4", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareVersions(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d; want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestParseVersionPart(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"1", 1},
		{"12", 12},
		{"0", 0},
		{"123", 123},
		{"1-beta", 1},
		{"2-alpha", 2},
		{"0-rc1", 0},
		{"abc", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseVersionPart(tt.input)
			if got != tt.expected {
				t.Errorf("parseVersionPart(%q) = %d; want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsNewerRelease(t *testing.T) {
	// Save original version
	origVersion := Version
	defer func() { Version = origVersion }()

	tests := []struct {
		name         string
		current      string
		otherVersion string
		expected     bool
	}{
		{"newer available", "0.9.0", "1.0.0", true},
		{"same version", "1.0.0", "1.0.0", false},
		{"already newer", "1.0.0", "0.9.0", false},
		{"with v prefix", "0.9.0", "v1.0.0", true},
		{"current has v", "v0.9.0", "1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.current
			got := IsNewerRelease(tt.otherVersion)
			if got != tt.expected {
				t.Errorf("IsNewerRelease(%q) with current %q = %v; want %v", tt.otherVersion, tt.current, got, tt.expected)
			}
		})
	}
}

func TestSummarizeReleaseNotes(t *testing.T) {
	tests := []struct {
		name     string
		notes    string
		expected string
	}{
		{
			name:     "short notes",
			notes:    "This is a release",
			expected: "This is a release",
		},
		{
			name:     "long notes",
			notes:    string(make([]byte, 600)),
			expected: string(make([]byte, 500)) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeReleaseNotes(tt.notes)
			if len(got) > 503 { // Account for "..."
				if got[:500] != tt.expected[:500] {
					t.Errorf("summarizeReleaseNotes truncated text mismatch")
				}
			} else if got != tt.expected {
				t.Errorf("summarizeReleaseNotes() = %q; want %q", got, tt.expected)
			}
		})
	}
}

func TestGetInfo(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	origDate := Date
	defer func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	}()

	Version = "1.0.0"
	Commit = "abc123"
	Date = "2024-01-01"

	info := GetInfo()
	if info.Version != "1.0.0" {
		t.Errorf("GetInfo().Version = %q; want %q", info.Version, "1.0.0")
	}
	if info.Commit != "abc123" {
		t.Errorf("GetInfo().Commit = %q; want %q", info.Commit, "abc123")
	}
	if info.Date != "2024-01-01" {
		t.Errorf("GetInfo().Date = %q; want %q", info.Date, "2024-01-01")
	}
	if info.GoVersion == "" {
		t.Errorf("GetInfo().GoVersion should not be empty")
	}
}

func TestIsDev(t *testing.T) {
	// Save original version
	origVersion := Version
	defer func() { Version = origVersion }()

	tests := []struct {
		version  string
		expected bool
	}{
		{"dev", true},
		{"", true},
		{"1.0.0", false},
		{"v1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			Version = tt.version
			got := IsDev()
			if got != tt.expected {
				t.Errorf("IsDev() with version %q = %v; want %v", tt.version, got, tt.expected)
			}
		})
	}
}
