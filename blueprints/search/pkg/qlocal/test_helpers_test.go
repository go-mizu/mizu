package qlocal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type testEnv struct {
	App     *App
	RootDir string
	DBPath  string
	CfgPath string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	td := t.TempDir()
	root := filepath.Join(td, "data")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(td, "index.sqlite")
	cfgPath := filepath.Join(td, "index.yml")
	app, err := Open(OpenOptions{
		IndexName:  "test",
		DBPath:     dbPath,
		ConfigPath: cfgPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })
	return &testEnv{App: app, RootDir: root, DBPath: dbPath, CfgPath: cfgPath}
}

func (e *testEnv) writeFile(t *testing.T, rel, body string) string {
	t.Helper()
	full := filepath.Join(e.RootDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func (e *testEnv) addCollectionAndIndex(t *testing.T, name, relPath string) {
	t.Helper()
	abs := filepath.Join(e.RootDir, filepath.FromSlash(relPath))
	if _, err := e.App.CollectionAdd(abs, name, "**/*.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.App.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
}
