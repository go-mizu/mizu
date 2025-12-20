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

func TestListTemplatesIncludesParentWithSubTemplates(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	// Check that frontend is listed as a parent template with sub-templates
	var frontendSPA *templateMeta
	for i, tmpl := range templates {
		if tmpl.Name == "frontend" {
			frontendSPA = &templates[i]
			break
		}
	}

	if frontendSPA == nil {
		t.Fatal("listTemplates() did not include 'frontend' template")
	}

	// Verify it has sub-templates
	if len(frontendSPA.SubTemplates) == 0 {
		t.Error("frontend template should have sub-templates")
	}

	// Expected sub-templates
	expected := map[string]bool{
		"react":     false,
		"vue":       false,
		"svelte":    false,
		"sveltekit": false,
		"angular":   false,
		"alpine":    false,
		"preact":    false,
		"next":      false,
		"nuxt":      false,
		"htmx":      false,
	}

	for _, st := range frontendSPA.SubTemplates {
		if _, ok := expected[st.Name]; ok {
			expected[st.Name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("frontend should have sub-template %q", name)
		}
	}

	// Verify that individual sub-templates are NOT listed separately
	for _, tmpl := range templates {
		if strings.HasPrefix(tmpl.Name, "frontend/") {
			t.Errorf("sub-template %q should not be listed separately", tmpl.Name)
		}
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
		{"frontend/react", true},
		{"frontend/vue", true},
		{"frontend/svelte", true},
		{"frontend/angular", true},
		{"frontend/alpine", true},
		{"frontend/htmx", true},
		{"frontend/next", true},
		{"frontend/nuxt", true},
		{"frontend/preact", true},
		{"frontend/sveltekit", true},
		{"frontend/nonexistent", false},
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
		{"frontend/react", "frontend/react", true},
		{"frontend/vue", "frontend/vue", true},
		{"frontend/svelte", "frontend/svelte", true},
		{"frontend/angular", "frontend/angular", true},
		{"frontend/alpine", "frontend/alpine", true},
		{"frontend/htmx", "frontend/htmx", true},
		{"frontend/next", "frontend/next", true},
		{"frontend/nuxt", "frontend/nuxt", true},
		{"frontend/preact", "frontend/preact", true},
		{"frontend/sveltekit", "frontend/sveltekit", true},
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
	files, err := loadTemplateFiles("frontend/react")
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
		"frontend/package.json",       // template specific
		"frontend/vite.config.ts",     // template specific
		"frontend/src/App.tsx",        // template specific
		"frontend/src/main.tsx",       // template specific
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
	files, err := loadTemplateFiles("frontend/react")
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

	p, err := buildPlan("frontend/react", tmpDir, vars)
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

	p, err := buildPlan("frontend/react", tmpDir, vars)
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
		"frontend/package.json",
		"frontend/vite.config.ts",
		"frontend/tsconfig.json",
		"frontend/index.html",
		"frontend/src/App.tsx",
		"frontend/src/main.tsx",
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

	p, err := buildPlan("frontend/react", tmpDir, vars)
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
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/package.json"))
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
	files, err := loadTemplateFiles("frontend/vue")
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
		"frontend/package.json",    // template specific
		"frontend/vite.config.ts",  // template specific
		"frontend/src/App.vue",     // template specific (Vue SFC)
		"frontend/src/main.ts",     // template specific
		"frontend/src/router/index.ts", // template specific
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

	p, err := buildPlan("frontend/vue", tmpDir, vars)
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

	p, err := buildPlan("frontend/vue", tmpDir, vars)
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
		"frontend/package.json",
		"frontend/vite.config.ts",
		"frontend/tsconfig.json",
		"frontend/index.html",
		"frontend/src/App.vue",
		"frontend/src/main.ts",
		"frontend/src/router/index.ts",
		"frontend/src/components/Layout.vue",
		"frontend/src/pages/Home.vue",
		"frontend/src/pages/About.vue",
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

	p, err := buildPlan("frontend/vue", tmpDir, vars)
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
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/package.json"))
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

	p, err := buildPlan("frontend/vue", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check App.vue contains Vue-specific content
	appVue, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/App.vue"))
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
	viteConfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/vite.config.ts"))
	if err != nil {
		t.Fatalf("ReadFile(vite.config.ts) error: %v", err)
	}

	if !strings.Contains(string(viteConfig), "@vitejs/plugin-vue") {
		t.Error("vite.config.ts does not import Vue plugin")
	}

	// Check main.ts uses Vue createApp
	mainTs, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/main.ts"))
	if err != nil {
		t.Fatalf("ReadFile(main.ts) error: %v", err)
	}

	if !strings.Contains(string(mainTs), "createApp") {
		t.Error("main.ts does not contain Vue createApp")
	}
}

// Svelte template tests

func TestSubTemplateIncludesSvelte(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	// Find frontend parent template
	var frontendSPA *templateMeta
	for i, tmpl := range templates {
		if tmpl.Name == "frontend" {
			frontendSPA = &templates[i]
			break
		}
	}

	if frontendSPA == nil {
		t.Fatal("frontend template not found")
	}

	// Check that svelte is listed as a sub-template
	found := false
	for _, st := range frontendSPA.SubTemplates {
		if st.Name == "svelte" {
			found = true
			break
		}
	}

	if !found {
		t.Error("frontend should include 'svelte' sub-template")
	}
}

func TestTemplateExistsSvelte(t *testing.T) {
	if !templateExists("frontend/svelte") {
		t.Error("templateExists('frontend/svelte') returned false")
	}
}

func TestLoadTemplateFilesSvelte(t *testing.T) {
	files, err := loadTemplateFiles("frontend/svelte")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",               // from _common
		"go.mod",                   // from _common
		"cmd/server/main.go",       // template specific
		"app/server/app.go",        // template specific
		"app/server/config.go",     // template specific
		"app/server/routes.go",     // template specific
		"frontend/package.json",      // template specific
		"frontend/vite.config.ts",    // template specific
		"frontend/svelte.config.js",  // template specific (Svelte)
		"frontend/src/App.svelte",    // template specific (Svelte SFC)
		"frontend/src/main.ts",       // template specific
		"Makefile",                 // template specific
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

func TestBuildPlanSvelteTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/svelte", tmpDir, vars)
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

func TestApplyPlanSvelteTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/svelte", tmpDir, vars)
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
		"frontend/package.json",
		"frontend/vite.config.ts",
		"frontend/tsconfig.json",
		"frontend/svelte.config.js",
		"frontend/index.html",
		"frontend/src/App.svelte",
		"frontend/src/main.ts",
		"frontend/src/components/Layout.svelte",
		"frontend/src/pages/Home.svelte",
		"frontend/src/pages/About.svelte",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}
}

func TestSvelteTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("mysvelteapp", "github.com/user/mysvelteapp", "Apache-2.0", nil)

	p, err := buildPlan("frontend/svelte", tmpDir, vars)
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

	if !strings.Contains(string(gomod), "github.com/user/mysvelteapp") {
		t.Error("go.mod does not contain module path")
	}

	// Check package.json contains the project name
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/package.json"))
	if err != nil {
		t.Fatalf("ReadFile(package.json) error: %v", err)
	}

	if !strings.Contains(string(pkgjson), "mysvelteapp-client") {
		t.Error("package.json does not contain project name")
	}

	// Check that package.json has Svelte dependencies
	if !strings.Contains(string(pkgjson), `"svelte"`) {
		t.Error("package.json does not contain svelte dependency")
	}
	if !strings.Contains(string(pkgjson), `"svelte-routing"`) {
		t.Error("package.json does not contain svelte-routing dependency")
	}
}

func TestSvelteTemplateHasSvelteSpecificContent(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/svelte", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check App.svelte contains Svelte-specific content
	appSvelte, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/App.svelte"))
	if err != nil {
		t.Fatalf("ReadFile(App.svelte) error: %v", err)
	}

	if !strings.Contains(string(appSvelte), "<script lang=\"ts\">") {
		t.Error("App.svelte does not contain Svelte <script lang=\"ts\"> syntax")
	}
	if !strings.Contains(string(appSvelte), "import { Router }") {
		t.Error("App.svelte does not import Router from svelte-routing")
	}
	if !strings.Contains(string(appSvelte), "svelte-routing") {
		t.Error("App.svelte does not reference svelte-routing")
	}

	// Check vite.config.ts uses Svelte plugin
	viteConfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/vite.config.ts"))
	if err != nil {
		t.Fatalf("ReadFile(vite.config.ts) error: %v", err)
	}

	if !strings.Contains(string(viteConfig), "@sveltejs/vite-plugin-svelte") {
		t.Error("vite.config.ts does not import Svelte plugin")
	}

	// Check svelte.config.js exists and contains vitePreprocess
	svelteConfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/svelte.config.js"))
	if err != nil {
		t.Fatalf("ReadFile(svelte.config.js) error: %v", err)
	}

	if !strings.Contains(string(svelteConfig), "vitePreprocess") {
		t.Error("svelte.config.js does not contain vitePreprocess")
	}

	// Check Home.svelte uses Svelte 5 runes syntax
	homeSvelte, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/pages/Home.svelte"))
	if err != nil {
		t.Fatalf("ReadFile(Home.svelte) error: %v", err)
	}

	if !strings.Contains(string(homeSvelte), "$state") {
		t.Error("Home.svelte does not contain Svelte 5 $state rune")
	}
	if !strings.Contains(string(homeSvelte), "onMount") {
		t.Error("Home.svelte does not contain onMount lifecycle function")
	}
}

// Angular template tests

func TestSubTemplateIncludesAngular(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	var frontendSPA *templateMeta
	for i, tmpl := range templates {
		if tmpl.Name == "frontend" {
			frontendSPA = &templates[i]
			break
		}
	}

	if frontendSPA == nil {
		t.Fatal("frontend template not found")
	}

	found := false
	for _, st := range frontendSPA.SubTemplates {
		if st.Name == "angular" {
			found = true
			break
		}
	}

	if !found {
		t.Error("frontend should include 'angular' sub-template")
	}
}

func TestTemplateExistsAngular(t *testing.T) {
	if !templateExists("frontend/angular") {
		t.Error("templateExists('frontend/angular') returned false")
	}
}

func TestLoadTemplateFilesAngular(t *testing.T) {
	files, err := loadTemplateFiles("frontend/angular")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",                                      // from _common
		"go.mod",                                          // from _common
		"cmd/server/main.go",                              // template specific
		"app/server/app.go",                               // template specific
		"app/server/config.go",                            // template specific
		"app/server/routes.go",                            // template specific
		"frontend/package.json",                             // template specific
		"frontend/angular.json",                             // template specific (Angular)
		"frontend/src/main.ts",                              // template specific
		"frontend/src/app/app.component.ts",                 // template specific (Angular)
		"frontend/src/app/app.routes.ts",                    // template specific (Angular)
		"Makefile",                                        // template specific
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

func TestBuildPlanAngularTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/angular", tmpDir, vars)
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

func TestApplyPlanAngularTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/angular", tmpDir, vars)
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
		"frontend/package.json",
		"frontend/angular.json",
		"frontend/tsconfig.json",
		"frontend/src/index.html",
		"frontend/src/main.ts",
		"frontend/src/app/app.component.ts",
		"frontend/src/app/app.routes.ts",
		"frontend/src/app/app.config.ts",
		"frontend/src/app/components/layout/layout.component.ts",
		"frontend/src/app/pages/home/home.component.ts",
		"frontend/src/app/pages/about/about.component.ts",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}
}

func TestAngularTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myangularapp", "github.com/user/myangularapp", "Apache-2.0", nil)

	p, err := buildPlan("frontend/angular", tmpDir, vars)
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

	if !strings.Contains(string(gomod), "github.com/user/myangularapp") {
		t.Error("go.mod does not contain module path")
	}

	// Check package.json contains the project name
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/package.json"))
	if err != nil {
		t.Fatalf("ReadFile(package.json) error: %v", err)
	}

	if !strings.Contains(string(pkgjson), "myangularapp-client") {
		t.Error("package.json does not contain project name")
	}

	// Check that package.json has Angular dependencies
	if !strings.Contains(string(pkgjson), `"@angular/core"`) {
		t.Error("package.json does not contain @angular/core dependency")
	}
	if !strings.Contains(string(pkgjson), `"@angular/router"`) {
		t.Error("package.json does not contain @angular/router dependency")
	}
}

func TestAngularTemplateHasAngularSpecificContent(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/angular", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check app.component.ts contains Angular-specific content
	appComponent, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/app/app.component.ts"))
	if err != nil {
		t.Fatalf("ReadFile(app.component.ts) error: %v", err)
	}

	if !strings.Contains(string(appComponent), "@Component") {
		t.Error("app.component.ts does not contain @Component decorator")
	}
	if !strings.Contains(string(appComponent), "standalone: true") {
		t.Error("app.component.ts does not contain standalone: true")
	}
	if !strings.Contains(string(appComponent), "@angular/core") {
		t.Error("app.component.ts does not import from @angular/core")
	}

	// Check angular.json exists and contains proper configuration
	angularJson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/angular.json"))
	if err != nil {
		t.Fatalf("ReadFile(angular.json) error: %v", err)
	}

	if !strings.Contains(string(angularJson), "@angular-devkit/build-angular") {
		t.Error("angular.json does not contain @angular-devkit/build-angular")
	}

	// Check main.ts uses Angular bootstrapApplication
	mainTs, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/main.ts"))
	if err != nil {
		t.Fatalf("ReadFile(main.ts) error: %v", err)
	}

	if !strings.Contains(string(mainTs), "bootstrapApplication") {
		t.Error("main.ts does not contain Angular bootstrapApplication")
	}

	// Check home.component.ts uses Angular signals
	homeComponent, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/app/pages/home/home.component.ts"))
	if err != nil {
		t.Fatalf("ReadFile(home.component.ts) error: %v", err)
	}

	if !strings.Contains(string(homeComponent), "signal") {
		t.Error("home.component.ts does not contain Angular signals")
	}
	if !strings.Contains(string(homeComponent), "HttpClient") {
		t.Error("home.component.ts does not contain HttpClient")
	}
}

