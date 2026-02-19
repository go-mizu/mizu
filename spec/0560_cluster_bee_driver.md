# 0560 - Bee Cluster Driver (Distributed, Sharded, Replicated)

Date: 2026-02-19
Author: Codex

## 1. Scope

This pass delivers:

1. Real Bee cluster architecture (distributed, sharded, replicated).
2. `cmd/bee` node binary and multi-node simulation (3-node and 5-node).
3. Fair S3/LiteIO benchmarking against `horse_s3` and `local_s3`.
4. Additional optimization pass with **no in-memory turbo mode** in benchmark runs.

## 2. Research and Design Inputs

Inspired by:

1. Haystack append-only log + in-memory metadata index.
2. SeaweedFS separation of placement policy from data node engine.
3. SeaweedFS replication/availability tradeoffs (foreground fast path + background convergence).

Recent papers considered (directional):

1. A Unified Understanding of Consistency Levels in Cloud Storage Services (2024).
2. MetaHive (2024).
3. SkyStore (2025).
4. BVLSM (2025).
5. HyRES (2025).

## 3. Implementation Summary

## 3.1 Bee cluster implementation

Files:

1. `blueprints/liteio/pkg/storage/driver/zoo/bee/engine.go`
2. `blueprints/liteio/pkg/storage/driver/zoo/bee/storage.go`
3. `blueprints/liteio/pkg/storage/driver/zoo/bee/remote.go`
4. `blueprints/liteio/pkg/storage/driver/zoo/bee/node_server.go`
5. `blueprints/liteio/pkg/storage/driver/zoo/bee/multipart.go`
6. `blueprints/liteio/pkg/storage/driver/zoo/bee/bee_test.go`
7. `blueprints/liteio/cmd/bee/main.go`

Key behavior:

1. Rendezvous-hash sharding.
2. Replica set with quorum `w`/`r`.
3. Background repair workers.
4. Remote node HTTP transport pooling.

## 3.2 Additional optimization pass (non-turbo fair mode)

Goal: improve Bee without in-memory object store shortcuts.

Changes made in `blueprints/liteio/pkg/storage/driver/zoo/bee/storage.go`:

1. Foreground write/delete changed to quorum-first minimal sync set.
2. Remaining replicas handled asynchronously by repair path.
3. Added pull-based repair tasks for large objects to avoid large in-memory repair payload retention.
4. Added configurable `repair_max_kb` payload cap.

Benchmark setup changes:

1. Removed turbo mode from benchmark DSNs (no `mode=turbo`).
2. Kept real network cluster mode with peers and replication enabled.
3. Added/kept fair S3 baselines: `local_s3`, `horse_s3`, `bee3net_s3`, `bee5net_s3`.

Files:

1. `blueprints/liteio/bench/config.go`
2. `blueprints/liteio/scripts/bench-bee-s3.sh`

## 4. How Benchmark Was Run

From `blueprints/liteio`:

```bash
./scripts/bench-bee-s3.sh
```

This script launches:

1. Bee nodes: `:9401,:9402,:9403` and `:9501,:9502,:9503,:9504,:9505`.
2. LiteIO S3 gateways:
   1. `local_s3` on `:9213`
   2. `horse_s3` on `:9210`
   3. `bee3net_s3` on `:9211`
   4. `bee5net_s3` on `:9212`

No in-memory turbo mode is used in these DSNs.

## 5. Latest Fair S3 Result (No Turbo)

Report timestamp: `2026-02-19T12:22:43+07:00`

Report files:

1. `blueprints/liteio/report/bee_s3_fair/summary.md`
2. `blueprints/liteio/report/bee_s3_fair/raw_results.json`

Summary:

1. `horse_s3` won 34/40 categories.
2. `local_s3` won 6/40 categories.
3. `bee3net_s3` and `bee5net_s3` won 0 categories in this run.
4. Zero benchmark errors for all drivers.

Selected metrics (throughput metric from report):

| Operation | local_s3 | horse_s3 | bee3net_s3 | bee5net_s3 |
|---|---:|---:|---:|---:|
| Write/1KB | 1.388 | 15.539 | 7.615 | 7.032 |
| Write/64KB | 142.009 | 525.328 | 116.931 | 143.719 |
| Write/1MB | 740.013 | 1022.938 | 181.138 | 290.899 |
| Write/10MB | 1223.261 | 647.547 | 202.231 | 418.471 |
| Read/10MB | 3840.511 | 6515.154 | 1535.810 | 1382.200 |
| Delete | 9822.864 | 18865.395 | 8656.504 | 8782.156 |
| List/100 | 1495.146 | 2495.532 | 1302.219 | 1360.558 |

Ratio summary vs `horse_s3` (from raw JSON):

1. `bee3net_s3` average ratio across 40 ops: `0.5406x`.
2. `bee5net_s3` average ratio across 40 ops: `0.5587x`.
3. Best Bee/horse per-op ratio was on `Read/64KB` (noise-sensitive read case), but aggregate remained below horse.

## 6. Interpretation

1. Bee is now truly distributed and replicated in network cluster mode.
2. Removing turbo/in-memory object fast path made comparison fairer, and performance dropped significantly versus turbo-mode runs.
3. Current non-turbo Bee remains slower than horse on most write/delete/small-object paths in this environment.
4. The new quorum-first + pull-repair path improves architecture correctness and memory behavior, but does not close the horse gap yet.

## 7. Next Technical Steps (No In-Memory Shortcut)

1. Add batch replication RPC (multi-object/segment write) to amortize per-request overhead.
2. Add binary protocol (or gRPC) between gateway and Bee nodes.
3. Add write-ahead segment replication (replicate log chunks, not per-object HTTP requests).
4. Add dedicated metadata/index service for faster list/stat fanout reduction.
5. Add worker backpressure telemetry and adaptive repair scheduling.

## 8. Sources

1. Haystack OSDI'10: https://www.usenix.org/conference/osdi10/finding-needle-haystack-facebook-s-photo-storage
2. SeaweedFS repository: https://github.com/seaweedfs/seaweedfs
3. SeaweedFS replication notes: https://github.com/seaweedfs/seaweedfs/wiki/Replication
4. Consistency levels paper (2024): https://arxiv.org/abs/2410.07480
5. MetaHive (2024): https://arxiv.org/abs/2409.09144
6. SkyStore (2025): https://arxiv.org/abs/2502.08373
7. BVLSM (2025): https://arxiv.org/abs/2503.21874
8. HyRES (2025): https://arxiv.org/abs/2504.12375
