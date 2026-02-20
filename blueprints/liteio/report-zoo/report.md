# Storage Benchmark Report

## System Info

| Property | Value |
|----------|-------|
| Timestamp | 2026-02-21T00:40:01+07:00 |
| Go Version | go1.26.0 |
| Platform | darwin/arm64 |
| CPUs | 10 |
| BenchTime | 500ms |
| Concurrency | 200 |
| Levels | [1 10 25 50 100 200] |
| Object Sizes | 1KB, 64KB, 1MB, 10MB, 100MB |
| Warmup | 10 iterations |
| Timeout | 30s |
| Drivers | 9 |

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
| 1 | **falcon** | ***** | **    | 20/48 | 10.8 GB | 26.1 GB | OK |
| 2 | **kangaroo** | *     | ***** | 3/48 | 10.8 GB | 576.0 MB | OK |
| 3 | **jaguar** | *     | ****  | 11/48 | 10.8 GB | 9.0 GB | 789 err |
| 4 | **gecko** | *     | *     | 0/48 | 10.8 GB | 7.2 GB | OK |
| 5 | **spider** | *     | *     | 0/48 | 10.8 GB | 1.9 GB | 3384 err |
| 6 | **narwhal** | *     | *     | 0/48 | 10.8 GB | 4.9 GB | OK |
| 7 | **ant** | *     | *     | 5/48 | 10.8 GB | 6.3 GB | OK |
| 8 | **owl** | *     |       | 9/48 | 10.8 GB | 6.8 GB | 6538340 err |
| 9 | **fox** | *     | *     | 0/48 | 10.8 GB | 673.6 MB | 1840209 err |

> **Overall Leader: falcon** -- won 20/48 benchmarks with combined write+read score of 65%.

## Performance Matrix

All drivers x key operations. Values show throughput (ops/s or MB/s). **Bold** = best in column.

| Driver | W/1KB | W/64KB | W/1MB | W/10MB | W/100MB | R/1KB | R/64KB | R/1MB | R/10MB | R/100MB | Stat | Delete | Copy/1KB | List/100 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| ant | 270.9 MB/s | 363.8 MB/s | 492.8 MB/s | 144.7 MB/s | 286.9 MB/s | 887.1 MB/s | 2.7 GB/s | 2.2 GB/s | 3.0 GB/s | 3.6 GB/s | 852.6K/s | 659.3K/s | 114.8 MB/s | **54.9K/s** |
| falcon | 1.7 GB/s | **10.0 GB/s** | **8.4 GB/s** | **8.2 GB/s** | **8.9 GB/s** | 2.9 GB/s | 8.3 GB/s | 5.4 GB/s | 4.8 GB/s | 4.4 GB/s | **10.8M/s** | 406.6K/s | **607.7 MB/s** | 2/s |
| fox | 145.5 MB/s | 1.0 GB/s | 2.2 GB/s | 2.0 GB/s | 1.6 GB/s | 329.2 MB/s | 0.00 MB/s | 0.00 MB/s | 0.00 MB/s | 0.00 MB/s | 7.4M/s | 98.3K/s | 30.9 MB/s | 13/s |
| gecko | 396.9 MB/s | 550.1 MB/s | 240.9 MB/s | 595.6 MB/s | 494.1 MB/s | 1.6 GB/s | 1.9 GB/s | 3.4 GB/s | 4.8 GB/s | 4.7 GB/s | 5.0M/s | 1.8M/s | 287.5 MB/s | 93/s |
| jaguar | 34.6 MB/s | 19.1 MB/s | 14.4 MB/s | 34.8 MB/s | 153.1 MB/s | **4.3 GB/s** | **49.0 GB/s** | 289.1 MB/s | 281.2 MB/s | 0.00 MB/s | 3.9M/s | 2.4K/s | 1.7 MB/s | 5/s |
| kangaroo | 48.1 MB/s | 134.8 MB/s | 1.5 GB/s | 3.3 GB/s | 2.4 GB/s | 3.6 GB/s | 47.1 GB/s | **58.5 GB/s** | **54.9 GB/s** | **54.2 GB/s** | 8.3M/s | - | - | 6/s |
| narwhal | 85.8 MB/s | 611.5 MB/s | 307.5 MB/s | 343.3 MB/s | 495.0 MB/s | 894.6 MB/s | 3.6 GB/s | 3.0 GB/s | 2.3 GB/s | 4.0 GB/s | 5.0M/s | 638.1K/s | 34.9 MB/s | 1.4K/s |
| owl | **2.5 GB/s** | 0.00 MB/s | 2.7 GB/s | 0.00 MB/s | 0.00 MB/s | - | - | - | - | - | 4.1M/s | **3.5M/s** | - | 15.0K/s |
| spider | 9.9 MB/s | 14.0 MB/s | 9.9 MB/s | 22.0 MB/s | 82.9 MB/s | 2.5 GB/s | 23.8 GB/s | 320.8 MB/s | 1.5 GB/s | 1.3 GB/s | 6.2M/s | 303/s | - | 1/s |

## Write Performance Deep Dive

### Write Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| ant | 277.4K/s | 270.9 MB/s | 3.0us | 14.3us | 5.8K/s | 363.8 MB/s | 21.0us | 725.9us | 493/s | 492.8 MB/s | 309.2us | 30.8ms | 14/s | 144.7 MB/s | 27.3ms | 232.3ms | 3/s | 286.9 MB/s | 195.6ms | 311.8ms |
| falcon | 1.7M/s | 1.7 GB/s | 292ns | 2.3us | 160.2K/s | 10.0 GB/s | 4.7us | 27.4us | 8.4K/s | 8.4 GB/s | 73.4us | 1.8ms | 822/s | 8.2 GB/s | 777.9us | 2.9ms | 89/s | 8.9 GB/s | 11.2ms | 15.4ms |
| fox | 149.0K/s | 145.5 MB/s | 750ns | 44.2us | 16.7K/s | 1.0 GB/s | 52.2us | 162.4us | 2.2K/s | 2.2 GB/s | 285.6us | 2.7ms | 198/s | 2.0 GB/s | 3.2ms | 5.0ms | 16/s | 1.6 GB/s | 58.4ms | 73.6ms |
| gecko | 406.5K/s | 396.9 MB/s | 500ns | 7.7us | 8.8K/s | 550.1 MB/s | 12.3us | 1.3ms | 241/s | 240.9 MB/s | 378.7us | 53.6ms | 60/s | 595.6 MB/s | 8.7ms | 74.6ms | 5/s | 494.1 MB/s | 211.1ms | 211.1ms |
| jaguar | 35.4K/s | 34.6 MB/s | 1.4us | 6.9us | 306/s | 19.1 MB/s | 22.1us | 227.5ms | 14/s | 14.4 MB/s | 309.1us | 252.6ms | 3/s | 34.8 MB/s | 285.0ms | 285.0ms | 2/s | 153.1 MB/s | 527.4ms | 527.4ms |
| kangaroo | 49.3K/s | 48.1 MB/s | 1.3us | 480.6us | 2.2K/s | 134.8 MB/s | 450.8us | 609.6us | 1.5K/s | 1.5 GB/s | 620.6us | 1.1ms | 327/s | 3.3 GB/s | 3.0ms | 4.9ms | 24/s | 2.4 GB/s | 32.5ms | 90.7ms |
| narwhal | 87.8K/s | 85.8 MB/s | 5.2us | 39.8us | 9.8K/s | 611.5 MB/s | 76.6us | 710.1us | 308/s | 307.5 MB/s | 871.8us | 9.3ms | 34/s | 343.3 MB/s | 12.6ms | 61.8ms | 5/s | 495.0 MB/s | 154.3ms | 217.1ms |
| owl | 2.5M/s | 2.5 GB/s | 209ns | 1.9us | 0/s | 0.00 MB/s | 0ns | 0ns | 2.7K/s | 2.7 GB/s | 139.0us | 592.0us | 0/s | 0.00 MB/s | 0ns | 0ns | 0/s | 0.00 MB/s | 0ns | 0ns |
| spider | 10.2K/s | 9.9 MB/s | 292ns | 3.5us | 224/s | 14.0 MB/s | 6.3us | 259.1ms | 10/s | 9.9 MB/s | 263.1us | 324.3ms | 2/s | 22.0 MB/s | 427.2ms | 427.2ms | 1/s | 82.9 MB/s | 947.2ms | 947.2ms |

**Write/1KB Throughput:**
```
  owl      |######################################## 2.5 GB/s (100%)
  falcon   |###########################............. 1.7 GB/s (69%)
  gecko    |######.................................. 396.9 MB/s (16%)
  ant      |####.................................... 270.9 MB/s (11%)
  fox      |##...................................... 145.5 MB/s (6%)
  narwhal  |#....................................... 85.8 MB/s (3%)
  kangaroo |#....................................... 48.1 MB/s (2%)
  jaguar   |#....................................... 34.6 MB/s (1%)
  spider   |#....................................... 9.9 MB/s (0%)
```

**Write/64KB Throughput:**
```
  falcon   |######################################## 10.0 GB/s (100%)
  fox      |####.................................... 1.0 GB/s (10%)
  narwhal  |##...................................... 611.5 MB/s (6%)
  gecko    |##...................................... 550.1 MB/s (5%)
  ant      |#....................................... 363.8 MB/s (4%)
  kangaroo |#....................................... 134.8 MB/s (1%)
  jaguar   |#....................................... 19.1 MB/s (0%)
  spider   |#....................................... 14.0 MB/s (0%)
  owl      | 0.00 MB/s (0%)
```

