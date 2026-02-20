# S3 Client Benchmark Results

**Date:** 2026-02-20T14:34:43+07:00

**BenchTime:** 500ms | **Warmup:** 3

## PutObject

| Driver | Operation | Iters | Avg | P50 | P95 | P99 | Throughput | Ops/s | Errors |
|--------|-----------|------:|----:|----:|----:|----:|-----------:|------:|-------:|
| herd_s3 | PutObject/10MB | 4 | 139.222ms | 117.401ms | 219.047ms | 219.047ms | 71.8 MB/s | 7 | 0 |
| liteio | PutObject/10MB | 7 | 85.078ms | 82.934ms | 109.22ms | 109.22ms | 117.5 MB/s | 12 | 0 |
| minio | PutObject/10MB | 9 | 57.604ms | 59.28ms | 61.268ms | 61.268ms | 173.6 MB/s | 17 | 0 |
| rustfs | PutObject/10MB | 9 | 57.52ms | 60.125ms | 64.961ms | 64.961ms | 173.9 MB/s | 17 | 0 |
| seaweedfs | PutObject/10MB | 7 | 82.526ms | 80.316ms | 92.887ms | 92.887ms | 121.2 MB/s | 12 | 0 |
| herd_s3 | PutObject/1KB | 319 | 1.798ms | 1.275ms | 4.521ms | 7.899ms | 0.5 MB/s | 556 | 0 |
| liteio | PutObject/1KB | 167 | 3.017ms | 3ms | 4.149ms | 4.64ms | 0.3 MB/s | 331 | 0 |
| minio | PutObject/1KB | 485 | 1.031ms | 902µs | 1.639ms | 3.892ms | 0.9 MB/s | 970 | 0 |
| rustfs | PutObject/1KB | 550 | 909µs | 832µs | 1.246ms | 2.458ms | 1.1 MB/s | 1100 | 0 |
| seaweedfs | PutObject/1KB | 302 | 1.657ms | 1.47ms | 3.148ms | 3.789ms | 0.6 MB/s | 603 | 0 |
| herd_s3 | PutObject/1MB | 14 | 52.042ms | 26.368ms | 267.399ms | 267.399ms | 19.2 MB/s | 19 | 0 |
| liteio | PutObject/1MB | 36 | 14.325ms | 10.424ms | 31.398ms | 81.506ms | 69.8 MB/s | 70 | 0 |
| minio | PutObject/1MB | 69 | 7.372ms | 7.1ms | 9.772ms | 12.494ms | 135.6 MB/s | 136 | 0 |
| rustfs | PutObject/1MB | 70 | 7.575ms | 7.089ms | 11.358ms | 16.937ms | 132.0 MB/s | 132 | 0 |
| seaweedfs | PutObject/1MB | 49 | 10.225ms | 9.55ms | 15.226ms | 17.55ms | 97.8 MB/s | 98 | 0 |
| herd_s3 | PutObject/64KB | 111 | 5.016ms | 3.83ms | 13.134ms | 16.701ms | 12.5 MB/s | 199 | 0 |
| liteio | PutObject/64KB | 207 | 2.417ms | 1.893ms | 4.497ms | 6.549ms | 25.9 MB/s | 414 | 0 |
| minio | PutObject/64KB | 380 | 1.318ms | 1.227ms | 1.868ms | 3.27ms | 47.4 MB/s | 759 | 0 |
| rustfs | PutObject/64KB | 451 | 1.117ms | 1.1ms | 1.287ms | 1.359ms | 56.0 MB/s | 895 | 0 |
| seaweedfs | PutObject/64KB | 292 | 1.823ms | 1.508ms | 3.46ms | 6.027ms | 34.3 MB/s | 549 | 0 |

## GetObject

