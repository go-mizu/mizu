# Spreadsheet Storage Benchmark Report

**Generated**: 2026-01-09T01:26:04+07:00

## System Information

| Property | Value |
|----------|-------|
| OS | darwin |
| Architecture | arm64 |
| CPUs | 10 |
| Go Version | go1.25.5 |
| GOMAXPROCS | 10 |

## Configuration

- **Drivers**: cached_sqlite, optimized_cached_sqlite
- **Categories**: all
- **Iterations**: 2
- **Warmup**: 1
- **Quick Mode**: true

## Summary

- **Total Duration**: 626ms
- **Benchmarks Run**: 48
- **By Category**:
  - cells: 16
  - rows: 8
  - merge: 8
  - format: 6
  - query: 4
  - usecase: 6


## Cells Benchmarks

### BatchSet_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 23.9us | 4184801 cells/sec | 100 | 248 |
| optimized_cached_sqlite | 17.2us | 5804167 cells/sec | 100 | 248 |

**Fastest**: optimized_cached_sqlite

### BatchSet_500

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 63.4us | 7886933 cells/sec | 500 | 1062 |
| optimized_cached_sqlite | 65.8us | 7604563 cells/sec | 500 | 1063 |

**Fastest**: cached_sqlite

### GetByPositions_Dense_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 14.8us | 6770022 cells/sec | 100 | 0 |
| optimized_cached_sqlite | 1.2us | 80000000 cells/sec | 100 | 0 |

**Fastest**: optimized_cached_sqlite

### GetByPositions_Dense_20x20

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 53.5us | 7476636 cells/sec | 400 | 0 |
| optimized_cached_sqlite | 4.5us | 89726335 cells/sec | 400 | 0 |

**Fastest**: optimized_cached_sqlite

### GetByPositions_Sparse_10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 1.9us | 5219207 cells/sec | 10 | 0 |
| optimized_cached_sqlite | 250ns | 40000000 cells/sec | 10 | 0 |

**Fastest**: optimized_cached_sqlite

### GetByPositions_Sparse_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 21.0us | 4762132 cells/sec | 100 | 0 |
| optimized_cached_sqlite | 3.4us | 29629630 cells/sec | 100 | 0 |

**Fastest**: optimized_cached_sqlite

### GetRange_100x50

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 23.73ms | 210705 cells/sec | 5000 | 0 |
| optimized_cached_sqlite | 187ns | 26737967914 cells/sec | 5000 | 0 |

**Fastest**: optimized_cached_sqlite

### GetRange_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 19.0us | 5251825 cells/sec | 100 | 0 |
| optimized_cached_sqlite | 104ns | 961538462 cells/sec | 100 | 0 |

**Fastest**: optimized_cached_sqlite

## Format Benchmarks

### BatchSet_NoFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 63.5us | 7879227 cells/sec | 500 | 0 |
| optimized_cached_sqlite | 62.8us | 7965462 cells/sec | 500 | 0 |

**Fastest**: optimized_cached_sqlite

### BatchSet_PartialFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 74.8us | 6688963 cells/sec | 500 | 0 |
| optimized_cached_sqlite | 72.9us | 6857112 cells/sec | 500 | 0 |

**Fastest**: optimized_cached_sqlite

### BatchSet_WithFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 72.9us | 6859181 cells/sec | 500 | 0 |
| optimized_cached_sqlite | 192.3us | 2599658 cells/sec | 500 | 0 |

**Fastest**: cached_sqlite

## Merge Benchmarks

### BatchCreateMerge_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 56.7us | 56708 | 10 |
| optimized_cached_sqlite | 47.6us | 47646 | 10 |

**Fastest**: optimized_cached_sqlite

### BatchCreateMerge_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 100.2us | 100208 | 50 |
| optimized_cached_sqlite | 95.8us | 95833 | 50 |

**Fastest**: optimized_cached_sqlite

### CreateMerge_Individual_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 161.0us | 160999 | 10 |
| optimized_cached_sqlite | 150.5us | 150500 | 10 |

**Fastest**: optimized_cached_sqlite

### CreateMerge_Individual_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 740.1us | 740104 | 50 |
| optimized_cached_sqlite | 729.0us | 729020 | 50 |

**Fastest**: optimized_cached_sqlite

## Query Benchmarks

### Query_NonEmpty_1000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 1.41ms | 1409250 | 1000 |
| optimized_cached_sqlite | 19.8us | 19750 | 1000 |

**Fastest**: optimized_cached_sqlite

### Query_NonEmpty_5000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 19.09ms | 19090521 | 5000 |
| optimized_cached_sqlite | 18.1us | 18146 | 5000 |

**Fastest**: optimized_cached_sqlite

## Rows Benchmarks

### ShiftCols_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 25.89ms | 25885250 | 1 |
| optimized_cached_sqlite | 27.19ms | 27187500 | 1 |

**Fastest**: cached_sqlite

### ShiftCols_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 24.57ms | 24571958 | 10 |
| optimized_cached_sqlite | 23.44ms | 23438958 | 10 |

**Fastest**: optimized_cached_sqlite

### ShiftRows_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 13.49ms | 13489146 | 1 |
| optimized_cached_sqlite | 8.89ms | 8890271 | 1 |

**Fastest**: optimized_cached_sqlite

### ShiftRows_100

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 8.65ms | 8654500 | 100 |
| optimized_cached_sqlite | 9.06ms | 9062333 | 100 |

**Fastest**: cached_sqlite

## Usecase Benchmarks

### Financial_Workbook

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 1.97ms | 1016518 cells/sec | 2000 | 0 |
| optimized_cached_sqlite | 333.4us | 5999250 cells/sec | 2000 | 0 |

**Fastest**: optimized_cached_sqlite

### Import_CSV_10000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 4.73ms | 2115973 cells/sec | 10000 | 0 |
| optimized_cached_sqlite | 4.74ms | 2108009 cells/sec | 10000 | 0 |

**Fastest**: cached_sqlite

### Import_CSV_50000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 34.42ms | 1452472 cells/sec | 50000 | 0 |
| optimized_cached_sqlite | 30.04ms | 1664700 cells/sec | 50000 | 0 |

**Fastest**: optimized_cached_sqlite

## Driver Comparison

### Performance Wins by Category

| Category | cached_sqlite | optimized_cached_sqlite |
|----------|------|------|
| cells | 1 | 7 |
| format | 1 | 2 |
| merge | - | 4 |
| query | - | 2 |
| rows | 2 | 2 |
| usecase | 1 | 2 |

### Overall Winners

- **1st**: optimized_cached_sqlite (19 wins)
- **2nd**: cached_sqlite (5 wins)

### Relative Performance (vs Fastest)

| Driver | Avg Relative Time | Interpretation |
|--------|-------------------|----------------|
| cached_sqlite | 5344.33x | Significantly slower |
| optimized_cached_sqlite | 1.07x | Fastest or near-fastest |

## Recommendations

### Financial Modeling

**Recommended**: optimized_cached_sqlite

**Reasons**:
- Best performance for cell operations
- Efficient handling of formatted cells
- Good batch write performance for large models

### Data Import Pipeline

**Recommended**: optimized_cached_sqlite

**Reasons**:
- Highest batch import throughput
- Efficient handling of large datasets
- Good memory efficiency during bulk operations

### Report Generation

**Recommended**: optimized_cached_sqlite

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