**Write/1MB Throughput:**
```
  falcon   |######################################## 8.4 GB/s (100%)
  owl      |############............................ 2.7 GB/s (32%)
  fox      |##########.............................. 2.2 GB/s (26%)
  kangaroo |#######................................. 1.5 GB/s (18%)
  ant      |##...................................... 492.8 MB/s (6%)
  narwhal  |#....................................... 307.5 MB/s (4%)
  gecko    |#....................................... 240.9 MB/s (3%)
  jaguar   |#....................................... 14.4 MB/s (0%)
  spider   |#....................................... 9.9 MB/s (0%)
```

**Write/10MB Throughput:**
```
  falcon   |######################################## 8.2 GB/s (100%)
  kangaroo |###############......................... 3.3 GB/s (40%)
  fox      |#########............................... 2.0 GB/s (24%)
  gecko    |##...................................... 595.6 MB/s (7%)
  narwhal  |#....................................... 343.3 MB/s (4%)
  ant      |#....................................... 144.7 MB/s (2%)
  jaguar   |#....................................... 34.8 MB/s (0%)
  spider   |#....................................... 22.0 MB/s (0%)
  owl      | 0.00 MB/s (0%)
```

**Write/100MB Throughput:**
```
  falcon   |######################################## 8.9 GB/s (100%)
  kangaroo |##########.............................. 2.4 GB/s (27%)
  fox      |#######................................. 1.6 GB/s (18%)
  narwhal  |##...................................... 495.0 MB/s (6%)
  gecko    |##...................................... 494.1 MB/s (6%)
  ant      |#....................................... 286.9 MB/s (3%)
  jaguar   |#....................................... 153.1 MB/s (2%)
  spider   |#....................................... 82.9 MB/s (1%)
  owl      | 0.00 MB/s (0%)
```

## Read Performance Deep Dive

### Read Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| ant | 908.4K/s | 887.1 MB/s | 625ns | 3.9us | 42.9K/s | 2.7 GB/s | 9.0us | 231.4us | 2.2K/s | 2.2 GB/s | 293.5us | 2.6ms | 303/s | 3.0 GB/s | 3.1ms | 6.9ms | 36/s | 3.6 GB/s | 27.4ms | 36.4ms |
| falcon | 2.9M/s | 2.9 GB/s | 208ns | 1.9us | 132.5K/s | 8.3 GB/s | 6.3us | 17.3us | 5.4K/s | 5.4 GB/s | 108.2us | 1.8ms | 476/s | 4.8 GB/s | 1.8ms | 5.1ms | 44/s | 4.4 GB/s | 21.3ms | 31.4ms |
| fox | 337.1K/s | 329.2 MB/s | 1.3us | 28.0us | 0/s | 0.00 MB/s | 0ns | 0ns | 0/s | 0.00 MB/s | 0ns | 0ns | 0/s | 0.00 MB/s | 0ns | 0ns | 0/s | 0.00 MB/s | 0ns | 0ns |
| gecko | 1.6M/s | 1.6 GB/s | 209ns | 4.5us | 30.7K/s | 1.9 GB/s | 13.7us | 465.2us | 3.4K/s | 3.4 GB/s | 127.5us | 2.3ms | 477/s | 4.8 GB/s | 1.9ms | 4.3ms | 47/s | 4.7 GB/s | 20.2ms | 29.8ms |
| jaguar | 4.4M/s | 4.3 GB/s | 125ns | 1.7us | 783.7K/s | 49.0 GB/s | 1.2us | 2.9us | 289/s | 289.1 MB/s | 3.5ms | 7.7ms | 28/s | 281.2 MB/s | 35.9ms | 39.7ms | 0/s | 0.00 MB/s | 0ns | 0ns |
| kangaroo | 3.7M/s | 3.6 GB/s | 167ns | 1.6us | 753.6K/s | 47.1 GB/s | 1.2us | 2.8us | 58.5K/s | 58.5 GB/s | 16.7us | 25.7us | 5.5K/s | 54.9 GB/s | 176.2us | 278.5us | 542/s | 54.2 GB/s | 1.8ms | 2.4ms |
| narwhal | 916.1K/s | 894.6 MB/s | 666ns | 4.0us | 57.2K/s | 3.6 GB/s | 8.2us | 190.0us | 3.0K/s | 3.0 GB/s | 168.0us | 2.6ms | 235/s | 2.3 GB/s | 2.3ms | 6.5ms | 40/s | 4.0 GB/s | 24.9ms | 28.4ms |
| owl | - | - | - | - | - | - | - | - | - | - | - | - | - | - | - | - | - | - | - | - |
| spider | 2.6M/s | 2.5 GB/s | 250ns | 2.2us | 380.6K/s | 23.8 GB/s | 1.6us | 12.7us | 321/s | 320.8 MB/s | 3.1ms | 8.6ms | 147/s | 1.5 GB/s | 6.6ms | 10.8ms | 13/s | 1.3 GB/s | 72.9ms | 88.7ms |

**Read/1KB Throughput:**
```
  jaguar   |######################################## 4.3 GB/s (100%)
  kangaroo |#################################....... 3.6 GB/s (83%)
  falcon   |##########################.............. 2.9 GB/s (66%)
  spider   |#######################................. 2.5 GB/s (58%)
  gecko    |##############.......................... 1.6 GB/s (36%)
  narwhal  |########................................ 894.6 MB/s (21%)
  ant      |########................................ 887.1 MB/s (20%)
  fox      |###..................................... 329.2 MB/s (8%)
```

**Read/64KB Throughput:**
```
  jaguar   |######################################## 49.0 GB/s (100%)
  kangaroo |######################################.. 47.1 GB/s (96%)
  spider   |###################..................... 23.8 GB/s (49%)
  falcon   |######.................................. 8.3 GB/s (17%)
  narwhal  |##...................................... 3.6 GB/s (7%)
  ant      |##...................................... 2.7 GB/s (5%)
  gecko    |#....................................... 1.9 GB/s (4%)
  fox      | 0.00 MB/s (0%)
```

**Read/1MB Throughput:**
```
  kangaroo |######################################## 58.5 GB/s (100%)
  falcon   |###..................................... 5.4 GB/s (9%)
  gecko    |##...................................... 3.4 GB/s (6%)
  narwhal  |##...................................... 3.0 GB/s (5%)
  ant      |#....................................... 2.2 GB/s (4%)
  spider   |#....................................... 320.8 MB/s (1%)
  jaguar   |#....................................... 289.1 MB/s (0%)
  fox      | 0.00 MB/s (0%)
```

**Read/10MB Throughput:**
```
  kangaroo |######################################## 54.9 GB/s (100%)
  gecko    |###..................................... 4.8 GB/s (9%)
  falcon   |###..................................... 4.8 GB/s (9%)
  ant      |##...................................... 3.0 GB/s (6%)
  narwhal  |#....................................... 2.3 GB/s (4%)
  spider   |#....................................... 1.5 GB/s (3%)
  jaguar   |#....................................... 281.2 MB/s (1%)
  fox      | 0.00 MB/s (0%)
```

**Read/100MB Throughput:**
```
  kangaroo |######################################## 54.2 GB/s (100%)
  gecko    |###..................................... 4.7 GB/s (9%)
  falcon   |###..................................... 4.4 GB/s (8%)
  narwhal  |##...................................... 4.0 GB/s (7%)
  ant      |##...................................... 3.6 GB/s (7%)
  spider   |#....................................... 1.3 GB/s (2%)
  fox      | 0.00 MB/s (0%)
  jaguar   | 0.00 MB/s (0%)
```

## Parallel Scalability

### Parallel Write Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| ant | 134.6 MB/s | 18.3 MB/s | 7.1 MB/s | 1.5 MB/s | 1.5 MB/s | 0.97 MB/s | 0% |
| falcon | 1.0 GB/s | 327.0 MB/s | 301.2 MB/s | 385.4 MB/s | 110.8 MB/s | 439.9 MB/s | 0% |
| fox | 66.9 MB/s | 3.0 MB/s | 0.99 MB/s | 0.72 MB/s | 0.30 MB/s | 0.13 MB/s | 0% |
| gecko | 279.8 MB/s | 32.9 MB/s | 14.2 MB/s | 5.9 MB/s | 2.7 MB/s | 1.3 MB/s | 0% |
| jaguar | 16.7 MB/s | 12.1 MB/s | 7.5 MB/s | 4.1 MB/s | 1.8 MB/s | 0.61 MB/s | 0% |
| narwhal | 53.8 MB/s | 4.9 MB/s | 2.2 MB/s | 1.2 MB/s | 0.63 MB/s | 0.38 MB/s | 0% |
| owl | 0.00 MB/s* | 0.00 MB/s* | 0.00 MB/s* | 0.00 MB/s* | 0.00 MB/s* | 0.00 MB/s* | - |

> Scaling = (throughput at C200 / throughput at C1) / (200/1). 100% = perfect linear scaling.

### Parallel Read Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| ant | 709.8 MB/s | 127.5 MB/s | 88.8 MB/s | 79.8 MB/s | 86.3 MB/s | 91.6 MB/s | 0% |
| falcon | 1.4 GB/s | 363.0 MB/s | 323.8 MB/s | 680.0 MB/s | 564.4 MB/s | 348.0 MB/s | 0% |
| fox | 330.0 MB/s* | 29.9 MB/s* | 35.0 MB/s* | 19.3 MB/s* | 30.9 MB/s* | 10.4 MB/s* | 0% |
| gecko | 1.0 GB/s | 294.6 MB/s | 398.1 MB/s | 389.3 MB/s | 260.6 MB/s | 196.1 MB/s | 0% |
| jaguar | 2.7 GB/s | 1.8 GB/s | 1.9 GB/s | 1.7 GB/s | 528.4 MB/s | 1.3 GB/s | 0% |
| narwhal | 328.2 MB/s | 291.3 MB/s | 187.8 MB/s | 186.6 MB/s | 131.0 MB/s | 157.8 MB/s | 0% |

## Latency Analysis

### Latency Distribution by Operation

