# S3 Client Benchmark Results

**Date:** 2026-02-20T14:54:43+07:00

**BenchTime:** 500ms | **Warmup:** 3

## Summary

### Leaderboard

| Rank | Driver | Wins | Share |
|-----:|--------|-----:|------:|
| 1 | **liteio_herd** | **12** | **80%** |
| 2 | liteio_local | 3 | 20% |

### Winner per Operation

| Operation | Winner | Avg | Runner-up | Avg | Speedup |
|-----------|--------|----:|-----------|----:|--------:|
| DeleteObject | **liteio_herd** | 67µs | liteio_local | 113µs | 1.7x |
| GetObject/10MB | **liteio_herd** | 2.135ms | liteio_local | 2.44ms | 1.1x |
| GetObject/1KB | **liteio_local** | 72µs | liteio_herd | 72µs | 1.0x |
| GetObject/1MB | **liteio_local** | 216µs | liteio_herd | 227µs | 1.1x |
| GetObject/64KB | **liteio_local** | 78µs | liteio_herd | 81µs | 1.0x |
| HeadObject | **liteio_herd** | 69µs | liteio_local | 73µs | 1.1x |
| ListObjects | **liteio_herd** | 412µs | liteio_local | 686µs | 1.7x |
| Mixed/10r90w | **liteio_herd** | 1.697ms | liteio_local | 3.784ms | 2.2x |
| Mixed/50r50w | **liteio_herd** | 1.317ms | liteio_local | 1.92ms | 1.5x |
| Mixed/90r10w | **liteio_herd** | 1.14ms | liteio_local | 1.381ms | 1.2x |
| Multipart/20MB | **liteio_herd** | 49.115ms | minio | 53.167ms | 1.1x |
| PutObject/10MB | **liteio_herd** | 7.146ms | liteio_local | 7.461ms | 1.0x |
| PutObject/1KB | **liteio_herd** | 94µs | liteio_local | 134µs | 1.4x |
| PutObject/1MB | **liteio_herd** | 583µs | liteio_local | 928µs | 1.6x |
| PutObject/64KB | **liteio_herd** | 114µs | liteio_local | 199µs | 1.7x |

### Throughput Highlights

| Operation | Winner | Throughput | Runner-up | Throughput | Speedup |
|-----------|--------|----------:|-----------|----------:|--------:|
| DeleteObject | **liteio_herd** | 14969 ops/s | liteio_local | 8881 ops/s | 1.7x |
| GetObject/10MB | **liteio_herd** | 4684.3 MB/s | liteio_local | 4099.2 MB/s | 1.1x |
| GetObject/1KB | **liteio_local** | 13.6 MB/s | liteio_herd | 13.5 MB/s | 1.0x |
| GetObject/1MB | **liteio_local** | 4625.1 MB/s | liteio_herd | 4400.6 MB/s | 1.1x |
| GetObject/64KB | **liteio_local** | 799.7 MB/s | liteio_herd | 770.7 MB/s | 1.0x |
| HeadObject | **liteio_herd** | 14485 ops/s | liteio_local | 13724 ops/s | 1.1x |
| ListObjects | **liteio_herd** | 2429 ops/s | liteio_local | 1458 ops/s | 1.7x |
| Mixed/10r90w | **liteio_herd** | 36.8 MB/s | liteio_local | 16.5 MB/s | 2.2x |
| Mixed/50r50w | **liteio_herd** | 47.4 MB/s | liteio_local | 32.6 MB/s | 1.5x |
| Mixed/90r10w | **liteio_herd** | 54.8 MB/s | liteio_local | 45.3 MB/s | 1.2x |
| Multipart/20MB | **liteio_herd** | 407.2 MB/s | minio | 376.2 MB/s | 1.1x |
| PutObject/10MB | **liteio_herd** | 1399.3 MB/s | liteio_local | 1340.3 MB/s | 1.0x |
| PutObject/1KB | **liteio_herd** | 10.4 MB/s | liteio_local | 7.3 MB/s | 1.4x |
| PutObject/1MB | **liteio_herd** | 1714.2 MB/s | liteio_local | 1077.8 MB/s | 1.6x |
| PutObject/64KB | **liteio_herd** | 545.9 MB/s | liteio_local | 314.8 MB/s | 1.7x |

---

## PutObject

