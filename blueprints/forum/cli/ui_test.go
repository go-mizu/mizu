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

	ui.Header(iconDatabase, "Test Header")

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

func TestUI_UserRow(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.UserRow("testuser", "test@example.com", true, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("UserRow should produce output")
	}
}

func TestUI_BoardRow(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.BoardRow("golang", "Go Programming", 100)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("BoardRow should produce output")
	}
}

func TestUI_ThreadRow(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ui.ThreadRow("Test Thread Title", "author", 42, 10)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("ThreadRow should produce output")
	}
}

func TestUI_ThreadRow_LongTitle(t *testing.T) {
	ui := NewUI()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Long title should be truncated
	ui.ThreadRow("This is a very long title that should be truncated because it exceeds forty characters", "author", 42, 10)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if buf.Len() == 0 {
		t.Error("ThreadRow should produce output for long titles")
	}
}

func TestIsTerminal(t *testing.T) {
	// This tests the isTerminal function
	// Result depends on whether we're running in a terminal or not
	_ = isTerminal()
}