| Driver | Operation | Iters | Avg | P50 | P95 | P99 | Throughput | Ops/s | Errors |
|--------|-----------|------:|----:|----:|----:|----:|-----------:|------:|-------:|
| herd_s3 | GetObject/10MB | 3 | 187.059ms | 166.763ms | 234.12ms | 234.12ms | 53.5 MB/s | 5 | 0 |
| liteio | GetObject/10MB | 6 | 87.423ms | 83.5ms | 100.012ms | 100.012ms | 114.4 MB/s | 11 | 0 |
| minio | GetObject/10MB | 9 | 64.083ms | 65.593ms | 87.303ms | 87.303ms | 156.0 MB/s | 16 | 0 |
| rustfs | GetObject/10MB | 14 | 38.694ms | 36.775ms | 49.491ms | 49.491ms | 258.4 MB/s | 26 | 0 |
| seaweedfs | GetObject/10MB | 13 | 41.051ms | 39.05ms | 55.621ms | 55.621ms | 243.6 MB/s | 24 | 0 |
| herd_s3 | GetObject/1KB | 673 | 743µs | 611µs | 1.643ms | 2.765ms | 1.3 MB/s | 1346 | 0 |
| liteio | GetObject/1KB | 1116 | 573µs | 382µs | 1.694ms | 2.121ms | 1.7 MB/s | 1746 | 0 |
| minio | GetObject/1KB | 1820 | 287µs | 266µs | 379µs | 647µs | 3.4 MB/s | 3481 | 0 |
| rustfs | GetObject/1KB | 901 | 561µs | 545µs | 635µs | 796µs | 1.7 MB/s | 1783 | 0 |
| seaweedfs | GetObject/1KB | 936 | 535µs | 418µs | 1.251ms | 2.415ms | 1.8 MB/s | 1869 | 0 |
| herd_s3 | GetObject/1MB | 95 | 6.929ms | 5.013ms | 21.27ms | 34.495ms | 144.3 MB/s | 144 | 0 |
| liteio | GetObject/1MB | 51 | 9.935ms | 8.981ms | 15.129ms | 18.558ms | 100.6 MB/s | 101 | 0 |
| minio | GetObject/1MB | 129 | 3.917ms | 3.761ms | 4.651ms | 7.024ms | 255.3 MB/s | 255 | 0 |
| rustfs | GetObject/1MB | 107 | 4.687ms | 4.576ms | 5.318ms | 5.868ms | 213.4 MB/s | 213 | 0 |
| seaweedfs | GetObject/1MB | 114 | 4.42ms | 4.206ms | 5.384ms | 7.75ms | 226.2 MB/s | 226 | 0 |
| herd_s3 | GetObject/64KB | 490 | 1.021ms | 817µs | 2.301ms | 3.836ms | 61.2 MB/s | 979 | 0 |
| liteio | GetObject/64KB | 259 | 1.957ms | 1.874ms | 2.78ms | 3.748ms | 31.9 MB/s | 511 | 0 |
| minio | GetObject/64KB | 1019 | 576µs | 482µs | 1.214ms | 1.915ms | 108.6 MB/s | 1737 | 0 |
| rustfs | GetObject/64KB | 639 | 783µs | 770µs | 876µs | 1.094ms | 79.8 MB/s | 1277 | 0 |
| seaweedfs | GetObject/64KB | 694 | 724µs | 683µs | 946µs | 1.467ms | 86.3 MB/s | 1381 | 0 |

## HeadObject

| Driver | Operation | Iters | Avg | P50 | P95 | P99 | Throughput | Ops/s | Errors |
|--------|-----------|------:|----:|----:|----:|----:|-----------:|------:|-------:|
| herd_s3 | HeadObject | 702 | 712µs | 451µs | 1.738ms | 6.511ms | - | 1404 | 0 |
| liteio | HeadObject | 312 | 1.606ms | 1.581ms | 2.146ms | 2.516ms | - | 623 | 0 |
| minio | HeadObject | 1152 | 434µs | 284µs | 1.271ms | 2.584ms | - | 2303 | 0 |
| rustfs | HeadObject | 1056 | 474µs | 407µs | 827µs | 1.381ms | - | 2110 | 0 |
| seaweedfs | HeadObject | 1431 | 349µs | 316µs | 531µs | 872µs | - | 2862 | 0 |

## DeleteObject

| Driver | Operation | Iters | Avg | P50 | P95 | P99 | Throughput | Ops/s | Errors |
|--------|-----------|------:|----:|----:|----:|----:|-----------:|------:|-------:|
| herd_s3 | DeleteObject | 881 | 641µs | 479µs | 1.445ms | 2.758ms | - | 1561 | 0 |
| liteio | DeleteObject | 1833 | 282µs | 234µs | 503µs | 895µs | - | 3545 | 0 |
| minio | DeleteObject | 243 | 328µs | 312µs | 450µs | 693µs | - | 3048 | 1370 |
| rustfs | DeleteObject | 389 | 1.288ms | 1.082ms | 2.429ms | 2.799ms | - | 777 | 0 |
| seaweedfs | DeleteObject | 1199 | 417µs | 350µs | 648µs | 1.858ms | - | 2398 | 0 |

## ListObjects

| Driver | Operation | Iters | Avg | P50 | P95 | P99 | Throughput | Ops/s | Errors |
|--------|-----------|------:|----:|----:|----:|----:|-----------:|------:|-------:|
| herd_s3 | ListObjects | 428 | 1.169ms | 1.038ms | 2.013ms | 2.798ms | - | 855 | 0 |
| liteio | ListObjects | 587 | 852µs | 816µs | 1.114ms | 1.405ms | - | 1173 | 0 |
| minio | ListObjects | 0 | 0s | 0s | 0s | 0s | - | 0 | 1853 |
| rustfs | ListObjects | 72 | 7.006ms | 6.994ms | 7.531ms | 13.196ms | - | 143 | 0 |
| seaweedfs | ListObjects | 347 | 1.445ms | 1.326ms | 2.174ms | 3.471ms | - | 692 | 0 |