| | Driver | Operation | Iters | Avg | P50 | P99 | Throughput | Ops/s | vs Best |
|---|--------|-----------|------:|----:|----:|----:|-----------:|------:|--------:|
| 1 | **liteio_herd** | PutObject/10MB | 72 | **7.146ms** | 7.01ms | 9.665ms | 1399.3 MB/s | 140 | **5x** |
| 2 | liteio_local | PutObject/10MB | 70 | 7.461ms | 7.055ms | 14.897ms | 1340.3 MB/s | 134 | 1.0x |
| 3 | minio | PutObject/10MB | 22 | 23.291ms | 23.232ms | 23.942ms | 429.4 MB/s | 43 | 3.3x |
|    | rustfs | PutObject/10MB | 19 | 26.999ms | 26.916ms | 27.641ms | 370.4 MB/s | 37 | 3.8x |
|    | seaweedfs | PutObject/10MB | 15 | 33.699ms | 33.238ms | 36.267ms | 296.7 MB/s | 30 | 4.7x |
| 1 | **liteio_herd** | PutObject/1KB | 5845 | **94µs** | 74µs | 410µs | 10.4 MB/s | 10618 | **6x** |
| 2 | liteio_local | PutObject/1KB | 3724 | 134µs | 125µs | 275µs | 7.3 MB/s | 7455 | 1.4x |
| 3 | seaweedfs | PutObject/1KB | 1396 | 358µs | 327µs | 934µs | 2.7 MB/s | 2793 | 3.8x |
|    | minio | PutObject/1KB | 961 | 521µs | 491µs | 961µs | 1.9 MB/s | 1921 | 5.5x |
|    | rustfs | PutObject/1KB | 890 | 562µs | 497µs | 1.486ms | 1.7 MB/s | 1779 | 6.0x |
| 1 | **liteio_herd** | PutObject/1MB | 860 | **583µs** | 566µs | 839µs | 1714.2 MB/s | 1714 | **8x** |
| 2 | liteio_local | PutObject/1MB | 539 | 928µs | 851µs | 1.674ms | 1077.8 MB/s | 1078 | 1.6x |
| 3 | minio | PutObject/1MB | 145 | 3.455ms | 3.187ms | 6.081ms | 289.4 MB/s | 289 | 5.9x |
|    | rustfs | PutObject/1MB | 134 | 3.745ms | 3.702ms | 4.968ms | 267.0 MB/s | 267 | 6.4x |
|    | seaweedfs | PutObject/1MB | 109 | 4.601ms | 4.291ms | 6.926ms | 217.3 MB/s | 217 | 7.9x |
| 1 | **liteio_herd** | PutObject/64KB | 4361 | **114µs** | 104µs | 341µs | 545.9 MB/s | 8734 | **11x** |
| 2 | liteio_local | PutObject/64KB | 2516 | 199µs | 162µs | 697µs | 314.8 MB/s | 5036 | 1.7x |
| 3 | seaweedfs | PutObject/64KB | 885 | 571µs | 556µs | 899µs | 109.5 MB/s | 1752 | 5.0x |
|    | minio | PutObject/64KB | 729 | 686µs | 662µs | 1.236ms | 91.1 MB/s | 1458 | 6.0x |
|    | rustfs | PutObject/64KB | 714 | 1.226ms | 715µs | 4.361ms | 51.0 MB/s | 816 | 10.7x |

## GetObject

