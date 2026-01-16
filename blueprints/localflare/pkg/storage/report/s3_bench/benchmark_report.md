# S3 Benchmark Report

**Generated:** 2026-01-16 03:19:06 UTC
**Total Samples:** 7500

---

## Executive Summary

### Overall Winner: **minio**

Based on weighted scoring across all object sizes and thread configurations:

| Rank | Driver | Score | Throughput | TTFB p50 | Consistency |
|------|--------|-------|------------|----------|-------------|
| 1st | minio | 47.4 | 18.1 MB/s | 20 ms | 86.0% |
| 2nd | rustfs | 44.3 | 17.9 MB/s | 30 ms | 86.0% |
| 3rd | localstack | 35.8 | 17.1 MB/s | 57 ms | 85.7% |
| 4 | liteio | 35.1 | 18.6 MB/s | 62 ms | 86.0% |
| 5 | seaweedfs | 34.9 | 18.6 MB/s | 64 ms | 86.9% |

### Key Findings

- **Best Throughput:** liteio achieves 3% higher throughput than runner-up
- **Lowest Latency:** minio has 0% lower TTFB p50 than runner-up
- **Most Consistent:** minio shows lowest variance in throughput

---

## Category Winners

### By Object Size

| Object Size | Winner | Throughput | Runner-up |
|------------|--------|------------|----------|
| 4 MB | liteio | 23.7 MB/s | - |
| 8 MB | liteio | 21.4 MB/s | - |
| 16 MB | seaweedfs | 23.1 MB/s | - |

### By Metric

| Metric | Winner | Value | vs Average |
|--------|--------|-------|------------|
| Throughput | liteio | 23.7 MB/s | +31% |
| TTFB p50 | minio | 17 ms | -64% |

---

## Recommendations

### High-Throughput Workloads
**Use seaweedfs** - Delivers highest average throughput at 18.6 MB/s

### Latency-Sensitive Workloads
**Use minio** - Lowest median latency at 20 ms (p50)

### Consistent Performance
**Use localstack** - Most consistent throughput with lowest variance

### Trade-offs

| Driver | Strengths | Considerations |
|--------|-----------|----------------|
| minio | High throughput, Low latency | None observed |
| rustfs | High throughput | Higher latency |
| localstack | Balanced | Lower throughput, Higher latency |
| liteio | High throughput | Higher latency |
| seaweedfs | High throughput | Higher latency |

---

## Detailed Results

### 4 MB Objects

| Driver | Threads | Throughput | TTFB p50 | TTFB p99 | TTLB p50 | TTLB p99 |
|--------|---------|------------|----------|----------|----------|----------|
| liteio | 8 | 23.7 MB/s | 36 ms | 63 ms | 155 ms | 660 ms |
| liteio | 9 | 20.2 MB/s | 39 ms | 76 ms | 181 ms | 599 ms |
| liteio | 10 | 18.2 MB/s | 44 ms | 71 ms | 184 ms | 626 ms |
| liteio | 11 | 16.4 MB/s | 49 ms | 79 ms | 236 ms | 458 ms |
| liteio | 12 | 15.1 MB/s | 47 ms | 85 ms | 226 ms | 590 ms |
| localstack | 8 | 21.2 MB/s | 30 ms | 56 ms | 140 ms | 590 ms |
| localstack | 9 | 18.6 MB/s | 39 ms | 60 ms | 170 ms | 668 ms |
| localstack | 10 | 17.4 MB/s | 46 ms | 66 ms | 187 ms | 603 ms |
| localstack | 11 | 15.5 MB/s | 48 ms | 78 ms | 201 ms | 571 ms |
| localstack | 12 | 14.4 MB/s | 55 ms | 82 ms | 260 ms | 536 ms |
| minio | 8 | 21.5 MB/s | 17 ms | 23 ms | 176 ms | 307 ms |
| minio | 9 | 19.1 MB/s | 19 ms | 170 ms | 195 ms | 340 ms |
| minio | 10 | 17.5 MB/s | 20 ms | 26 ms | 235 ms | 341 ms |
| minio | 11 | 15.8 MB/s | 24 ms | 31 ms | 228 ms | 386 ms |
| minio | 12 | 14.4 MB/s | 22 ms | 28 ms | 254 ms | 459 ms |
| rustfs | 8 | 21.5 MB/s | 20 ms | 41 ms | 180 ms | 279 ms |
| rustfs | 9 | 19.2 MB/s | 21 ms | 28 ms | 197 ms | 296 ms |
| rustfs | 10 | 17.2 MB/s | 22 ms | 33 ms | 216 ms | 382 ms |
| rustfs | 11 | 16.1 MB/s | 24 ms | 29 ms | 239 ms | 418 ms |
| rustfs | 12 | 14.4 MB/s | 26 ms | 33 ms | 256 ms | 441 ms |
| seaweedfs | 8 | 21.8 MB/s | 47 ms | 82 ms | 172 ms | 443 ms |
| seaweedfs | 9 | 20.0 MB/s | 41 ms | 70 ms | 167 ms | 440 ms |
| seaweedfs | 10 | 18.2 MB/s | 37 ms | 80 ms | 189 ms | 522 ms |
| seaweedfs | 11 | 16.6 MB/s | 49 ms | 78 ms | 203 ms | 735 ms |
| seaweedfs | 12 | 15.1 MB/s | 51 ms | 89 ms | 236 ms | 788 ms |

