# Spreadsheet Storage Benchmark Report

**Generated**: 2026-01-09T00:39:17+07:00

## System Information

| Property | Value |
|----------|-------|
| OS | darwin |
| Architecture | arm64 |
| CPUs | 10 |
| Go Version | go1.25.5 |
| GOMAXPROCS | 10 |

## Configuration

- **Drivers**: sqlite, swandb
- **Categories**: all
- **Iterations**: 5
- **Warmup**: 3
- **Quick Mode**: false

## Summary

- **Total Duration**: 11m18.54s
- **Benchmarks Run**: 80
- **By Category**:
  - merge: 12
  - format: 6
  - query: 6
  - usecase: 14
  - cells: 28
  - rows: 14


## Cells Benchmarks

### BatchSet_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 171.3us | 583631 cells/sec | 100 | 442 |
| swandb | 2.84ms | 35271 cells/sec | 100 | 573 |

**Fastest**: sqlite

### BatchSet_1000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 923.3us | 1083053 cells/sec | 1000 | 3166 |
| swandb | 3.48ms | 287359 cells/sec | 1000 | 3305 |

**Fastest**: sqlite

### BatchSet_10000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 11.77ms | 849797 cells/sec | 10000 | 30306 |
| swandb | 12.87ms | 777120 cells/sec | 10000 | 30428 |

**Fastest**: sqlite

### BatchSet_500

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 521.0us | 959603 cells/sec | 500 | 1667 |
| swandb | 3.04ms | 164320 cells/sec | 500 | 1791 |

**Fastest**: sqlite

### BatchSet_5000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 5.66ms | 883629 cells/sec | 5000 | 15252 |
| swandb | 7.64ms | 654656 cells/sec | 5000 | 15361 |

**Fastest**: sqlite

### GetByPositions_Dense_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 153.6us | 650970 cells/sec | 100 | 0 |
| swandb | 276.6us | 361490 cells/sec | 100 | 0 |

**Fastest**: sqlite

### GetByPositions_Dense_20x20

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 624.4us | 640581 cells/sec | 400 | 0 |
| swandb | 762.2us | 524808 cells/sec | 400 | 0 |

**Fastest**: sqlite

### GetByPositions_Dense_50x20

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 1.56ms | 640588 cells/sec | 1000 | 0 |
| swandb | 1.86ms | 538329 cells/sec | 1000 | 0 |

**Fastest**: sqlite

### GetByPositions_Sparse_10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 64.7us | 154679 cells/sec | 10 | 0 |
| swandb | 1.39ms | 7174 cells/sec | 10 | 0 |

**Fastest**: sqlite

### GetByPositions_Sparse_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 620.8us | 161078 cells/sec | 100 | 0 |
| swandb | 12.87ms | 7772 cells/sec | 100 | 0 |

**Fastest**: sqlite

### GetByPositions_Sparse_50

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 302.9us | 165049 cells/sec | 50 | 0 |
| swandb | 6.51ms | 7679 cells/sec | 50 | 0 |

**Fastest**: sqlite

### GetRange_100x50

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 30.38ms | 164588 cells/sec | 5000 | 0 |
| swandb | 29.19ms | 171291 cells/sec | 5000 | 0 |

**Fastest**: swandb

### GetRange_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 217.6us | 459664 cells/sec | 100 | 0 |
| swandb | 357.1us | 280021 cells/sec | 100 | 0 |

**Fastest**: sqlite

### GetRange_500x100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 1.28s | 39055 cells/sec | 50000 | 0 |
| swandb | 1.64s | 30434 cells/sec | 50000 | 0 |

**Fastest**: sqlite

## Format Benchmarks

### BatchSet_NoFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 1.37ms | 730340 cells/sec | 1000 | 0 |
| swandb | 5.56ms | 179758 cells/sec | 1000 | 0 |

**Fastest**: sqlite

### BatchSet_PartialFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 1.37ms | 728518 cells/sec | 1000 | 0 |
| swandb | 4.91ms | 203531 cells/sec | 1000 | 0 |

**Fastest**: sqlite

### BatchSet_WithFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 2.76ms | 361894 cells/sec | 1000 | 0 |
| swandb | 6.43ms | 155564 cells/sec | 1000 | 0 |

**Fastest**: sqlite

## Merge Benchmarks

### BatchCreateMerge_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 57.1us | 57141 | 10 |
| swandb | 1.77ms | 1771325 | 10 |

**Fastest**: sqlite

### BatchCreateMerge_100

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 271.6us | 271558 | 100 |
| swandb | 16.06ms | 16063125 | 100 |

**Fastest**: sqlite

### BatchCreateMerge_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 152.8us | 152841 | 50 |
| swandb | 8.33ms | 8325783 | 50 |

**Fastest**: sqlite

### CreateMerge_Individual_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 181.4us | 181425 | 10 |
| swandb | 3.29ms | 3289208 | 10 |

