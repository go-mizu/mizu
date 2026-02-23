package qlocal

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestGetMCPDaemonStatus_FromPIDFile(t *testing.T) {
	td := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", td)

	pidPath := MCPPIDPathForIndex("status_test")
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := GetMCPDaemonStatus("status_test")
	if st.PID != os.Getpid() {
		t.Fatalf("pid=%d want %d", st.PID, os.Getpid())
	}
	if !st.Running {
		t.Fatal("expected current process pid to be reported running")
	}
}