### 8 MB Objects

| Driver | Threads | Throughput | TTFB p50 | TTFB p99 | TTLB p50 | TTLB p99 |
|--------|---------|------------|----------|----------|----------|----------|
| liteio | 8 | 21.4 MB/s | 68 ms | 98 ms | 370 ms | 731 ms |
| liteio | 9 | 19.6 MB/s | 64 ms | 85 ms | 369 ms | 1276 ms |
| liteio | 10 | 18.8 MB/s | 61 ms | 90 ms | 367 ms | 1502 ms |
| liteio | 11 | 14.0 MB/s | 45 ms | 75 ms | 474 ms | 1510 ms |
| liteio | 12 | 15.4 MB/s | 80 ms | 120 ms | 526 ms | 953 ms |
| localstack | 8 | 18.6 MB/s | 48 ms | 66 ms | 364 ms | 1237 ms |
| localstack | 9 | 17.7 MB/s | 49 ms | 66 ms | 412 ms | 989 ms |
| localstack | 10 | 16.1 MB/s | 57 ms | 78 ms | 355 ms | 1449 ms |
| localstack | 11 | 14.8 MB/s | 60 ms | 84 ms | 400 ms | 1736 ms |
| localstack | 12 | 13.7 MB/s | 57 ms | 72 ms | 501 ms | 1863 ms |
| minio | 8 | 21.0 MB/s | 21 ms | 29 ms | 381 ms | 499 ms |
| minio | 9 | 18.7 MB/s | 17 ms | 23 ms | 426 ms | 567 ms |
| minio | 10 | 16.8 MB/s | 17 ms | 22 ms | 477 ms | 608 ms |
| minio | 11 | 15.6 MB/s | 20 ms | 27 ms | 468 ms | 835 ms |
| minio | 12 | 14.2 MB/s | 22 ms | 29 ms | 559 ms | 742 ms |
| rustfs | 8 | 20.5 MB/s | 34 ms | 175 ms | 367 ms | 676 ms |
| rustfs | 9 | 18.4 MB/s | 42 ms | 84 ms | 402 ms | 714 ms |
| rustfs | 10 | 16.9 MB/s | 40 ms | 69 ms | 438 ms | 768 ms |
| rustfs | 11 | 15.2 MB/s | 40 ms | 219 ms | 489 ms | 865 ms |
| rustfs | 12 | 14.2 MB/s | 42 ms | 199 ms | 522 ms | 868 ms |
| seaweedfs | 8 | 20.7 MB/s | 69 ms | 175 ms | 354 ms | 874 ms |
| seaweedfs | 9 | 18.9 MB/s | 70 ms | 98 ms | 420 ms | 700 ms |
| seaweedfs | 10 | 17.7 MB/s | 74 ms | 106 ms | 421 ms | 1120 ms |
| seaweedfs | 11 | 16.2 MB/s | 73 ms | 105 ms | 434 ms | 1346 ms |
| seaweedfs | 12 | 14.8 MB/s | 74 ms | 116 ms | 487 ms | 1506 ms |