| Driver | Operation | Min | P50 | P95 | P99 | Max | Tail Ratio |
|--------|-----------|-----|-----|-----|-----|-----|------------|
| falcon | Copy/1KB | 208ns | 625ns | 3.8us | 19.9us | 1.6ms | 31.8x (!) |
| gecko | Copy/1KB | 542ns | 1.1us | 4.2us | 19.0us | 11.8ms | 17.6x (!) |
| ant | Copy/1KB | 2.8us | 4.2us | 22.7us | 67.5us | 5.3ms | 15.9x (!) |
| narwhal | Copy/1KB | 4.2us | 8.9us | 46.1us | 287.6us | 30.0ms | 32.3x (!) |
| fox | Copy/1KB | 1.1us | 3.6us | 64.7us | 64.7us | 201.2us | 17.9x (!) |
| jaguar | Copy/1KB | 1.3us | 553.4us | 696.8us | 1.2ms | 505.9ms | 2.3x |
| owl | Delete | 0ns | 208ns | 459ns | 1.5us | 1.6ms | 7.4x |
| gecko | Delete | 125ns | 334ns | 708ns | 2.3us | 6.8ms | 6.9x |
| ant | Delete | 1.0us | 1.2us | 2.1us | 3.4us | 9.5ms | 2.7x |
| narwhal | Delete | 958ns | 1.3us | 2.0us | 3.0us | 15.5ms | 2.2x |
| falcon | Delete | 416ns | 958ns | 2.7us | 4.3us | 938.1ms | 4.5x |
| fox | Delete | 792ns | 3.3us | 25.5us | 97.8us | 14.0ms | 29.3x (!) |
| jaguar | Delete | 917ns | 546.9us | 653.8us | 1.2ms | 17.0ms | 2.2x |
| spider | Delete | 166ns | 1.2ms | 10.0ms | 25.2ms | 248.3ms | 20.5x (!) |
| falcon | EdgeCase/DeepNested | 166ns | 541ns | 1.4us | 3.8us | 1.1ms | 6.9x |
| gecko | EdgeCase/DeepNested | 375ns | 667ns | 2.3us | 9.4us | 19.2ms | 14.1x (!) |
| ant | EdgeCase/DeepNested | 2.3us | 2.9us | 9.5us | 31.7us | 1.6ms | 10.9x (!) |
| fox | EdgeCase/DeepNested | 125ns | 209ns | 16.0us | 84.0us | 3.01s | 401.7x (!) |
| narwhal | EdgeCase/DeepNested | 2.8us | 4.8us | 35.0us | 74.9us | 48.0ms | 15.8x (!) |
| jaguar | EdgeCase/DeepNested | 958ns | 1.2us | 1.8us | 5.2us | 1.22s | 4.5x |
| owl | EdgeCase/DeepNested | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| falcon | EdgeCase/EmptyObject | 125ns | 500ns | 1.1us | 2.6us | 1.7ms | 5.2x |
| narwhal | EdgeCase/EmptyObject | 1.1us | 1.5us | 2.7us | 5.5us | 13.5ms | 3.6x |
| fox | EdgeCase/EmptyObject | 83ns | 208ns | 750ns | 91.8us | 2.9ms | 441.3x (!) |
| gecko | EdgeCase/EmptyObject | 125ns | 333ns | 750ns | 2.8us | 101.8ms | 8.3x |
| jaguar | EdgeCase/EmptyObject | 958ns | 1.1us | 2.2us | 4.4us | 605.9ms | 3.9x |
| ant | EdgeCase/EmptyObject | 2.2us | 6.8us | 35.0us | 200.8us | 2.8ms | 29.6x (!) |
| owl | EdgeCase/EmptyObject | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| falcon | EdgeCase/LongKey256 | 500ns | 958ns | 2.1us | 6.2us | 1.7ms | 6.5x |
| gecko | EdgeCase/LongKey256 | 583ns | 792ns | 2.0us | 3.5us | 106.4ms | 4.4x |
| narwhal | EdgeCase/LongKey256 | 3.2us | 5.5us | 32.0us | 43.2us | 68.5ms | 7.9x |
| ant | EdgeCase/LongKey256 | 2.8us | 6.0us | 27.8us | 187.4us | 4.0ms | 31.2x (!) |
| fox | EdgeCase/LongKey256 | 166ns | 334ns | 143.7us | 196.0us | 2.1ms | 586.9x (!) |
| jaguar | EdgeCase/LongKey256 | 1.0us | 1.2us | 4.7us | 25.0us | 1.33s | 20.0x (!) |
| owl | EdgeCase/LongKey256 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| ant | List/100 | 4.7us | 7.2us | 33.3us | 128.6us | 20.2ms | 17.9x (!) |
| owl | List/100 | 41.2us | 47.4us | 112.4us | 526.7us | 2.7ms | 11.1x (!) |
| narwhal | List/100 | 526.2us | 668.9us | 900.0us | 1.3ms | 2.9ms | 2.0x |
| gecko | List/100 | 8.5ms | 10.7ms | 13.4ms | 13.7ms | 13.9ms | 1.3x |
| fox | List/100 | 56.6ms | 78.9ms | 92.1ms | 92.1ms | 103.5ms | 1.2x |
| kangaroo | List/100 | 163.4ms | 173.8ms | 176.7ms | 176.7ms | 193.4ms | 1.0x |
| jaguar | List/100 | 187.1ms | 193.2ms | 193.2ms | 193.2ms | 194.7ms | 1.0x |
| falcon | List/100 | 642.2ms | 644.2ms | 644.2ms | 644.2ms | 687.5ms | 1.0x |
| spider | List/100 | 846.5ms | 958.3ms | 958.3ms | 958.3ms | 1.90s | 1.0x |
| falcon | MixedWorkload/Balanced_50_50 | 1.7us | 11.3us | 140.6us | 401.2us | 3.9ms | 35.5x (!) |
| jaguar | MixedWorkload/Balanced_50_50 | 6.3us | 109.3us | 1.1ms | 1.7ms | 564.0ms | 15.1x (!) |
| ant | MixedWorkload/Balanced_50_50 | 1.6us | 190.0us | 5.8ms | 8.4ms | 131.4ms | 44.4x (!) |
| gecko | MixedWorkload/Balanced_50_50 | 1.6us | 160.8us | 4.1ms | 18.7ms | 522.1ms | 116.6x (!) |
| narwhal | MixedWorkload/Balanced_50_50 | 2.2us | 2.6ms | 23.5ms | 47.8ms | 74.2ms | 18.1x (!) |
| fox | MixedWorkload/Balanced_50_50 | 77.2us | 17.3ms | 21.8ms | 33.6ms | 34.9ms | 1.9x |
| falcon | MixedWorkload/ReadHeavy_90_10 | 1.7us | 11.1us | 133.1us | 294.1us | 8.9ms | 26.4x (!) |
| jaguar | MixedWorkload/ReadHeavy_90_10 | 375ns | 8.1us | 943.8us | 1.7ms | 514.8ms | 212.2x (!) |
| ant | MixedWorkload/ReadHeavy_90_10 | 1.5us | 35.9us | 9.0ms | 13.1ms | 34.4ms | 364.7x (!) |
| narwhal | MixedWorkload/ReadHeavy_90_10 | 2.1us | 10.4us | 13.0ms | 21.6ms | 49.2ms | 2069.3x (!) |
| gecko | MixedWorkload/ReadHeavy_90_10 | 1.8us | 75.8us | 18.5ms | 48.6ms | 1.24s | 641.5x (!) |
| fox | MixedWorkload/ReadHeavy_90_10 | 72.5us | 16.7ms | 21.8ms | 23.8ms | 45.0ms | 1.4x |
| falcon | MixedWorkload/WriteHeavy_10_90 | 1.6us | 9.0us | 98.9us | 615.9us | 19.0ms | 68.1x (!) |
| gecko | MixedWorkload/WriteHeavy_10_90 | 3.7us | 1.6ms | 4.5ms | 133.3ms | 354.2ms | 84.0x (!) |
| ant | MixedWorkload/WriteHeavy_10_90 | 2.2us | 3.3ms | 8.1ms | 22.8ms | 207.8ms | 6.9x |
| narwhal | MixedWorkload/WriteHeavy_10_90 | 3.2us | 5.9ms | 28.7ms | 46.9ms | 75.1ms | 7.9x |
| jaguar | MixedWorkload/WriteHeavy_10_90 | 6.9us | 364.9us | 2.4ms | 3.4ms | 11.44s | 9.3x |
| fox | MixedWorkload/WriteHeavy_10_90 | 82.6us | 19.7ms | 32.6ms | 617.7ms | 625.3ms | 31.4x (!) |
| falcon | Multipart/15MB_3Parts | 27.0ms | 30.2ms | 35.7ms | 35.9ms | 37.1ms | 1.2x |
| fox | Multipart/15MB_3Parts | 27.9ms | 31.8ms | 36.2ms | 37.2ms | 40.9ms | 1.2x |
| ant | Multipart/15MB_3Parts | 34.5ms | 43.4ms | 45.7ms | 46.8ms | 95.7ms | 1.1x |
| narwhal | Multipart/15MB_3Parts | 44.9ms | 62.6ms | 88.6ms | 88.6ms | 99.6ms | 1.4x |
| gecko | Multipart/15MB_3Parts | 65.6ms | 90.8ms | 102.2ms | 102.2ms | 130.2ms | 1.1x |
| jaguar | Multipart/15MB_3Parts | 608.5ms | 616.8ms | 616.8ms | 616.8ms | 669.0ms | 1.0x |
| owl | Multipart/15MB_3Parts | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| jaguar | ParallelRead/1KB/C1 | 83ns | 167ns | 916ns | 3.1us | 2.2ms | 18.5x (!) |
| falcon | ParallelRead/1KB/C1 | 125ns | 291ns | 1.7us | 2.9us | 3.5ms | 10.0x (!) |
| gecko | ParallelRead/1KB/C1 | 166ns | 250ns | 2.5us | 14.1us | 2.4ms | 56.3x (!) |
| ant | ParallelRead/1KB/C1 | 542ns | 750ns | 2.8us | 6.9us | 2.4ms | 9.2x |
| fox | ParallelRead/1KB/C1 | 500ns | 2.1us | 5.3us | 20.3us | 1.2ms | 9.8x |
| narwhal | ParallelRead/1KB/C1 | 541ns | 875ns | 7.5us | 22.6us | 10.2ms | 25.9x (!) |
| jaguar | ParallelRead/1KB/C10 | 83ns | 208ns | 1.3us | 5.4us | 3.3ms | 26.0x (!) |
| falcon | ParallelRead/1KB/C10 | 166ns | 292ns | 2.6us | 10.0us | 6.5ms | 34.1x (!) |
| gecko | ParallelRead/1KB/C10 | 166ns | 583ns | 6.5us | 46.0us | 3.1ms | 78.8x (!) |
| narwhal | ParallelRead/1KB/C10 | 583ns | 2.0us | 10.0us | 25.6us | 2.8ms | 13.1x (!) |
| ant | ParallelRead/1KB/C10 | 583ns | 2.7us | 26.5us | 65.7us | 6.6ms | 24.6x (!) |
| fox | ParallelRead/1KB/C10 | 541ns | 7.9us | 104.0us | 389.0us | 16.0ms | 49.1x (!) |
| falcon | ParallelRead/1KB/C100 | 166ns | 416ns | 3.5us | 14.2us | 3.4ms | 34.3x (!) |
| jaguar | ParallelRead/1KB/C100 | 83ns | 375ns | 2.5us | 12.8us | 33.4ms | 34.2x (!) |
| gecko | ParallelRead/1KB/C100 | 166ns | 792ns | 6.9us | 67.8us | 3.0ms | 85.6x (!) |
| narwhal | ParallelRead/1KB/C100 | 583ns | 2.3us | 15.0us | 72.7us | 6.0ms | 31.1x (!) |
| ant | ParallelRead/1KB/C100 | 583ns | 4.8us | 37.2us | 111.4us | 1.5ms | 23.4x (!) |
| fox | ParallelRead/1KB/C100 | 250ns | 9.0us | 120.4us | 438.6us | 2.9ms | 48.7x (!) |
| jaguar | ParallelRead/1KB/C200 | 125ns | 291ns | 1.8us | 11.1us | 959.7us | 38.1x (!) |
| falcon | ParallelRead/1KB/C200 | 166ns | 542ns | 7.0us | 50.5us | 2.9ms | 93.1x (!) |
| gecko | ParallelRead/1KB/C200 | 166ns | 750ns | 11.5us | 93.2us | 3.8ms | 124.3x (!) |
| narwhal | ParallelRead/1KB/C200 | 583ns | 2.3us | 13.6us | 62.1us | 4.9ms | 26.6x (!) |
| ant | ParallelRead/1KB/C200 | 583ns | 4.6us | 35.6us | 97.5us | 1.4ms | 21.3x (!) |
| fox | ParallelRead/1KB/C200 | 208ns | 10.5us | 396.7us | 2.0ms | 12.3ms | 189.2x (!) |
| jaguar | ParallelRead/1KB/C25 | 83ns | 208ns | 1.4us | 7.4us | 192.2us | 35.5x (!) |
| gecko | ParallelRead/1KB/C25 | 166ns | 500ns | 5.0us | 29.5us | 3.2ms | 59.0x (!) |
| falcon | ParallelRead/1KB/C25 | 166ns | 625ns | 7.3us | 32.8us | 4.9ms | 52.4x (!) |
| narwhal | ParallelRead/1KB/C25 | 583ns | 2.3us | 14.3us | 50.2us | 2.5ms | 21.9x (!) |
| ant | ParallelRead/1KB/C25 | 583ns | 3.7us | 32.6us | 127.5us | 3.0ms | 34.4x (!) |
| fox | ParallelRead/1KB/C25 | 542ns | 8.5us | 96.4us | 352.5us | 3.8ms | 41.5x (!) |
| jaguar | ParallelRead/1KB/C50 | 83ns | 208ns | 1.5us | 8.5us | 1.6ms | 40.9x (!) |
| falcon | ParallelRead/1KB/C50 | 166ns | 375ns | 3.2us | 22.0us | 3.0ms | 58.7x (!) |
| gecko | ParallelRead/1KB/C50 | 166ns | 625ns | 4.6us | 21.7us | 14.3ms | 34.7x (!) |
| narwhal | ParallelRead/1KB/C50 | 583ns | 2.3us | 14.7us | 55.0us | 1.6ms | 24.0x (!) |
| ant | ParallelRead/1KB/C50 | 625ns | 4.2us | 43.8us | 153.8us | 2.0ms | 36.5x (!) |
| fox | ParallelRead/1KB/C50 | 250ns | 7.6us | 160.1us | 778.7us | 53.0ms | 102.7x (!) |
| falcon | ParallelWrite/1KB/C1 | 166ns | 458ns | 2.5us | 3.2us | 5.9ms | 7.1x |
| gecko | ParallelWrite/1KB/C1 | 291ns | 667ns | 3.8us | 28.1us | 63.7ms | 42.2x (!) |
| ant | ParallelWrite/1KB/C1 | 3.7us | 5.0us | 13.0us | 30.0us | 4.4ms | 6.0x |
| fox | ParallelWrite/1KB/C1 | 500ns | 1.2us | 57.5us | 72.9us | 278.6us | 58.3x (!) |
| narwhal | ParallelWrite/1KB/C1 | 3.8us | 6.9us | 36.8us | 49.8us | 12.8ms | 7.2x |
| jaguar | ParallelWrite/1KB/C1 | 1.6us | 2.2us | 6.3us | 28.7us | 312.6ms | 13.2x (!) |
| owl | ParallelWrite/1KB/C1 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| falcon | ParallelWrite/1KB/C10 | 166ns | 625ns | 4.0us | 15.3us | 5.8ms | 24.5x (!) |
| gecko | ParallelWrite/1KB/C10 | 333ns | 3.4us | 99.9us | 245.2us | 56.5ms | 72.6x (!) |
| ant | ParallelWrite/1KB/C10 | 4.0us | 20.7us | 156.8us | 351.1us | 5.4ms | 17.0x (!) |
| jaguar | ParallelWrite/1KB/C10 | 1.7us | 3.0us | 150.5us | 253.5us | 368.3ms | 83.3x (!) |
| narwhal | ParallelWrite/1KB/C10 | 4.2us | 138.1us | 778.5us | 1.5ms | 14.6ms | 10.5x (!) |
| fox | ParallelWrite/1KB/C10 | 583ns | 15.0us | 1.1ms | 1.7ms | 3.7ms | 114.1x (!) |
| owl | ParallelWrite/1KB/C10 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| falcon | ParallelWrite/1KB/C100 | 166ns | 833ns | 7.5us | 86.2us | 24.5ms | 103.4x (!) |
| gecko | ParallelWrite/1KB/C100 | 333ns | 4.6us | 1.3ms | 2.8ms | 77.5ms | 601.8x (!) |
| jaguar | ParallelWrite/1KB/C100 | 1.7us | 13.9us | 1.9ms | 2.9ms | 476.9ms | 211.3x (!) |
| ant | ParallelWrite/1KB/C100 | 4.2us | 685.8us | 1.7ms | 2.4ms | 6.6ms | 3.5x |
| narwhal | ParallelWrite/1KB/C100 | 4.9us | 1.1ms | 4.1ms | 8.3ms | 26.5ms | 7.5x |
| fox | ParallelWrite/1KB/C100 | 250ns | 3.0ms | 5.8ms | 7.6ms | 12.8ms | 2.5x |
| owl | ParallelWrite/1KB/C100 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| falcon | ParallelWrite/1KB/C200 | 208ns | 709ns | 6.1us | 29.0us | 1.5ms | 40.8x (!) |
| gecko | ParallelWrite/1KB/C200 | 333ns | 952.2us | 2.0ms | 2.9ms | 92.3ms | 3.1x |
| ant | ParallelWrite/1KB/C200 | 4.5us | 1.1ms | 1.9ms | 2.7ms | 7.2ms | 2.5x |
| jaguar | ParallelWrite/1KB/C200 | 2.3us | 1.9ms | 3.8ms | 5.5ms | 626.2ms | 3.0x |
| narwhal | ParallelWrite/1KB/C200 | 5.0us | 1.9ms | 6.6ms | 11.4ms | 34.8ms | 6.1x |
| fox | ParallelWrite/1KB/C200 | 291ns | 7.2ms | 12.7ms | 22.5ms | 40.4ms | 3.1x |
| owl | ParallelWrite/1KB/C200 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| falcon | ParallelWrite/1KB/C25 | 125ns | 792ns | 6.0us | 20.0us | 6.2ms | 25.3x (!) |
| gecko | ParallelWrite/1KB/C25 | 333ns | 4.2us | 275.7us | 618.5us | 76.1ms | 146.9x (!) |
| jaguar | ParallelWrite/1KB/C25 | 1.7us | 3.2us | 510.1us | 851.7us | 361.8ms | 268.9x (!) |
| ant | ParallelWrite/1KB/C25 | 4.0us | 47.3us | 416.8us | 895.2us | 7.6ms | 18.9x (!) |
| narwhal | ParallelWrite/1KB/C25 | 4.8us | 348.0us | 1.4ms | 2.2ms | 10.5ms | 6.2x |
| fox | ParallelWrite/1KB/C25 | 666ns | 1.1ms | 2.1ms | 3.6ms | 11.6ms | 3.2x |
| owl | ParallelWrite/1KB/C25 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| falcon | ParallelWrite/1KB/C50 | 167ns | 792ns | 5.5us | 26.9us | 1.8ms | 33.9x (!) |
| gecko | ParallelWrite/1KB/C50 | 333ns | 4.0us | 707.1us | 1.5ms | 101.5ms | 370.0x (!) |
| jaguar | ParallelWrite/1KB/C50 | 1.7us | 3.7us | 930.2us | 1.6ms | 428.5ms | 425.2x (!) |
| ant | ParallelWrite/1KB/C50 | 4.0us | 70.8us | 2.5ms | 7.8ms | 33.9ms | 109.6x (!) |
| narwhal | ParallelWrite/1KB/C50 | 4.7us | 628.6us | 2.1ms | 3.7ms | 27.2ms | 5.9x |
| fox | ParallelWrite/1KB/C50 | 584ns | 1.3ms | 2.0ms | 3.5ms | 10.4ms | 2.7x |
| owl | ParallelWrite/1KB/C50 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| jaguar | RangeRead/End_256KB | 3.5us | 4.2us | 4.9us | 8.4us | 1.4ms | 2.0x |
| falcon | RangeRead/End_256KB | 11.1us | 54.8us | 214.2us | 628.1us | 2.7ms | 11.5x (!) |
| gecko | RangeRead/End_256KB | 9.7us | 63.4us | 232.5us | 567.2us | 2.8ms | 8.9x |
| narwhal | RangeRead/End_256KB | 80.1us | 167.2us | 1.4ms | 2.2ms | 18.4ms | 13.0x (!) |
| ant | RangeRead/End_256KB | 101.5us | 277.6us | 972.0us | 2.0ms | 8.4ms | 7.3x |
| fox | RangeRead/End_256KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| jaguar | RangeRead/Middle_256KB | 3.5us | 4.3us | 4.8us | 7.4us | 1.7ms | 1.7x |
| falcon | RangeRead/Middle_256KB | 11.0us | 30.6us | 129.7us | 1.0ms | 12.4ms | 33.3x (!) |
| gecko | RangeRead/Middle_256KB | 22.1us | 62.8us | 574.9us | 1.4ms | 5.1ms | 22.7x (!) |
| narwhal | RangeRead/Middle_256KB | 79.1us | 123.3us | 693.3us | 1.7ms | 3.0ms | 13.4x (!) |
| ant | RangeRead/Middle_256KB | 79.0us | 158.0us | 1.6ms | 3.4ms | 23.0ms | 21.6x (!) |
| fox | RangeRead/Middle_256KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| jaguar | RangeRead/Start_256KB | 3.8us | 4.8us | 5.5us | 8.4us | 3.4ms | 1.8x |
| falcon | RangeRead/Start_256KB | 18.0us | 26.3us | 101.9us | 362.9us | 3.4ms | 13.8x (!) |
| gecko | RangeRead/Start_256KB | 23.3us | 55.5us | 241.3us | 1.8ms | 9.1ms | 31.7x (!) |
| ant | RangeRead/Start_256KB | 79.3us | 117.4us | 705.5us | 1.4ms | 3.6ms | 12.3x (!) |
| narwhal | RangeRead/Start_256KB | 95.7us | 128.3us | 944.9us | 1.8ms | 41.4ms | 14.4x (!) |
| fox | RangeRead/Start_256KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| kangaroo | Read/100MB | 1.7ms | 1.8ms | 2.1ms | 2.4ms | 2.6ms | 1.4x |
| gecko | Read/100MB | 18.0ms | 20.2ms | 27.2ms | 29.8ms | 31.4ms | 1.5x |
| falcon | Read/100MB | 16.9ms | 21.3ms | 30.8ms | 31.4ms | 38.4ms | 1.5x |
| narwhal | Read/100MB | 20.0ms | 24.9ms | 28.1ms | 28.4ms | 30.8ms | 1.1x |
| ant | Read/100MB | 21.3ms | 27.4ms | 34.1ms | 36.4ms | 42.2ms | 1.3x |
| spider | Read/100MB | 55.4ms | 72.9ms | 88.7ms | 88.7ms | 138.0ms | 1.2x |
| fox | Read/100MB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| jaguar | Read/100MB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| kangaroo | Read/10MB | 163.5us | 176.2us | 215.4us | 278.5us | 442.4us | 1.6x |
| gecko | Read/10MB | 1.1ms | 1.9ms | 3.6ms | 4.3ms | 5.6ms | 2.3x |
| falcon | Read/10MB | 842.9us | 1.8ms | 4.3ms | 5.1ms | 6.7ms | 2.9x |
| ant | Read/10MB | 1.6ms | 3.1ms | 6.0ms | 6.9ms | 8.1ms | 2.3x |
| narwhal | Read/10MB | 1.1ms | 2.3ms | 5.3ms | 6.5ms | 198.9ms | 2.8x |
| spider | Read/10MB | 4.1ms | 6.6ms | 9.2ms | 10.8ms | 11.2ms | 1.6x |
| jaguar | Read/10MB | 23.3ms | 35.9ms | 39.2ms | 39.7ms | 47.8ms | 1.1x |
| fox | Read/10MB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| jaguar | Read/1KB | 41ns | 125ns | 208ns | 1.7us | 4.2ms | 13.3x (!) |
| kangaroo | Read/1KB | 83ns | 167ns | 292ns | 1.6us | 2.4ms | 9.7x |
| falcon | Read/1KB | 125ns | 208ns | 1.2us | 1.9us | 3.3ms | 9.0x |
| spider | Read/1KB | 125ns | 250ns | 417ns | 2.2us | 1.2ms | 9.0x |
| gecko | Read/1KB | 125ns | 209ns | 1.6us | 4.5us | 4.7ms | 21.3x (!) |
| narwhal | Read/1KB | 500ns | 666ns | 2.0us | 4.0us | 2.7ms | 6.1x |
| ant | Read/1KB | 500ns | 625ns | 1.8us | 3.9us | 4.2ms | 6.3x |
| fox | Read/1KB | 166ns | 1.3us | 7.7us | 28.0us | 1.5ms | 21.0x (!) |
| kangaroo | Read/1MB | 14.8us | 16.7us | 18.8us | 25.7us | 57.3us | 1.5x |
| falcon | Read/1MB | 49.0us | 108.2us | 648.2us | 1.8ms | 4.3ms | 16.4x (!) |
| gecko | Read/1MB | 81.1us | 127.5us | 1.0ms | 2.3ms | 29.8ms | 17.8x (!) |
| narwhal | Read/1MB | 100.4us | 168.0us | 1.3ms | 2.6ms | 9.1ms | 15.6x (!) |
| ant | Read/1MB | 145.9us | 293.5us | 1.4ms | 2.6ms | 8.2ms | 8.8x |
| spider | Read/1MB | 25.5us | 3.1ms | 5.8ms | 8.6ms | 10.7ms | 2.8x |
| jaguar | Read/1MB | 19.5us | 3.5ms | 5.7ms | 7.7ms | 8.9ms | 2.2x |
| fox | Read/1MB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| jaguar | Read/64KB | 1.0us | 1.2us | 1.3us | 2.9us | 1.8ms | 2.5x |
| kangaroo | Read/64KB | 1.0us | 1.2us | 1.5us | 2.8us | 1.1ms | 2.2x |
| spider | Read/64KB | 1.3us | 1.6us | 3.8us | 12.7us | 2.0ms | 7.8x |
| falcon | Read/64KB | 4.6us | 6.3us | 10.7us | 17.3us | 2.2ms | 2.8x |
| narwhal | Read/64KB | 6.2us | 8.2us | 32.2us | 190.0us | 3.4ms | 23.2x (!) |
| ant | Read/64KB | 6.4us | 9.0us | 39.3us | 231.4us | 12.7ms | 25.8x (!) |
| gecko | Read/64KB | 5.5us | 13.7us | 51.3us | 465.2us | 4.8ms | 33.9x (!) |
| fox | Read/64KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| owl | Scale/Delete/10 | 10.4us | 10.4us | 10.4us | 10.4us | 10.4us | 1.0x |
| gecko | Scale/Delete/10 | 19.3us | 19.3us | 19.3us | 19.3us | 19.3us | 1.0x |
| ant | Scale/Delete/10 | 19.7us | 19.7us | 19.7us | 19.7us | 19.7us | 1.0x |
| fox | Scale/Delete/10 | 19.8us | 19.8us | 19.8us | 19.8us | 19.8us | 1.0x |
| jaguar | Scale/Delete/10 | 31.5us | 31.5us | 31.5us | 31.5us | 31.5us | 1.0x |
| narwhal | Scale/Delete/10 | 34.0us | 34.0us | 34.0us | 34.0us | 34.0us | 1.0x |
| falcon | Scale/Delete/10 | 10.3ms | 10.3ms | 10.3ms | 10.3ms | 10.3ms | 1.0x |
| owl | Scale/Delete/100 | 60.8us | 60.8us | 60.8us | 60.8us | 60.8us | 1.0x |
| gecko | Scale/Delete/100 | 110.0us | 110.0us | 110.0us | 110.0us | 110.0us | 1.0x |
| ant | Scale/Delete/100 | 136.8us | 136.8us | 136.8us | 136.8us | 136.8us | 1.0x |
| jaguar | Scale/Delete/100 | 146.1us | 146.1us | 146.1us | 146.1us | 146.1us | 1.0x |
| narwhal | Scale/Delete/100 | 221.6us | 221.6us | 221.6us | 221.6us | 221.6us | 1.0x |
| fox | Scale/Delete/100 | 1.1ms | 1.1ms | 1.1ms | 1.1ms | 1.1ms | 1.0x |
| owl | Scale/Delete/1000 | 461.6us | 461.6us | 461.6us | 461.6us | 461.6us | 1.0x |
| gecko | Scale/Delete/1000 | 687.2us | 687.2us | 687.2us | 687.2us | 687.2us | 1.0x |
| ant | Scale/Delete/1000 | 1.3ms | 1.3ms | 1.3ms | 1.3ms | 1.3ms | 1.0x |
| narwhal | Scale/Delete/1000 | 2.0ms | 2.0ms | 2.0ms | 2.0ms | 2.0ms | 1.0x |
| fox | Scale/Delete/1000 | 4.6ms | 4.6ms | 4.6ms | 4.6ms | 4.6ms | 1.0x |
| jaguar | Scale/Delete/1000 | 4.7ms | 4.7ms | 4.7ms | 4.7ms | 4.7ms | 1.0x |
| owl | Scale/Delete/10000 | 4.8ms | 4.8ms | 4.8ms | 4.8ms | 4.8ms | 1.0x |
| gecko | Scale/Delete/10000 | 5.9ms | 5.9ms | 5.9ms | 5.9ms | 5.9ms | 1.0x |
| ant | Scale/Delete/10000 | 14.6ms | 14.6ms | 14.6ms | 14.6ms | 14.6ms | 1.0x |
| narwhal | Scale/Delete/10000 | 15.8ms | 15.8ms | 15.8ms | 15.8ms | 15.8ms | 1.0x |
| fox | Scale/Delete/10000 | 66.0ms | 66.0ms | 66.0ms | 66.0ms | 66.0ms | 1.0x |
| jaguar | Scale/Delete/10000 | 21.78s | 21.78s | 21.78s | 21.78s | 21.78s | 1.0x |
| ant | Scale/List/10 | 9.9us | 9.9us | 9.9us | 9.9us | 9.9us | 1.0x |
| narwhal | Scale/List/10 | 18.8ms | 18.8ms | 18.8ms | 18.8ms | 18.8ms | 1.0x |
| gecko | Scale/List/10 | 100.7ms | 100.7ms | 100.7ms | 100.7ms | 100.7ms | 1.0x |
| owl | Scale/List/10 | 226.5ms | 226.5ms | 226.5ms | 226.5ms | 226.5ms | 1.0x |
| fox | Scale/List/10 | 842.6ms | 842.6ms | 842.6ms | 842.6ms | 842.6ms | 1.0x |
| jaguar | Scale/List/10 | 1.04s | 1.04s | 1.04s | 1.04s | 1.04s | 1.0x |
| falcon | Scale/List/10 | 273.80s | 273.80s | 273.80s | 273.80s | 273.80s | 1.0x |
| ant | Scale/List/100 | 20.1us | 20.1us | 20.1us | 20.1us | 20.1us | 1.0x |
| narwhal | Scale/List/100 | 18.7ms | 18.7ms | 18.7ms | 18.7ms | 18.7ms | 1.0x |
| gecko | Scale/List/100 | 84.2ms | 84.2ms | 84.2ms | 84.2ms | 84.2ms | 1.0x |
| owl | Scale/List/100 | 196.3ms | 196.3ms | 196.3ms | 196.3ms | 196.3ms | 1.0x |
| fox | Scale/List/100 | 505.6ms | 505.6ms | 505.6ms | 505.6ms | 505.6ms | 1.0x |
| jaguar | Scale/List/100 | 1.07s | 1.07s | 1.07s | 1.07s | 1.07s | 1.0x |
| falcon | Scale/List/100 | 295.92s | 295.92s | 295.92s | 295.92s | 295.92s | 1.0x |
| ant | Scale/List/1000 | 111.4us | 111.4us | 111.4us | 111.4us | 111.4us | 1.0x |
| narwhal | Scale/List/1000 | 19.1ms | 19.1ms | 19.1ms | 19.1ms | 19.1ms | 1.0x |
| gecko | Scale/List/1000 | 77.9ms | 77.9ms | 77.9ms | 77.9ms | 77.9ms | 1.0x |
| owl | Scale/List/1000 | 200.8ms | 200.8ms | 200.8ms | 200.8ms | 200.8ms | 1.0x |
| fox | Scale/List/1000 | 1.04s | 1.04s | 1.04s | 1.04s | 1.04s | 1.0x |
| jaguar | Scale/List/1000 | 1.10s | 1.10s | 1.10s | 1.10s | 1.10s | 1.0x |
| ant | Scale/List/10000 | 1.9ms | 1.9ms | 1.9ms | 1.9ms | 1.9ms | 1.0x |
| narwhal | Scale/List/10000 | 20.7ms | 20.7ms | 20.7ms | 20.7ms | 20.7ms | 1.0x |
| gecko | Scale/List/10000 | 79.6ms | 79.6ms | 79.6ms | 79.6ms | 79.6ms | 1.0x |
| owl | Scale/List/10000 | 205.2ms | 205.2ms | 205.2ms | 205.2ms | 205.2ms | 1.0x |
| jaguar | Scale/List/10000 | 1.10s | 1.10s | 1.10s | 1.10s | 1.10s | 1.0x |
| fox | Scale/List/10000 | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| jaguar | Scale/Write/10 | 28.1us | 28.1us | 28.1us | 28.1us | 28.1us | 1.0x |
| gecko | Scale/Write/10 | 38.1us | 38.1us | 38.1us | 38.1us | 38.1us | 1.0x |
| falcon | Scale/Write/10 | 44.9us | 44.9us | 44.9us | 44.9us | 44.9us | 1.0x |
| ant | Scale/Write/10 | 62.3us | 62.3us | 62.3us | 62.3us | 62.3us | 1.0x |
| owl | Scale/Write/10 | 76.6us | 76.6us | 76.6us | 76.6us | 76.6us | 1.0x |
| narwhal | Scale/Write/10 | 304.0us | 304.0us | 304.0us | 304.0us | 304.0us | 1.0x |
| fox | Scale/Write/10 | 348.5us | 348.5us | 348.5us | 348.5us | 348.5us | 1.0x |
| owl | Scale/Write/100 | 75.5us | 75.5us | 75.5us | 75.5us | 75.5us | 1.0x |
| jaguar | Scale/Write/100 | 223.0us | 223.0us | 223.0us | 223.0us | 223.0us | 1.0x |
| gecko | Scale/Write/100 | 265.8us | 265.8us | 265.8us | 265.8us | 265.8us | 1.0x |
| falcon | Scale/Write/100 | 312.1us | 312.1us | 312.1us | 312.1us | 312.1us | 1.0x |
| ant | Scale/Write/100 | 314.9us | 314.9us | 314.9us | 314.9us | 314.9us | 1.0x |
| fox | Scale/Write/100 | 3.3ms | 3.3ms | 3.3ms | 3.3ms | 3.3ms | 1.0x |
| narwhal | Scale/Write/100 | 6.2ms | 6.2ms | 6.2ms | 6.2ms | 6.2ms | 1.0x |
| owl | Scale/Write/1000 | 617.3us | 617.3us | 617.3us | 617.3us | 617.3us | 1.0x |
| jaguar | Scale/Write/1000 | 1.5ms | 1.5ms | 1.5ms | 1.5ms | 1.5ms | 1.0x |
| gecko | Scale/Write/1000 | 4.9ms | 4.9ms | 4.9ms | 4.9ms | 4.9ms | 1.0x |
| ant | Scale/Write/1000 | 5.1ms | 5.1ms | 5.1ms | 5.1ms | 5.1ms | 1.0x |
| fox | Scale/Write/1000 | 28.9ms | 28.9ms | 28.9ms | 28.9ms | 28.9ms | 1.0x |
| narwhal | Scale/Write/1000 | 117.2ms | 117.2ms | 117.2ms | 117.2ms | 117.2ms | 1.0x |
| owl | Scale/Write/10000 | 5.9ms | 5.9ms | 5.9ms | 5.9ms | 5.9ms | 1.0x |
| gecko | Scale/Write/10000 | 16.3ms | 16.3ms | 16.3ms | 16.3ms | 16.3ms | 1.0x |
| ant | Scale/Write/10000 | 44.7ms | 44.7ms | 44.7ms | 44.7ms | 44.7ms | 1.0x |
| fox | Scale/Write/10000 | 489.6ms | 489.6ms | 489.6ms | 489.6ms | 489.6ms | 1.0x |
| narwhal | Scale/Write/10000 | 541.3ms | 541.3ms | 541.3ms | 541.3ms | 541.3ms | 1.0x |
| jaguar | Scale/Write/10000 | 1.35s | 1.35s | 1.35s | 1.35s | 1.35s | 1.0x |
| falcon | Stat | 0ns | 83ns | 84ns | 250ns | 5.8ms | 3.0x |
| kangaroo | Stat | 0ns | 83ns | 125ns | 1.1us | 4.1ms | 13.0x (!) |
| fox | Stat | 0ns | 83ns | 125ns | 958ns | 4.4ms | 11.5x (!) |
| spider | Stat | 0ns | 83ns | 125ns | 1.3us | 5.4ms | 15.6x (!) |
| narwhal | Stat | 41ns | 125ns | 166ns | 1.5us | 4.3ms | 11.7x (!) |
| gecko | Stat | 0ns | 83ns | 125ns | 1.7us | 2.8ms | 20.6x (!) |
| owl | Stat | 0ns | 42ns | 125ns | 1.9us | 2.1ms | 45.6x (!) |
| jaguar | Stat | 0ns | 83ns | 167ns | 2.1us | 7.7ms | 25.1x (!) |
| ant | Stat | 375ns | 459ns | 2.5us | 5.9us | 3.8ms | 12.8x (!) |
| falcon | Write/100MB | 8.2ms | 11.2ms | 14.3ms | 15.4ms | 20.3ms | 1.4x |
| kangaroo | Write/100MB | 7.2ms | 32.5ms | 90.6ms | 90.7ms | 92.6ms | 2.8x |
| fox | Write/100MB | 49.1ms | 58.4ms | 73.6ms | 73.6ms | 82.9ms | 1.3x |
| narwhal | Write/100MB | 123.9ms | 154.3ms | 217.1ms | 217.1ms | 384.0ms | 1.4x |
| gecko | Write/100MB | 178.1ms | 211.1ms | 211.1ms | 211.1ms | 218.0ms | 1.0x |
| ant | Write/100MB | 65.2ms | 195.6ms | 311.8ms | 311.8ms | 1.13s | 1.6x |
| jaguar | Write/100MB | 507.0ms | 527.4ms | 527.4ms | 527.4ms | 925.0ms | 1.0x |
| spider | Write/100MB | 841.5ms | 947.2ms | 947.2ms | 947.2ms | 1.83s | 1.0x |
| owl | Write/100MB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| falcon | Write/10MB | 625.3us | 777.9us | 2.5ms | 2.9ms | 6.6ms | 3.8x |
| kangaroo | Write/10MB | 1.9ms | 3.0ms | 4.1ms | 4.9ms | 5.7ms | 1.6x |
| fox | Write/10MB | 2.4ms | 3.2ms | 4.4ms | 5.0ms | 198.8ms | 1.6x |
| gecko | Write/10MB | 6.4ms | 8.7ms | 17.2ms | 74.6ms | 252.3ms | 8.6x |
| narwhal | Write/10MB | 10.4ms | 12.6ms | 59.9ms | 61.8ms | 223.0ms | 4.9x |
| ant | Write/10MB | 4.6ms | 27.3ms | 216.3ms | 232.3ms | 487.0ms | 8.5x |
| jaguar | Write/10MB | 280.7ms | 285.0ms | 285.0ms | 285.0ms | 296.9ms | 1.0x |
| spider | Write/10MB | 415.1ms | 427.2ms | 427.2ms | 427.2ms | 523.9ms | 1.0x |
| owl | Write/10MB | 0ns | 0ns | 0ns | 0ns | 0ns | - |
| owl | Write/1KB | 83ns | 209ns | 1.2us | 1.9us | 383.0us | 9.2x |
| falcon | Write/1KB | 83ns | 292ns | 1.2us | 2.3us | 2.8ms | 7.8x |
| gecko | Write/1KB | 208ns | 500ns | 1.9us | 7.7us | 132.3ms | 15.3x (!) |
| ant | Write/1KB | 2.2us | 3.0us | 5.2us | 14.3us | 3.2ms | 4.8x |
| fox | Write/1KB | 167ns | 750ns | 31.5us | 44.2us | 1.6ms | 59.0x (!) |
| narwhal | Write/1KB | 3.3us | 5.2us | 32.7us | 39.8us | 1.5ms | 7.6x |
| kangaroo | Write/1KB | 166ns | 1.3us | 10.2us | 480.6us | 1.6ms | 372.3x (!) |
| jaguar | Write/1KB | 1.1us | 1.4us | 3.0us | 6.9us | 205.9ms | 4.9x |
| spider | Write/1KB | 125ns | 292ns | 1.3us | 3.5us | 471.0ms | 12.1x (!) |
| falcon | Write/1MB | 51.5us | 73.4us | 185.4us | 1.8ms | 4.8ms | 24.1x (!) |
| owl | Write/1MB | 96.4us | 139.0us | 592.0us | 592.0us | 1.3ms | 4.3x |
| fox | Write/1MB | 99.5us | 285.6us | 1.2ms | 2.7ms | 39.1ms | 9.3x |
| kangaroo | Write/1MB | 519.8us | 620.6us | 792.2us | 1.1ms | 1.8ms | 1.8x |
| ant | Write/1MB | 208.2us | 309.2us | 2.7ms | 30.8ms | 165.0ms | 99.5x (!) |
| narwhal | Write/1MB | 649.8us | 871.8us | 3.1ms | 9.3ms | 654.2ms | 10.7x (!) |
| gecko | Write/1MB | 271.5us | 378.7us | 1.5ms | 53.6ms | 1.28s | 141.6x (!) |
| jaguar | Write/1MB | 228.1us | 309.1us | 252.6ms | 252.6ms | 271.9ms | 817.3x (!) |
| spider | Write/1MB | 91.5us | 263.1us | 324.3ms | 324.3ms | 571.9ms | 1232.6x (!) |
| falcon | Write/64KB | 1.8us | 4.7us | 11.6us | 27.4us | 2.5ms | 5.8x |
| fox | Write/64KB | 41.7us | 52.2us | 88.8us | 162.4us | 4.4ms | 3.1x |
| narwhal | Write/64KB | 42.8us | 76.6us | 184.5us | 710.1us | 4.2ms | 9.3x |
| gecko | Write/64KB | 8.5us | 12.3us | 248.0us | 1.3ms | 94.3ms | 104.9x (!) |
| ant | Write/64KB | 15.6us | 21.0us | 114.8us | 725.9us | 556.5ms | 34.5x (!) |
| kangaroo | Write/64KB | 411.7us | 450.8us | 552.1us | 609.6us | 821.5us | 1.4x |
| jaguar | Write/64KB | 14.5us | 22.1us | 49.2us | 227.5ms | 251.1ms | 10281.9x (!) |
| spider | Write/64KB | 5.2us | 6.3us | 65.3us | 259.1ms | 315.7ms | 40920.2x (!) |
| owl | Write/64KB | 0ns | 0ns | 0ns | 0ns | 0ns | - |

