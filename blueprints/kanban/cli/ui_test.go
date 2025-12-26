package cli

import (
	"testing"
	"time"
)

func TestModeString(t *testing.T) {
	tests := []struct {
		dev  bool
		want string
	}{
		{true, "development"},
		{false, "production"},
	}

	for _, tt := range tests {
		got := modeString(tt.dev)
		if got != tt.want {
			t.Errorf("modeString(%v) = %s; want %s", tt.dev, got, tt.want)
		}
	}
}

func TestStartSpinner(t *testing.T) {
	// Test that spinner starts and can be stopped
	stop := StartSpinner("Testing...")

	// Give spinner time to start
	time.Sleep(50 * time.Millisecond)

	// Stop should not panic
	stop()

	// Give cleanup time
	time.Sleep(100 * time.Millisecond)
}

func TestColorConstants(t *testing.T) {
	// Verify color constants are defined
	colors := []struct {
		name  string
		color interface{}
	}{
		{"primaryColor", primaryColor},
		{"secondaryColor", secondaryColor},
		{"errorColor", errorColor},
		{"warnColor", warnColor},
		{"mutedColor", mutedColor},
	}

	for _, c := range colors {
		if c.color == nil {
			t.Errorf("%s is nil", c.name)
		}
	}
}

func TestStyleConstants(t *testing.T) {
	// Verify style constants are defined
	styles := []struct {
		name  string
		style interface{}
	}{
		{"headerStyle", headerStyle},
		{"successStyle", successStyle},
		{"errorStyle", errorStyle},
		{"warnStyle", warnStyle},
		{"mutedStyle", mutedStyle},
		{"keyStyle", keyStyle},
		{"valueStyle", valueStyle},
	}

	for _, s := range styles {
		if s.style == nil {
			t.Errorf("%s is nil", s.name)
		}
	}
}