// HTMX template tests

func TestSubTemplateIncludesHtmx(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	var frontendSPA *templateMeta
	for i, tmpl := range templates {
		if tmpl.Name == "frontend" {
			frontendSPA = &templates[i]
			break
		}
	}

	if frontendSPA == nil {
		t.Fatal("frontend template not found")
	}

	found := false
	for _, st := range frontendSPA.SubTemplates {
		if st.Name == "htmx" {
			found = true
			break
		}
	}

	if !found {
		t.Error("frontend should include 'htmx' sub-template")
	}
}

func TestTemplateExistsHtmx(t *testing.T) {
	if !templateExists("frontend/htmx") {
		t.Error("templateExists('frontend/htmx') returned false")
	}
}

func TestLoadTemplateFilesHtmx(t *testing.T) {
	files, err := loadTemplateFiles("frontend/htmx")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",                      // from _common
		"go.mod",                          // from _common
		"cmd/server/main.go",              // template specific
		"app/server/app.go",               // template specific
		"app/server/config.go",            // template specific
		"app/server/routes.go",            // template specific
		"app/server/handlers.go",          // template specific (HTMX)
		"views/embed.go",                  // template specific (HTMX)
		"views/layouts/default.html",      // template specific (HTMX)
		"views/pages/home.html",           // template specific (HTMX)
		"views/pages/about.html",          // template specific (HTMX)
		"views/components/greeting.html",  // template specific (HTMX)
		"views/components/counter.html",   // template specific (HTMX)
		"static/embed.go",                 // template specific (HTMX)
		"static/css/app.css",              // template specific (HTMX)
		"static/js/app.js",                // template specific (HTMX)
		"Makefile",                        // template specific
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

func TestBuildPlanHtmxTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/htmx", tmpDir, vars)
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

func TestApplyPlanHtmxTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/htmx", tmpDir, vars)
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
		"app/server/handlers.go",
		"views/embed.go",
		"views/layouts/default.html",
		"views/pages/home.html",
		"views/pages/about.html",
		"views/components/greeting.html",
		"views/components/counter.html",
		"static/embed.go",
		"static/css/app.css",
		"static/js/app.js",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}

	// Verify NO client/package.json (HTMX doesn't use npm)
	clientPkg := filepath.Join(tmpDir, "frontend/package.json")
	if _, err := os.Stat(clientPkg); err == nil {
		t.Error("HTMX template should NOT have client/package.json")
	}
}

func TestHtmxTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myhtmxapp", "github.com/user/myhtmxapp", "Apache-2.0", nil)

	p, err := buildPlan("frontend/htmx", tmpDir, vars)
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

	if !strings.Contains(string(gomod), "github.com/user/myhtmxapp") {
		t.Error("go.mod does not contain module path")
	}

	// Check handlers.go contains the project name
	handlers, err := os.ReadFile(filepath.Join(tmpDir, "app/server/handlers.go"))
	if err != nil {
		t.Fatalf("ReadFile(handlers.go) error: %v", err)
	}

	if !strings.Contains(string(handlers), "myhtmxapp") {
		t.Error("handlers.go does not contain project name")
	}
}

func TestHtmxTemplateHasHtmxSpecificContent(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/htmx", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check default.html layout contains HTMX script
	layout, err := os.ReadFile(filepath.Join(tmpDir, "views/layouts/default.html"))
	if err != nil {
		t.Fatalf("ReadFile(default.html) error: %v", err)
	}

	if !strings.Contains(string(layout), "htmx.org") {
		t.Error("default.html does not include HTMX script")
	}
	if !strings.Contains(string(layout), "alpinejs") {
		t.Error("default.html does not include Alpine.js script")
	}
	if !strings.Contains(string(layout), "tailwindcss") {
		t.Error("default.html does not include Tailwind CSS")
	}

	// Check home.html contains HTMX attributes
	home, err := os.ReadFile(filepath.Join(tmpDir, "views/pages/home.html"))
	if err != nil {
		t.Fatalf("ReadFile(home.html) error: %v", err)
	}

	if !strings.Contains(string(home), "hx-get") {
		t.Error("home.html does not contain hx-get attribute")
	}
	if !strings.Contains(string(home), "hx-post") {
		t.Error("home.html does not contain hx-post attribute")
	}
	if !strings.Contains(string(home), "hx-target") {
		t.Error("home.html does not contain hx-target attribute")
	}
	if !strings.Contains(string(home), "x-data") {
		t.Error("home.html does not contain Alpine.js x-data attribute")
	}

	// Check counter.html contains HTMX attributes
	counter, err := os.ReadFile(filepath.Join(tmpDir, "views/components/counter.html"))
	if err != nil {
		t.Fatalf("ReadFile(counter.html) error: %v", err)
	}

	if !strings.Contains(string(counter), "hx-post") {
		t.Error("counter.html does not contain hx-post attribute")
	}

	// Check app.js contains HTMX event handlers
	appJs, err := os.ReadFile(filepath.Join(tmpDir, "static/js/app.js"))
	if err != nil {
		t.Fatalf("ReadFile(app.js) error: %v", err)
	}

	if !strings.Contains(string(appJs), "htmx:") {
		t.Error("app.js does not contain HTMX event handlers")
	}

	// Check app.css contains HTMX-related styles
	appCss, err := os.ReadFile(filepath.Join(tmpDir, "static/css/app.css"))
	if err != nil {
		t.Fatalf("ReadFile(app.css) error: %v", err)
	}

	if !strings.Contains(string(appCss), "htmx-request") {
		t.Error("app.css does not contain htmx-request styles")
	}

	// Check app.go uses view package (not frontend middleware)
	appGo, err := os.ReadFile(filepath.Join(tmpDir, "app/server/app.go"))
	if err != nil {
		t.Fatalf("ReadFile(app.go) error: %v", err)
	}

	if !strings.Contains(string(appGo), "github.com/go-mizu/mizu/view") {
		t.Error("app.go does not import view package")
	}
	if strings.Contains(string(appGo), "frontend.WithOptions") {
		t.Error("app.go should NOT use frontend middleware (HTMX uses server-side rendering)")
	}

	// Check handlers.go uses view.Render
	handlersGo, err := os.ReadFile(filepath.Join(tmpDir, "app/server/handlers.go"))
	if err != nil {
		t.Fatalf("ReadFile(handlers.go) error: %v", err)
	}

	if !strings.Contains(string(handlersGo), "view.Render") {
		t.Error("handlers.go does not use view.Render")
	}
	if !strings.Contains(string(handlersGo), "view.NoLayout()") {
		t.Error("handlers.go does not use view.NoLayout() for partials")
	}
}

