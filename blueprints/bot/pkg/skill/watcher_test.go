package skill

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBumpVersion_Increments(t *testing.T) {
	v1 := CurrentVersion()
	v2 := BumpVersion()
	if v2 != v1+1 {
		t.Errorf("BumpVersion() = %d; want %d", v2, v1+1)
	}
	if CurrentVersion() != v2 {
		t.Errorf("CurrentVersion() = %d; want %d", CurrentVersion(), v2)
	}
}

func TestMapsEqual(t *testing.T) {
	a := map[string]int64{"a": 1, "b": 2}
	b := map[string]int64{"a": 1, "b": 2}
	if !mapsEqual(a, b) {
		t.Error("identical maps should be equal")
	}

	c := map[string]int64{"a": 1, "b": 3}
	if mapsEqual(a, c) {
		t.Error("different values should not be equal")
	}

	d := map[string]int64{"a": 1}
	if mapsEqual(a, d) {
		t.Error("different lengths should not be equal")
	}
}

func TestWatcher_Snapshot(t *testing.T) {
	dir := t.TempDir()

	// Create a skill directory with SKILL.md.
	skillDir := filepath.Join(dir, "test-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\n---\nBody"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := &Watcher{dirs: []string{dir}}
	snap := w.snapshot()

	expected := filepath.Join(skillDir, "SKILL.md")
	if _, ok := snap[expected]; !ok {
		t.Errorf("snapshot missing %q", expected)
	}
	if len(snap) != 1 {
		t.Errorf("snapshot has %d entries; want 1", len(snap))
	}
}

func TestWatcher_DetectsChange(t *testing.T) {
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "test-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("---\nname: test\n---\nBody"), 0o644); err != nil {
		t.Fatal(err)
	}

	vBefore := CurrentVersion()

	changed := make(chan struct{}, 1)
	w := StartWatcher([]string{dir}, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	defer w.Stop()

	// Wait a moment then modify the file.
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(skillFile, []byte("---\nname: test\n---\nUpdated body"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the poll cycle to detect the change.
	select {
	case <-changed:
		// OK — change detected.
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for change notification")
	}

	vAfter := CurrentVersion()
	if vAfter <= vBefore {
		t.Errorf("version did not increase: before=%d, after=%d", vBefore, vAfter)
	}
}