**Fastest**: sqlite

### CreateMerge_Individual_100

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 2.44ms | 2439191 | 100 |
| swandb | 31.90ms | 31900458 | 100 |

**Fastest**: sqlite

### CreateMerge_Individual_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 1.51ms | 1507566 | 50 |
| swandb | 14.44ms | 14440974 | 50 |

**Fastest**: sqlite

## Query Benchmarks

### Query_NonEmpty_1000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 3.37ms | 3368308 | 1000 |
| swandb | 3.41ms | 3414433 | 1000 |

**Fastest**: sqlite

### Query_NonEmpty_10000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 172.03ms | 172033308 | 10000 |
| swandb | 129.46ms | 129463900 | 10000 |

**Fastest**: swandb

### Query_NonEmpty_5000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 34.33ms | 34325324 | 5000 |
| swandb | 33.22ms | 33220416 | 5000 |

**Fastest**: swandb

## Rows Benchmarks

### ShiftCols_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 24.23ms | 24234866 | 1 |
| swandb | 26.72ms | 26723092 | 1 |

**Fastest**: sqlite

### ShiftCols_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 19.86ms | 19857475 | 10 |
| swandb | 21.50ms | 21498933 | 10 |

**Fastest**: sqlite

### ShiftCols_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 13.38ms | 13379166 | 50 |
| swandb | 16.38ms | 16382508 | 50 |

**Fastest**: sqlite

### ShiftRows_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 10.45ms | 10452658 | 1 |
| swandb | 12.60ms | 12601916 | 1 |

**Fastest**: sqlite

### ShiftRows_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 10.47ms | 10466266 | 10 |
| swandb | 12.49ms | 12486316 | 10 |

**Fastest**: sqlite

### ShiftRows_100

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 7.58ms | 7584816 | 100 |
| swandb | 8.23ms | 8229458 | 100 |

**Fastest**: sqlite

### ShiftRows_1000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 5.34ms | 5343833 | 1000 |
| swandb | 7.03ms | 7029950 | 1000 |

**Fastest**: sqlite

## Usecase Benchmarks

### Bulk_Operations

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 645.57ms | 645569816 | 50000 |
| swandb | 684.16ms | 684157366 | 50000 |

**Fastest**: sqlite

### Financial_Workbook

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 84.30ms | 296546 cells/sec | 25000 | 0 |
| swandb | 82.91ms | 301517 cells/sec | 25000 | 0 |

**Fastest**: swandb

### Import_CSV_10000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 89.74ms | 111433 cells/sec | 10000 | 0 |
| swandb | 101.88ms | 98152 cells/sec | 10000 | 0 |

**Fastest**: sqlite

### Import_CSV_100000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 958.99ms | 104276 cells/sec | 100000 | 0 |
| swandb | 1.09s | 91772 cells/sec | 100000 | 0 |

**Fastest**: sqlite

### Import_CSV_50000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 501.34ms | 99732 cells/sec | 50000 | 0 |
| swandb | 583.37ms | 85710 cells/sec | 50000 | 0 |

**Fastest**: sqlite

### Report_Generation

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| sqlite | 44.78s | 44783882441 | 300000 |
| swandb | 78.62s | 78618750591 | 300000 |

**Fastest**: sqlite

### Sparse_Data

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| sqlite | 447.02ms | 447407 cells/sec | 200000 | 0 |
| swandb | 790.61ms | 252970 cells/sec | 200000 | 0 |

**Fastest**: sqlite

## Driver Comparison

### Performance Wins by Category

| Category | sqlite | swandb |
|----------|------|------|
| cells | 13 | 1 |
| format | 3 | - |
| merge | 6 | - |
| query | 1 | 2 |
| rows | 7 | - |
| usecase | 6 | 1 |

### Overall Winners

- **1st**: sqlite (36 wins)
- **2nd**: swandb (4 wins)

### Relative Performance (vs Fastest)

| Driver | Avg Relative Time | Interpretation |
|--------|-------------------|----------------|
| sqlite | 1.01x | Fastest or near-fastest |
| swandb | 7.90x | Significantly slower |

## Recommendations

### Financial Modeling

**Recommended**: sqlite

**Reasons**:
- Best performance for cell operations
- Efficient handling of formatted cells
- Good batch write performance for large models

### Data Import Pipeline

**Recommended**: sqlite

**Reasons**:
- Highest batch import throughput
- Efficient handling of large datasets
- Good memory efficiency during bulk operations

### Report Generation

**Recommended**: sqlite

**Reasons**:
- Fast range query performance
- Efficient large data retrieval
- Good aggregation query support

### Desktop/Embedded Use

**Recommended**: sqlite

**Reasons**:
- Zero server configuration required
- Single-file database deployment
- Good single-user performance
- WAL mode for concurrent reads

