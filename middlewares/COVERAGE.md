# Test Coverage Plan for Mizu Middlewares

## Current Coverage Status

**Overall: 89.4%** (improved from 86.6%)

### Packages at 100%
| Package | Coverage |
|---------|----------|
| bodylimit | 100.0% |
| contenttype | 100.0% |
| h2c | 100.0% |
| helmet | 100.0% |
| keepalive | 100.0% |
| methodoverride | 100.0% |
| nocache | 100.0% |
| recover | 100.0% |
| rewrite | 100.0% |
| slash | 100.0% |
| timeout | 100.0% |
| timing | 100.0% |
| vary | 100.0% |
| xrequestedwith | 100.0% |

### Excellent Coverage (95-99%)
| Package | Coverage |
|---------|----------|
| oauth2 | 98.9% |
| cors | 98.0% |
| jwt | 98.0% |
| fallback | 98.2% |
| keyauth | 97.8% |
| metrics | 97.6% |
| hedge | 97.5% |
| sse | 97.4% |
| adaptive | 97.4% |
| lastmodified | 97.4% |
| filter | 97.3% |
| concurrency | 97.1% |
| cors2 | 96.9% |
| timezone | 96.8% |
| version | 96.1% |
| signature | 95.7% |
| fingerprint | 95.9% |
| requestid | 95.8% |
| bearerauth | 95.7% |
| bodydump | 95.6% |
| ipfilter | 95.6% |
| header | 95.5% |
| captcha | 95.3% |

### Good Coverage (90-95%)
| Package | Coverage |
|---------|----------|
| embed | 94.7% |
| etag | 94.6% |
| maintenance | 93.9% |
| redirect | 93.7% |
| bodyclose | 93.3% |
| expvar | 93.3% |
| mock | 93.2% |
| secure | 93.2% |
| realip | 93.0% |
| bot | 92.7% |
| bulkhead | 92.7% |
| cache | 92.7% |
| idempotency | 92.4% |
| spa | 92.3% |
| requestsize | 91.7% |
| mirror | 91.7% |
| responselog | 91.5% |
| responsesize | 91.3% |
| audit | 90.9% |
| conditional | 90.9% |
| graphql | 90.9% |
| retry | 90.7% |
| pprof | 90.5% |
| canary | 90.2% |

### Needs Improvement (80-90%)
| Package | Coverage |
|---------|----------|
| basicauth | 89.7% |
| csrf | 89.8% |
| prometheus | 88.9% |
| static | 88.6% |
| circuitbreaker | 88.3% |
| proxy | 88.2% |
| compress | 88.0% |
| requestlog | 88.0% |
| validator | 87.8% |
| xml | 87.8% |
| transformer | 87.9% |
| ratelimit | 87.5% |
| maxconns | 87.3% |
| throttle | 87.0% |
| forwarded | 87.2% |
| favicon | 86.8% |
| language | 86.5% |
| nonce | 86.2% |
| otel | 85.8% |
| multitenancy | 85.9% |
| feature | 85.5% |
| honeypot | 85.5% |
| trace | 85.2% |
| logger | 85.3% |
| errorpage | 84.8% |
| hypermedia | 83.8% |
| websocket | 83.8% |
| envelope | 83.6% |
| session | 83.5% |
| chaos | 83.3% |
| sanitizer | 82.8% |
| csrf2 | 82.3% |
| surrogate | 81.0% |

### Below 80%
| Package | Coverage |
|---------|----------|
| sentry | 79.0% |
| healthcheck | 76.6% |
| rbac | 76.1% |
| jsonrpc | 75.0% |
| oidc | 74.4% |
| msgpack | 70.8% |

## Improvements Made

| Package | Before | After | Change |
|---------|--------|-------|--------|
| h2c | 55.0% | 100.0% | +45.0% |
| oauth2 | 65.2% | 98.9% | +33.7% |
| jwt | 67.7% | 98.0% | +30.3% |
| signature | 67.5% | 95.7% | +28.2% |
| concurrency | 73.5% | 97.1% | +23.6% |
| embed | 71.1% | 94.7% | +23.6% |
| mirror | 72.9% | 91.7% | +18.8% |

## Running Coverage

```bash
# Full coverage report
go test -coverprofile=coverage.out ./middlewares/...
go tool cover -func=coverage.out

# HTML report
go tool cover -html=coverage.out -o coverage.html

# Single package with uncovered lines
go test -cover -coverprofile=pkg.out ./middlewares/h2c/
go tool cover -func=pkg.out

# Get total coverage
go tool cover -func=coverage.out | grep total
```

## Notes

- Many packages have functions that are difficult to test (error handling for rare conditions)
- Some packages rely on external services (sentry, oidc) which require complex mocking
- Async functions and cleanup goroutines are challenging to test with high coverage