### Tail Latency Warnings

> Drivers with P99/P50 ratio > 10x indicate significant tail latency.

- **falcon** on Copy/1KB: P99 is 32x the P50 latency
- **gecko** on Copy/1KB: P99 is 18x the P50 latency
- **ant** on Copy/1KB: P99 is 16x the P50 latency
- **narwhal** on Copy/1KB: P99 is 32x the P50 latency
- **fox** on Copy/1KB: P99 is 18x the P50 latency
- **fox** on Delete: P99 is 29x the P50 latency
- **spider** on Delete: P99 is 21x the P50 latency
- **gecko** on EdgeCase/DeepNested: P99 is 14x the P50 latency
- **ant** on EdgeCase/DeepNested: P99 is 11x the P50 latency
- **fox** on EdgeCase/DeepNested: P99 is 402x the P50 latency
- **narwhal** on EdgeCase/DeepNested: P99 is 16x the P50 latency
- **fox** on EdgeCase/EmptyObject: P99 is 441x the P50 latency
- **ant** on EdgeCase/EmptyObject: P99 is 30x the P50 latency
- **ant** on EdgeCase/LongKey256: P99 is 31x the P50 latency
- **fox** on EdgeCase/LongKey256: P99 is 587x the P50 latency
- **jaguar** on EdgeCase/LongKey256: P99 is 20x the P50 latency
- **ant** on List/100: P99 is 18x the P50 latency
- **owl** on List/100: P99 is 11x the P50 latency
- **falcon** on MixedWorkload/Balanced_50_50: P99 is 36x the P50 latency
- **jaguar** on MixedWorkload/Balanced_50_50: P99 is 15x the P50 latency
- **ant** on MixedWorkload/Balanced_50_50: P99 is 44x the P50 latency
- **gecko** on MixedWorkload/Balanced_50_50: P99 is 117x the P50 latency
- **narwhal** on MixedWorkload/Balanced_50_50: P99 is 18x the P50 latency
- **falcon** on MixedWorkload/ReadHeavy_90_10: P99 is 26x the P50 latency
- **jaguar** on MixedWorkload/ReadHeavy_90_10: P99 is 212x the P50 latency
- **ant** on MixedWorkload/ReadHeavy_90_10: P99 is 365x the P50 latency
- **narwhal** on MixedWorkload/ReadHeavy_90_10: P99 is 2069x the P50 latency
- **gecko** on MixedWorkload/ReadHeavy_90_10: P99 is 642x the P50 latency
- **falcon** on MixedWorkload/WriteHeavy_10_90: P99 is 68x the P50 latency
- **gecko** on MixedWorkload/WriteHeavy_10_90: P99 is 84x the P50 latency
- **fox** on MixedWorkload/WriteHeavy_10_90: P99 is 31x the P50 latency
- **jaguar** on ParallelRead/1KB/C1: P99 is 18x the P50 latency
- **falcon** on ParallelRead/1KB/C1: P99 is 10x the P50 latency
- **gecko** on ParallelRead/1KB/C1: P99 is 56x the P50 latency
- **narwhal** on ParallelRead/1KB/C1: P99 is 26x the P50 latency
- **jaguar** on ParallelRead/1KB/C10: P99 is 26x the P50 latency
- **falcon** on ParallelRead/1KB/C10: P99 is 34x the P50 latency
- **gecko** on ParallelRead/1KB/C10: P99 is 79x the P50 latency
- **narwhal** on ParallelRead/1KB/C10: P99 is 13x the P50 latency
- **ant** on ParallelRead/1KB/C10: P99 is 25x the P50 latency
- **fox** on ParallelRead/1KB/C10: P99 is 49x the P50 latency
- **falcon** on ParallelRead/1KB/C100: P99 is 34x the P50 latency
- **jaguar** on ParallelRead/1KB/C100: P99 is 34x the P50 latency
- **gecko** on ParallelRead/1KB/C100: P99 is 86x the P50 latency
- **narwhal** on ParallelRead/1KB/C100: P99 is 31x the P50 latency
- **ant** on ParallelRead/1KB/C100: P99 is 23x the P50 latency
- **fox** on ParallelRead/1KB/C100: P99 is 49x the P50 latency
- **jaguar** on ParallelRead/1KB/C200: P99 is 38x the P50 latency
- **falcon** on ParallelRead/1KB/C200: P99 is 93x the P50 latency
- **gecko** on ParallelRead/1KB/C200: P99 is 124x the P50 latency
- **narwhal** on ParallelRead/1KB/C200: P99 is 27x the P50 latency
- **ant** on ParallelRead/1KB/C200: P99 is 21x the P50 latency
- **fox** on ParallelRead/1KB/C200: P99 is 189x the P50 latency
- **jaguar** on ParallelRead/1KB/C25: P99 is 35x the P50 latency
- **gecko** on ParallelRead/1KB/C25: P99 is 59x the P50 latency
- **falcon** on ParallelRead/1KB/C25: P99 is 52x the P50 latency
- **narwhal** on ParallelRead/1KB/C25: P99 is 22x the P50 latency
- **ant** on ParallelRead/1KB/C25: P99 is 34x the P50 latency
- **fox** on ParallelRead/1KB/C25: P99 is 41x the P50 latency
- **jaguar** on ParallelRead/1KB/C50: P99 is 41x the P50 latency
- **falcon** on ParallelRead/1KB/C50: P99 is 59x the P50 latency
- **gecko** on ParallelRead/1KB/C50: P99 is 35x the P50 latency
- **narwhal** on ParallelRead/1KB/C50: P99 is 24x the P50 latency
- **ant** on ParallelRead/1KB/C50: P99 is 37x the P50 latency
- **fox** on ParallelRead/1KB/C50: P99 is 103x the P50 latency
- **gecko** on ParallelWrite/1KB/C1: P99 is 42x the P50 latency
- **fox** on ParallelWrite/1KB/C1: P99 is 58x the P50 latency
- **jaguar** on ParallelWrite/1KB/C1: P99 is 13x the P50 latency
- **falcon** on ParallelWrite/1KB/C10: P99 is 25x the P50 latency
- **gecko** on ParallelWrite/1KB/C10: P99 is 73x the P50 latency
- **ant** on ParallelWrite/1KB/C10: P99 is 17x the P50 latency
- **jaguar** on ParallelWrite/1KB/C10: P99 is 83x the P50 latency
- **narwhal** on ParallelWrite/1KB/C10: P99 is 11x the P50 latency
- **fox** on ParallelWrite/1KB/C10: P99 is 114x the P50 latency
- **falcon** on ParallelWrite/1KB/C100: P99 is 103x the P50 latency
- **gecko** on ParallelWrite/1KB/C100: P99 is 602x the P50 latency
- **jaguar** on ParallelWrite/1KB/C100: P99 is 211x the P50 latency
- **falcon** on ParallelWrite/1KB/C200: P99 is 41x the P50 latency
- **falcon** on ParallelWrite/1KB/C25: P99 is 25x the P50 latency
- **gecko** on ParallelWrite/1KB/C25: P99 is 147x the P50 latency
- **jaguar** on ParallelWrite/1KB/C25: P99 is 269x the P50 latency
- **ant** on ParallelWrite/1KB/C25: P99 is 19x the P50 latency
- **falcon** on ParallelWrite/1KB/C50: P99 is 34x the P50 latency
- **gecko** on ParallelWrite/1KB/C50: P99 is 370x the P50 latency
- **jaguar** on ParallelWrite/1KB/C50: P99 is 425x the P50 latency
- **ant** on ParallelWrite/1KB/C50: P99 is 110x the P50 latency
- **falcon** on RangeRead/End_256KB: P99 is 11x the P50 latency
- **narwhal** on RangeRead/End_256KB: P99 is 13x the P50 latency
- **falcon** on RangeRead/Middle_256KB: P99 is 33x the P50 latency
- **gecko** on RangeRead/Middle_256KB: P99 is 23x the P50 latency
- **narwhal** on RangeRead/Middle_256KB: P99 is 13x the P50 latency
- **ant** on RangeRead/Middle_256KB: P99 is 22x the P50 latency
- **falcon** on RangeRead/Start_256KB: P99 is 14x the P50 latency
- **gecko** on RangeRead/Start_256KB: P99 is 32x the P50 latency
- **ant** on RangeRead/Start_256KB: P99 is 12x the P50 latency
- **narwhal** on RangeRead/Start_256KB: P99 is 14x the P50 latency
- **jaguar** on Read/1KB: P99 is 13x the P50 latency
- **gecko** on Read/1KB: P99 is 21x the P50 latency
- **fox** on Read/1KB: P99 is 21x the P50 latency
- **falcon** on Read/1MB: P99 is 16x the P50 latency
- **gecko** on Read/1MB: P99 is 18x the P50 latency
- **narwhal** on Read/1MB: P99 is 16x the P50 latency
- **narwhal** on Read/64KB: P99 is 23x the P50 latency
- **ant** on Read/64KB: P99 is 26x the P50 latency
- **gecko** on Read/64KB: P99 is 34x the P50 latency
- **kangaroo** on Stat: P99 is 13x the P50 latency
- **fox** on Stat: P99 is 12x the P50 latency
- **spider** on Stat: P99 is 16x the P50 latency
- **narwhal** on Stat: P99 is 12x the P50 latency
- **gecko** on Stat: P99 is 21x the P50 latency
- **owl** on Stat: P99 is 46x the P50 latency
- **jaguar** on Stat: P99 is 25x the P50 latency
- **ant** on Stat: P99 is 13x the P50 latency
- **gecko** on Write/1KB: P99 is 15x the P50 latency
- **fox** on Write/1KB: P99 is 59x the P50 latency
- **kangaroo** on Write/1KB: P99 is 372x the P50 latency
- **spider** on Write/1KB: P99 is 12x the P50 latency
- **falcon** on Write/1MB: P99 is 24x the P50 latency
- **ant** on Write/1MB: P99 is 100x the P50 latency
- **narwhal** on Write/1MB: P99 is 11x the P50 latency
- **gecko** on Write/1MB: P99 is 142x the P50 latency
- **jaguar** on Write/1MB: P99 is 817x the P50 latency
- **spider** on Write/1MB: P99 is 1233x the P50 latency
- **gecko** on Write/64KB: P99 is 105x the P50 latency
- **ant** on Write/64KB: P99 is 35x the P50 latency
- **jaguar** on Write/64KB: P99 is 10282x the P50 latency
- **spider** on Write/64KB: P99 is 40920x the P50 latency

