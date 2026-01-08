# Spreadsheet Storage Benchmark Report

**Generated**: 2026-01-09T01:14:41+07:00

## System Information

| Property | Value |
|----------|-------|
| OS | darwin |
| Architecture | arm64 |
| CPUs | 10 |
| Go Version | go1.25.5 |
| GOMAXPROCS | 10 |

## Configuration

- **Drivers**: optimized_cached_sqlite, cached_sqlite, sqlite
- **Categories**: all
- **Iterations**: 2
- **Warmup**: 1
- **Quick Mode**: true

## Summary

- **Total Duration**: 1.824s
- **Benchmarks Run**: 72
- **By Category**:
  - merge: 12
  - format: 9
  - query: 6
  - usecase: 9
  - cells: 24
  - rows: 12


## Cells Benchmarks

### BatchSet_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 17.2us | 5804167 cells/sec | 100 | 246 |
| optimized_cached_sqlite | 20.8us | 4809773 cells/sec | 100 | 246 |
| sqlite | 168.1us | 595019 cells/sec | 100 | 451 |

**Fastest**: cached_sqlite

### BatchSet_500

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 63.8us | 7835517 cells/sec | 500 | 1060 |
| optimized_cached_sqlite | 66.4us | 7530688 cells/sec | 500 | 1061 |
| sqlite | 554.1us | 902392 cells/sec | 500 | 1669 |

**Fastest**: cached_sqlite

### GetByPositions_Dense_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 38.9us | 2572347 cells/sec | 100 | 0 |
| optimized_cached_sqlite | 13.2us | 7559722 cells/sec | 100 | 0 |
| sqlite | 160.2us | 624185 cells/sec | 100 | 0 |

**Fastest**: optimized_cached_sqlite

### GetByPositions_Dense_20x20

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 36.5us | 10958904 cells/sec | 400 | 0 |
| optimized_cached_sqlite | 81.9us | 4883051 cells/sec | 400 | 0 |
| sqlite | 584.1us | 684760 cells/sec | 400 | 0 |

**Fastest**: cached_sqlite

### GetByPositions_Sparse_10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 1.5us | 6489293 cells/sec | 10 | 0 |
| optimized_cached_sqlite | 2.2us | 4528986 cells/sec | 10 | 0 |
| sqlite | 62.7us | 159574 cells/sec | 10 | 0 |

**Fastest**: cached_sqlite

### GetByPositions_Sparse_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 17.4us | 5748778 cells/sec | 100 | 0 |
| optimized_cached_sqlite | 20.5us | 4883051 cells/sec | 100 | 0 |
| sqlite | 674.0us | 148359 cells/sec | 100 | 0 |

**Fastest**: cached_sqlite

### GetRange_100x50

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 23.13ms | 216139 cells/sec | 5000 | 0 |
| optimized_cached_sqlite | 903.4us | 5534800 cells/sec | 5000 | 0 |
| sqlite | 30.99ms | 161320 cells/sec | 5000 | 0 |

**Fastest**: optimized_cached_sqlite

### GetRange_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 17.3us | 5790388 cells/sec | 100 | 0 |
| optimized_cached_sqlite | 35.4us | 2828534 cells/sec | 100 | 0 |
| sqlite | 167.3us | 597611 cells/sec | 100 | 0 |

**Fastest**: cached_sqlite

## Format Benchmarks

### BatchSet_NoFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 223.3us | 2239020 cells/sec | 500 | 0 |
| optimized_cached_sqlite | 190.8us | 2620092 cells/sec | 500 | 0 |
| sqlite | 1.51ms | 330283 cells/sec | 500 | 0 |

**Fastest**: optimized_cached_sqlite

### BatchSet_PartialFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 77.1us | 6486515 cells/sec | 500 | 0 |
| optimized_cached_sqlite | 81.6us | 6130306 cells/sec | 500 | 0 |
| sqlite | 555.3us | 900463 cells/sec | 500 | 0 |

**Fastest**: cached_sqlite

### BatchSet_WithFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 79.0us | 6329114 cells/sec | 500 | 0 |
| optimized_cached_sqlite | 94.0us | 5321527 cells/sec | 500 | 0 |
| sqlite | 702.3us | 711934 cells/sec | 500 | 0 |

**Fastest**: cached_sqlite

## Merge Benchmarks

