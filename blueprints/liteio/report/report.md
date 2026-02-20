# Storage Benchmark Report

## System Info

| Property | Value |
|----------|-------|
| Timestamp | 2026-02-20T23:58:59+07:00 |
| Go Version | go1.26.0 |
| Platform | darwin/arm64 |
| CPUs | 10 |
| BenchTime | 500ms |
| Concurrency | 200 |
| Levels | [1 10 25 50 100 200] |
| Object Sizes | 1KB, 64KB, 1MB, 10MB, 100MB |
| Warmup | 10 iterations |
| Timeout | 30s |
| Drivers | 5 |

## Table of Contents

1. [Executive Summary Dashboard](#executive-summary-dashboard)
2. [Performance Matrix](#performance-matrix)
3. [Write Performance Deep Dive](#write-performance-deep-dive)
4. [Read Performance Deep Dive](#read-performance-deep-dive)
5. [Parallel Scalability](#parallel-scalability)
6. [Latency Analysis](#latency-analysis)
7. [Resource Efficiency](#resource-efficiency)
8. [Error & Timeout Summary](#error--timeout-summary)
9. [Recommendations](#recommendations)

## Executive Summary Dashboard

| # | Driver | Write | Read | Wins | Memory | Disk | Status |
|---|--------|-------|------|------|--------|------|--------|
| 1 | **horse** | ***** | ***** | 42/48 | 6.5 GB | 32.0 GB | OK |
| 2 | **usagi** | ***   | **    | 4/48 | 6.5 GB | 12.0 GB | OK |
| 3 | **rabbit** | *     | ***   | 1/48 | 2.0 GB | 5.4 GB | OK |
| 4 | **local** | *     | **    | 0/48 | 6.5 GB | 5.9 GB | OK |
| 5 | **devnull_s3** | *     |       | 1/48 | - | - | 319223 err |

> **Overall Leader: horse** -- won 42/48 benchmarks with combined write+read score of 98%.

## Performance Matrix

All drivers x key operations. Values show throughput (ops/s or MB/s). **Bold** = best in column.

| Driver | W/1KB | W/64KB | W/1MB | W/10MB | W/100MB | R/1KB | R/64KB | R/1MB | R/10MB | R/100MB | Stat | Delete | Copy/1KB | List/100 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| devnull_s3 | 13.5 MB/s | 591.4 MB/s | 1.3 GB/s | 1.3 GB/s | 1.5 GB/s | 0.00 MB/s | 0.00 MB/s | 0.00 MB/s | 0.00 MB/s | 0.00 MB/s | 0/s | 18.8K/s | 15.5 MB/s | 1.4K/s |
| horse | **1.4 GB/s** | **1.6 GB/s** | **2.1 GB/s** | **2.4 GB/s** | 1.7 GB/s | **6.2 GB/s** | 50.0 GB/s | **54.2 GB/s** | **15.4 GB/s** | 7.3 GB/s | 15.1M/s | **2.5M/s** | **719.9 MB/s** | 120.6K/s |
| local | 2.0 MB/s | 192.9 MB/s | 1.4 GB/s | 1.7 GB/s | 1.6 GB/s | 5.3 GB/s | 15.8 GB/s | 17.5 GB/s | 10.1 GB/s | 11.0 GB/s | 5.3M/s | 22.7K/s | 2.2 MB/s | 4.5K/s |
| rabbit | 19.7 MB/s | 890.0 MB/s | 1.8 GB/s | 1.6 GB/s | 1.6 GB/s | 4.8 GB/s | **50.7 GB/s** | 13.1 GB/s | 10.7 GB/s | 10.9 GB/s | 5.1M/s | 27.8K/s | 14.0 MB/s | 5.4K/s |
| usagi | 295.3 MB/s | 1.4 GB/s | 1.8 GB/s | 1.5 GB/s | **1.9 GB/s** | 3.5 GB/s | 12.3 GB/s | 15.7 GB/s | 12.1 GB/s | **11.4 GB/s** | **19.0M/s** | 592.6K/s | 306.9 MB/s | **174.0K/s** |

## Write Performance Deep Dive

### Write Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| devnull_s3 | 13.8K/s | 13.5 MB/s | 63.8us | 292.9us | 9.5K/s | 591.4 MB/s | 97.8us | 331.7us | 1.3K/s | 1.3 GB/s | 667.5us | 1.1ms | 125/s | 1.3 GB/s | 8.0ms | 8.9ms | 15/s | 1.5 GB/s | 65.4ms | 65.9ms |
| horse | 1.4M/s | 1.4 GB/s | 500ns | 1.7us | 25.3K/s | 1.6 GB/s | 2.0us | 6.3us | 2.1K/s | 2.1 GB/s | 21.1us | 14.5ms | 244/s | 2.4 GB/s | 253.6us | 46.7ms | 17/s | 1.7 GB/s | 56.8ms | 69.6ms |
| local | 2.1K/s | 2.0 MB/s | 309.1us | 1.7ms | 3.1K/s | 192.9 MB/s | 317.8us | 437.2us | 1.4K/s | 1.4 GB/s | 670.2us | 1.2ms | 170/s | 1.7 GB/s | 3.6ms | 16.4ms | 16/s | 1.6 GB/s | 45.5ms | 99.7ms |
| rabbit | 20.2K/s | 19.7 MB/s | 43.2us | 152.3us | 14.2K/s | 890.0 MB/s | 58.8us | 265.5us | 1.8K/s | 1.8 GB/s | 339.7us | 2.4ms | 155/s | 1.6 GB/s | 6.3ms | 9.1ms | 16/s | 1.6 GB/s | 58.7ms | 67.6ms |
| usagi | 302.4K/s | 295.3 MB/s | 2.3us | 7.7us | 22.3K/s | 1.4 GB/s | 18.8us | 131.7us | 1.8K/s | 1.8 GB/s | 186.5us | 3.3ms | 154/s | 1.5 GB/s | 4.4ms | 41.8ms | 19/s | 1.9 GB/s | 53.3ms | 57.3ms |

**Write/1KB Throughput:**
```
  horse      |######################################## 1.4 GB/s (100%)
  usagi      |########................................ 295.3 MB/s (21%)
  rabbit     |#....................................... 19.7 MB/s (1%)
  devnull_s3 |#....................................... 13.5 MB/s (1%)
  local      |#....................................... 2.0 MB/s (0%)
```

**Write/64KB Throughput:**
```
  horse      |######################################## 1.6 GB/s (100%)
  usagi      |###################################..... 1.4 GB/s (88%)
  rabbit     |######################.................. 890.0 MB/s (56%)
  devnull_s3 |##############.......................... 591.4 MB/s (37%)
  local      |####.................................... 192.9 MB/s (12%)
```

**Write/1MB Throughput:**
```
  horse      |######################################## 2.1 GB/s (100%)
  rabbit     |##################################...... 1.8 GB/s (85%)
  usagi      |#################################....... 1.8 GB/s (85%)
  local      |##########################.............. 1.4 GB/s (66%)
  devnull_s3 |#########################............... 1.3 GB/s (65%)
```

**Write/10MB Throughput:**
```
  horse      |######################################## 2.4 GB/s (100%)
  local      |###########################............. 1.7 GB/s (70%)
  rabbit     |#########################............... 1.6 GB/s (64%)
  usagi      |#########################............... 1.5 GB/s (63%)
  devnull_s3 |####################.................... 1.3 GB/s (51%)
```

**Write/100MB Throughput:**
```
  usagi      |######################################## 1.9 GB/s (100%)
  horse      |####################################.... 1.7 GB/s (92%)
  local      |##################################...... 1.6 GB/s (87%)
  rabbit     |##################################...... 1.6 GB/s (86%)
  devnull_s3 |#################################....... 1.5 GB/s (83%)
```

## Read Performance Deep Dive

### Read Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| devnull_s3 | 0/s | 0.00 MB/s | 0ns | 0ns | 0/s | 0.00 MB/s | 0ns | 0ns | 0/s | 0.00 MB/s | 0ns | 0ns | 0/s | 0.00 MB/s | 0ns | 0ns | 0/s | 0.00 MB/s | 0ns | 0ns |
| horse | 6.3M/s | 6.2 GB/s | 125ns | 458ns | 800.3K/s | 50.0 GB/s | 1.2us | 3.0us | 54.2K/s | 54.2 GB/s | 18.0us | 25.3us | 1.5K/s | 15.4 GB/s | 227.0us | 3.5ms | 73/s | 7.3 GB/s | 11.3ms | 25.9ms |
| local | 5.4M/s | 5.3 GB/s | 167ns | 417ns | 253.6K/s | 15.8 GB/s | 3.5us | 10.7us | 17.5K/s | 17.5 GB/s | 55.8us | 79.8us | 1.0K/s | 10.1 GB/s | 961.1us | 1.5ms | 110/s | 11.0 GB/s | 8.6ms | 12.6ms |
| rabbit | 4.9M/s | 4.8 GB/s | 167ns | 458ns | 810.9K/s | 50.7 GB/s | 1.2us | 1.6us | 13.1K/s | 13.1 GB/s | 75.1us | 98.0us | 1.1K/s | 10.7 GB/s | 919.0us | 1.2ms | 109/s | 10.9 GB/s | 9.1ms | 10.8ms |
| usagi | 3.6M/s | 3.5 GB/s | 209ns | 1.1us | 197.5K/s | 12.3 GB/s | 3.6us | 34.8us | 15.7K/s | 15.7 GB/s | 62.9us | 76.0us | 1.2K/s | 12.1 GB/s | 813.2us | 1.0ms | 114/s | 11.4 GB/s | 8.8ms | 9.1ms |

**Read/1KB Throughput:**
```
  horse      |######################################## 6.2 GB/s (100%)
  local      |##################################...... 5.3 GB/s (86%)
  rabbit     |###############################......... 4.8 GB/s (78%)
  usagi      |######################.................. 3.5 GB/s (56%)
  devnull_s3 | 0.00 MB/s (0%)
```

**Read/64KB Throughput:**
```
  rabbit     |######################################## 50.7 GB/s (100%)
  horse      |#######################################. 50.0 GB/s (99%)
  local      |############............................ 15.8 GB/s (31%)
  usagi      |#########............................... 12.3 GB/s (24%)
  devnull_s3 | 0.00 MB/s (0%)
```

**Read/1MB Throughput:**
```
  horse      |######################################## 54.2 GB/s (100%)
  local      |############............................ 17.5 GB/s (32%)
  usagi      |###########............................. 15.7 GB/s (29%)
  rabbit     |#########............................... 13.1 GB/s (24%)
  devnull_s3 | 0.00 MB/s (0%)
```

**Read/10MB Throughput:**
```
  horse      |######################################## 15.4 GB/s (100%)
  usagi      |###############################......... 12.1 GB/s (79%)
  rabbit     |###########################............. 10.7 GB/s (70%)
  local      |##########################.............. 10.1 GB/s (66%)
  devnull_s3 | 0.00 MB/s (0%)
```

**Read/100MB Throughput:**
```
  usagi      |######################################## 11.4 GB/s (100%)
  local      |######################################.. 11.0 GB/s (97%)
  rabbit     |######################################.. 10.9 GB/s (96%)
  horse      |#########################............... 7.3 GB/s (64%)
  devnull_s3 | 0.00 MB/s (0%)
```

## Parallel Scalability

### Parallel Write Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| devnull_s3 | 9.8 MB/s | 4.2 MB/s | 2.1 MB/s | 1.3 MB/s | 0.72 MB/s | 0.43 MB/s | 0% |
| horse | 932.8 MB/s | 125.8 MB/s | 83.8 MB/s | 31.5 MB/s | 12.5 MB/s | 6.6 MB/s | 0% |
| local | 1.7 MB/s | 0.73 MB/s | 0.21 MB/s | 0.10 MB/s | 0.04 MB/s | 0.02 MB/s | 0% |
| rabbit | 20.3 MB/s | 2.6 MB/s | 1.3 MB/s | 0.78 MB/s | 0.43 MB/s | 0.05 MB/s | 0% |
| usagi | 247.5 MB/s | 53.4 MB/s | 18.6 MB/s | 12.3 MB/s | 6.1 MB/s | 3.5 MB/s | 0% |

> Scaling = (throughput at C200 / throughput at C1) / (200/1). 100% = perfect linear scaling.

### Parallel Read Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| devnull_s3 | 0.00 MB/s* | 0.00 MB/s* | 0.00 MB/s* | 0.00 MB/s* | 0.00 MB/s* | 0.00 MB/s* | - |
| horse | 4.5 GB/s | 3.2 GB/s | 3.4 GB/s | 3.1 GB/s | 2.9 GB/s | 2.5 GB/s | 0% |
| local | 1.4 GB/s | 843.3 MB/s | 713.7 MB/s | 639.1 MB/s | 559.5 MB/s | 534.5 MB/s | 0% |
| rabbit | 1.8 GB/s | 1.4 GB/s | 1.2 GB/s | 1.1 GB/s | 905.6 MB/s | 834.5 MB/s | 0% |
| usagi | 1.9 GB/s | 142.1 MB/s | 55.2 MB/s | 27.5 MB/s | 13.1 MB/s | 6.9 MB/s | 0% |

## Latency Analysis

### Latency Distribution by Operation

| Driver | Operation | Min | P50 | P95 | P99 | Max | Tail Ratio |
|--------|-----------|-----|-----|-----|-----|-----|------------|
| horse | Copy/1KB | 250ns | 958ns | 2.2us | 4.2us | 6.3ms | 4.4x |
| usagi | Copy/1KB | 1.8us | 2.5us | 4.1us | 9.2us | 19.3ms | 3.6x |
| devnull_s3 | Copy/1KB | 49.2us | 59.8us | 71.2us | 157.7us | 1.6ms | 2.6x |
| rabbit | Copy/1KB | 49.6us | 56.9us | 127.4us | 194.6us | 13.3ms | 3.4x |
| local | Copy/1KB | 225.2us | 448.2us | 540.0us | 665.2us | 1.7ms | 1.5x |
| horse | Delete | 83ns | 375ns | 583ns | 791ns | 767.7us | 2.1x |
| usagi | Delete | 1.2us | 1.5us | 1.8us | 2.2us | 36.2ms | 1.5x |
| rabbit | Delete | 20.8us | 35.1us | 45.9us | 53.0us | 1.1ms | 1.5x |
| local | Delete | 26.6us | 43.3us | 55.7us | 71.2us | 741.0us | 1.6x |
| devnull_s3 | Delete | 41.7us | 52.3us | 61.6us | 75.6us | 520.5us | 1.4x |
| horse | EdgeCase/DeepNested | 250ns | 542ns | 917ns | 8.8us | 293.7us | 16.3x (!) |
| usagi | EdgeCase/DeepNested | 1.5us | 2.1us | 2.9us | 8.9us | 6.55s | 4.2x |
| devnull_s3 | EdgeCase/DeepNested | 48.7us | 60.2us | 72.8us | 153.0us | 1.5ms | 2.5x |
| rabbit | EdgeCase/DeepNested | 39.2us | 75.9us | 118.0us | 138.3us | 11.1ms | 1.8x |
| local | EdgeCase/DeepNested | 255.0us | 377.2us | 630.9us | 997.0us | 5.7ms | 2.6x |
| horse | EdgeCase/EmptyObject | 208ns | 500ns | 917ns | 1.6us | 1.4ms | 3.2x |
| usagi | EdgeCase/EmptyObject | 1.3us | 1.6us | 2.2us | 4.8us | 3.7ms | 2.9x |
| rabbit | EdgeCase/EmptyObject | 27.8us | 47.6us | 108.0us | 122.1us | 3.5ms | 2.6x |
| devnull_s3 | EdgeCase/EmptyObject | 47.3us | 57.9us | 76.1us | 110.6us | 1.1ms | 1.9x |
| local | EdgeCase/EmptyObject | 228.2us | 329.0us | 653.7us | 1.0ms | 7.3ms | 3.1x |
| horse | EdgeCase/LongKey256 | 416ns | 792ns | 1.8us | 4.4us | 22.3ms | 5.5x |
| usagi | EdgeCase/LongKey256 | 1.8us | 2.3us | 3.8us | 7.0us | 24.0ms | 3.0x |
| devnull_s3 | EdgeCase/LongKey256 | 52.5us | 64.0us | 73.8us | 86.6us | 1.2ms | 1.4x |
| rabbit | EdgeCase/LongKey256 | 44.4us | 63.2us | 124.0us | 158.3us | 9.9ms | 2.5x |
| local | EdgeCase/LongKey256 | 258.2us | 348.7us | 455.1us | 557.9us | 755.3us | 1.6x |
| usagi | List/100 | 4.5us | 4.9us | 8.6us | 17.8us | 3.9ms | 3.7x |
| horse | List/100 | 5.8us | 7.2us | 13.4us | 28.0us | 958.2us | 3.9x |
| rabbit | List/100 | 169.9us | 181.9us | 197.3us | 226.7us | 684.3us | 1.2x |
| local | List/100 | 208.2us | 222.2us | 237.6us | 253.4us | 843.1us | 1.1x |
| devnull_s3 | List/100 | 639.5us | 710.9us | 770.5us | 825.7us | 1.5ms | 1.2x |
| horse | MixedWorkload/Balanced_50_50 | 1.5us | 6.3us | 368.0us | 1.6ms | 24.7ms | 250.2x (!) |
| usagi | MixedWorkload/Balanced_50_50 | 959ns | 201.0us | 3.6ms | 17.4ms | 42.3ms | 86.4x (!) |
| devnull_s3 | MixedWorkload/Balanced_50_50 | 106.7us | 1.5ms | 8.0ms | 18.9ms | 42.9ms | 12.3x (!) |
| rabbit | MixedWorkload/Balanced_50_50 | 709ns | 73.4us | 58.5ms | 94.2ms | 201.9ms | 1284.3x (!) |
| local | MixedWorkload/Balanced_50_50 | 1.2us | 85.8us | 90.5ms | 105.3ms | 115.6ms | 1226.5x (!) |
| horse | MixedWorkload/ReadHeavy_90_10 | 334ns | 6.2us | 88.0us | 315.3us | 48.1ms | 51.1x (!) |
| rabbit | MixedWorkload/ReadHeavy_90_10 | 625ns | 958ns | 380.1us | 6.5ms | 157.8ms | 6779.1x (!) |
| usagi | MixedWorkload/ReadHeavy_90_10 | 1.4us | 9.6us | 2.0ms | 7.5ms | 88.3ms | 782.7x (!) |
| devnull_s3 | MixedWorkload/ReadHeavy_90_10 | 117.9us | 1.4ms | 6.9ms | 12.9ms | 21.2ms | 8.9x |
| local | MixedWorkload/ReadHeavy_90_10 | 958ns | 2.5us | 57.9ms | 91.6ms | 125.5ms | 36053.2x (!) |
| horse | MixedWorkload/WriteHeavy_10_90 | 1.3us | 5.1us | 1.9ms | 35.5ms | 123.8ms | 6932.8x (!) |
| usagi | MixedWorkload/WriteHeavy_10_90 | 1.5us | 400.2us | 7.9ms | 23.7ms | 43.7ms | 59.3x (!) |
| devnull_s3 | MixedWorkload/WriteHeavy_10_90 | 105.6us | 1.7ms | 8.7ms | 15.3ms | 37.7ms | 8.9x |
| rabbit | MixedWorkload/WriteHeavy_10_90 | 1.1us | 390.6us | 69.3ms | 100.7ms | 159.3ms | 257.9x (!) |
| local | MixedWorkload/WriteHeavy_10_90 | 1.6us | 53.4ms | 96.7ms | 107.0ms | 127.2ms | 2.0x |
| devnull_s3 | Multipart/15MB_3Parts | 9.1ms | 11.4ms | 13.4ms | 13.8ms | 13.9ms | 1.2x |
| usagi | Multipart/15MB_3Parts | 8.4ms | 13.7ms | 43.6ms | 53.0ms | 57.7ms | 3.9x |
| local | Multipart/15MB_3Parts | 25.2ms | 28.1ms | 33.1ms | 43.6ms | 44.1ms | 1.6x |
| horse | Multipart/15MB_3Parts | 24.7ms | 28.7ms | 42.1ms | 44.5ms | 78.3ms | 1.6x |
| rabbit | Multipart/15MB_3Parts | 28.1ms | 32.5ms | 36.3ms | 38.4ms | 44.3ms | 1.2x |
| horse | ParallelRead/1KB/C1 | 42ns | 208ns | 375ns | 584ns | 420.5us | 2.8x |
| usagi | ParallelRead/1KB/C1 | 125ns | 292ns | 1.3us | 1.8us | 1.6ms | 6.3x |
| rabbit | ParallelRead/1KB/C1 | 375ns | 500ns | 708ns | 1.0us | 75.5us | 2.1x |
| local | ParallelRead/1KB/C1 | 458ns | 666ns | 958ns | 1.6us | 168.1us | 2.4x |
| devnull_s3 | ParallelRead/1KB/C1 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | ParallelRead/1KB/C10 | 83ns | 208ns | 625ns | 1.5us | 1.7ms | 7.4x |
| rabbit | ParallelRead/1KB/C10 | 375ns | 542ns | 1.2us | 3.2us | 645.7us | 6.0x |
| local | ParallelRead/1KB/C10 | 500ns | 792ns | 2.2us | 8.2us | 753.3us | 10.4x (!) |
| usagi | ParallelRead/1KB/C10 | 166ns | 583ns | 35.8us | 67.8us | 8.4ms | 116.3x (!) |
| devnull_s3 | ParallelRead/1KB/C10 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | ParallelRead/1KB/C100 | 41ns | 208ns | 666ns | 1.7us | 2.4ms | 8.0x |
| rabbit | ParallelRead/1KB/C100 | 375ns | 583ns | 2.0us | 13.5us | 1.2ms | 23.1x (!) |
| local | ParallelRead/1KB/C100 | 458ns | 917ns | 3.3us | 16.8us | 1.1ms | 18.3x (!) |
| usagi | ParallelRead/1KB/C100 | 208ns | 625ns | 406.2us | 789.0us | 23.7ms | 1262.5x (!) |
| devnull_s3 | ParallelRead/1KB/C100 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | ParallelRead/1KB/C200 | 42ns | 208ns | 709ns | 2.6us | 1.3ms | 12.4x (!) |
| rabbit | ParallelRead/1KB/C200 | 416ns | 583ns | 2.1us | 14.4us | 2.3ms | 24.7x (!) |
| local | ParallelRead/1KB/C200 | 458ns | 958ns | 3.5us | 18.4us | 774.9us | 19.2x (!) |
| usagi | ParallelRead/1KB/C200 | 208ns | 667ns | 808.6us | 1.5ms | 11.1ms | 2205.5x (!) |
| devnull_s3 | ParallelRead/1KB/C200 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | ParallelRead/1KB/C25 | 83ns | 208ns | 583ns | 1.5us | 868.0us | 7.0x |
| rabbit | ParallelRead/1KB/C25 | 375ns | 542ns | 1.4us | 3.6us | 1.5ms | 6.7x |
| local | ParallelRead/1KB/C25 | 458ns | 834ns | 2.4us | 11.0us | 1.9ms | 13.1x (!) |
| usagi | ParallelRead/1KB/C25 | 167ns | 583ns | 93.4us | 181.4us | 14.4ms | 311.1x (!) |
| devnull_s3 | ParallelRead/1KB/C25 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | ParallelRead/1KB/C50 | 83ns | 208ns | 625ns | 1.6us | 834.3us | 7.6x |
| rabbit | ParallelRead/1KB/C50 | 375ns | 542ns | 1.7us | 9.5us | 3.1ms | 17.6x (!) |
| local | ParallelRead/1KB/C50 | 458ns | 875ns | 3.2us | 14.7us | 2.6ms | 16.8x (!) |
| usagi | ParallelRead/1KB/C50 | 166ns | 625ns | 188.6us | 368.8us | 16.2ms | 590.0x (!) |
| devnull_s3 | ParallelRead/1KB/C50 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | ParallelWrite/1KB/C1 | 458ns | 916ns | 1.5us | 2.3us | 2.4ms | 2.5x |
| usagi | ParallelWrite/1KB/C1 | 2.1us | 3.2us | 5.6us | 8.0us | 7.9ms | 2.4x |
| rabbit | ParallelWrite/1KB/C1 | 39.3us | 45.0us | 55.0us | 61.4us | 7.3ms | 1.4x |
| devnull_s3 | ParallelWrite/1KB/C1 | 84.2us | 97.2us | 110.8us | 153.9us | 682.2us | 1.6x |
| local | ParallelWrite/1KB/C1 | 223.7us | 280.2us | 2.5ms | 3.5ms | 23.8ms | 12.4x (!) |
| horse | ParallelWrite/1KB/C10 | 667ns | 1.8us | 23.2us | 62.1us | 61.7ms | 35.5x (!) |
| usagi | ParallelWrite/1KB/C10 | 2.5us | 9.2us | 48.9us | 86.2us | 7.2ms | 9.4x |
| devnull_s3 | ParallelWrite/1KB/C10 | 83.4us | 213.1us | 391.7us | 624.6us | 6.2ms | 2.9x |
| rabbit | ParallelWrite/1KB/C10 | 48.8us | 210.6us | 851.8us | 2.6ms | 30.9ms | 12.2x (!) |
| local | ParallelWrite/1KB/C10 | 298.6us | 1.3ms | 2.2ms | 2.7ms | 4.1ms | 2.1x |
| horse | ParallelWrite/1KB/C100 | 708ns | 2.8us | 135.2us | 301.3us | 62.1ms | 108.0x (!) |
| usagi | ParallelWrite/1KB/C100 | 2.3us | 27.7us | 578.2us | 1.4ms | 24.4ms | 50.2x (!) |
| devnull_s3 | ParallelWrite/1KB/C100 | 95.5us | 929.8us | 3.9ms | 6.4ms | 13.5ms | 6.8x |
| rabbit | ParallelWrite/1KB/C100 | 48.9us | 244.8us | 17.4ms | 36.6ms | 66.7ms | 149.4x (!) |
| local | ParallelWrite/1KB/C100 | 435.5us | 18.4ms | 29.8ms | 195.9ms | 209.1ms | 10.6x (!) |
| horse | ParallelWrite/1KB/C200 | 708ns | 3.0us | 361.3us | 767.1us | 81.6ms | 252.2x (!) |
| usagi | ParallelWrite/1KB/C200 | 2.4us | 33.0us | 1.2ms | 2.5ms | 22.1ms | 77.1x (!) |
| devnull_s3 | ParallelWrite/1KB/C200 | 87.1us | 1.5ms | 6.8ms | 12.2ms | 36.2ms | 7.9x |
| rabbit | ParallelWrite/1KB/C200 | 53.6us | 525.3us | 95.5ms | 133.0ms | 177.4ms | 253.3x (!) |
| local | ParallelWrite/1KB/C200 | 471.0us | 41.4ms | 77.2ms | 91.2ms | 104.7ms | 2.2x |
| horse | ParallelWrite/1KB/C25 | 708ns | 2.0us | 48.3us | 96.2us | 9.3ms | 49.1x (!) |
| usagi | ParallelWrite/1KB/C25 | 2.5us | 22.0us | 148.0us | 298.2us | 14.1ms | 13.6x (!) |
| devnull_s3 | ParallelWrite/1KB/C25 | 86.8us | 376.1us | 1.0ms | 1.6ms | 15.7ms | 4.3x |
| rabbit | ParallelWrite/1KB/C25 | 51.7us | 265.2us | 2.0ms | 15.5ms | 28.3ms | 58.5x (!) |
| local | ParallelWrite/1KB/C25 | 302.8us | 4.5ms | 7.4ms | 9.1ms | 15.6ms | 2.0x |
| horse | ParallelWrite/1KB/C50 | 667ns | 2.3us | 126.0us | 273.6us | 24.5ms | 119.4x (!) |
| usagi | ParallelWrite/1KB/C50 | 2.6us | 21.1us | 256.2us | 580.4us | 15.0ms | 27.5x (!) |
| devnull_s3 | ParallelWrite/1KB/C50 | 88.8us | 568.3us | 2.1ms | 3.2ms | 9.8ms | 5.6x |
| rabbit | ParallelWrite/1KB/C50 | 50.4us | 274.8us | 6.4ms | 18.3ms | 46.4ms | 66.5x (!) |
| local | ParallelWrite/1KB/C50 | 453.7us | 9.7ms | 15.7ms | 18.4ms | 23.3ms | 1.9x |
| horse | RangeRead/End_256KB | 4.1us | 4.5us | 5.5us | 5.9us | 18.9us | 1.3x |
| usagi | RangeRead/End_256KB | 14.5us | 15.2us | 18.0us | 19.8us | 102.7us | 1.3x |
| local | RangeRead/End_256KB | 19.7us | 21.4us | 25.1us | 28.0us | 177.5us | 1.3x |
| rabbit | RangeRead/End_256KB | 23.5us | 24.7us | 28.8us | 31.2us | 146.5us | 1.3x |
| devnull_s3 | RangeRead/End_256KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | RangeRead/Middle_256KB | 3.5us | 4.3us | 5.3us | 5.7us | 67.9us | 1.3x |
| usagi | RangeRead/Middle_256KB | 14.5us | 15.3us | 17.8us | 19.3us | 129.1us | 1.3x |
| local | RangeRead/Middle_256KB | 19.5us | 21.3us | 25.0us | 28.0us | 294.8us | 1.3x |
| rabbit | RangeRead/Middle_256KB | 23.6us | 24.8us | 28.8us | 31.6us | 263.0us | 1.3x |
| devnull_s3 | RangeRead/Middle_256KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | RangeRead/Start_256KB | 3.6us | 4.5us | 5.5us | 5.8us | 96.4us | 1.3x |
| usagi | RangeRead/Start_256KB | 14.4us | 15.5us | 17.8us | 20.0us | 213.8us | 1.3x |
| local | RangeRead/Start_256KB | 19.3us | 21.6us | 24.8us | 28.4us | 169.7us | 1.3x |
| rabbit | RangeRead/Start_256KB | 23.3us | 24.5us | 28.5us | 31.9us | 192.8us | 1.3x |
| devnull_s3 | RangeRead/Start_256KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| usagi | Read/100MB | 8.5ms | 8.8ms | 9.1ms | 9.1ms | 9.4ms | 1.0x |
| local | Read/100MB | 7.6ms | 8.6ms | 11.3ms | 12.6ms | 13.5ms | 1.5x |
| rabbit | Read/100MB | 8.8ms | 9.1ms | 9.5ms | 10.8ms | 11.8ms | 1.2x |
| horse | Read/100MB | 10.3ms | 11.3ms | 24.2ms | 25.9ms | 33.3ms | 2.3x |
| devnull_s3 | Read/100MB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | Read/10MB | 176.5us | 227.0us | 1.7ms | 3.5ms | 7.8ms | 15.3x (!) |
| usagi | Read/10MB | 711.0us | 813.2us | 891.8us | 1.0ms | 1.8ms | 1.3x |
| rabbit | Read/10MB | 768.3us | 919.0us | 1.0ms | 1.2ms | 1.7ms | 1.3x |
| local | Read/10MB | 768.8us | 961.1us | 1.1ms | 1.5ms | 1.7ms | 1.5x |
| devnull_s3 | Read/10MB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | Read/1KB | 41ns | 125ns | 208ns | 458ns | 17.0ms | 3.7x |
| local | Read/1KB | 83ns | 167ns | 250ns | 417ns | 340.7us | 2.5x |
| rabbit | Read/1KB | 83ns | 167ns | 292ns | 458ns | 435.0us | 2.7x |
| usagi | Read/1KB | 125ns | 209ns | 541ns | 1.1us | 3.8ms | 5.2x |
| devnull_s3 | Read/1KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | Read/1MB | 16.4us | 18.0us | 21.4us | 25.3us | 51.6us | 1.4x |
| local | Read/1MB | 50.0us | 55.8us | 65.8us | 79.8us | 206.4us | 1.4x |
| usagi | Read/1MB | 58.9us | 62.9us | 70.3us | 76.0us | 149.7us | 1.2x |
| rabbit | Read/1MB | 69.9us | 75.1us | 85.2us | 98.0us | 254.0us | 1.3x |
| devnull_s3 | Read/1MB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| rabbit | Read/64KB | 1.1us | 1.2us | 1.5us | 1.6us | 13.9us | 1.4x |
| horse | Read/64KB | 958ns | 1.2us | 1.5us | 3.0us | 122.8us | 2.5x |
| local | Read/64KB | 2.7us | 3.5us | 4.6us | 10.7us | 712.0us | 3.0x |
| usagi | Read/64KB | 2.6us | 3.6us | 9.2us | 34.8us | 816.2us | 9.7x |
| devnull_s3 | Read/64KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | Scale/Delete/10 | 5.5us | 5.5us | 5.5us | 5.5us | 5.5us | 1.0x |
| usagi | Scale/Delete/10 | 68.3us | 68.3us | 68.3us | 68.3us | 68.3us | 1.0x |
| rabbit | Scale/Delete/10 | 360.6us | 360.6us | 360.6us | 360.6us | 360.6us | 1.0x |
| local | Scale/Delete/10 | 504.7us | 504.7us | 504.7us | 504.7us | 504.7us | 1.0x |
| devnull_s3 | Scale/Delete/10 | 660.2us | 660.2us | 660.2us | 660.2us | 660.2us | 1.0x |
| horse | Scale/Delete/100 | 28.0us | 28.0us | 28.0us | 28.0us | 28.0us | 1.0x |
| usagi | Scale/Delete/100 | 250.9us | 250.9us | 250.9us | 250.9us | 250.9us | 1.0x |
| rabbit | Scale/Delete/100 | 3.8ms | 3.8ms | 3.8ms | 3.8ms | 3.8ms | 1.0x |
| local | Scale/Delete/100 | 5.0ms | 5.0ms | 5.0ms | 5.0ms | 5.0ms | 1.0x |
| devnull_s3 | Scale/Delete/100 | 6.6ms | 6.6ms | 6.6ms | 6.6ms | 6.6ms | 1.0x |
| horse | Scale/Delete/1000 | 279.0us | 279.0us | 279.0us | 279.0us | 279.0us | 1.0x |
| usagi | Scale/Delete/1000 | 2.2ms | 2.2ms | 2.2ms | 2.2ms | 2.2ms | 1.0x |
| rabbit | Scale/Delete/1000 | 39.1ms | 39.1ms | 39.1ms | 39.1ms | 39.1ms | 1.0x |
| local | Scale/Delete/1000 | 52.3ms | 52.3ms | 52.3ms | 52.3ms | 52.3ms | 1.0x |
| devnull_s3 | Scale/Delete/1000 | 52.9ms | 52.9ms | 52.9ms | 52.9ms | 52.9ms | 1.0x |
| horse | Scale/Delete/10000 | 3.6ms | 3.6ms | 3.6ms | 3.6ms | 3.6ms | 1.0x |
| usagi | Scale/Delete/10000 | 17.3ms | 17.3ms | 17.3ms | 17.3ms | 17.3ms | 1.0x |
| rabbit | Scale/Delete/10000 | 459.1ms | 459.1ms | 459.1ms | 459.1ms | 459.1ms | 1.0x |
| devnull_s3 | Scale/Delete/10000 | 528.9ms | 528.9ms | 528.9ms | 528.9ms | 528.9ms | 1.0x |
| local | Scale/Delete/10000 | 619.4ms | 619.4ms | 619.4ms | 619.4ms | 619.4ms | 1.0x |
| horse | Scale/List/10 | 5.9us | 5.9us | 5.9us | 5.9us | 5.9us | 1.0x |
| rabbit | Scale/List/10 | 76.4us | 76.4us | 76.4us | 76.4us | 76.4us | 1.0x |
| local | Scale/List/10 | 96.3us | 96.3us | 96.3us | 96.3us | 96.3us | 1.0x |
| usagi | Scale/List/10 | 798.4ms | 798.4ms | 798.4ms | 798.4ms | 798.4ms | 1.0x |
| devnull_s3 | Scale/List/10 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | Scale/List/100 | 15.2us | 15.2us | 15.2us | 15.2us | 15.2us | 1.0x |
| rabbit | Scale/List/100 | 276.0us | 276.0us | 276.0us | 276.0us | 276.0us | 1.0x |
| local | Scale/List/100 | 346.2us | 346.2us | 346.2us | 346.2us | 346.2us | 1.0x |
| usagi | Scale/List/100 | 809.8ms | 809.8ms | 809.8ms | 809.8ms | 809.8ms | 1.0x |
| devnull_s3 | Scale/List/100 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | Scale/List/1000 | 206.8us | 206.8us | 206.8us | 206.8us | 206.8us | 1.0x |
| rabbit | Scale/List/1000 | 2.7ms | 2.7ms | 2.7ms | 2.7ms | 2.7ms | 1.0x |
| local | Scale/List/1000 | 3.5ms | 3.5ms | 3.5ms | 3.5ms | 3.5ms | 1.0x |
| usagi | Scale/List/1000 | 819.4ms | 819.4ms | 819.4ms | 819.4ms | 819.4ms | 1.0x |
| devnull_s3 | Scale/List/1000 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | Scale/List/10000 | 2.5ms | 2.5ms | 2.5ms | 2.5ms | 2.5ms | 1.0x |
| rabbit | Scale/List/10000 | 28.5ms | 28.5ms | 28.5ms | 28.5ms | 28.5ms | 1.0x |
| local | Scale/List/10000 | 36.6ms | 36.6ms | 36.6ms | 36.6ms | 36.6ms | 1.0x |
| usagi | Scale/List/10000 | 800.5ms | 800.5ms | 800.5ms | 800.5ms | 800.5ms | 1.0x |
| devnull_s3 | Scale/List/10000 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| horse | Scale/Write/10 | 25.6us | 25.6us | 25.6us | 25.6us | 25.6us | 1.0x |
| usagi | Scale/Write/10 | 69.3us | 69.3us | 69.3us | 69.3us | 69.3us | 1.0x |
| devnull_s3 | Scale/Write/10 | 662.8us | 662.8us | 662.8us | 662.8us | 662.8us | 1.0x |
| rabbit | Scale/Write/10 | 1.0ms | 1.0ms | 1.0ms | 1.0ms | 1.0ms | 1.0x |
| local | Scale/Write/10 | 3.8ms | 3.8ms | 3.8ms | 3.8ms | 3.8ms | 1.0x |
| horse | Scale/Write/100 | 78.9us | 78.9us | 78.9us | 78.9us | 78.9us | 1.0x |
| usagi | Scale/Write/100 | 443.3us | 443.3us | 443.3us | 443.3us | 443.3us | 1.0x |
| devnull_s3 | Scale/Write/100 | 6.8ms | 6.8ms | 6.8ms | 6.8ms | 6.8ms | 1.0x |
| rabbit | Scale/Write/100 | 8.5ms | 8.5ms | 8.5ms | 8.5ms | 8.5ms | 1.0x |
| local | Scale/Write/100 | 36.9ms | 36.9ms | 36.9ms | 36.9ms | 36.9ms | 1.0x |
| horse | Scale/Write/1000 | 914.3us | 914.3us | 914.3us | 914.3us | 914.3us | 1.0x |
| usagi | Scale/Write/1000 | 3.4ms | 3.4ms | 3.4ms | 3.4ms | 3.4ms | 1.0x |
| devnull_s3 | Scale/Write/1000 | 61.3ms | 61.3ms | 61.3ms | 61.3ms | 61.3ms | 1.0x |
| rabbit | Scale/Write/1000 | 76.0ms | 76.0ms | 76.0ms | 76.0ms | 76.0ms | 1.0x |
| local | Scale/Write/1000 | 408.3ms | 408.3ms | 408.3ms | 408.3ms | 408.3ms | 1.0x |
| horse | Scale/Write/10000 | 7.9ms | 7.9ms | 7.9ms | 7.9ms | 7.9ms | 1.0x |
| usagi | Scale/Write/10000 | 42.2ms | 42.2ms | 42.2ms | 42.2ms | 42.2ms | 1.0x |
| devnull_s3 | Scale/Write/10000 | 631.4ms | 631.4ms | 631.4ms | 631.4ms | 631.4ms | 1.0x |
| rabbit | Scale/Write/10000 | 776.8ms | 776.8ms | 776.8ms | 776.8ms | 776.8ms | 1.0x |
| local | Scale/Write/10000 | 4.59s | 4.59s | 4.59s | 4.59s | 4.59s | 1.0x |
| usagi | Stat | 0ns | 42ns | 84ns | 209ns | 3.2ms | 5.0x |
| horse | Stat | 0ns | 42ns | 125ns | 250ns | 3.0ms | 6.0x |
| local | Stat | 41ns | 125ns | 292ns | 667ns | 424.2us | 5.3x |
| rabbit | Stat | 83ns | 167ns | 333ns | 666ns | 442.2us | 4.0x |
| devnull_s3 | Stat | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| usagi | Write/100MB | 48.1ms | 53.3ms | 57.3ms | 57.3ms | 59.1ms | 1.1x |
| horse | Write/100MB | 19.9ms | 56.8ms | 69.6ms | 69.6ms | 176.2ms | 1.2x |
| local | Write/100MB | 36.9ms | 45.5ms | 99.7ms | 99.7ms | 111.8ms | 2.2x |
| rabbit | Write/100MB | 54.1ms | 58.7ms | 67.6ms | 67.6ms | 92.5ms | 1.2x |
| devnull_s3 | Write/100MB | 62.2ms | 65.4ms | 65.9ms | 65.9ms | 66.6ms | 1.0x |
| horse | Write/10MB | 188.0us | 253.6us | 20.2ms | 46.7ms | 126.0ms | 184.3x (!) |
| local | Write/10MB | 2.9ms | 3.6ms | 15.4ms | 16.4ms | 17.0ms | 4.5x |
| rabbit | Write/10MB | 2.1ms | 6.3ms | 7.8ms | 9.1ms | 68.7ms | 1.5x |
| usagi | Write/10MB | 2.1ms | 4.4ms | 14.2ms | 41.8ms | 46.1ms | 9.5x |
| devnull_s3 | Write/10MB | 6.8ms | 8.0ms | 8.8ms | 8.9ms | 8.9ms | 1.1x |
| horse | Write/1KB | 166ns | 500ns | 958ns | 1.7us | 4.7ms | 3.4x |
| usagi | Write/1KB | 1.5us | 2.3us | 4.9us | 7.7us | 17.7ms | 3.3x |
| rabbit | Write/1KB | 37.7us | 43.2us | 59.0us | 152.3us | 9.2ms | 3.5x |
| devnull_s3 | Write/1KB | 47.5us | 63.8us | 103.2us | 292.9us | 1.0ms | 4.6x |
| local | Write/1KB | 228.1us | 309.1us | 1.3ms | 1.7ms | 4.5ms | 5.5x |
| horse | Write/1MB | 18.1us | 21.1us | 26.9us | 14.5ms | 91.7ms | 688.9x (!) |
| rabbit | Write/1MB | 170.7us | 339.7us | 1.4ms | 2.4ms | 18.8ms | 7.1x |
| usagi | Write/1MB | 157.9us | 186.5us | 951.5us | 3.3ms | 44.7ms | 17.6x (!) |
| local | Write/1MB | 538.8us | 670.2us | 1.0ms | 1.2ms | 1.4ms | 1.8x |
| devnull_s3 | Write/1MB | 572.9us | 667.5us | 1.0ms | 1.1ms | 1.4ms | 1.6x |
| horse | Write/64KB | 1.0us | 2.0us | 3.3us | 6.3us | 101.2ms | 3.1x |
| usagi | Write/64KB | 14.7us | 18.8us | 31.5us | 131.7us | 34.8ms | 7.0x |
| rabbit | Write/64KB | 49.3us | 58.8us | 74.7us | 265.5us | 26.4ms | 4.5x |
| devnull_s3 | Write/64KB | 80.5us | 97.8us | 118.1us | 331.7us | 928.2us | 3.4x |
| local | Write/64KB | 257.7us | 317.8us | 386.0us | 437.2us | 683.2us | 1.4x |

### Tail Latency Warnings

> Drivers with P99/P50 ratio > 10x indicate significant tail latency.

- **horse** on EdgeCase/DeepNested: P99 is 16x the P50 latency
- **horse** on MixedWorkload/Balanced_50_50: P99 is 250x the P50 latency
- **usagi** on MixedWorkload/Balanced_50_50: P99 is 86x the P50 latency
- **devnull_s3** on MixedWorkload/Balanced_50_50: P99 is 12x the P50 latency
- **rabbit** on MixedWorkload/Balanced_50_50: P99 is 1284x the P50 latency
- **local** on MixedWorkload/Balanced_50_50: P99 is 1227x the P50 latency
- **horse** on MixedWorkload/ReadHeavy_90_10: P99 is 51x the P50 latency
- **rabbit** on MixedWorkload/ReadHeavy_90_10: P99 is 6779x the P50 latency
- **usagi** on MixedWorkload/ReadHeavy_90_10: P99 is 783x the P50 latency
- **local** on MixedWorkload/ReadHeavy_90_10: P99 is 36053x the P50 latency
- **horse** on MixedWorkload/WriteHeavy_10_90: P99 is 6933x the P50 latency
- **usagi** on MixedWorkload/WriteHeavy_10_90: P99 is 59x the P50 latency
- **rabbit** on MixedWorkload/WriteHeavy_10_90: P99 is 258x the P50 latency
- **local** on ParallelRead/1KB/C10: P99 is 10x the P50 latency
- **usagi** on ParallelRead/1KB/C10: P99 is 116x the P50 latency
- **rabbit** on ParallelRead/1KB/C100: P99 is 23x the P50 latency
- **local** on ParallelRead/1KB/C100: P99 is 18x the P50 latency
- **usagi** on ParallelRead/1KB/C100: P99 is 1262x the P50 latency
- **horse** on ParallelRead/1KB/C200: P99 is 12x the P50 latency
- **rabbit** on ParallelRead/1KB/C200: P99 is 25x the P50 latency
- **local** on ParallelRead/1KB/C200: P99 is 19x the P50 latency
- **usagi** on ParallelRead/1KB/C200: P99 is 2205x the P50 latency
- **local** on ParallelRead/1KB/C25: P99 is 13x the P50 latency
- **usagi** on ParallelRead/1KB/C25: P99 is 311x the P50 latency
- **rabbit** on ParallelRead/1KB/C50: P99 is 18x the P50 latency
- **local** on ParallelRead/1KB/C50: P99 is 17x the P50 latency
- **usagi** on ParallelRead/1KB/C50: P99 is 590x the P50 latency
- **local** on ParallelWrite/1KB/C1: P99 is 12x the P50 latency
- **horse** on ParallelWrite/1KB/C10: P99 is 36x the P50 latency
- **rabbit** on ParallelWrite/1KB/C10: P99 is 12x the P50 latency
- **horse** on ParallelWrite/1KB/C100: P99 is 108x the P50 latency
- **usagi** on ParallelWrite/1KB/C100: P99 is 50x the P50 latency
- **rabbit** on ParallelWrite/1KB/C100: P99 is 149x the P50 latency
- **local** on ParallelWrite/1KB/C100: P99 is 11x the P50 latency
- **horse** on ParallelWrite/1KB/C200: P99 is 252x the P50 latency
- **usagi** on ParallelWrite/1KB/C200: P99 is 77x the P50 latency
- **rabbit** on ParallelWrite/1KB/C200: P99 is 253x the P50 latency
- **horse** on ParallelWrite/1KB/C25: P99 is 49x the P50 latency
- **usagi** on ParallelWrite/1KB/C25: P99 is 14x the P50 latency
- **rabbit** on ParallelWrite/1KB/C25: P99 is 59x the P50 latency
- **horse** on ParallelWrite/1KB/C50: P99 is 119x the P50 latency
- **usagi** on ParallelWrite/1KB/C50: P99 is 28x the P50 latency
- **rabbit** on ParallelWrite/1KB/C50: P99 is 66x the P50 latency
- **horse** on Read/10MB: P99 is 15x the P50 latency
- **horse** on Write/10MB: P99 is 184x the P50 latency
- **horse** on Write/1MB: P99 is 689x the P50 latency
- **usagi** on Write/1MB: P99 is 18x the P50 latency

## Resource Efficiency

### Runtime Resources

| Driver | Peak RSS | Go Heap | Disk Used | GC Cycles | Total Ops | Efficiency |
|--------|----------|---------|-----------|-----------|-----------|------------|
| horse | 6.5 GB | 3.0 GB | 32.0 GB | 1502 | 32.2M/s | 4.8K ops/MB |
| local | 6.5 GB | 1.3 GB | 5.9 GB | 1432 | 13.5M/s | 2.0K ops/MB |
| rabbit | 2.0 GB | 1.8 GB | 5.4 GB | 997 | 13.8M/s | 6.8K ops/MB |
| usagi | 6.5 GB | 5.3 GB | 12.0 GB | 1295 | 20.2M/s | 3.0K ops/MB |

> Peak RSS = process-level resident memory. Go Heap = Go runtime heap. Efficiency = total iterations / peak RSS.

**Memory Usage (Peak RSS, ascending):**
```
  rabbit |############............................ 2.0 GB
  usagi  |#######################################. 6.5 GB
  horse  |######################################## 6.5 GB
  local  |######################################## 6.5 GB
```

## Error & Timeout Summary

### Errors

| Driver | Total Errors | Affected Operations | Last Error |
|--------|-------------|--------------------|-----------|
| devnull_s3 | 319223 | Read/1KB (9159), Read/64KB (9605), Read/1MB (9803), Read/10MB (1111), Read/100MB... | listed 0, expected 10000 |

## Recommendations

### Best for Write-Heavy Workloads

> **horse** -- highest average write throughput across all object sizes.

| Rank | Driver | Avg Write Throughput |
|------|--------|---------------------|
| 1 | horse | 1.8 GB/s |
| 2 | usagi | 1.4 GB/s |
| 3 | rabbit | 1.2 GB/s |
| 4 | local | 977.5 MB/s |
| 5 | devnull_s3 | 949.7 MB/s |

### Best for Read-Heavy Workloads

> **horse** -- highest average read throughput across all object sizes.

| Rank | Driver | Avg Read Throughput |
|------|--------|--------------------|
| 1 | horse | 26.6 GB/s |
| 2 | rabbit | 18.0 GB/s |
| 3 | local | 12.0 GB/s |
| 4 | usagi | 11.0 GB/s |
| 5 | devnull_s3 | 0.00 MB/s |

### Most Memory Efficient

> **rabbit** -- lowest peak RSS at 2.0 GB.

### Best Overall

> **horse** -- won 42/48 competitive benchmarks.

| Rank | Driver | Wins |
|------|--------|------|
| 1 | horse | 42 |
| 2 | usagi | 4 |
| 3 | devnull_s3 | 1 |
| 4 | rabbit | 1 |


---

*Generated by storage benchmark CLI*
