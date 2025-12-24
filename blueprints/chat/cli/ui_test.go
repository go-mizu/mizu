package cli

import (
	"testing"
	"time"
)

func TestNewUI(t *testing.T) {
	ui := NewUI()
	if ui == nil {
		t.Fatal("NewUI should not return nil")
	}
}

func TestUI_Spinner(t *testing.T) {
	ui := NewUI()

	// Start spinner
	ui.StartSpinner("Loading...")

	// Update spinner
	ui.UpdateSpinner("Still loading...")

	// Let spinner run briefly
	time.Sleep(100 * time.Millisecond)

	// Stop spinner
	ui.StopSpinner("Done", 50*time.Millisecond)
}

func TestUI_SpinnerError(t *testing.T) {
	ui := NewUI()

	ui.StartSpinner("Loading...")
	time.Sleep(50 * time.Millisecond)
	ui.StopSpinnerError("Failed")
}

func TestUI_DoubleStartSpinner(t *testing.T) {
	ui := NewUI()

	// Start first spinner
	ui.StartSpinner("First")

	// Second start should be no-op
	ui.StartSpinner("Second")

	// Stop should still work
	ui.StopSpinner("Done", time.Millisecond)
}

func TestUI_StopSpinnerWithoutStart(t *testing.T) {
	ui := NewUI()

	// Should not panic
	ui.StopSpinner("Done", time.Millisecond)
	ui.StopSpinnerError("Error")
}

func TestUI_UpdateSpinner(t *testing.T) {
	ui := NewUI()

	ui.StartSpinner("Initial")
	ui.UpdateSpinner("Updated")

	time.Sleep(50 * time.Millisecond)
	ui.StopSpinner("Done", time.Millisecond)
}

func TestUI_Icons(t *testing.T) {
	// Verify icons are defined
	icons := []string{
		iconCheck,
		iconCross,
		iconServer,
		iconChannel,
		iconUser,
		iconMessage,
		iconInfo,
		iconWarning,
	}

	for _, icon := range icons {
		if icon == "" {
			t.Error("icon should not be empty")
		}
	}
}

func TestUI_SpinnerFrames(t *testing.T) {
	if len(spinnerFrames) == 0 {
		t.Error("spinnerFrames should not be empty")
	}

	for i, frame := range spinnerFrames {
		if frame == "" {
			t.Errorf("spinnerFrame[%d] should not be empty", i)
		}
	}
}

func TestUI_Styles(t *testing.T) {
	// Test that styles can be rendered without panic
	testStrings := []string{
		titleStyle.Render("test"),
		subtitleStyle.Render("test"),
		labelStyle.Render("test"),
		valueStyle.Render("test"),
		progressStyle.Render("test"),
		successStyle.Render("test"),
		errorStyle.Render("test"),
		warnStyle.Render("test"),
		hintStyle.Render("test"),
		serverStyle.Render("test"),
		channelStyle.Render("test"),
		usernameStyle.Render("test"),
	}

	for _, s := range testStrings {
		if s == "" {
			t.Error("rendered style should not be empty")
		}
	}
}

func TestUI_Colors(t *testing.T) {
	// Verify colors are defined
	colors := []interface{}{
		primaryColor,
		secondaryColor,
		accentColor,
		successColor,
		errorColor,
		warnColor,
		dimColor,
	}

	for i, color := range colors {
		if color == nil {
			t.Errorf("color[%d] should not be nil", i)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	// Just test that it doesn't panic
	_ = isTerminal()
}