func TestHtmxTemplateMakefileNoBuildStep(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/htmx", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check Makefile does NOT contain npm commands
	makefile, err := os.ReadFile(filepath.Join(tmpDir, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error: %v", err)
	}

	if strings.Contains(string(makefile), "npm") {
		t.Error("HTMX Makefile should NOT contain npm commands")
	}
	if strings.Contains(string(makefile), "frontend") {
		t.Error("HTMX Makefile should NOT reference frontend directory")
	}
}

// Next.js template tests

func TestSubTemplateIncludesNext(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	var frontendSPA *templateMeta
	for i, tmpl := range templates {
		if tmpl.Name == "frontend" {
			frontendSPA = &templates[i]
			break
		}
	}

	if frontendSPA == nil {
		t.Fatal("frontend template not found")
	}

	found := false
	for _, st := range frontendSPA.SubTemplates {
		if st.Name == "next" {
			found = true
			break
		}
	}

	if !found {
		t.Error("frontend should include 'next' sub-template")
	}
}

func TestTemplateExistsNext(t *testing.T) {
	if !templateExists("frontend/next") {
		t.Error("templateExists('frontend/next') returned false")
	}
}

func TestLoadTemplateFilesNext(t *testing.T) {
	files, err := loadTemplateFiles("frontend/next")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",                       // from _common
		"go.mod",                           // from _common
		"cmd/server/main.go",               // template specific
		"app/server/app.go",                // template specific
		"app/server/config.go",             // template specific
		"app/server/routes.go",             // template specific
		"frontend/package.json",              // template specific
		"frontend/next.config.ts",            // template specific (Next.js)
		"frontend/src/app/layout.tsx",        // template specific (Next.js App Router)
		"frontend/src/app/page.tsx",          // template specific
		"Makefile",                         // template specific
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

func TestBuildPlanNextTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/next", tmpDir, vars)
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

func TestApplyPlanNextTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/next", tmpDir, vars)
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
		"frontend/package.json",
		"frontend/next.config.ts",
		"frontend/tsconfig.json",
		"frontend/tailwind.config.ts",
		"frontend/postcss.config.mjs",
		"frontend/src/app/layout.tsx",
		"frontend/src/app/page.tsx",
		"frontend/src/app/about/page.tsx",
		"frontend/src/components/Navigation.tsx",
		"frontend/src/styles/globals.css",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}
}

func TestNextTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("mynextapp", "github.com/user/mynextapp", "Apache-2.0", nil)

	p, err := buildPlan("frontend/next", tmpDir, vars)
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

	if !strings.Contains(string(gomod), "github.com/user/mynextapp") {
		t.Error("go.mod does not contain module path")
	}

	// Check package.json contains the project name
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/package.json"))
	if err != nil {
		t.Fatalf("ReadFile(package.json) error: %v", err)
	}

	if !strings.Contains(string(pkgjson), "mynextapp-client") {
		t.Error("package.json does not contain project name")
	}

	// Check that package.json has Next.js dependencies
	if !strings.Contains(string(pkgjson), `"next"`) {
		t.Error("package.json does not contain next dependency")
	}
	if !strings.Contains(string(pkgjson), `"react"`) {
		t.Error("package.json does not contain react dependency")
	}
}

func TestNextTemplateHasNextSpecificContent(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/next", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check next.config.ts contains Next.js specific configuration
	nextConfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/next.config.ts"))
	if err != nil {
		t.Fatalf("ReadFile(next.config.ts) error: %v", err)
	}

	if !strings.Contains(string(nextConfig), "output: 'export'") {
		t.Error("next.config.ts does not contain output: 'export' for static export")
	}
	if !strings.Contains(string(nextConfig), "NextConfig") {
		t.Error("next.config.ts does not contain NextConfig type")
	}

	// Check layout.tsx contains Next.js App Router patterns
	layout, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/app/layout.tsx"))
	if err != nil {
		t.Fatalf("ReadFile(layout.tsx) error: %v", err)
	}

	if !strings.Contains(string(layout), "RootLayout") {
		t.Error("layout.tsx does not contain RootLayout function")
	}
	if !strings.Contains(string(layout), "Metadata") {
		t.Error("layout.tsx does not contain Metadata type")
	}

	// Check page.tsx uses 'use client' directive
	page, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/app/page.tsx"))
	if err != nil {
		t.Fatalf("ReadFile(page.tsx) error: %v", err)
	}

	if !strings.Contains(string(page), "'use client'") {
		t.Error("page.tsx does not contain 'use client' directive")
	}
	if !strings.Contains(string(page), "useState") {
		t.Error("page.tsx does not use useState hook")
	}

	// Check Navigation.tsx uses Next.js navigation
	navigation, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/components/Navigation.tsx"))
	if err != nil {
		t.Fatalf("ReadFile(Navigation.tsx) error: %v", err)
	}

	if !strings.Contains(string(navigation), "next/link") {
		t.Error("Navigation.tsx does not import from next/link")
	}
	if !strings.Contains(string(navigation), "next/navigation") {
		t.Error("Navigation.tsx does not import from next/navigation")
	}
	if !strings.Contains(string(navigation), "usePathname") {
		t.Error("Navigation.tsx does not use usePathname hook")
	}

	// Check tailwind.config.ts exists
	tailwindConfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/tailwind.config.ts"))
	if err != nil {
		t.Fatalf("ReadFile(tailwind.config.ts) error: %v", err)
	}

	if !strings.Contains(string(tailwindConfig), "tailwindcss") {
		t.Error("tailwind.config.ts does not reference tailwindcss")
	}

	// Check app.go uses frontend middleware
	appGo, err := os.ReadFile(filepath.Join(tmpDir, "app/server/app.go"))
	if err != nil {
		t.Fatalf("ReadFile(app.go) error: %v", err)
	}

	if !strings.Contains(string(appGo), "frontend.WithOptions") {
		t.Error("app.go does not use frontend.WithOptions")
	}
	if !strings.Contains(string(appGo), "frontend.ModeAuto") {
		t.Error("app.go does not use frontend.ModeAuto")
	}
}

