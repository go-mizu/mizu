# 0668 Benchmark Vector

Date: 2026-03-05 06:38:08 +07

## Best Open Source Benchmark

Best baseline: **ANN-Benchmarks** ([GitHub](https://github.com/erikbern/ann-benchmarks), [Website](https://ann-benchmarks.com/)).

Reason:
- Widely accepted ANN benchmark baseline in the community.
- Emphasizes recall/latency/QPS trade-offs.
- Vendor-neutral and reproducible.

Alternatives considered:
- VectorDBBench ([GitHub](https://github.com/zilliztech/VectorDBBench)) for DB operational scenarios.
- Big ANN Benchmarks ([GitHub](https://github.com/harsha-simhadri/big-ann-benchmarks)) for billion-scale tracks.

## Harness

Implemented command: `cmd/vector-bench`.

Workload parameters:
- corpus=1000
- queries=50
- dim=64
- k=10
- seed=42

## Results

| Driver | Status | compose_up_s | ready_s | init_ms | index_ms | search_p50_ms | search_p95_ms | qps | recall@10 |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| qdrant | PASS | 0.293 | 0.049 | 0.0 | 64.0 | 46.0 | 47.2 | 22.3 | 1.000 |
| weaviate | PASS | 0.469 | 9.268 | 0.0 | 277.0 | 1.0 | 1.3 | 954.0 | 1.000 |
| milvus | PASS | 0.346 | 2.712 | 0.0 | 114.7 | 4.0 | 6.3 | 48.9 | 0.000 |
| chroma | PASS | 0.236 | 0.035 | 0.0 | 286.8 | 2.1 | 2.3 | 462.5 | 1.000 |
| elasticsearch | PASS | 0.240 | 6.024 | 0.0 | 1181.1 | 3.6 | 6.6 | 197.3 | 0.864 |
| opensearch | PASS | 0.280 | 7.145 | 0.0 | 39705.8 | 3.0 | 5.7 | 229.3 | 1.000 |
| meilisearch | PASS | 0.229 | 0.033 | 0.0 | 2260.2 | 1.7 | 2.7 | 519.9 | 1.000 |
| typesense | PASS | 0.212 | 3.308 | 0.0 | 90.5 | 1.3 | 1.8 | 736.1 | 0.966 |
| pgvector | PASS | 0.231 | 0.001 | 0.0 | 476.6 | 0.3 | 0.5 | 2594.4 | 0.122 |
| solr | PASS | 0.216 | 1.522 | 0.0 | 1708.5 | 4.1 | 12.3 | 186.6 | 0.938 |
## Reproduce

```bash
go run ./cmd/vector-bench \
  -manage-compose=true \
  -drivers qdrant,weaviate,milvus,chroma,elasticsearch,opensearch,meilisearch,typesense,pgvector,solr \
  -corpus 1000 -queries 50 -dim 64 -k 10 -seed 42 \
  -report spec/0668_benchmark_vector.md
```