## Multipart

| Driver | Operation | Iters | Avg | P50 | P95 | P99 | Throughput | Ops/s | Errors |
|--------|-----------|------:|----:|----:|----:|----:|-----------:|------:|-------:|
| herd_s3 | Multipart/20MB | 3 | 702.674ms | 840.824ms | 1.028209s | 1.028209s | 28.5 MB/s | 1 | 0 |
| liteio | Multipart/20MB | 4 | 152.984ms | 146.239ms | 169.893ms | 169.893ms | 130.7 MB/s | 7 | 0 |
| minio | Multipart/20MB | 0 | 0s | 0s | 0s | 0s | - | 0 | 2038 |
| rustfs | Multipart/20MB | 5 | 117.939ms | 117.33ms | 128.797ms | 128.797ms | 169.6 MB/s | 8 | 0 |
| seaweedfs | Multipart/20MB | 3 | 195.377ms | 187.404ms | 214.267ms | 214.267ms | 102.4 MB/s | 5 | 0 |

## Mixed

| Driver | Operation | Iters | Avg | P50 | P95 | P99 | Throughput | Ops/s | Errors |
|--------|-----------|------:|----:|----:|----:|----:|-----------:|------:|-------:|
| herd_s3 | Mixed/10r90w | 1473 | 20.347ms | 15.306ms | 25.258ms | 220.338ms | 3.1 MB/s | 49 | 0 |
| liteio | Mixed/10r90w | 1606 | 15.71ms | 15.36ms | 23.44ms | 31.307ms | 4.0 MB/s | 64 | 0 |
| minio | Mixed/10r90w | 0 | 0s | 0s | 0s | 0s | - | 0 | 2126 |
| rustfs | Mixed/10r90w | 1098 | 31.502ms | 19.82ms | 38.653ms | 64.34ms | 2.0 MB/s | 32 | 0 |
| seaweedfs | Mixed/10r90w | 1275 | 28.123ms | 16.444ms | 36.101ms | 66.732ms | 2.2 MB/s | 36 | 0 |
| herd_s3 | Mixed/50r50w | 1438 | 17.616ms | 17.054ms | 23.34ms | 24.65ms | 3.5 MB/s | 57 | 0 |
| liteio | Mixed/50r50w | 1604 | 15.796ms | 15.048ms | 27.031ms | 29.823ms | 4.0 MB/s | 63 | 0 |
| minio | Mixed/50r50w | 0 | 0s | 0s | 0s | 0s | - | 0 | 2987 |
| rustfs | Mixed/50r50w | 1257 | 20.199ms | 19.194ms | 32.471ms | 37.78ms | 3.1 MB/s | 50 | 0 |
| seaweedfs | Mixed/50r50w | 1488 | 16.98ms | 16.263ms | 24.683ms | 33.948ms | 3.7 MB/s | 59 | 0 |
| herd_s3 | Mixed/90r10w | 1331 | 19.091ms | 18.47ms | 23.49ms | 27.822ms | 3.3 MB/s | 52 | 0 |
| liteio | Mixed/90r10w | 1523 | 16.666ms | 16.112ms | 23.699ms | 28.763ms | 3.8 MB/s | 60 | 0 |
| minio | Mixed/90r10w | 0 | 0s | 0s | 0s | 0s | - | 0 | 8335 |
| rustfs | Mixed/90r10w | 1278 | 19.891ms | 19.023ms | 31.37ms | 40.035ms | 3.1 MB/s | 50 | 0 |
| seaweedfs | Mixed/90r10w | 1579 | 16.034ms | 15.29ms | 20.441ms | 36.5ms | 3.9 MB/s | 62 | 0 |

## Concurrency

| Driver | C1 | C10 | C50 | C100 | C200 |
|--------|----:|----:|----:|----:|----:|
| herd_s3 | 50.0 MB/s | 15.7 MB/s | 2.2 MB/s | 1.4 MB/s | 0.5 MB/s |
| liteio | 49.9 MB/s | 11.9 MB/s | 2.7 MB/s | 1.4 MB/s | 0.6 MB/s |
| minio | 0 ops/s | 0 ops/s | 0 ops/s | 0 ops/s | 0 ops/s |
| rustfs | 50.3 MB/s | 14.1 MB/s | 1.8 MB/s | 1.1 MB/s | 0.5 MB/s |
| seaweedfs | 38.5 MB/s | 12.2 MB/s | 2.4 MB/s | 0.9 MB/s | 0.5 MB/s |

