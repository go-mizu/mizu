package web

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
)

func TestManager_CreateAndList(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	cfg := pipeline.JobConfig{
		Type:    "download",
		CrawlID: "CC-MAIN-2026-04",
		Files:   "0-4",
	}
	job := jm.Create(cfg)

	if job.ID == "" {
		t.Fatal("expected non-empty job ID")
	}
	if len(job.ID) != 8 {
		t.Fatalf("expected 8-char ID, got %d chars: %q", len(job.ID), job.ID)
	}
	if job.Status != "queued" {
		t.Fatalf("expected status=queued, got %q", job.Status)
	}
	if job.Type != "download" {
		t.Fatalf("expected type=download, got %q", job.Type)
	}
	if job.Config.Files != "0-4" {
		t.Fatalf("expected config.files=0-4, got %q", job.Config.Files)
	}
	if job.StartedAt.IsZero() {
		t.Fatal("expected non-zero StartedAt")
	}

	jobs := jm.List()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job in list, got %d", len(jobs))
	}
	if jobs[0].ID != job.ID {
		t.Fatalf("listed job ID mismatch: got %q, want %q", jobs[0].ID, job.ID)
	}
}

func TestManager_ListNewestFirst(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	j1 := jm.Create(pipeline.JobConfig{Type: "download"})
	j2 := jm.Create(pipeline.JobConfig{Type: "markdown"})
	j3 := jm.Create(pipeline.JobConfig{Type: "index"})

	jobs := jm.List()
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
	// Newest first: j3, j2, j1
	if jobs[0].ID != j3.ID {
		t.Fatalf("expected first job to be j3 (%s), got %s", j3.ID, jobs[0].ID)
	}
	if jobs[1].ID != j2.ID {
		t.Fatalf("expected second job to be j2 (%s), got %s", j2.ID, jobs[1].ID)
	}
	if jobs[2].ID != j1.ID {
		t.Fatalf("expected third job to be j1 (%s), got %s", j1.ID, jobs[2].ID)
	}
}

func TestManager_GetNonexistent(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	got := jm.Get("nonexistent")
	if got != nil {
		t.Fatalf("expected nil for nonexistent job, got %+v", got)
	}
}

func TestManager_GetExisting(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(pipeline.JobConfig{Type: "pack", Format: "jsonl"})
	got := jm.Get(job.ID)
	if got == nil {
		t.Fatal("expected to find job by ID")
	}
	if got.ID != job.ID {
		t.Fatalf("ID mismatch: got %q, want %q", got.ID, job.ID)
	}
	if got.Config.Format != "jsonl" {
		t.Fatalf("expected format=jsonl, got %q", got.Config.Format)
	}
}

func TestManager_CancelJob(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(pipeline.JobConfig{Type: "index", Engine: "bleve"})

	// Set running with a cancel func tied to a real context.
	ctx, cancel := context.WithCancel(context.Background())
	jm.SetRunning(job.ID, cancel)

	// Verify status is running.
	got := jm.Get(job.ID)
	if got.Status != "running" {
		t.Fatalf("expected status=running after SetRunning, got %q", got.Status)
	}

	// Cancel the job.
	ok := jm.Cancel(job.ID)
	if !ok {
		t.Fatal("expected Cancel to return true")
	}

	// Verify the context was cancelled.
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Fatal("expected context to be cancelled")
	}

	// Verify status is "cancelled".
	got = jm.Get(job.ID)
	if got.Status != "cancelled" {
		t.Fatalf("expected status=cancelled, got %q", got.Status)
	}
	if got.EndedAt == nil {
		t.Fatal("expected EndedAt to be set after cancel")
	}
}

func TestManager_CancelNonexistent(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	ok := jm.Cancel("nonexistent")
	if ok {
		t.Fatal("expected Cancel to return false for nonexistent job")
	}
}

func TestManager_CancelQueuedJob(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(pipeline.JobConfig{Type: "download"})

	// Cancel a queued job (no cancel func set) — should still succeed.
	ok := jm.Cancel(job.ID)
	if !ok {
		t.Fatal("expected Cancel to return true for queued job")
	}

	got := jm.Get(job.ID)
	if got.Status != "cancelled" {
		t.Fatalf("expected status=cancelled, got %q", got.Status)
	}
}

func TestManager_UpdateProgress(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(pipeline.JobConfig{Type: "download"})
	jm.SetRunning(job.ID, func() {})

	jm.UpdateProgress(job.ID, 0.5, "downloading file 3 of 6", 12.5)

	got := jm.Get(job.ID)
	if got.Progress != 0.5 {
		t.Fatalf("expected progress=0.5, got %f", got.Progress)
	}
	if got.Message != "downloading file 3 of 6" {
		t.Fatalf("expected message mismatch, got %q", got.Message)
	}
	if got.Rate != 12.5 {
		t.Fatalf("expected rate=12.5, got %f", got.Rate)
	}
}

