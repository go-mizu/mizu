package web

import (
	"context"
	"testing"
)

func TestJobManager_CreateAndList(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	cfg := JobConfig{
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

func TestJobManager_ListNewestFirst(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	j1 := jm.Create(JobConfig{Type: "download"})
	j2 := jm.Create(JobConfig{Type: "markdown"})
	j3 := jm.Create(JobConfig{Type: "index"})

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

func TestJobManager_GetNonexistent(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	got := jm.Get("nonexistent")
	if got != nil {
		t.Fatalf("expected nil for nonexistent job, got %+v", got)
	}
}

func TestJobManager_GetExisting(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(JobConfig{Type: "pack", Format: "jsonl"})
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

func TestJobManager_CancelJob(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(JobConfig{Type: "index", Engine: "bleve"})

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

func TestJobManager_CancelNonexistent(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	ok := jm.Cancel("nonexistent")
	if ok {
		t.Fatal("expected Cancel to return false for nonexistent job")
	}
}

func TestJobManager_CancelQueuedJob(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(JobConfig{Type: "download"})

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

func TestJobManager_UpdateProgress(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(JobConfig{Type: "download"})
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

func TestJobManager_Complete(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(JobConfig{Type: "markdown"})
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

func TestJobManager_Fail(t *testing.T) {
	hub := NewWSHub()
	defer hub.Close()

	jm := NewJobManager(hub, "/tmp/test-base", "CC-MAIN-2026-04")

	job := jm.Create(JobConfig{Type: "index", Engine: "bleve"})
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
