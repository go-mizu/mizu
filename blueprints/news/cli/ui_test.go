package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestNewUI(t *testing.T) {
	ui := NewUI()
	if ui == nil {
		t.Error("NewUI should not return nil")
	}
}

func TestUI_Header(t *testing.T) {
	ui := NewUI()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Header(iconInfo, "Test Header")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Header should produce output")
	}
}

func TestUI_Info(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Info("Label", "Value")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Info should produce output")
	}
}

func TestUI_Spinner(t *testing.T) {
	ui := NewUI()

	// Start spinner
	ui.StartSpinner("Loading...")

	// Update spinner
	ui.UpdateSpinner("Still loading...")

	// Starting again should be no-op
	ui.StartSpinner("Again...")

	// Stop with error
	ui.StopSpinnerError("Failed")
}

func TestUI_Success(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Success("Operation completed")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Success should produce output")
	}
}

func TestUI_Error(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Error("Something went wrong")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Error should produce output")
	}
}

func TestUI_Warn(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Warn("Warning message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Warn should produce output")
	}
}

func TestUI_Summary(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Summary([][2]string{
		{"Key1", "Value1"},
		{"Key2", "Value2"},
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Summary should produce output")
	}
}

func TestUI_StoryRow(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.StoryRow("Test Story Title", 42, "author")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("StoryRow should produce output")
	}
}

func TestUI_StoryRow_LongTitle(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Long title should be truncated
	ui.StoryRow("This is a very long title that should be truncated because it exceeds fifty characters", 42, "author")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if buf.Len() == 0 {
		t.Error("StoryRow should produce output for long titles")
	}
}

func TestUI_UserRow(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.UserRow("testuser", 100, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("UserRow should produce output")
	}
}

func TestUI_UserRow_Admin(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.UserRow("adminuser", 1000, true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("UserRow should produce output for admin users")
	}
}

func TestIsTerminal(t *testing.T) {
	// This tests the isTerminal function
	// Result depends on whether we're running in a terminal or not
	_ = isTerminal()
}

func TestUI_Hint(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Hint("This is a hint")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Hint should produce output")
	}
}

func TestUI_Step(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Step("Performing step...")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Step should produce output")
	}
}

func TestUI_Divider(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Divider()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Divider should produce output")
	}
}

func TestUI_Blank(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Blank()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Blank should produce a newline")
	}
}

func TestUI_Progress(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.Progress(iconStory, "Processing...")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Progress should produce output")
	}
}