func TestNextTemplateMakefileHasNpmCommands(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/next", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check Makefile contains npm commands (unlike HTMX)
	makefile, err := os.ReadFile(filepath.Join(tmpDir, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error: %v", err)
	}

	if !strings.Contains(string(makefile), "npm run build") {
		t.Error("Next.js Makefile should contain npm run build")
	}
	if !strings.Contains(string(makefile), "npm run dev") {
		t.Error("Next.js Makefile should contain npm run dev")
	}
	if !strings.Contains(string(makefile), "frontend") {
		t.Error("Next.js Makefile should reference frontend directory")
	}
}

// Nuxt template tests

func TestSubTemplateIncludesNuxt(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	var frontendSPA *templateMeta
	for i, tmpl := range templates {
		if tmpl.Name == "frontend" {
			frontendSPA = &templates[i]
			break
		}
	}

	if frontendSPA == nil {
		t.Fatal("frontend template not found")
	}

	found := false
	for _, st := range frontendSPA.SubTemplates {
		if st.Name == "nuxt" {
			found = true
			break
		}
	}

	if !found {
		t.Error("frontend should include 'nuxt' sub-template")
	}
}

func TestTemplateExistsNuxt(t *testing.T) {
	if !templateExists("frontend/nuxt") {
		t.Error("templateExists('frontend/nuxt') returned false")
	}
}

func TestLoadTemplateFilesNuxt(t *testing.T) {
	files, err := loadTemplateFiles("frontend/nuxt")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",                         // from _common
		"go.mod",                             // from _common
		"cmd/server/main.go",                 // template specific
		"app/server/app.go",                  // template specific
		"app/server/config.go",               // template specific
		"app/server/routes.go",               // template specific
		"frontend/package.json",                // template specific
		"frontend/nuxt.config.ts",              // template specific (Nuxt)
		"frontend/app.vue",                     // template specific (Nuxt)
		"frontend/pages/index.vue",             // template specific (Nuxt pages)
		"frontend/pages/about.vue",             // template specific
		"frontend/layouts/default.vue",         // template specific (Nuxt layouts)
		"frontend/components/AppNavigation.vue", // template specific
		"Makefile",                           // template specific
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

func TestBuildPlanNuxtTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/nuxt", tmpDir, vars)
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

func TestApplyPlanNuxtTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/nuxt", tmpDir, vars)
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
		"frontend/package.json",
		"frontend/nuxt.config.ts",
		"frontend/tsconfig.json",
		"frontend/tailwind.config.ts",
		"frontend/app.vue",
		"frontend/pages/index.vue",
		"frontend/pages/about.vue",
		"frontend/layouts/default.vue",
		"frontend/components/AppNavigation.vue",
		"frontend/assets/css/main.css",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}
}

func TestNuxtTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("mynuxtapp", "github.com/user/mynuxtapp", "Apache-2.0", nil)

	p, err := buildPlan("frontend/nuxt", tmpDir, vars)
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

	if !strings.Contains(string(gomod), "github.com/user/mynuxtapp") {
		t.Error("go.mod does not contain module path")
	}

	// Check package.json contains the project name
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/package.json"))
	if err != nil {
		t.Fatalf("ReadFile(package.json) error: %v", err)
	}

	if !strings.Contains(string(pkgjson), "mynuxtapp-client") {
		t.Error("package.json does not contain project name")
	}

	// Check that package.json has Nuxt dependencies
	if !strings.Contains(string(pkgjson), `"nuxt"`) {
		t.Error("package.json does not contain nuxt dependency")
	}
	if !strings.Contains(string(pkgjson), `"vue"`) {
		t.Error("package.json does not contain vue dependency")
	}
}

func TestNuxtTemplateHasNuxtSpecificContent(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/nuxt", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check nuxt.config.ts contains Nuxt specific configuration
	nuxtConfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/nuxt.config.ts"))
	if err != nil {
		t.Fatalf("ReadFile(nuxt.config.ts) error: %v", err)
	}

	if !strings.Contains(string(nuxtConfig), "ssr: false") {
		t.Error("nuxt.config.ts does not contain ssr: false for SPA mode")
	}
	if !strings.Contains(string(nuxtConfig), "defineNuxtConfig") {
		t.Error("nuxt.config.ts does not contain defineNuxtConfig")
	}
	if !strings.Contains(string(nuxtConfig), "@nuxtjs/tailwindcss") {
		t.Error("nuxt.config.ts does not contain @nuxtjs/tailwindcss module")
	}

	// Check app.vue uses Nuxt components
	appVue, err := os.ReadFile(filepath.Join(tmpDir, "frontend/app.vue"))
	if err != nil {
		t.Fatalf("ReadFile(app.vue) error: %v", err)
	}

	if !strings.Contains(string(appVue), "<NuxtLayout>") {
		t.Error("app.vue does not contain NuxtLayout component")
	}
	if !strings.Contains(string(appVue), "<NuxtPage />") {
		t.Error("app.vue does not contain NuxtPage component")
	}

	// Check default.vue layout
	layout, err := os.ReadFile(filepath.Join(tmpDir, "frontend/layouts/default.vue"))
	if err != nil {
		t.Fatalf("ReadFile(default.vue) error: %v", err)
	}

	if !strings.Contains(string(layout), "<slot />") {
		t.Error("default.vue layout does not contain slot")
	}
	if !strings.Contains(string(layout), "<AppNavigation />") {
		t.Error("default.vue layout does not contain AppNavigation component")
	}

	// Check index.vue uses Nuxt auto-imports
	indexVue, err := os.ReadFile(filepath.Join(tmpDir, "frontend/pages/index.vue"))
	if err != nil {
		t.Fatalf("ReadFile(index.vue) error: %v", err)
	}

	if !strings.Contains(string(indexVue), "ref(") {
		t.Error("index.vue does not use ref (auto-imported)")
	}
	if !strings.Contains(string(indexVue), "onMounted") {
		t.Error("index.vue does not use onMounted (auto-imported)")
	}
	if !strings.Contains(string(indexVue), "$fetch") {
		t.Error("index.vue does not use $fetch (Nuxt composable)")
	}
	if !strings.Contains(string(indexVue), "useHead") {
		t.Error("index.vue does not use useHead (Nuxt composable)")
	}

	// Check AppNavigation.vue uses NuxtLink
	navigation, err := os.ReadFile(filepath.Join(tmpDir, "frontend/components/AppNavigation.vue"))
	if err != nil {
		t.Fatalf("ReadFile(AppNavigation.vue) error: %v", err)
	}

	if !strings.Contains(string(navigation), "<NuxtLink") {
		t.Error("AppNavigation.vue does not use NuxtLink component")
	}
	if !strings.Contains(string(navigation), "useRoute") {
		t.Error("AppNavigation.vue does not use useRoute composable")
	}

	// Check app.go uses frontend middleware
	appGo, err := os.ReadFile(filepath.Join(tmpDir, "app/server/app.go"))
	if err != nil {
		t.Fatalf("ReadFile(app.go) error: %v", err)
	}

	if !strings.Contains(string(appGo), "frontend.WithOptions") {
		t.Error("app.go does not use frontend.WithOptions")
	}
	if !strings.Contains(string(appGo), "frontend.ModeAuto") {
		t.Error("app.go does not use frontend.ModeAuto")
	}
}

