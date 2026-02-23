package qlocal

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestCoreFeatures_EndToEnd(t *testing.T) {
	env := newTestEnv(t)
	env.writeFile(t, "notes/README.md", "# Project Notes\n\nCompiler project timeline and milestones.\n")
	env.writeFile(t, "notes/arch/design.md", "# Design\n\nThe parser uses recursive descent.\n")
	env.writeFile(t, "docs/api.md", "# API Reference\n\nAuthentication token and session docs.\n")
	env.writeFile(t, "docs/ops.md", "# Ops\n\nDeployment runbook and rollback procedure.\n")

	if _, err := env.App.CollectionAdd(env.RootDir+"/notes", "notes", "**/*.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.CollectionAdd(env.RootDir+"/docs", "docs", "**/*.md"); err != nil {
		t.Fatal(err)
	}

	if _, err := env.App.ContextAdd("qmd://notes/", "Personal engineering notes", env.RootDir); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.ContextAdd("/", "Global knowledge base", env.RootDir); err != nil {
		t.Fatal(err)
	}

	upd, err := env.App.Update(context.Background(), UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Added != 4 {
		t.Fatalf("added=%d want 4", upd.Added)
	}

	st, err := env.App.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st.TotalDocuments != 4 {
		t.Fatalf("totalDocs=%d want 4", st.TotalDocuments)
	}

	searchRes, err := env.App.SearchFTS(`parser -"authentication token"`, SearchOptions{Limit: 10, IncludeBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(searchRes) == 0 || !strings.Contains(searchRes[0].DisplayPath, "notes/arch/design.md") {
		t.Fatalf("unexpected search results: %#v", searchRes)
	}

	doc, err := env.App.Get("notes/arch/design.md", GetOptions{Full: true})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Context != "Personal engineering notes" {
		t.Fatalf("context=%q want %q", doc.Context, "Personal engineering notes")
	}

	docByID, err := env.App.Get("#"+doc.DocID, GetOptions{Full: true})
	if err != nil {
		t.Fatal(err)
	}
	if docByID.DisplayPath != doc.DisplayPath {
		t.Fatalf("docid lookup mismatch: got %s want %s", docByID.DisplayPath, doc.DisplayPath)
	}

	multi, errs, err := env.App.MultiGet("notes/*.md", 0, DefaultMultiGetMaxBytes, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 0 {
		t.Fatalf("multi-get errors: %v", errs)
	}
	if len(multi) == 0 {
		t.Fatal("expected multi-get results")
	}

	list, err := env.App.List("notes", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 {
		t.Fatalf("list len=%d want >=2", len(list))
	}

	if _, err := env.App.Embed(context.Background(), EmbedOptions{}); err != nil {
		t.Fatal(err)
	}
	vres, err := env.App.VectorSearch("recursive descent parser", SearchOptions{Limit: 5, IncludeBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(vres) == 0 {
		t.Fatal("expected vector results")
	}

	qres, err := env.App.QueryContext(context.Background(), "lex: parser\nvec: recursive descent parsing", HybridOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(qres) == 0 {
		t.Fatal("expected hybrid query results")
	}

	if _, err := env.App.Cleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestUpdate_DeactivatesMissingFiles(t *testing.T) {
	env := newTestEnv(t)
	path := env.writeFile(t, "notes/tmp.md", "# Temp\n\nHello\n")
	if _, err := env.App.CollectionAdd(env.RootDir+"/notes", "notes", "**/*.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil { // os imported? not yet
		t.Fatal(err)
	}
	upd, err := env.App.Update(context.Background(), UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Deactivated < 1 {
		t.Fatalf("expected deactivated file, got %d", upd.Deactivated)
	}
}