## Resource Efficiency

### Runtime Resources

| Driver | Peak RSS | Go Heap | Disk Used | GC Cycles | Total Ops | Efficiency |
|--------|----------|---------|-----------|-----------|-----------|------------|
| ant | 10.8 GB | 30.7 GB | 6.3 GB | 94 | 6.3M/s | 569 ops/MB |
| falcon | 10.8 GB | 36.5 GB | 26.1 GB | 34 | 22.6M/s | 2.0K ops/MB |
| fox | 10.8 GB | 24.4 GB | 673.6 MB | 64 | 7.2M/s | 645 ops/MB |
| gecko | 10.8 GB | 44.2 GB | 7.2 GB | 80 | 12.7M/s | 1.1K ops/MB |
| jaguar | 10.8 GB | 37.0 GB | 9.0 GB | 72 | 13.3M/s | 1.2K ops/MB |
| kangaroo | 10.8 GB | 34.8 GB | 576.0 MB | 99 | 7.9M/s | 713 ops/MB |
| narwhal | 10.8 GB | 42.9 GB | 4.9 GB | 104 | 9.4M/s | 848 ops/MB |
| owl | 10.8 GB | 69.2 GB | 6.8 GB | 87 | 6.4M/s | 579 ops/MB |
| spider | 10.8 GB | 52.1 GB | 1.9 GB | 52 | 4.4M/s | 395 ops/MB |