func TestNuxtTemplateMakefileHasNpmCommands(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/nuxt", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check Makefile contains npm commands
	makefile, err := os.ReadFile(filepath.Join(tmpDir, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error: %v", err)
	}

	if !strings.Contains(string(makefile), "npm run build") {
		t.Error("Nuxt Makefile should contain npm run build")
	}
	if !strings.Contains(string(makefile), "npm run dev") {
		t.Error("Nuxt Makefile should contain npm run dev")
	}
	if !strings.Contains(string(makefile), "frontend") {
		t.Error("Nuxt Makefile should reference frontend directory")
	}
	if !strings.Contains(string(makefile), ".nuxt") {
		t.Error("Nuxt Makefile should clean .nuxt directory")
	}
}

// Preact template tests

func TestSubTemplateIncludesPreact(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	var frontendSPA *templateMeta
	for i, tmpl := range templates {
		if tmpl.Name == "frontend" {
			frontendSPA = &templates[i]
			break
		}
	}

	if frontendSPA == nil {
		t.Fatal("frontend template not found")
	}

	found := false
	for _, st := range frontendSPA.SubTemplates {
		if st.Name == "preact" {
			found = true
			break
		}
	}

	if !found {
		t.Error("frontend should include 'preact' sub-template")
	}
}

func TestTemplateExistsPreact(t *testing.T) {
	if !templateExists("frontend/preact") {
		t.Error("templateExists('frontend/preact') returned false")
	}
}

func TestLoadTemplateFilesPreact(t *testing.T) {
	files, err := loadTemplateFiles("frontend/preact")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",            // from _common
		"go.mod",                // from _common
		"cmd/server/main.go",    // template specific
		"app/server/app.go",     // template specific
		"app/server/config.go",  // template specific
		"app/server/routes.go",  // template specific
		"frontend/package.json",   // template specific
		"frontend/vite.config.ts", // template specific
		"frontend/src/App.tsx",    // template specific (Preact)
		"frontend/src/main.tsx",   // template specific
		"Makefile",              // template specific
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

func TestBuildPlanPreactTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/preact", tmpDir, vars)
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

func TestApplyPlanPreactTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/preact", tmpDir, vars)
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
		"frontend/package.json",
		"frontend/vite.config.ts",
		"frontend/tsconfig.json",
		"frontend/index.html",
		"frontend/src/App.tsx",
		"frontend/src/main.tsx",
		"frontend/src/components/Layout.tsx",
		"frontend/src/pages/Home.tsx",
		"frontend/src/pages/About.tsx",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}
}

func TestPreactTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("mypreactapp", "github.com/user/mypreactapp", "Apache-2.0", nil)

	p, err := buildPlan("frontend/preact", tmpDir, vars)
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

	if !strings.Contains(string(gomod), "github.com/user/mypreactapp") {
		t.Error("go.mod does not contain module path")
	}

	// Check package.json contains the project name
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/package.json"))
	if err != nil {
		t.Fatalf("ReadFile(package.json) error: %v", err)
	}

	if !strings.Contains(string(pkgjson), "mypreactapp-client") {
		t.Error("package.json does not contain project name")
	}

	// Check that package.json has Preact dependencies
	if !strings.Contains(string(pkgjson), `"preact"`) {
		t.Error("package.json does not contain preact dependency")
	}
	if !strings.Contains(string(pkgjson), `"preact-router"`) {
		t.Error("package.json does not contain preact-router dependency")
	}
}

func TestPreactTemplateHasPreactSpecificContent(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/preact", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check App.tsx contains Preact-specific content
	appTsx, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/App.tsx"))
	if err != nil {
		t.Fatalf("ReadFile(App.tsx) error: %v", err)
	}

	if !strings.Contains(string(appTsx), "preact-router") {
		t.Error("App.tsx does not import from preact-router")
	}
	if !strings.Contains(string(appTsx), "Router") {
		t.Error("App.tsx does not contain Router component")
	}

	// Check main.tsx uses Preact render
	mainTsx, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/main.tsx"))
	if err != nil {
		t.Fatalf("ReadFile(main.tsx) error: %v", err)
	}

	if !strings.Contains(string(mainTsx), "import { render } from 'preact'") {
		t.Error("main.tsx does not import render from preact")
	}

	// Check vite.config.ts uses Preact preset
	viteConfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/vite.config.ts"))
	if err != nil {
		t.Fatalf("ReadFile(vite.config.ts) error: %v", err)
	}

	if !strings.Contains(string(viteConfig), "@preact/preset-vite") {
		t.Error("vite.config.ts does not import @preact/preset-vite")
	}

	// Check Home.tsx uses preact/hooks
	homeTsx, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/pages/Home.tsx"))
	if err != nil {
		t.Fatalf("ReadFile(Home.tsx) error: %v", err)
	}

	if !strings.Contains(string(homeTsx), "preact/hooks") {
		t.Error("Home.tsx does not import from preact/hooks")
	}
	if !strings.Contains(string(homeTsx), "useState") {
		t.Error("Home.tsx does not use useState hook")
	}
	if !strings.Contains(string(homeTsx), "useEffect") {
		t.Error("Home.tsx does not use useEffect hook")
	}

	// Check Layout.tsx uses ComponentChildren from preact
	layoutTsx, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/components/Layout.tsx"))
	if err != nil {
		t.Fatalf("ReadFile(Layout.tsx) error: %v", err)
	}

	if !strings.Contains(string(layoutTsx), "ComponentChildren") {
		t.Error("Layout.tsx does not use ComponentChildren type from preact")
	}

	// Check tsconfig.json has Preact JSX configuration
	tsconfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/tsconfig.json"))
	if err != nil {
		t.Fatalf("ReadFile(tsconfig.json) error: %v", err)
	}

	if !strings.Contains(string(tsconfig), `"jsxImportSource": "preact"`) {
		t.Error("tsconfig.json does not contain jsxImportSource: preact")
	}

	// Check app.go uses frontend middleware
	appGo, err := os.ReadFile(filepath.Join(tmpDir, "app/server/app.go"))
	if err != nil {
		t.Fatalf("ReadFile(app.go) error: %v", err)
	}

	if !strings.Contains(string(appGo), "frontend.WithOptions") {
		t.Error("app.go does not use frontend.WithOptions")
	}
	if !strings.Contains(string(appGo), "frontend.ModeAuto") {
		t.Error("app.go does not use frontend.ModeAuto")
	}
}