### 16 MB Objects

| Driver | Threads | Throughput | TTFB p50 | TTFB p99 | TTLB p50 | TTLB p99 |
|--------|---------|------------|----------|----------|----------|----------|
| liteio | 8 | 21.2 MB/s | 81 ms | 655 ms | 672 ms | 1826 ms |
| liteio | 9 | 21.0 MB/s | 72 ms | 239 ms | 714 ms | 1964 ms |
| liteio | 10 | 19.5 MB/s | 83 ms | 107 ms | 808 ms | 1564 ms |
| liteio | 11 | 18.0 MB/s | 88 ms | 119 ms | 876 ms | 1378 ms |
| liteio | 12 | 17.0 MB/s | 78 ms | 115 ms | 914 ms | 1938 ms |
| localstack | 8 | 21.8 MB/s | 63 ms | 83 ms | 636 ms | 1606 ms |
| localstack | 9 | 20.4 MB/s | 58 ms | 77 ms | 593 ms | 2681 ms |
| localstack | 10 | 15.8 MB/s | 77 ms | 1486 ms | 772 ms | 2780 ms |
| localstack | 11 | 16.5 MB/s | 76 ms | 98 ms | 793 ms | 2637 ms |
| localstack | 12 | 14.7 MB/s | 93 ms | 120 ms | 964 ms | 2310 ms |
| minio | 8 | 22.7 MB/s | 19 ms | 43 ms | 721 ms | 843 ms |
| minio | 9 | 20.7 MB/s | 21 ms | 29 ms | 783 ms | 921 ms |
| minio | 10 | 19.3 MB/s | 20 ms | 176 ms | 835 ms | 1041 ms |
| minio | 11 | 17.3 MB/s | 21 ms | 24 ms | 881 ms | 1394 ms |
| minio | 12 | 16.4 MB/s | 22 ms | 27 ms | 980 ms | 1356 ms |
| rustfs | 8 | 22.6 MB/s | 25 ms | 81 ms | 711 ms | 936 ms |
| rustfs | 9 | 20.6 MB/s | 26 ms | 190 ms | 782 ms | 981 ms |
| rustfs | 10 | 18.7 MB/s | 27 ms | 179 ms | 851 ms | 1261 ms |
| rustfs | 11 | 17.1 MB/s | 30 ms | 446 ms | 926 ms | 1573 ms |
| rustfs | 12 | 15.9 MB/s | 28 ms | 59 ms | 980 ms | 1423 ms |
| seaweedfs | 8 | 23.1 MB/s | 72 ms | 94 ms | 689 ms | 1037 ms |
| seaweedfs | 9 | 21.8 MB/s | 71 ms | 95 ms | 714 ms | 1394 ms |
| seaweedfs | 10 | 19.7 MB/s | 68 ms | 89 ms | 707 ms | 2450 ms |
| seaweedfs | 11 | 18.1 MB/s | 70 ms | 173 ms | 774 ms | 3801 ms |
| seaweedfs | 12 | 16.9 MB/s | 87 ms | 112 ms | 911 ms | 1555 ms |

---

## Methodology

- **Samples per Configuration:** 100
- **Metrics:** TTFB (Time to First Byte), TTLB (Time to Last Byte)
- **Scoring:** Weighted composite: 50% throughput + 30% latency + 20% consistency

## Configuration

| Parameter | Value |
|-----------|-------|
| Thread Range | 8 - 12 |
| Object Sizes | 4 MB - 16 MB |
| Samples/Config | 100 |
| Drivers | liteio, localstack, minio, rustfs, seaweedfs |
