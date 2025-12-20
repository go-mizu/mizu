package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListTemplates(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	if len(templates) == 0 {
		t.Error("listTemplates() returned no templates")
	}

	// Check that all templates have required fields
	for _, tmpl := range templates {
		if tmpl.Name == "" {
			t.Error("template has empty name")
		}
		if tmpl.Description == "" {
			t.Errorf("template %q has empty description", tmpl.Name)
		}
	}
}

func TestListTemplatesIncludesNested(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	// Check for the nested react template
	foundReact := false
	foundVue := false
	for _, tmpl := range templates {
		if tmpl.Name == "frontend/spa/react" {
			foundReact = true
		}
		if tmpl.Name == "frontend/spa/vue" {
			foundVue = true
		}
	}

	if !foundReact {
		t.Error("listTemplates() did not include nested template 'frontend/spa/react'")
	}
	if !foundVue {
		t.Error("listTemplates() did not include nested template 'frontend/spa/vue'")
	}
}

func TestTemplateExistsNested(t *testing.T) {
	tests := []struct {
		name   string
		exists bool
	}{
		{"minimal", true},
		{"api", true},
		{"web", true},
		{"frontend/spa/react", true},
		{"frontend/spa/vue", true},
		{"frontend/spa/nonexistent", false},
		{"frontend/nonexistent", false},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := templateExists(tt.name)
			if got != tt.exists {
				t.Errorf("templateExists(%q) = %v, want %v", tt.name, got, tt.exists)
			}
		})
	}
}

func TestLoadTemplateMeta(t *testing.T) {
	tests := []struct {
		name        string
		wantName    string
		wantHasTags bool
	}{
		{"minimal", "minimal", true},
		{"api", "api", true},
		{"frontend/spa/react", "frontend/spa/react", true},
		{"frontend/spa/vue", "frontend/spa/vue", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := loadTemplateMeta(tt.name)
			if err != nil {
				t.Fatalf("loadTemplateMeta(%q) error: %v", tt.name, err)
			}

			if meta.Name != tt.wantName {
				t.Errorf("meta.Name = %q, want %q", meta.Name, tt.wantName)
			}

			if tt.wantHasTags && len(meta.Tags) == 0 {
				t.Errorf("meta.Tags is empty, expected tags")
			}
		})
	}
}

func TestLoadTemplateFilesNested(t *testing.T) {
	files, err := loadTemplateFiles("frontend/spa/react")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",                // from _common
		"go.mod",                    // from _common
		"cmd/server/main.go",        // template specific
		"app/server/app.go",         // template specific
		"app/server/config.go",      // template specific
		"app/server/routes.go",      // template specific
		"client/package.json",       // template specific
		"client/vite.config.ts",     // template specific
		"client/src/App.tsx",        // template specific
		"client/src/main.tsx",       // template specific
		"Makefile",                  // template specific
	}

	fileMap := make(map[string]bool)
	for _, f := range files {
		fileMap[f.path] = true
	}

	for _, expected := range expectedFiles {
		if !fileMap[expected] {
			t.Errorf("expected file %q not found in template files", expected)
		}
	}
}

func TestLoadTemplateFilesHierarchy(t *testing.T) {
	// Test that nested common directories override properly
	files, err := loadTemplateFiles("frontend/spa/react")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	// Find the Makefile - it should come from the react template
	var makefileContent []byte
	for _, f := range files {
		if f.path == "Makefile" {
			makefileContent = f.content
			break
		}
	}

	if makefileContent == nil {
		t.Fatal("Makefile not found in template files")
	}

	// The react Makefile should contain npm-specific content
	if !strings.Contains(string(makefileContent), "npm run build") {
		t.Error("Makefile does not contain expected content from react template")
	}
}