func TestPreactTemplateMakefileHasNpmCommands(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/preact", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check Makefile contains npm commands
	makefile, err := os.ReadFile(filepath.Join(tmpDir, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error: %v", err)
	}

	if !strings.Contains(string(makefile), "npm run build") {
		t.Error("Preact Makefile should contain npm run build")
	}
	if !strings.Contains(string(makefile), "npm run dev") {
		t.Error("Preact Makefile should contain npm run dev")
	}
	if !strings.Contains(string(makefile), "frontend") {
		t.Error("Preact Makefile should reference frontend directory")
	}
}

// SvelteKit template tests

func TestSubTemplateIncludesSvelteKit(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	var frontendSPA *templateMeta
	for i, tmpl := range templates {
		if tmpl.Name == "frontend" {
			frontendSPA = &templates[i]
			break
		}
	}

	if frontendSPA == nil {
		t.Fatal("frontend template not found")
	}

	found := false
	for _, st := range frontendSPA.SubTemplates {
		if st.Name == "sveltekit" {
			found = true
			break
		}
	}

	if !found {
		t.Error("frontend should include 'sveltekit' sub-template")
	}
}

func TestTemplateExistsSvelteKit(t *testing.T) {
	if !templateExists("frontend/sveltekit") {
		t.Error("templateExists('frontend/sveltekit') returned false")
	}
}

func TestLoadTemplateFilesSvelteKit(t *testing.T) {
	files, err := loadTemplateFiles("frontend/sveltekit")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",                          // from _common
		"go.mod",                              // from _common
		"cmd/server/main.go",                  // template specific
		"app/server/app.go",                   // template specific
		"app/server/config.go",                // template specific
		"app/server/routes.go",                // template specific
		"frontend/package.json",                 // template specific
		"frontend/svelte.config.js",             // template specific (SvelteKit)
		"frontend/vite.config.ts",               // template specific
		"frontend/src/app.html",                 // template specific (SvelteKit)
		"frontend/src/routes/+layout.svelte",    // template specific (SvelteKit routing)
		"frontend/src/routes/+page.svelte",      // template specific
		"Makefile",                            // template specific
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

func TestBuildPlanSvelteKitTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/sveltekit", tmpDir, vars)
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

func TestApplyPlanSvelteKitTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/sveltekit", tmpDir, vars)
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
		"frontend/package.json",
		"frontend/svelte.config.js",
		"frontend/vite.config.ts",
		"frontend/tsconfig.json",
		"frontend/tailwind.config.ts",
		"frontend/postcss.config.js",
		"frontend/src/app.html",
		"frontend/src/app.css",
		"frontend/src/app.d.ts",
		"frontend/src/routes/+layout.svelte",
		"frontend/src/routes/+page.svelte",
		"frontend/src/routes/about/+page.svelte",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}
}

func TestSvelteKitTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("mysveltekitapp", "github.com/user/mysveltekitapp", "Apache-2.0", nil)

	p, err := buildPlan("frontend/sveltekit", tmpDir, vars)
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

	if !strings.Contains(string(gomod), "github.com/user/mysveltekitapp") {
		t.Error("go.mod does not contain module path")
	}

	// Check package.json contains the project name
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/package.json"))
	if err != nil {
		t.Fatalf("ReadFile(package.json) error: %v", err)
	}

	if !strings.Contains(string(pkgjson), "mysveltekitapp-client") {
		t.Error("package.json does not contain project name")
	}

	// Check that package.json has SvelteKit dependencies
	if !strings.Contains(string(pkgjson), `"svelte"`) {
		t.Error("package.json does not contain svelte dependency")
	}
	if !strings.Contains(string(pkgjson), `"@sveltejs/kit"`) {
		t.Error("package.json does not contain @sveltejs/kit dependency")
	}
	if !strings.Contains(string(pkgjson), `"@sveltejs/adapter-static"`) {
		t.Error("package.json does not contain @sveltejs/adapter-static dependency")
	}
}

func TestSvelteKitTemplateHasSvelteKitSpecificContent(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/sveltekit", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check svelte.config.js contains adapter-static
	svelteConfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/svelte.config.js"))
	if err != nil {
		t.Fatalf("ReadFile(svelte.config.js) error: %v", err)
	}

	if !strings.Contains(string(svelteConfig), "adapter-static") {
		t.Error("svelte.config.js does not import adapter-static")
	}
	if !strings.Contains(string(svelteConfig), "fallback: 'index.html'") {
		t.Error("svelte.config.js does not configure SPA fallback")
	}

	// Check +layout.svelte uses Svelte 5 syntax
	layout, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/routes/+layout.svelte"))
	if err != nil {
		t.Fatalf("ReadFile(+layout.svelte) error: %v", err)
	}

	if !strings.Contains(string(layout), "$props()") {
		t.Error("+layout.svelte does not use Svelte 5 $props()")
	}
	if !strings.Contains(string(layout), "{@render children()}") {
		t.Error("+layout.svelte does not use Svelte 5 {@render} for children")
	}
	if !strings.Contains(string(layout), "svelte:head") {
		t.Error("+layout.svelte does not use svelte:head")
	}

	// Check +page.svelte uses Svelte 5 runes
	page, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/routes/+page.svelte"))
	if err != nil {
		t.Fatalf("ReadFile(+page.svelte) error: %v", err)
	}

	if !strings.Contains(string(page), "$state(") {
		t.Error("+page.svelte does not use Svelte 5 $state rune")
	}
	if !strings.Contains(string(page), "onMount") {
		t.Error("+page.svelte does not use onMount lifecycle function")
	}

	// Check vite.config.ts uses SvelteKit plugin
	viteConfig, err := os.ReadFile(filepath.Join(tmpDir, "frontend/vite.config.ts"))
	if err != nil {
		t.Fatalf("ReadFile(vite.config.ts) error: %v", err)
	}

	if !strings.Contains(string(viteConfig), "@sveltejs/kit/vite") {
		t.Error("vite.config.ts does not import from @sveltejs/kit/vite")
	}
	if !strings.Contains(string(viteConfig), "sveltekit()") {
		t.Error("vite.config.ts does not use sveltekit() plugin")
	}

	// Check app.html contains SvelteKit placeholders
	appHtml, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/app.html"))
	if err != nil {
		t.Fatalf("ReadFile(app.html) error: %v", err)
	}

	if !strings.Contains(string(appHtml), "%sveltekit.head%") {
		t.Error("app.html does not contain sveltekit.head placeholder")
	}
	if !strings.Contains(string(appHtml), "%sveltekit.body%") {
		t.Error("app.html does not contain sveltekit.body placeholder")
	}

	// Check app.css imports Tailwind
	appCss, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/app.css"))
	if err != nil {
		t.Fatalf("ReadFile(app.css) error: %v", err)
	}

	if !strings.Contains(string(appCss), "tailwindcss") {
		t.Error("app.css does not import tailwindcss")
	}

	// Check app.go uses frontend middleware
	appGo, err := os.ReadFile(filepath.Join(tmpDir, "app/server/app.go"))
	if err != nil {
		t.Fatalf("ReadFile(app.go) error: %v", err)
	}

	if !strings.Contains(string(appGo), "frontend.WithOptions") {
		t.Error("app.go does not use frontend.WithOptions")
	}
	if !strings.Contains(string(appGo), "frontend.ModeAuto") {
		t.Error("app.go does not use frontend.ModeAuto")
	}
}