> Peak RSS = process-level resident memory. Go Heap = Go runtime heap. Efficiency = total iterations / peak RSS.

**Memory Usage (Peak RSS, ascending):**
```
  ant      |######################################## 10.8 GB
  falcon   |######################################## 10.8 GB
  fox      |######################################## 10.8 GB
  gecko    |######################################## 10.8 GB
  jaguar   |######################################## 10.8 GB
  kangaroo |######################################## 10.8 GB
  narwhal  |######################################## 10.8 GB
  owl      |######################################## 10.8 GB
  spider   |######################################## 10.8 GB
```

## Error & Timeout Summary

### Errors

| Driver | Total Errors | Affected Operations | Last Error |
|--------|-------------|--------------------|-----------|
| owl | 6538340 | Write/1KB (1049671), Write/64KB (61111), Write/1MB (5119), Write/10MB (430), Wri... | owl: write buffer full, try again |
| fox | 1840209 | Read/1KB (48430), Read/64KB (61111), Read/1MB (11111), Read/10MB (1111), Read/10... | storage: not exist |
| spider | 3384 | Delete (3384) | storage: not exist |
| jaguar | 789 | Read/100MB (111), MixedWorkload/Balanced_50_50 (560), MixedWorkload/WriteHeavy_1... | storage: not exist |