func TestRenderTemplateFile(t *testing.T) {
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	tests := []struct {
		name     string
		file     templateFile
		contains []string
	}{
		{
			name: "go.mod",
			file: templateFile{
				path:    "go.mod",
				content: []byte("module {{.Module}}\n"),
			},
			contains: []string{"module example.com/myapp"},
		},
		{
			name: "no template markers",
			file: templateFile{
				path:    "static.txt",
				content: []byte("static content"),
			},
			contains: []string{"static content"},
		},
		{
			name: "package.json",
			file: templateFile{
				path:    "package.json",
				content: []byte(`{"name": "{{.Name}}-client"}`),
			},
			contains: []string{`"name": "myapp-client"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderTemplateFile(tt.file, vars)
			if err != nil {
				t.Fatalf("renderTemplateFile() error: %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(string(result), want) {
					t.Errorf("result does not contain %q\ngot: %s", want, result)
				}
			}
		})
	}
}

func TestBuildPlanReactTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/spa/react", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if len(p.ops) == 0 {
		t.Error("buildPlan() returned empty plan")
	}

	// Check for expected operations
	hasWrite := false
	hasMkdir := false
	for _, op := range p.ops {
		switch op.kind {
		case opWrite:
			hasWrite = true
		case opMkdir:
			hasMkdir = true
		}
	}

	if !hasWrite {
		t.Error("plan has no write operations")
	}
	if !hasMkdir {
		t.Error("plan has no mkdir operations")
	}
}

func TestApplyPlanReactTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/spa/react", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Verify key files exist
	expectedFiles := []string{
		"go.mod",
		"cmd/server/main.go",
		"app/server/app.go",
		"app/server/config.go",
		"app/server/routes.go",
		"client/package.json",
		"client/vite.config.ts",
		"client/tsconfig.json",
		"client/index.html",
		"client/src/App.tsx",
		"client/src/main.tsx",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}
}

func TestTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myproject", "github.com/user/myproject", "Apache-2.0", nil)

	p, err := buildPlan("frontend/spa/react", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check go.mod contains the module path
	gomod, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(go.mod) error: %v", err)
	}

	if !strings.Contains(string(gomod), "github.com/user/myproject") {
		t.Error("go.mod does not contain module path")
	}

	// Check package.json contains the project name
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "client/package.json"))
	if err != nil {
		t.Fatalf("ReadFile(package.json) error: %v", err)
	}

	if !strings.Contains(string(pkgjson), "myproject-client") {
		t.Error("package.json does not contain project name")
	}
}

func TestMapOutputFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"gitignore", ".gitignore"},
		{"dockerignore", ".dockerignore"},
		{"env", ".env"},
		{"env.example", ".env.example"},
		{"main.go", "main.go"},
		{"config/gitignore", "config/.gitignore"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapOutputFilename(tt.input)
			if got != tt.expected {
				t.Errorf("mapOutputFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNewTemplateVars(t *testing.T) {
	customVars := map[string]string{"port": "8080"}
	vars := newTemplateVars("test", "example.com/test", "", customVars)

	if vars.Name != "test" {
		t.Errorf("vars.Name = %q, want %q", vars.Name, "test")
	}
	if vars.Module != "example.com/test" {
		t.Errorf("vars.Module = %q, want %q", vars.Module, "example.com/test")
	}
	if vars.License != "MIT" {
		t.Errorf("vars.License = %q, want %q (default)", vars.License, "MIT")
	}
	if vars.Year == 0 {
		t.Error("vars.Year should not be 0")
	}
	if vars.Vars["port"] != "8080" {
		t.Errorf("vars.Vars[port] = %q, want %q", vars.Vars["port"], "8080")
	}
}

// Vue template tests

func TestLoadTemplateFilesVue(t *testing.T) {
	files, err := loadTemplateFiles("frontend/spa/vue")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",             // from _common
		"go.mod",                 // from _common
		"cmd/server/main.go",     // template specific
		"app/server/app.go",      // template specific
		"app/server/config.go",   // template specific
		"app/server/routes.go",   // template specific
		"client/package.json",    // template specific
		"client/vite.config.ts",  // template specific
		"client/src/App.vue",     // template specific (Vue SFC)
		"client/src/main.ts",     // template specific
		"client/src/router/index.ts", // template specific
		"Makefile",               // template specific
	}

	fileMap := make(map[string]bool)
	for _, f := range files {
		fileMap[f.path] = true
	}

	for _, expected := range expectedFiles {
		if !fileMap[expected] {
			t.Errorf("expected file %q not found in template files", expected)
		}
	}
}

func TestBuildPlanVueTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/spa/vue", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if len(p.ops) == 0 {
		t.Error("buildPlan() returned empty plan")
	}

	// Check for expected operations
	hasWrite := false
	hasMkdir := false
	for _, op := range p.ops {
		switch op.kind {
		case opWrite:
			hasWrite = true
		case opMkdir:
			hasMkdir = true
		}
	}

	if !hasWrite {
		t.Error("plan has no write operations")
	}
	if !hasMkdir {
		t.Error("plan has no mkdir operations")
	}
}

func TestApplyPlanVueTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/spa/vue", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Verify key files exist
	expectedFiles := []string{
		"go.mod",
		"cmd/server/main.go",
		"app/server/app.go",
		"app/server/config.go",
		"app/server/routes.go",
		"client/package.json",
		"client/vite.config.ts",
		"client/tsconfig.json",
		"client/index.html",
		"client/src/App.vue",
		"client/src/main.ts",
		"client/src/router/index.ts",
		"client/src/components/Layout.vue",
		"client/src/pages/Home.vue",
		"client/src/pages/About.vue",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}
}

func TestVueTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myvueapp", "github.com/user/myvueapp", "Apache-2.0", nil)

	p, err := buildPlan("frontend/spa/vue", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check go.mod contains the module path
	gomod, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(go.mod) error: %v", err)
	}

	if !strings.Contains(string(gomod), "github.com/user/myvueapp") {
		t.Error("go.mod does not contain module path")
	}

	// Check package.json contains the project name
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "client/package.json"))
	if err != nil {
		t.Fatalf("ReadFile(package.json) error: %v", err)
	}

	if !strings.Contains(string(pkgjson), "myvueapp-client") {
		t.Error("package.json does not contain project name")
	}

	// Check that package.json has Vue dependencies
	if !strings.Contains(string(pkgjson), `"vue"`) {
		t.Error("package.json does not contain vue dependency")
	}
	if !strings.Contains(string(pkgjson), `"vue-router"`) {
		t.Error("package.json does not contain vue-router dependency")
	}
}

func TestVueTemplateHasVueSpecificContent(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/spa/vue", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check App.vue contains Vue-specific content
	appVue, err := os.ReadFile(filepath.Join(tmpDir, "client/src/App.vue"))
	if err != nil {
		t.Fatalf("ReadFile(App.vue) error: %v", err)
	}

	if !strings.Contains(string(appVue), "<script setup") {
		t.Error("App.vue does not contain Vue <script setup> syntax")
	}
	if !strings.Contains(string(appVue), "<template>") {
		t.Error("App.vue does not contain Vue <template> block")
	}

	// Check vite.config.ts uses Vue plugin
	viteConfig, err := os.ReadFile(filepath.Join(tmpDir, "client/vite.config.ts"))
	if err != nil {
		t.Fatalf("ReadFile(vite.config.ts) error: %v", err)
	}

	if !strings.Contains(string(viteConfig), "@vitejs/plugin-vue") {
		t.Error("vite.config.ts does not import Vue plugin")
	}

	// Check main.ts uses Vue createApp
	mainTs, err := os.ReadFile(filepath.Join(tmpDir, "client/src/main.ts"))
	if err != nil {
		t.Fatalf("ReadFile(main.ts) error: %v", err)
	}

	if !strings.Contains(string(mainTs), "createApp") {
		t.Error("main.ts does not contain Vue createApp")
	}
}
