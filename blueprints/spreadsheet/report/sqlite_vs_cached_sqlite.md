# Spreadsheet Storage Benchmark Report

**Generated**: 2026-01-09T01:05:10+07:00

## System Information

| Property | Value |
|----------|-------|
| OS | darwin |
| Architecture | arm64 |
| CPUs | 10 |
| Go Version | go1.25.5 |
| GOMAXPROCS | 10 |

## Configuration

- **Drivers**: sqlite, cached_sqlite
- **Categories**: cells
- **Iterations**: 2
- **Warmup**: 1
- **Quick Mode**: true

## Summary

- **Total Duration**: 201ms
- **Benchmarks Run**: 16
- **By Category**:
  - cells: 16


## Cells Benchmarks

### BatchSet_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 19.5us | 5128205 cells/sec | 100 | 245 |
| sqlite | 197.1us | 507346 cells/sec | 100 | 441 |

**Fastest**: cached_sqlite

### BatchSet_500

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 66.9us | 7476636 cells/sec | 500 | 1059 |
| sqlite | 541.2us | 923895 cells/sec | 500 | 1671 |

**Fastest**: cached_sqlite

### GetByPositions_Dense_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 88.4us | 1131273 cells/sec | 100 | 0 |
| sqlite | 159.5us | 626881 cells/sec | 100 | 0 |

**Fastest**: cached_sqlite

### GetByPositions_Dense_20x20

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 35.5us | 11254291 cells/sec | 400 | 0 |
| sqlite | 798.9us | 500665 cells/sec | 400 | 0 |

**Fastest**: cached_sqlite

### GetByPositions_Sparse_10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 1.9us | 5277045 cells/sec | 10 | 0 |
| sqlite | 74.2us | 134795 cells/sec | 10 | 0 |

**Fastest**: cached_sqlite

### GetByPositions_Sparse_100

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 64.5us | 1549883 cells/sec | 100 | 0 |
| sqlite | 660.5us | 151405 cells/sec | 100 | 0 |

**Fastest**: cached_sqlite

### GetRange_100x50

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 21.45ms | 233151 cells/sec | 5000 | 0 |
| sqlite | 32.47ms | 153994 cells/sec | 5000 | 0 |

**Fastest**: cached_sqlite

### GetRange_10x10

| Driver | Duration | Throughput | Cells/Op | Allocs/Op |
|--------|----------|------------|----------|----------|
| cached_sqlite | 95.8us | 1043711 cells/sec | 100 | 0 |
| sqlite | 173.3us | 576994 cells/sec | 100 | 0 |

**Fastest**: cached_sqlite

## Driver Comparison

### Performance Wins by Category

| Category | cached_sqlite | sqlite |
|----------|------|------|
| cells | 8 | - |

### Overall Winners

- **1st**: cached_sqlite (8 wins)

### Relative Performance (vs Fastest)

| Driver | Avg Relative Time | Interpretation |
|--------|-------------------|----------------|
| cached_sqlite | 1.00x | Fastest or near-fastest |
| sqlite | 11.90x | Significantly slower |

## Recommendations

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

### Financial Modeling

**Recommended**: cached_sqlite

**Reasons**:
- Best performance for cell operations
- Efficient handling of formatted cells
- Good batch write performance for large models