## Recommendations

### Best for Write-Heavy Workloads

> **falcon** -- highest average write throughput across all object sizes.

| Rank | Driver | Avg Write Throughput |
|------|--------|---------------------|
| 1 | falcon | 7.4 GB/s |
| 2 | kangaroo | 1.5 GB/s |
| 3 | fox | 1.4 GB/s |
| 4 | owl | 1.0 GB/s |
| 5 | gecko | 455.5 MB/s |

### Best for Read-Heavy Workloads

> **kangaroo** -- highest average read throughput across all object sizes.

| Rank | Driver | Avg Read Throughput |
|------|--------|--------------------|
| 1 | kangaroo | 43.7 GB/s |
| 2 | jaguar | 10.8 GB/s |
| 3 | spider | 5.9 GB/s |
| 4 | falcon | 5.1 GB/s |
| 5 | gecko | 3.3 GB/s |

### Most Memory Efficient

> **ant** -- lowest peak RSS at 10.8 GB.

### Best Overall

> **falcon** -- won 20/48 competitive benchmarks.

| Rank | Driver | Wins |
|------|--------|------|
| 1 | falcon | 20 |
| 2 | jaguar | 11 |
| 3 | owl | 9 |
| 4 | ant | 5 |
| 5 | kangaroo | 3 |


---

*Generated by storage benchmark CLI*
