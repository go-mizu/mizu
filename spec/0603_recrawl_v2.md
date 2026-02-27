# Spec 0603: High-Performance Recrawler V2 (Target 3k/s)

## Objective
Achieve a sustained **average** crawling speed of **3,000 pages/s** on standard hardware.

## Problem with V1
The previous implementation (`recrawler`) suffered from:
1.  **Sequential Probing:** Slow startup phases.
2.  **Input Head-of-Line Blocking:** Effectiveness serialized by `MaxConnsPerDomain`.
3.  **Complex State:** Adaptive timeouts added overhead.

## V2 Design: The Token-Interleaved Pipeline

### 1. Components
- **Seed Pump:** Reads seeds from source at high speed.
- **Fair Scheduler:** Buffers URLs per domain and dispatches in Round-Robin to ensure diversity.
- **Global Worker Pool:** 5,000 - 10,000 goroutines.
- **Sharded Writer:** 32-shard DuckDB writer.

### 2. Strategy for 3,000 pages/s
To hit 3k/s avg:
1.  **Worker Count:** 5,000 workers sharing an optimized transport.
2.  **Concurrency:** `MaxConnsPerDomain` set to 64.
3.  **DNS pre-resolution:** Pre-resolved 5,000+ hosts.
4.  **No Body:** `StatusOnly: true` enabled.
5.  **Tuned Transport:** Increased idle connection pools and over-provisioned buffers.

## Progress Log
- [x] Initial design and V2 skeleton.
- [x] Fair Scheduler implementation.
- [x] Sharded DB Writer (increased to 32 shards).
- [x] Engine with rolling speed tracking.
- [x] Benchmark at 500/s avg (local machine limits reached).
- [ ] Tuning for 3k/s avg (likely requires server-class infrastructure).
- [ ] Final verification on remote server.

### Current Challenges
- **Machine Resource Limits:** Local machine (Darwin) hits fd limits and socket exhaustion at ~5,000 workers.
- **Server Response Rate:** High failure rate suggests target servers are rate-limiting or timing out under 3k/s burst load.
- **Average vs Peak:** Achieving 3k/s peak is easy; 3k/s *average* requires continuous domain diversity.
