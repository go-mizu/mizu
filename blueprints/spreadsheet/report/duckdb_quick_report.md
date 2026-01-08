# Spreadsheet Storage Benchmark Report

**Generated**: 2026-01-09T00:35:22+07:00

## System Information

| Property | Value |
|----------|-------|
| OS | darwin |
| Architecture | arm64 |
| CPUs | 10 |
| Go Version | go1.25.5 |
| GOMAXPROCS | 10 |

## Configuration

- **Drivers**: duckdb
- **Categories**: all
- **Iterations**: 2
- **Warmup**: 1
- **Quick Mode**: true

## Summary

- **Total Duration**: 3m41.861s
- **Benchmarks Run**: 24
- **By Category**:
  - cells: 8
  - rows: 4
  - merge: 4
  - format: 3
  - query: 2
  - usecase: 3

- **Errors**: 3 benchmarks failed
  - rows/ShiftRows_1 (duckdb): Constraint Error: Duplicate key "sheet_id: 01KEFATZQR6SVMAJP4H0V7K1ZD, row_num: 87, col_num: 10" violates unique constraint.
  - rows/ShiftCols_1 (duckdb): Constraint Error: Duplicate key "sheet_id: 01KEFAV68VX48H2ECCRG4FE9Z7, row_num: 38, col_num: 46" violates unique constraint.
  - rows/ShiftCols_10 (duckdb): Constraint Error: Duplicate key "sheet_id: 01KEFAVDG65GFBX8XMG10DXDF1, row_num: 66, col_num: 47" violates unique constraint.

## Cells Benchmarks

### BatchSet_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 86.48ms | 1156 cells/sec | 100 | 6914 |

**Fastest**: duckdb

### BatchSet_500

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 469.78ms | 1064 cells/sec | 500 | 34217 |

**Fastest**: duckdb

### GetByPositions_Dense_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 542.1us | 184466 cells/sec | 100 | 0 |

**Fastest**: duckdb

### GetByPositions_Dense_20x20

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 1.38ms | 290715 cells/sec | 400 | 0 |

**Fastest**: duckdb

### GetByPositions_Sparse_10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 565.0us | 17699 cells/sec | 10 | 0 |

**Fastest**: duckdb

### GetByPositions_Sparse_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 607.0us | 164750 cells/sec | 100 | 0 |

**Fastest**: duckdb

### GetRange_100x50

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 17.29ms | 289171 cells/sec | 5000 | 0 |

**Fastest**: duckdb

### GetRange_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 729.4us | 137092 cells/sec | 100 | 0 |

**Fastest**: duckdb

## Format Benchmarks

### BatchSet_NoFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 786.29ms | 636 cells/sec | 500 | 0 |

**Fastest**: duckdb

### BatchSet_PartialFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 803.07ms | 623 cells/sec | 500 | 0 |

**Fastest**: duckdb

### BatchSet_WithFormat

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 746.41ms | 670 cells/sec | 500 | 0 |

**Fastest**: duckdb

## Merge Benchmarks

### BatchCreateMerge_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 2.26ms | 2256749 | 10 |

**Fastest**: duckdb

### BatchCreateMerge_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 7.82ms | 7815979 | 50 |

**Fastest**: duckdb

### CreateMerge_Individual_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 4.36ms | 4361521 | 10 |

**Fastest**: duckdb

### CreateMerge_Individual_50

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 19.78ms | 19775854 | 50 |

**Fastest**: duckdb

## Query Benchmarks

### Query_NonEmpty_1000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 4.09ms | 4093541 | 1000 |

**Fastest**: duckdb

### Query_NonEmpty_5000

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 17.47ms | 17469458 | 5000 |

**Fastest**: duckdb

## Rows Benchmarks

### ShiftCols_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | ERROR | - | - |

### ShiftCols_10

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | ERROR | - | - |

### ShiftRows_1

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | ERROR | - | - |

### ShiftRows_100

| Driver | Duration | ns/op | Cells/Op |
|--------|----------|-------|----------|
| duckdb | 7.45ms | 7454166 | 100 |

**Fastest**: duckdb

## Usecase Benchmarks

### Financial_Workbook

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 3.09s | 648 cells/sec | 2000 | 0 |

**Fastest**: duckdb

### Import_CSV_10000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 14.67s | 682 cells/sec | 10000 | 0 |

**Fastest**: duckdb

### Import_CSV_50000

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| duckdb | 72.20s | 692 cells/sec | 50000 | 0 |

**Fastest**: duckdb

## Driver Comparison

### Performance Wins by Category

| Category | duckdb |
|----------|------|
| cells | 8 |
| format | 3 |
| merge | 4 |
| query | 2 |
| rows | 1 |
| usecase | 3 |

### Overall Winners

- **1st**: duckdb (21 wins)

### Relative Performance (vs Fastest)

| Driver | Avg Relative Time | Interpretation |
|--------|-------------------|----------------|
| duckdb | 1.00x | Fastest or near-fastest |

## Recommendations

### Report Generation

**Recommended**: duckdb

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

### Financial Modeling

**Recommended**: duckdb

**Reasons**:
- Best performance for cell operations
- Efficient handling of formatted cells
- Good batch write performance for large models

### Data Import Pipeline

**Recommended**: duckdb

**Reasons**:
- Highest batch import throughput
- Efficient handling of large datasets
- Good memory efficiency during bulk operations