| | Driver | Operation | Iters | Avg | P50 | P99 | Throughput | Ops/s | vs Best |
|---|--------|-----------|------:|----:|----:|----:|-----------:|------:|--------:|
| 1 | **liteio_herd** | GetObject/10MB | 235 | **2.135ms** | 1.994ms | 3.709ms | 4684.3 MB/s | 468 | **3x** |
| 2 | liteio_local | GetObject/10MB | 207 | 2.44ms | 2.274ms | 4.67ms | 4099.2 MB/s | 410 | 1.1x |
| 3 | seaweedfs | GetObject/10MB | 149 | 3.365ms | 3.162ms | 5.961ms | 2972.0 MB/s | 297 | 1.6x |
|    | minio | GetObject/10MB | 128 | 3.917ms | 3.151ms | 7.014ms | 2553.1 MB/s | 255 | 1.8x |
|    | rustfs | GetObject/10MB | 91 | 5.549ms | 5.551ms | 7.969ms | 1802.0 MB/s | 180 | 2.6x |
| 1 | **liteio_local** | GetObject/1KB | 6953 | **72µs** | 69µs | 125µs | 13.6 MB/s | 13920 | **3x** |
| 2 | liteio_herd | GetObject/1KB | 6923 | 72µs | 68µs | 131µs | 13.5 MB/s | 13858 | 1.0x |
| 3 | minio | GetObject/1KB | 2720 | 184µs | 178µs | 276µs | 5.3 MB/s | 5424 | 2.6x |
|    | seaweedfs | GetObject/1KB | 2742 | 184µs | 175µs | 411µs | 5.3 MB/s | 5421 | 2.6x |
|    | rustfs | GetObject/1KB | 2187 | 230µs | 215µs | 547µs | 4.2 MB/s | 4339 | 3.2x |
| 1 | **liteio_local** | GetObject/1MB | 2312 | **216µs** | 197µs | 505µs | 4625.1 MB/s | 4625 | **4x** |
| 2 | liteio_herd | GetObject/1MB | 2200 | 227µs | 201µs | 521µs | 4400.6 MB/s | 4401 | 1.1x |
| 3 | seaweedfs | GetObject/1MB | 1101 | 476µs | 446µs | 1.112ms | 2103.0 MB/s | 2103 | 2.2x |
|    | minio | GetObject/1MB | 1053 | 608µs | 455µs | 1.677ms | 1645.0 MB/s | 1645 | 2.8x |
|    | rustfs | GetObject/1MB | 729 | 759µs | 752µs | 1.195ms | 1317.0 MB/s | 1317 | 3.5x |
| 1 | **liteio_local** | GetObject/64KB | 6561 | **78µs** | 73µs | 202µs | 799.7 MB/s | 12796 | **3x** |
| 2 | liteio_herd | GetObject/64KB | 6561 | 81µs | 74µs | 222µs | 770.7 MB/s | 12331 | 1.0x |
| 3 | minio | GetObject/64KB | 2572 | 194µs | 186µs | 424µs | 321.6 MB/s | 5145 | 2.5x |
|    | seaweedfs | GetObject/64KB | 2497 | 201µs | 198µs | 294µs | 311.0 MB/s | 4976 | 2.6x |
|    | rustfs | GetObject/64KB | 2019 | 248µs | 243µs | 342µs | 252.5 MB/s | 4039 | 3.2x |

## HeadObject

| | Driver | Operation | Iters | Avg | P50 | P99 | Throughput | Ops/s | vs Best |
|---|--------|-----------|------:|----:|----:|----:|-----------:|------:|--------:|
| 1 | **liteio_herd** | HeadObject | 7236 | **69µs** | 66µs | 138µs | - | 14485 | **2x** |
| 2 | liteio_local | HeadObject | 6856 | 73µs | 69µs | 183µs | - | 13724 | 1.1x |
| 3 | seaweedfs | HeadObject | 3811 | 131µs | 128µs | 206µs | - | 7616 | 1.9x |
|    | minio | HeadObject | 3005 | 166µs | 154µs | 353µs | - | 6012 | 2.4x |
|    | rustfs | HeadObject | 2978 | 168µs | 163µs | 256µs | - | 5957 | 2.4x |

## DeleteObject

| | Driver | Operation | Iters | Avg | P50 | P99 | Throughput | Ops/s | vs Best |
|---|--------|-----------|------:|----:|----:|----:|-----------:|------:|--------:|
| 1 | **liteio_herd** | DeleteObject | 7655 | **67µs** | 63µs | 171µs | - | 14969 | **10x** |
| 2 | liteio_local | DeleteObject | 4576 | 113µs | 108µs | 210µs | - | 8881 | 1.7x |
| 3 | seaweedfs | DeleteObject | 3529 | 145µs | 135µs | 350µs | - | 6891 | 2.2x |
|    | minio | DeleteObject | 1112 | 460µs | 450µs | 635µs | - | 2175 | 6.9x |
|    | rustfs | DeleteObject | 750 | 667µs | 597µs | 2.253ms | - | 1500 | 10.0x |

## ListObjects

| | Driver | Operation | Iters | Avg | P50 | P99 | Throughput | Ops/s | vs Best |
|---|--------|-----------|------:|----:|----:|----:|-----------:|------:|--------:|
| 1 | **liteio_herd** | ListObjects | 1215 | **412µs** | 406µs | 674µs | - | 2429 | **9x** |
| 2 | liteio_local | ListObjects | 729 | 686µs | 679µs | 921µs | - | 1458 | 1.7x |
| 3 | seaweedfs | ListObjects | 428 | 1.17ms | 1.155ms | 1.441ms | - | 855 | 2.8x |
|    | minio | ListObjects | 207 | 2.423ms | 2.392ms | 3.506ms | - | 413 | 5.9x |
|    | rustfs | ListObjects | 142 | 3.687ms | 3.604ms | 4.909ms | - | 271 | 9.0x |

## Multipart

