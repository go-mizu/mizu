# Spreadsheet Storage Benchmark Report

**Generated**: 2026-01-09T08:07:43+07:00

## System Information

| Property | Value |
|----------|-------|
| OS | darwin |
| Architecture | arm64 |
| CPUs | 10 |
| Go Version | go1.25.5 |
| GOMAXPROCS | 10 |

## Configuration

- **Drivers**: cached_sqlite, sqlite
- **Categories**: all
- **Iterations**: 2
- **Warmup**: 1
- **Quick Mode**: true

## Summary

- **Total Duration**: 1.526s
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
| cached_sqlite | 21.4us | 4664832 cells/sec | 100 | 248 |
| sqlite | 204.1us | 490047 cells/sec | 100 | 451 |

**Fastest**: cached_sqlite

### BatchSet_500

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 65.5us | 7628812 cells/sec | 500 | 1063 |
| sqlite | 533.5us | 937133 cells/sec | 500 | 1670 |

**Fastest**: cached_sqlite

### GetByPositions_Dense_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 1.7us | 57836900 cells/sec | 100 | 0 |
| sqlite | 149.1us | 670862 cells/sec | 100 | 0 |

**Fastest**: cached_sqlite

### GetByPositions_Dense_20x20

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 1.2us | 336983993 cells/sec | 400 | 0 |
| sqlite | 806.9us | 495714 cells/sec | 400 | 0 |

**Fastest**: cached_sqlite

### GetByPositions_Sparse_10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 333ns | 30030030 cells/sec | 10 | 0 |
| sqlite | 79.1us | 126416 cells/sec | 10 | 0 |

**Fastest**: cached_sqlite

### GetByPositions_Sparse_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 729ns | 137174211 cells/sec | 100 | 0 |
| sqlite | 661.6us | 151143 cells/sec | 100 | 0 |

**Fastest**: cached_sqlite

### GetRange_100x50

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 188ns | 26595744681 cells/sec | 5000 | 0 |
| sqlite | 30.22ms | 165437 cells/sec | 5000 | 0 |

**Fastest**: cached_sqlite

### GetRange_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 104ns | 961538462 cells/sec | 100 | 0 |
| sqlite | 164.1us | 609448 cells/sec | 100 | 0 |

**Fastest**: cached_sqlite

## Format Benchmarks

### BatchSet_NoFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 62.0us | 8067249 cells/sec | 500 | 0 |
| sqlite | 1.72ms | 291542 cells/sec | 500 | 0 |

**Fastest**: cached_sqlite

### BatchSet_PartialFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 80.4us | 6219292 cells/sec | 500 | 0 |
| sqlite | 510.5us | 979513 cells/sec | 500 | 0 |

**Fastest**: cached_sqlite

### BatchSet_WithFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 68.6us | 7288205 cells/sec | 500 | 0 |
| sqlite | 848.3us | 589420 cells/sec | 500 | 0 |

**Fastest**: cached_sqlite

## Merge Benchmarks

### BatchCreateMerge_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 57.7us | 57708 | 10 |
| sqlite | 46.8us | 46750 | 10 |

**Fastest**: sqlite

### BatchCreateMerge_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 96.6us | 96583 | 50 |
| sqlite | 93.2us | 93208 | 50 |

**Fastest**: sqlite

### CreateMerge_Individual_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 161.7us | 161749 | 10 |
| sqlite | 140.6us | 140563 | 10 |

**Fastest**: sqlite

### CreateMerge_Individual_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 722.3us | 722271 | 50 |
| sqlite | 721.8us | 721791 | 50 |

**Fastest**: sqlite

## Query Benchmarks

### Query_NonEmpty_1000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 12.3us | 12271 | 1000 |
| sqlite | 2.25ms | 2245417 | 1000 |

**Fastest**: cached_sqlite

### Query_NonEmpty_5000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 26.1us | 26083 | 5000 |
| sqlite | 26.14ms | 26140271 | 5000 |

**Fastest**: cached_sqlite

## Rows Benchmarks

### ShiftCols_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 26.52ms | 26519771 | 1 |
| sqlite | 18.96ms | 18963875 | 1 |

**Fastest**: sqlite

### ShiftCols_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 25.78ms | 25776145 | 10 |
| sqlite | 18.90ms | 18901312 | 10 |

**Fastest**: sqlite

### ShiftRows_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 9.15ms | 9148458 | 1 |
| sqlite | 10.85ms | 10854104 | 1 |

**Fastest**: cached_sqlite

### ShiftRows_100

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 8.98ms | 8982104 | 100 |
| sqlite | 7.14ms | 7137374 | 100 |

**Fastest**: sqlite

## Usecase Benchmarks

### Financial_Workbook

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 330.6us | 6048765 cells/sec | 2000 | 0 |
| sqlite | 8.98ms | 222685 cells/sec | 2000 | 0 |

**Fastest**: cached_sqlite

### Import_CSV_10000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 5.11ms | 1956955 cells/sec | 10000 | 0 |
| sqlite | 76.40ms | 130888 cells/sec | 10000 | 0 |

**Fastest**: cached_sqlite

### Import_CSV_50000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 26.52ms | 1885304 cells/sec | 50000 | 0 |
| sqlite | 392.85ms | 127276 cells/sec | 50000 | 0 |

**Fastest**: cached_sqlite

## Driver Comparison

### Performance Wins by Category

| Category | cached_sqlite | sqlite |
|----------|------|------|
| cells | 8 | - |
| format | 3 | - |
| merge | - | 4 |
| query | 2 | - |
| rows | 1 | 3 |
| usecase | 3 | - |

### Overall Winners

- **1st**: cached_sqlite (17 wins)
- **2nd**: sqlite (7 wins)

### Relative Performance (vs Fastest)

| Driver | Avg Relative Time | Interpretation |
|--------|-------------------|----------------|
| cached_sqlite | 1.06x | Fastest or near-fastest |
| sqlite | 6898.47x | Significantly slower |

## Recommendations

### Financial Modeling

**Recommended**: cached_sqlite

**Reasons**:
- Best performance for cell operations
- Efficient handling of formatted cells
- Good batch write performance for large models

### Data Import Pipeline

**Recommended**: cached_sqlite

**Reasons**:
- Highest batch import throughput
- Efficient handling of large datasets
- Good memory efficiency during bulk operations

### Report Generation

**Recommended**: cached_sqlite

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