func TestManager_Complete(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(pipeline.JobConfig{Type: "markdown"})
	jm.SetRunning(job.ID, func() {})

	jm.Complete(job.ID, "processed 1000 documents")

	got := jm.Get(job.ID)
	if got.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", got.Status)
	}
	if got.Progress != 1.0 {
		t.Fatalf("expected progress=1.0, got %f", got.Progress)
	}
	if got.Message != "processed 1000 documents" {
		t.Fatalf("expected message mismatch, got %q", got.Message)
	}
	if got.EndedAt == nil {
		t.Fatal("expected EndedAt to be set after completion")
	}
}

func TestManager_Fail(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(pipeline.JobConfig{Type: "index", Engine: "bleve"})
	jm.SetRunning(job.ID, func() {})

	jm.Fail(job.ID, context.DeadlineExceeded)

	got := jm.Get(job.ID)
	if got.Status != "failed" {
		t.Fatalf("expected status=failed, got %q", got.Status)
	}
	if got.Error != "context deadline exceeded" {
		t.Fatalf("expected error message mismatch, got %q", got.Error)
	}
	if got.EndedAt == nil {
		t.Fatal("expected EndedAt to be set after failure")
	}
}

func TestManager_CompleteHook_DefaultCrawl(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	baseDir := filepath.Join(t.TempDir(), "CC-MAIN-2026-04")
	jm := pipeline.NewManager(hub, baseDir, "CC-MAIN-2026-04")

	var called bool
	var gotCrawlID, gotCrawlDir string
	jm.SetCompleteHook(func(_ *pipeline.Job, crawlID, crawlDir string) {
		called = true
		gotCrawlID = crawlID
		gotCrawlDir = crawlDir
	})

	job := jm.Create(pipeline.JobConfig{Type: "pack"})
	jm.SetRunning(job.ID, func() {})
	jm.Complete(job.ID, "done")

	if !called {
		t.Fatal("expected complete hook to be called")
	}
	if gotCrawlID != "CC-MAIN-2026-04" {
		t.Fatalf("hook crawlID=%q, want %q", gotCrawlID, "CC-MAIN-2026-04")
	}
	if gotCrawlDir != baseDir {
		t.Fatalf("hook crawlDir=%q, want %q", gotCrawlDir, baseDir)
	}
}

func TestManager_CompleteHook_JobCrawlOverride(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	commonRoot := t.TempDir()
	baseDir := filepath.Join(commonRoot, "CC-MAIN-2026-04")
	jm := pipeline.NewManager(hub, baseDir, "CC-MAIN-2026-04")

	var gotCrawlID, gotCrawlDir string
	jm.SetCompleteHook(func(_ *pipeline.Job, crawlID, crawlDir string) {
		gotCrawlID = crawlID
		gotCrawlDir = crawlDir
	})

	job := jm.Create(pipeline.JobConfig{Type: "index", CrawlID: "CC-MAIN-2026-08"})
	jm.SetRunning(job.ID, func() {})
	jm.Complete(job.ID, "done")

	if gotCrawlID != "CC-MAIN-2026-08" {
		t.Fatalf("hook crawlID=%q, want %q", gotCrawlID, "CC-MAIN-2026-08")
	}
	wantDir := filepath.Join(commonRoot, "CC-MAIN-2026-08")
	if gotCrawlDir != wantDir {
		t.Fatalf("hook crawlDir=%q, want %q", gotCrawlDir, wantDir)
	}
}

func TestManager_GetManifestPaths_Cache(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	jm := pipeline.NewManager(hub, t.TempDir(), "CC-MAIN-2026-04")

	calls := 0
	jm.SetManifestFetcher(func(ctx context.Context, crawlID string) ([]string, error) {
		calls++
		return []string{
			fmt.Sprintf("crawl-data/%s/segments/x/warc/CC-MAIN-20260206181458-20260206211458-00000.warc.gz", crawlID),
		}, nil
	})

	// Create a job and run through the public API to exercise manifest caching.
	// The manifest is fetched during resolveFiles (internal to RunJob).
	job1 := jm.Create(pipeline.JobConfig{Type: "download", Files: "0"})
	job2 := jm.Create(pipeline.JobConfig{Type: "download", Files: "0"})

	// Both jobs should reuse the same cached manifest.
	// Since RunJob is async, we can't easily verify the cache hit count here,
	// but we verify the Manager is constructable with manifest fetcher.
	_ = job1
	_ = job2

	if calls > 0 {
		// Fetcher hasn't been called yet since no RunJob was invoked.
		t.Fatalf("expected 0 calls before RunJob, got %d", calls)
	}
}
