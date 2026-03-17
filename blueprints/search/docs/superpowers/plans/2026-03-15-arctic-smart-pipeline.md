# Arctic Smart Pipeline Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace sequential download→process→upload with a pipelined architecture that overlaps stages across different (month,type) pairs, auto-tuned to available hardware.

**Architecture:** Three-stage pipeline (download → process → upload) connected by bounded Go channels. Hardware detected at startup to compute concurrency budget. Per-job work directories prevent file collisions. Existing sequential mode preserved as fallback.

**Tech Stack:** Go stdlib (runtime, syscall, os, sync), existing DuckDB/torrent/HF infrastructure.

---

## Chunk 1: Hardware Detection + Budget

### Task 1: Hardware Detection

**Files:**
- Create: `blueprints/search/pkg/arctic/hwdetect.go`

- [ ] **Step 1: Create hwdetect.go with cross-platform detection**

- [ ] **Step 2: Test manually with `go build`**

- [ ] **Step 3: Commit**

### Task 2: Resource Budget

**Files:**
- Create: `blueprints/search/pkg/arctic/budget.go`

- [ ] **Step 1: Create budget.go**

- [ ] **Step 2: Commit**

---

## Chunk 2: Per-Job Work Directories + Pipeline State

### Task 3: Per-Job Config

**Files:**
- Modify: `blueprints/search/pkg/arctic/config.go`

- [ ] **Step 1: Add JobWorkDir method to Config**

- [ ] **Step 2: Commit**

### Task 4: Enhanced Live State

**Files:**
- Modify: `blueprints/search/pkg/arctic/live_state.go`

- [ ] **Step 1: Add pipeline types to live_state.go**

- [ ] **Step 2: Commit**

---

## Chunk 3: Pipeline Orchestrator

### Task 5: Pipeline Task

**Files:**
- Create: `blueprints/search/pkg/arctic/pipeline.go`

- [ ] **Step 1: Create pipeline.go with three-stage pipeline**

- [ ] **Step 2: Commit**

---

## Chunk 4: README Enhancement + CLI Integration

### Task 6: README pipeline section

**Files:**
- Modify: `blueprints/search/pkg/arctic/readme.go`

- [ ] **Step 1: Add pipeline status section to README template**

- [ ] **Step 2: Commit**

### Task 7: CLI integration

**Files:**
- Modify: `blueprints/search/cli/arctic_publish.go`

- [ ] **Step 1: Add pipeline mode to CLI**

- [ ] **Step 2: Commit**