func TestSvelteKitTemplateMakefileHasNpmCommands(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/sveltekit", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check Makefile contains npm commands
	makefile, err := os.ReadFile(filepath.Join(tmpDir, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error: %v", err)
	}

	if !strings.Contains(string(makefile), "npm run build") {
		t.Error("SvelteKit Makefile should contain npm run build")
	}
	if !strings.Contains(string(makefile), "npm run dev") {
		t.Error("SvelteKit Makefile should contain npm run dev")
	}
	if !strings.Contains(string(makefile), "frontend") {
		t.Error("SvelteKit Makefile should reference frontend directory")
	}
	if !strings.Contains(string(makefile), ".svelte-kit") {
		t.Error("SvelteKit Makefile should clean .svelte-kit directory")
	}
}

// Alpine.js template tests

func TestSubTemplateIncludesAlpine(t *testing.T) {
	templates, err := listTemplates()
	if err != nil {
		t.Fatalf("listTemplates() error: %v", err)
	}

	var frontendSPA *templateMeta
	for i, tmpl := range templates {
		if tmpl.Name == "frontend" {
			frontendSPA = &templates[i]
			break
		}
	}

	if frontendSPA == nil {
		t.Fatal("frontend template not found")
	}

	found := false
	for _, st := range frontendSPA.SubTemplates {
		if st.Name == "alpine" {
			found = true
			break
		}
	}

	if !found {
		t.Error("frontend should include 'alpine' sub-template")
	}
}

func TestTemplateExistsAlpine(t *testing.T) {
	if !templateExists("frontend/alpine") {
		t.Error("templateExists('frontend/alpine') returned false")
	}
}

func TestLoadTemplateFilesAlpine(t *testing.T) {
	files, err := loadTemplateFiles("frontend/alpine")
	if err != nil {
		t.Fatalf("loadTemplateFiles() error: %v", err)
	}

	if len(files) == 0 {
		t.Error("loadTemplateFiles() returned no files")
	}

	// Check for expected files
	expectedFiles := []string{
		".gitignore",               // from _common
		"go.mod",                   // from _common
		"cmd/server/main.go",       // template specific
		"app/server/app.go",        // template specific
		"app/server/config.go",     // template specific
		"app/server/routes.go",     // template specific
		"frontend/package.json",      // template specific
		"frontend/vite.config.ts",    // template specific
		"frontend/index.html",        // template specific (Alpine uses HTML)
		"frontend/src/main.ts",       // template specific
		"frontend/src/app.ts",        // template specific (Alpine app data)
		"Makefile",                 // template specific
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

func TestBuildPlanAlpineTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/alpine", tmpDir, vars)
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

func TestApplyPlanAlpineTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

	p, err := buildPlan("frontend/alpine", tmpDir, vars)
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
		"frontend/package.json",
		"frontend/vite.config.ts",
		"frontend/tsconfig.json",
		"frontend/index.html",
		"frontend/src/main.ts",
		"frontend/src/app.ts",
		"frontend/src/styles/index.css",
		"Makefile",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q does not exist", file)
		}
	}
}

func TestAlpineTemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("myalpineapp", "github.com/user/myalpineapp", "Apache-2.0", nil)

	p, err := buildPlan("frontend/alpine", tmpDir, vars)
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

	if !strings.Contains(string(gomod), "github.com/user/myalpineapp") {
		t.Error("go.mod does not contain module path")
	}

	// Check package.json contains the project name
	pkgjson, err := os.ReadFile(filepath.Join(tmpDir, "frontend/package.json"))
	if err != nil {
		t.Fatalf("ReadFile(package.json) error: %v", err)
	}

	if !strings.Contains(string(pkgjson), "myalpineapp-client") {
		t.Error("package.json does not contain project name")
	}

	// Check that package.json has Alpine.js dependency
	if !strings.Contains(string(pkgjson), `"alpinejs"`) {
		t.Error("package.json does not contain alpinejs dependency")
	}
}

func TestAlpineTemplateHasAlpineSpecificContent(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/alpine", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check index.html contains Alpine-specific directives
	indexHtml, err := os.ReadFile(filepath.Join(tmpDir, "frontend/index.html"))
	if err != nil {
		t.Fatalf("ReadFile(index.html) error: %v", err)
	}

	alpineDirectives := []string{
		"x-data",
		"x-cloak",
		"@click.prevent",
		"x-if",
		"x-text",
	}

	for _, directive := range alpineDirectives {
		if !strings.Contains(string(indexHtml), directive) {
			t.Errorf("index.html does not contain Alpine directive: %s", directive)
		}
	}

	// Check main.ts imports and starts Alpine
	mainTs, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/main.ts"))
	if err != nil {
		t.Fatalf("ReadFile(main.ts) error: %v", err)
	}

	if !strings.Contains(string(mainTs), "import Alpine from 'alpinejs'") {
		t.Error("main.ts does not import Alpine")
	}
	if !strings.Contains(string(mainTs), "Alpine.start()") {
		t.Error("main.ts does not call Alpine.start()")
	}
	if !strings.Contains(string(mainTs), "Alpine.data") {
		t.Error("main.ts does not register Alpine data")
	}

	// Check app.ts contains the app data/methods
	appTs, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/app.ts"))
	if err != nil {
		t.Fatalf("ReadFile(app.ts) error: %v", err)
	}

	if !strings.Contains(string(appTs), "createApp") {
		t.Error("app.ts does not export createApp function")
	}
	if !strings.Contains(string(appTs), "navigate") {
		t.Error("app.ts does not contain navigate method")
	}
	if !strings.Contains(string(appTs), "fetchMessage") {
		t.Error("app.ts does not contain fetchMessage method")
	}
	if !strings.Contains(string(appTs), "/api/hello") {
		t.Error("app.ts does not fetch from /api/hello")
	}

	// Check styles include x-cloak rule
	indexCss, err := os.ReadFile(filepath.Join(tmpDir, "frontend/src/styles/index.css"))
	if err != nil {
		t.Fatalf("ReadFile(index.css) error: %v", err)
	}

	if !strings.Contains(string(indexCss), "[x-cloak]") {
		t.Error("index.css does not contain x-cloak style rule")
	}

	// Check app.go uses frontend middleware
	appGo, err := os.ReadFile(filepath.Join(tmpDir, "app/server/app.go"))
	if err != nil {
		t.Fatalf("ReadFile(app.go) error: %v", err)
	}

	if !strings.Contains(string(appGo), "frontend.WithOptions") {
		t.Error("app.go does not use frontend.WithOptions")
	}
	if !strings.Contains(string(appGo), "frontend.ModeAuto") {
		t.Error("app.go does not use frontend.ModeAuto")
	}
}

func TestAlpineTemplateMakefileHasNpmCommands(t *testing.T) {
	tmpDir := t.TempDir()
	vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)

	p, err := buildPlan("frontend/alpine", tmpDir, vars)
	if err != nil {
		t.Fatalf("buildPlan() error: %v", err)
	}

	if err := p.apply(false); err != nil {
		t.Fatalf("apply() error: %v", err)
	}

	// Check Makefile contains npm commands
	makefile, err := os.ReadFile(filepath.Join(tmpDir, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error: %v", err)
	}

	if !strings.Contains(string(makefile), "npm run build") {
		t.Error("Alpine Makefile should contain npm run build")
	}
	if !strings.Contains(string(makefile), "npm run dev") {
		t.Error("Alpine Makefile should contain npm run dev")
	}
	if !strings.Contains(string(makefile), "frontend") {
		t.Error("Alpine Makefile should reference frontend directory")
	}
}
