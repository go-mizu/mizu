# Spreadsheet Storage Benchmark Report

**Generated**: 2026-01-09T00:28:31+07:00

## System Information

| Property | Value |
|----------|-------|
| OS | darwin |
| Architecture | arm64 |
| CPUs | 10 |
| Go Version | go1.25.5 |
| GOMAXPROCS | 10 |

## Configuration

- **Drivers**: duckdb, sqlite, swandb
- **Categories**: all
- **Iterations**: 2
- **Warmup**: 1
- **Quick Mode**: true

## Summary

- **Total Duration**: 3m29.29s
- **Benchmarks Run**: 72
- **By Category**:
  - cells: 24
  - rows: 12
  - merge: 12
  - format: 9
  - query: 6
  - usecase: 9

- **Errors**: 3 benchmarks failed
  - rows/ShiftRows_1 (duckdb): Constraint Error: Duplicate key "sheet_id: 01KEFAEH7KG3THXFVCFGP523MM, row_num: 87, col_num: 10" violates unique constraint.
  - rows/ShiftCols_1 (duckdb): Constraint Error: Duplicate key "sheet_id: 01KEFAEQCG18WVH4ANNVSH1E21, row_num: 38, col_num: 46" violates unique constraint.
  - rows/ShiftCols_10 (duckdb): Constraint Error: Duplicate key "sheet_id: 01KEFAEY8YN5DNNQY04GH1QS5N, row_num: 66, col_num: 47" violates unique constraint.

## Cells Benchmarks

### BatchSet_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 90.90ms | 1100 cells/sec | 100 | 6898 |
| sqlite | 204.2us | 489697 cells/sec | 100 | 443 |
| swandb | 2.48ms | 40274 cells/sec | 100 | 581 |

**Fastest**: sqlite

### BatchSet_500

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 531.48ms | 941 cells/sec | 500 | 34195 |
| sqlite | 576.8us | 866834 cells/sec | 500 | 1665 |
| swandb | 3.78ms | 132169 cells/sec | 500 | 1787 |

**Fastest**: sqlite

### GetByPositions_Dense_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 794.1us | 125935 cells/sec | 100 | 0 |
| sqlite | 197.9us | 505316 cells/sec | 100 | 0 |
| swandb | 359.3us | 278342 cells/sec | 100 | 0 |

**Fastest**: sqlite

### GetByPositions_Dense_20x20

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 1.78ms | 225209 cells/sec | 400 | 0 |
| sqlite | 952.2us | 420086 cells/sec | 400 | 0 |
| swandb | 1.18ms | 340233 cells/sec | 400 | 0 |

**Fastest**: sqlite

### GetByPositions_Sparse_10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 660.2us | 15147 cells/sec | 10 | 0 |
| sqlite | 74.6us | 134077 cells/sec | 10 | 0 |
| swandb | 1.57ms | 6354 cells/sec | 10 | 0 |

**Fastest**: sqlite

### GetByPositions_Sparse_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 786.1us | 127210 cells/sec | 100 | 0 |
| sqlite | 943.4us | 105995 cells/sec | 100 | 0 |
| swandb | 14.78ms | 6767 cells/sec | 100 | 0 |

**Fastest**: duckdb

### GetRange_100x50

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 17.32ms | 288655 cells/sec | 5000 | 0 |
| sqlite | 38.75ms | 129043 cells/sec | 5000 | 0 |
| swandb | 37.83ms | 132166 cells/sec | 5000 | 0 |

**Fastest**: duckdb

### GetRange_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 820.6us | 121855 cells/sec | 100 | 0 |
| sqlite | 220.0us | 454632 cells/sec | 100 | 0 |
| swandb | 503.5us | 198626 cells/sec | 100 | 0 |

**Fastest**: sqlite

## Format Benchmarks

### BatchSet_NoFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 685.58ms | 729 cells/sec | 500 | 0 |
| sqlite | 1.78ms | 281031 cells/sec | 500 | 0 |
| swandb | 5.05ms | 99037 cells/sec | 500 | 0 |

**Fastest**: sqlite

### BatchSet_PartialFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 709.15ms | 705 cells/sec | 500 | 0 |
| sqlite | 668.9us | 747500 cells/sec | 500 | 0 |
| swandb | 4.50ms | 111169 cells/sec | 500 | 0 |

