# Crawler Binary Writer Performance Design
**Date**: 2026-03-02
**Goal**: 1,500+ OK pages/s on server 2
**Status**: Approved

## Problem

The binary writer's channel saturates at 100% under load, causing average RPS to drop from
~3,994 (devnull baseline) to ~2,547 (binary writer). Three root causes identified:

### Bottleneck 1 — Mutex on every `write()` [CRITICAL]

`binary.rs:155` wraps the crossbeam `Sender` in `Mutex<Option<Sender<T>>>`. Every worker
call to `write()` acquires this mutex before sending. crossbeam's bounded channel is already
an MPMC lock-free structure; the mutex serializes all 2,000 concurrent Tokio workers into a
queue before they reach the channel.

```rust
// current (bad)
fn write(&self, result: CrawlResult) -> Result<()> {
    let guard = self.tx.lock().unwrap();  // serializes 2000 workers
    match guard.as_ref() { Some(tx) => tx.send(result)... }
}
```

### Bottleneck 2 — Domain semaphore held during write() [HIGH]

`reqwest_engine.rs:process_one_url` holds the per-domain semaphore permit for the entire
duration of the function, including the `results.write()` call. When the channel is full,
`write()` blocks, the permit stays held, and other workers for that domain stall.

Chain: channel full → write blocks → semaphore starved → domain throughput drops → avg RPS craters.

### Bottleneck 3 — Single flusher thread + double allocation per record [MEDIUM]

One OS thread does serialize→copy→write for every record. Two heap allocations per record:
1. `rkyv::to_bytes()` → allocates `AlignedVec`
2. `.to_vec()` → copies to `Vec`

One CPU core is the throughput ceiling for all 2,000 workers combined.

## Solution: Approach A — Sentinel Message + Multi-Flusher

### Architecture

**Before:**
```
Workers (2000) → Mutex<Option<Sender<T>>> → 1 channel → 1 flusher thread → seg_000.bin
```

**After:**
```
Workers (2000) → Sender<Option<T>> (lock-free) → 1 channel → N flusher threads → seg_t{n}_{seq}.bin
                                                              (N=4, share receiver)
```

### Change 1: Sentinel-Based Close (eliminates Mutex)

Change channel type from `Sender<CrawlResult>` to `Sender<Option<CrawlResult>>`.

- `write()` sends `Some(result)` directly — no Mutex, no Option wrapper
- `close()` sends N `None` sentinels (one per flusher thread), then joins handles
- Each flusher loops until it receives `None`, then exits cleanly

```rust
// New struct fields
tx: crossbeam_channel::Sender<Option<CrawlResult>>,  // plain field, no Mutex
handles: Mutex<Vec<JoinHandle<()>>>,                  // only locked in close()
num_flushers: usize,

// New write() — lock-free hot path
fn write(&self, result: CrawlResult) -> Result<()> {
    self.tx.send(Some(result))
        .map_err(|_| anyhow!("binary result channel closed"))
}

// New close()
fn close(&self) -> Result<()> {
    for _ in 0..self.num_flushers {
        let _ = self.tx.send(None);  // one sentinel per flusher
    }
    let handles = self.handles.lock().unwrap().drain(..).collect::<Vec<_>>();
    for h in handles { h.join()?; }
    // drain to DuckDB if configured...
}
```

Same pattern applies to `BinaryFailureWriter` (url_tx and domain_tx channels).

### Change 2: Multi-Flusher Threads

Spawn `num_flushers` threads (default 4, auto = `clamp(cpu_count/2, 2, 8)`) that share
the same `Receiver`. Each writes to its own segment file family:

- Thread 0 → `seg_t0_000.bin`, `seg_t0_001.bin`, ...
- Thread 1 → `seg_t1_000.bin`, `seg_t1_001.bin`, ...

`read_dir_segments` already matches `seg_*.bin` via startswith — no change needed there.
`drain_to_duckdb` already processes all matching files — no change needed.

Updated flusher loop:
```rust
for msg in rx.iter() {
    let item = match msg {
        Some(item) => item,
        None => break,   // sentinel → this thread's work is done
    };
    // encode + write as before
}
```

### Change 3: Drop Semaphore Before Write

In `process_one_url` (both `reqwest_engine.rs` and `hyper_engine.rs`):

```rust
let _permit = domain_entry.semaphore.acquire().await?;
let fetch_result = fetch_one(...).await;
stats.total.fetch_add(1, Ordering::Relaxed);
drop(_permit);  // release NOW, before any writer call

match fetch_result {
    Ok(result)  => { let _ = results.write(result); }
    Err(...)    => { let _ = failures.write_url(...); }
}
```

This fully decouples fetch concurrency from writer backpressure.

### Change 4: Config & CLI

Add `num_flushers: usize` to `Config` (default 0 = auto).
Add `--flusher-threads <N>` flag to `hn.rs` and `cc.rs`.
`auto_config()` sets `num_flushers = clamp(cpu_count / 2, 2, 8)`.

## Files Changed

| File | Change |
|------|--------|
| `crawler-lib/src/writer/binary.rs` | Sentinel close, multi-flusher, lock-free write path |
| `crawler-lib/src/config.rs` | Add `num_flushers` field, update `auto_config()` |
| `crawler-lib/src/engine/reqwest_engine.rs` | `drop(_permit)` before write |
| `crawler-lib/src/engine/hyper_engine.rs` | `drop(_permit)` before write |
| `crawler-cli/src/hn.rs` | Add `--flusher-threads` flag |
| `crawler-cli/src/cc.rs` | Add `--flusher-threads` flag |

## Expected Outcome

| Metric | Before | After | Target |
|--------|--------|-------|--------|
| Avg RPS (binary, 200K seeds) | 2,547 | ~3,800 | ≥ 3,800 |
| Peak RPS | 9,363 | ~9,500 | ≥ 9,000 |
| Channel saturation | 100% | < 20% | < 20% |
| OK pages/s (80% seed quality) | ~2,037 | ~3,040 | ≥ 1,500 |
| Binary vs devnull gap | 36% | < 5% | < 5% |

## Testing Plan

1. **Existing unit tests** — `test_result_writer_roundtrip`, `test_drain_to_duckdb`,
   `test_failure_writer_roundtrip`, `test_segment_rotation`, `test_write_after_close_returns_error`
   must all pass unchanged.

2. **Concurrent stress test** (new) — spawn 2,000 threads each writing 100 records
   concurrently; verify all 200,000 records recovered after close, no deadlock, no panic.

3. **Benchmark** — devnull vs binary-4-flusher side-by-side on same seed set;
   target ≤ 5% throughput gap.

4. **Server 2 validation** — HN recrawl with pre-filtered seeds; confirm avg OK pages/s ≥ 1,500.