| | Driver | Operation | Iters | Avg | P50 | P99 | Throughput | Ops/s | vs Best |
|---|--------|-----------|------:|----:|----:|----:|-----------:|------:|--------:|
| 1 | **liteio_herd** | Multipart/20MB | 11 | **49.115ms** | 45.251ms | 75.969ms | 407.2 MB/s | 20 | **2x** |
| 2 | minio | Multipart/20MB | 10 | 53.167ms | 52.174ms | 56.565ms | 376.2 MB/s | 19 | 1.1x |
| 3 | liteio_local | Multipart/20MB | 10 | 54.448ms | 54.193ms | 65.548ms | 367.3 MB/s | 18 | 1.1x |
|    | rustfs | Multipart/20MB | 8 | 62.798ms | 62.339ms | 67.195ms | 318.5 MB/s | 16 | 1.3x |
|    | seaweedfs | Multipart/20MB | 6 | 87.719ms | 84.905ms | 94.837ms | 228.0 MB/s | 11 | 1.8x |

## Mixed

| | Driver | Operation | Iters | Avg | P50 | P99 | Throughput | Ops/s | vs Best |
|---|--------|-----------|------:|----:|----:|----:|-----------:|------:|--------:|
| 1 | **liteio_herd** | Mixed/10r90w | 14674 | **1.697ms** | 1.156ms | 8.408ms | 36.8 MB/s | 589 | **9x** |
| 2 | liteio_local | Mixed/10r90w | 6596 | 3.784ms | 2.292ms | 20.009ms | 16.5 MB/s | 264 | 2.2x |
| 3 | seaweedfs | Mixed/10r90w | 4542 | 5.523ms | 5.394ms | 11.464ms | 11.3 MB/s | 181 | 3.3x |
|    | minio | Mixed/10r90w | 1953 | 13.032ms | 9.82ms | 51.131ms | 4.8 MB/s | 77 | 7.7x |
|    | rustfs | Mixed/10r90w | 1674 | 15.244ms | 15.037ms | 26.06ms | 4.1 MB/s | 66 | 9.0x |
| 1 | **liteio_herd** | Mixed/50r50w | 18931 | **1.317ms** | 878µs | 7.191ms | 47.4 MB/s | 759 | **9x** |
| 2 | liteio_local | Mixed/50r50w | 13006 | 1.92ms | 1.212ms | 11.06ms | 32.6 MB/s | 521 | 1.5x |
| 3 | seaweedfs | Mixed/50r50w | 7501 | 3.336ms | 2.604ms | 8.93ms | 18.7 MB/s | 300 | 2.5x |
|    | minio | Mixed/50r50w | 3687 | 6.878ms | 3.656ms | 38.545ms | 9.1 MB/s | 145 | 5.2x |
|    | rustfs | Mixed/50r50w | 2126 | 11.883ms | 11.323ms | 28.918ms | 5.3 MB/s | 84 | 9.0x |
| 1 | **liteio_herd** | Mixed/90r10w | 21929 | **1.14ms** | 838µs | 6.12ms | 54.8 MB/s | 877 | **10x** |
| 2 | liteio_local | Mixed/90r10w | 18066 | 1.381ms | 966µs | 7.574ms | 45.3 MB/s | 724 | 1.2x |
| 3 | seaweedfs | Mixed/90r10w | 10943 | 2.285ms | 1.973ms | 6.605ms | 27.4 MB/s | 438 | 2.0x |
|    | minio | Mixed/90r10w | 7142 | 3.529ms | 2.166ms | 24.524ms | 17.7 MB/s | 283 | 3.1x |
|    | rustfs | Mixed/90r10w | 2192 | 11.539ms | 11.529ms | 15.391ms | 5.4 MB/s | 87 | 10.1x |

## Concurrency

| Driver | C1 | C10 | C50 | C100 | C200 |
|--------|----:|----:|----:|----:|----:|
| liteio_herd | **562.9 MB/s** | **174.6 MB/s** | **22.4 MB/s** | **7.3 MB/s** | 2.6 MB/s |
| liteio_local | 287.1 MB/s | 80.1 MB/s | 16.1 MB/s | 6.9 MB/s | **2.8 MB/s** |
| seaweedfs | 96.1 MB/s | 45.3 MB/s | 10.2 MB/s | 5.1 MB/s | 2.5 MB/s |
| minio | 62.7 MB/s | 22.6 MB/s | 4.0 MB/s | 1.7 MB/s | 1.0 MB/s |
| rustfs | 60.4 MB/s | 17.5 MB/s | 3.3 MB/s | 1.5 MB/s | 0.8 MB/s |