**Fastest**: sqlite

### BatchSet_WithFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 699.73ms | 715 cells/sec | 500 | 0 |
| sqlite | 882.2us | 566787 cells/sec | 500 | 0 |
| swandb | 5.06ms | 98720 cells/sec | 500 | 0 |

**Fastest**: sqlite

## Merge Benchmarks

### BatchCreateMerge_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 1.44ms | 1437562 | 10 |
| sqlite | 67.1us | 67145 | 10 |
| swandb | 1.51ms | 1514625 | 10 |

**Fastest**: sqlite

### BatchCreateMerge_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 7.16ms | 7162146 | 50 |
| sqlite | 127.7us | 127687 | 50 |
| swandb | 6.63ms | 6626229 | 50 |

**Fastest**: sqlite

### CreateMerge_Individual_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 2.98ms | 2975958 | 10 |
| sqlite | 409.9us | 409854 | 10 |
| swandb | 3.03ms | 3034417 | 10 |

**Fastest**: sqlite

### CreateMerge_Individual_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 13.22ms | 13223979 | 50 |
| sqlite | 1.60ms | 1599895 | 50 |
| swandb | 16.61ms | 16608750 | 50 |

**Fastest**: sqlite

## Query Benchmarks

### Query_NonEmpty_1000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 5.06ms | 5055374 | 1000 |
| sqlite | 3.23ms | 3231333 | 1000 |
| swandb | 3.74ms | 3738583 | 1000 |

**Fastest**: sqlite

### Query_NonEmpty_5000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 17.60ms | 17603166 | 5000 |
| sqlite | 32.54ms | 32537771 | 5000 |
| swandb | 34.95ms | 34953270 | 5000 |

**Fastest**: duckdb

## Rows Benchmarks

### ShiftCols_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | ERROR | - | - |
| sqlite | 25.62ms | 25619312 | 1 |
| swandb | 28.27ms | 28270062 | 1 |

**Fastest**: sqlite

### ShiftCols_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | ERROR | - | - |
| sqlite | 25.70ms | 25695083 | 10 |
| swandb | 28.01ms | 28006229 | 10 |

**Fastest**: sqlite

### ShiftRows_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | ERROR | - | - |
| sqlite | 12.01ms | 12009375 | 1 |
| swandb | 11.46ms | 11456104 | 1 |

**Fastest**: swandb

### ShiftRows_100

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 7.43ms | 7432354 | 100 |
| sqlite | 9.62ms | 9617583 | 100 |
| swandb | 11.20ms | 11196125 | 100 |

**Fastest**: duckdb

## Usecase Benchmarks

### Financial_Workbook

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 2.81s | 711 cells/sec | 2000 | 0 |
| sqlite | 11.33ms | 176598 cells/sec | 2000 | 0 |
| swandb | 13.21ms | 151433 cells/sec | 2000 | 0 |

**Fastest**: sqlite

### Import_CSV_10000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 14.69s | 681 cells/sec | 10000 | 0 |
| sqlite | 102.18ms | 97862 cells/sec | 10000 | 0 |
| swandb | 111.36ms | 89801 cells/sec | 10000 | 0 |

**Fastest**: sqlite

### Import_CSV_50000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 64.78s | 772 cells/sec | 50000 | 0 |
| sqlite | 494.75ms | 101061 cells/sec | 50000 | 0 |
| swandb | 570.61ms | 87626 cells/sec | 50000 | 0 |

**Fastest**: sqlite

## Driver Comparison

### Performance Wins by Category

| Category | duckdb | sqlite | swandb |
|----------|------|------|------|
| cells | 2 | 6 | - |
| format | - | 3 | - |
| merge | - | 4 | - |
| query | 1 | 1 | - |
| rows | 1 | 2 | 1 |
| usecase | - | 3 | - |

### Overall Winners

- **1st**: sqlite (19 wins)
- **2nd**: duckdb (4 wins)
- **3rd**: swandb (1 wins)

### Relative Performance (vs Fastest)

| Driver | Avg Relative Time | Interpretation |
|--------|-------------------|----------------|
| duckdb | 202.17x | Significantly slower |
| sqlite | 1.11x | Competitive |
| swandb | 7.71x | Significantly slower |

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

