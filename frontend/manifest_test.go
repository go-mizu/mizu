package frontend

import (
	"html/template"
	"strings"
	"testing"
	"testing/fstest"
)

//nolint:cyclop // Test functions can have multiple subtests
func TestLoadViteManifest(t *testing.T) {
	viteManifest := `{
  "src/main.tsx": {
    "file": "assets/main-abc123.js",
    "src": "src/main.tsx",
    "isEntry": true,
    "imports": ["src/vendor.tsx"],
    "css": ["assets/main-def456.css"]
  },
  "src/vendor.tsx": {
    "file": "assets/vendor-789xyz.js",
    "src": "src/vendor.tsx"
  },
  "src/lazy.tsx": {
    "file": "assets/lazy-111222.js",
    "src": "src/lazy.tsx",
    "isDynamicEntry": true
  }
}`

	fsys := fstest.MapFS{
		".vite/manifest.json": &fstest.MapFile{Data: []byte(viteManifest)},
	}

	manifest, err := LoadManifest(fsys, ".vite/manifest.json")
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	t.Run("Entry returns output file", func(t *testing.T) {
		entry := manifest.Entry("src/main.tsx")
		if entry != "assets/main-abc123.js" {
			t.Errorf("expected assets/main-abc123.js, got %s", entry)
		}
	})

	t.Run("Asset returns output path", func(t *testing.T) {
		asset := manifest.Asset("src/main.tsx")
		if asset != "/assets/main-abc123.js" {
			t.Errorf("expected /assets/main-abc123.js, got %s", asset)
		}
	})

	t.Run("CSS returns associated styles", func(t *testing.T) {
		css := manifest.CSS("src/main.tsx")
		if len(css) != 1 || css[0] != "assets/main-def456.css" {
			t.Errorf("expected [assets/main-def456.css], got %v", css)
		}
	})

	t.Run("Preloads includes imports", func(t *testing.T) {
		preloads := manifest.Preloads("src/main.tsx")
		if len(preloads) == 0 {
			t.Error("expected preloads to include imports")
		}
		found := false
		for _, p := range preloads {
			if p == "assets/vendor-789xyz.js" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected preloads to include vendor")
		}
	})

	t.Run("ScriptTag generates correct tag", func(t *testing.T) {
		tag := manifest.ScriptTag("src/main.tsx")
		expected := `<script type="module" src="/assets/main-abc123.js"></script>`
		if string(tag) != expected {
			t.Errorf("expected %q, got %q", expected, string(tag))
		}
	})

	t.Run("CSSTags generates correct tags", func(t *testing.T) {
		tags := manifest.CSSTags("src/main.tsx")
		if !strings.Contains(string(tags), `<link rel="stylesheet" href="/assets/main-def456.css">`) {
			t.Errorf("expected CSS link tag, got %s", string(tags))
		}
	})

	t.Run("PreloadTags generates modulepreload tags", func(t *testing.T) {
		tags := manifest.PreloadTags("src/main.tsx")
		if !strings.Contains(string(tags), `rel="modulepreload"`) {
			t.Errorf("expected modulepreload tags, got %s", string(tags))
		}
	})

	t.Run("EntryTags includes all resources", func(t *testing.T) {
		tags := manifest.EntryTags("src/main.tsx")
		s := string(tags)
		if !strings.Contains(s, "stylesheet") {
			t.Error("expected CSS in entry tags")
		}
		if !strings.Contains(s, "modulepreload") {
			t.Error("expected modulepreload in entry tags")
		}
		if !strings.Contains(s, `type="module"`) {
			t.Error("expected script module in entry tags")
		}
	})

	t.Run("Entries returns entry points", func(t *testing.T) {
		entries := manifest.Entries()
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
		if entries[0] != "src/main.tsx" {
			t.Errorf("expected src/main.tsx, got %s", entries[0])
		}
	})
}

func TestLoadWebpackManifest(t *testing.T) {
	webpackManifest := `{
  "main.js": "main.abc123.js",
  "main.css": "main.def456.css",
  "vendor.js": "vendor.789xyz.js"
}`

	fsys := fstest.MapFS{
		"manifest.json": &fstest.MapFile{Data: []byte(webpackManifest)},
	}

	manifest, err := LoadManifest(fsys, "manifest.json")
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	t.Run("Asset returns output path", func(t *testing.T) {
		asset := manifest.Asset("main.js")
		if asset != "/main.abc123.js" {
			t.Errorf("expected /main.abc123.js, got %s", asset)
		}
	})

	t.Run("Unknown asset returns original path", func(t *testing.T) {
		asset := manifest.Asset("unknown.js")
		if asset != "/unknown.js" {
			t.Errorf("expected /unknown.js, got %s", asset)
		}
	})
}

func TestViewHelpers(t *testing.T) {
	viteManifest := `{
  "src/main.tsx": {
    "file": "assets/main-abc123.js",
    "src": "src/main.tsx",
    "isEntry": true,
    "css": ["assets/main-def456.css"]
  }
}`

	fsys := fstest.MapFS{
		".vite/manifest.json": &fstest.MapFile{Data: []byte(viteManifest)},
	}

	manifest, err := LoadManifest(fsys, ".vite/manifest.json")
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	helpers := manifest.ViewHelpers()

	t.Run("vite_entry helper", func(t *testing.T) {
		fn := helpers["vite_entry"].(func(string) template.HTML)
		result := fn("src/main.tsx")
		if !strings.Contains(string(result), "main-abc123.js") {
			t.Error("expected entry tags")
		}
	})

	t.Run("vite_asset helper", func(t *testing.T) {
		fn := helpers["vite_asset"].(func(string) string)
		result := fn("src/main.tsx")
		if result != "/assets/main-abc123.js" {
			t.Errorf("expected /assets/main-abc123.js, got %s", result)
		}
	})
}

func TestManifestNotFound(t *testing.T) {
	fsys := fstest.MapFS{}

	_, err := LoadManifest(fsys, "nonexistent.json")
	if err == nil {
		t.Error("expected error for nonexistent manifest")
	}
}

func TestManifestInvalidJSON(t *testing.T) {
	fsys := fstest.MapFS{
		"manifest.json": &fstest.MapFile{Data: []byte("invalid json")},
	}

	// Should return empty manifest, not error
	manifest, err := LoadManifest(fsys, "manifest.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have empty entries
	if len(manifest.Entries()) != 0 {
		t.Error("expected empty manifest for invalid JSON")
	}
}
