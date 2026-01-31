package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListFiles_BasicDirectory(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"alpha.txt", "beta.txt", "gamma.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tool := ListFilesTool()
	result, err := tool.Execute(context.Background(), map[string]any{"path": dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"alpha.txt", "beta.txt", "gamma.txt"} {
		if !strings.Contains(result, name) {
			t.Errorf("result missing %q:\n%s", name, result)
		}
	}
}

func TestListFiles_WithPattern(t *testing.T) {
	dir := t.TempDir()

	files := map[string]bool{
		"report.pdf":  true,
		"invoice.pdf": true,
		"notes.txt":   false,
		"readme.md":   false,
	}
	for name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tool := ListFilesTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":    dir,
		"pattern": "*.pdf",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "report.pdf") {
		t.Errorf("result missing report.pdf:\n%s", result)
	}
	if !strings.Contains(result, "invoice.pdf") {
		t.Errorf("result missing invoice.pdf:\n%s", result)
	}
	if strings.Contains(result, "notes.txt") {
		t.Errorf("result should not contain notes.txt:\n%s", result)
	}
	if strings.Contains(result, "readme.md") {
		t.Errorf("result should not contain readme.md:\n%s", result)
	}
}

func TestListFiles_Recursive(t *testing.T) {
	dir := t.TempDir()

	subdir := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("r"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "mid.txt"), []byte("m"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "deep.txt"), []byte("d"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := ListFilesTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":      dir,
		"recursive": true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"root.txt", "mid.txt", "deep.txt"} {
		if !strings.Contains(result, name) {
			t.Errorf("recursive result missing %q:\n%s", name, result)
		}
	}
}

func TestListFiles_NonExistentDir(t *testing.T) {
	tool := ListFilesTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "/tmp/nonexistent_dir_builtins_test_xyz",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == "" {
		t.Fatal("expected error message, got empty string")
	}
	if !strings.Contains(result, "no such file or directory") {
		t.Errorf("expected 'no such file or directory' in result, got: %s", result)
	}
}

func TestReadFile_SmallFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	content := "Hello, World!\nSecond line."

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := ReadFileTool()
	result, err := tool.Execute(context.Background(), map[string]any{"path": path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != content {
		t.Errorf("result = %q, want %q", result, content)
	}
}

func TestReadFile_LargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.bin")

	// Create a file larger than 100KB.
	size := 120 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = 'A'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := ReadFileTool()
	result, err := tool.Execute(context.Background(), map[string]any{"path": path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[truncated") {
		t.Errorf("expected truncation notice in result:\n%s", result[:200])
	}
	if !strings.Contains(result, "122880 bytes total") {
		t.Errorf("expected total size in truncation notice:\n%s", result[len(result)-100:])
	}
}

func TestReadFile_NonExistent(t *testing.T) {
	tool := ReadFileTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "/tmp/nonexistent_file_builtins_test_xyz.txt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "no such file or directory") {
		t.Errorf("expected 'no such file or directory' in result, got: %s", result)
	}
}

func TestRunCommand_SimpleCommand(t *testing.T) {
	tool := RunCommandTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "echo hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "hello") {
		t.Errorf("expected 'hello' in result, got: %q", result)
	}
}

func TestRunCommand_WithWorkdir(t *testing.T) {
	dir := t.TempDir()

	tool := RunCommandTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "pwd",
		"workdir": dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resolve symlinks on both sides for comparison (macOS /var -> /private/var).
	resolvedDir, resolveErr := filepath.EvalSymlinks(dir)
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}

	trimmed := strings.TrimSpace(result)
	resolvedResult, resolveErr := filepath.EvalSymlinks(trimmed)
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}

	if resolvedResult != resolvedDir {
		t.Errorf("pwd result = %q, want %q", resolvedResult, resolvedDir)
	}
}

func TestRunCommand_OutputTruncation(t *testing.T) {
	tool := RunCommandTool()

	// Generate output > 50KB. Each line is "AAAA...A\n" (81 bytes).
	// 700 lines * 81 bytes = 56700 bytes > 50KB.
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "for i in $(seq 1 700); do printf 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\\n'; done",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[truncated") {
		t.Errorf("expected truncation notice in result (len=%d)", len(result))
	}
}