### BatchCreateMerge_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 43.6us | 43625 | 10 |
| optimized_cached_sqlite | 64.9us | 64854 | 10 |
| sqlite | 42.8us | 42791 | 10 |

**Fastest**: sqlite

### BatchCreateMerge_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 95.1us | 95146 | 50 |
| optimized_cached_sqlite | 97.1us | 97104 | 50 |
| sqlite | 91.6us | 91604 | 50 |

**Fastest**: sqlite

### CreateMerge_Individual_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 144.3us | 144333 | 10 |
| optimized_cached_sqlite | 155.6us | 155562 | 10 |
| sqlite | 137.4us | 137417 | 10 |

**Fastest**: sqlite

### CreateMerge_Individual_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 702.3us | 702291 | 50 |
| optimized_cached_sqlite | 732.4us | 732374 | 50 |
| sqlite | 697.9us | 697854 | 50 |

**Fastest**: sqlite

## Query Benchmarks

### Query_NonEmpty_1000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 828.1us | 828104 | 1000 |
| optimized_cached_sqlite | 105.5us | 105458 | 1000 |
| sqlite | 2.41ms | 2410563 | 1000 |

**Fastest**: optimized_cached_sqlite

### Query_NonEmpty_5000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 18.32ms | 18318103 | 5000 |
| optimized_cached_sqlite | 586.9us | 586875 | 5000 |
| sqlite | 28.70ms | 28695521 | 5000 |

**Fastest**: optimized_cached_sqlite

## Rows Benchmarks

### ShiftCols_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 24.51ms | 24505250 | 1 |
| optimized_cached_sqlite | 25.20ms | 25196875 | 1 |
| sqlite | 19.14ms | 19138646 | 1 |

**Fastest**: sqlite

### ShiftCols_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 24.47ms | 24472854 | 10 |
| optimized_cached_sqlite | 24.00ms | 24001250 | 10 |
| sqlite | 18.32ms | 18324375 | 10 |

**Fastest**: sqlite

### ShiftRows_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 9.15ms | 9146374 | 1 |
| optimized_cached_sqlite | 8.94ms | 8943708 | 1 |
| sqlite | 9.42ms | 9420374 | 1 |

**Fastest**: optimized_cached_sqlite

### ShiftRows_100

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| cached_sqlite | 8.84ms | 8840187 | 100 |
| optimized_cached_sqlite | 8.63ms | 8634333 | 100 |
| sqlite | 7.64ms | 7640124 | 100 |

**Fastest**: sqlite

## Usecase Benchmarks

### Financial_Workbook

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 2.02ms | 988132 cells/sec | 2000 | 0 |
| optimized_cached_sqlite | 488.1us | 4097135 cells/sec | 2000 | 0 |
| sqlite | 8.43ms | 237328 cells/sec | 2000 | 0 |

**Fastest**: optimized_cached_sqlite

### Import_CSV_10000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 4.56ms | 2193654 cells/sec | 10000 | 0 |
| optimized_cached_sqlite | 4.94ms | 2024454 cells/sec | 10000 | 0 |
| sqlite | 75.03ms | 133272 cells/sec | 10000 | 0 |

**Fastest**: cached_sqlite

### Import_CSV_50000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 36.34ms | 1376036 cells/sec | 50000 | 0 |
| optimized_cached_sqlite | 24.73ms | 2021875 cells/sec | 50000 | 0 |
| sqlite | 374.49ms | 133514 cells/sec | 50000 | 0 |

**Fastest**: optimized_cached_sqlite

## Driver Comparison

### Performance Wins by Category

| Category | cached_sqlite | optimized_cached_sqlite | sqlite |
|----------|------|------|------|
| cells | 6 | 2 | - |
| format | 2 | 1 | - |
| merge | - | - | 4 |
| query | - | 2 | - |
| rows | - | 1 | 3 |
| usecase | 1 | 2 | - |

### Overall Winners

- **1st**: cached_sqlite (9 wins)
- **2nd**: optimized_cached_sqlite (8 wins)
- **3rd**: sqlite (7 wins)

### Relative Performance (vs Fastest)

| Driver | Avg Relative Time | Interpretation |
|--------|-------------------|----------------|
| cached_sqlite | 3.85x | Significantly slower |
| optimized_cached_sqlite | 1.21x | Competitive |
| sqlite | 13.44x | Significantly slower |

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

