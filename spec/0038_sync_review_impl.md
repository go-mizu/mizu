# Sync Package Review Implementation Plan

This document provides a detailed verification and implementation plan for the issues identified in `spec/0037_sync_review.md`.

**Status: COMPLETED** - All items have been implemented and tested.

## Review Verification Summary

| # | Issue | Verified | Severity | Status |
|---|-------|----------|----------|--------|
| 1 | `Change.Cursor` never assigned | Partial | High | ✅ Documented in contract |
| 2 | `errCursorTooOld` never produced | Yes | High | ✅ Exported & implemented |
| 3 | `Applied.Put` errors ignored | Yes | High | ✅ Errors now handled |
| 4 | Push batch not atomic | Yes | Medium | ✅ Documented (as-is) |
| 5 | `[]byte` base64 in JSON | Yes | Medium | ✅ Using `json.RawMessage` |
| 6 | Store comment misleading | Yes | Low | ✅ Comment fixed |
| 7 | Unused error codes | Yes | Low | ✅ `codeOK` removed |
| 8 | Magic string `_default` | Yes | Low | ✅ `DefaultScope` constant |
| 9 | Limit unbounded | Yes | Medium | ✅ Max limits added |
| 10 | Time not injectable | Yes | Low | ✅ `Now` option added |
| 11 | Missing auth hook | Yes | Medium | ✅ `ScopeFunc` added |
| 12 | HTTP methods | N/A | Info | N/A (informational) |

---

## Detailed Verification

### 1. `Change.Cursor` is never assigned

**Review Claim:** `Log.Append` returns only the final cursor. Nothing populates `changes[i].Cursor`.

**Verification:**
- `sync.go:340` - Engine calls `e.log.Append(ctx, scope, changes)` which returns only final cursor
- `memory.go:146-148` - Memory implementation **DOES** assign cursors in-place:
  ```go
  for i := range changes {
      l.global++
      changes[i].Cursor = l.global
  }
  ```

**Finding:** PARTIALLY CORRECT. The memory implementation does assign cursors, but:
1. The `Log` interface contract (`sync.go:170-174`) doesn't document this requirement
2. Other `Log` implementations might not know to assign cursors
3. The API is implicit about mutation

**Recommendation:** Document explicitly in `Log.Append` that implementations must assign `Cursor` to each change, or change signature to return `(start, end uint64, error)`.

---

### 2. `errCursorTooOld` is defined but never produced

**Review Claim:** `handlePull()` checks for `errCursorTooOld` but nothing produces it.

**Verification:**
- `sync.go:89` - `errCursorTooOld` is defined (unexported)
- `sync.go:504-507` - `handlePull()` checks `errors.Is(err, errCursorTooOld)` to return HTTP 410
- `sync.go:388` - `Engine.Pull()` returns whatever `Log.Since()` returns
- `memory.go:160-181` - `Log.Since()` never returns this error

**Finding:** CORRECT. This is dead code. The error exists but:
1. `Engine.Pull` doesn't map any error to `errCursorTooOld`
2. `memory.Log.Since` returns `nil` even for trimmed cursors
3. No code path produces this error

**Recommendation:** Either:
- Export `ErrCursorTooOld` and document that `Log.Since` must return it when cursor is trimmed
- Or detect the condition in Engine (compare cursor vs oldest entry)

---

### 3. Idempotency storage errors are ignored

**Review Claim:** `Applied.Put` errors are discarded with `_ = e.applied.Put(...)`.

**Verification:**
- `sync.go:355-356`:
  ```go
  if e.applied != nil && mut.ID != "" {
      _ = e.applied.Put(ctx, scope, mut.ID, result)
  }
  ```

**Finding:** CORRECT. The error is explicitly ignored with blank identifier.

**Impact:** If `Applied.Put` fails:
1. The mutation was already applied to Store and Log
2. A replay could reapply the mutation (violates idempotency guarantee)

**Recommendation:** Return `OK=false` with internal error if `Applied.Put` fails, even after successful store/log operations. This maintains the idempotency contract.

---

### 4. Push does not provide atomicity across a batch

**Review Claim:** Push processes mutations one by one. If mutation 3 fails, 1 and 2 are committed.

**Verification:**
- `sync.go:279-292`:
  ```go
  for i, mut := range mutations {
      result := e.processMutation(ctx, mut)
      results[i] = result
      // ... continues regardless of result.OK
  }
  ```

**Finding:** CORRECT. Each mutation is processed independently. Failures don't roll back previous successes.

**Recommendation:** This is acceptable behavior but should be documented. Options:
1. Document "best effort per mutation" explicitly
2. Add `PushOne` for simpler single-mutation semantics
3. Future: Add optional transaction support via interface

---

### 5. `[]byte` in JSON will be base64 encoded

**Review Claim:** `Change.Data []byte` and snapshot data will be base64 encoded in JSON.

**Verification:**
- `sync.go:133`: `Data []byte json:"data,omitempty"`
- `sync.go:449`: `Data map[string]map[string][]byte json:"data"`
- Standard library `encoding/json` base64-encodes `[]byte` values

**Finding:** CORRECT. This creates:
1. DX friction for JS/TS clients (must decode base64)
2. ~33% wire overhead for binary data (base64 expansion)

**Recommendation:** Use `json.RawMessage` instead of `[]byte`:
```go
Data json.RawMessage `json:"data,omitempty"`
```
This keeps JSON payloads as plain JSON on the wire.

---

### 6. Store contract says "All data is stored as JSON bytes" but `Set` does not enforce JSON

**Review Claim:** The comment claims JSON storage but `Set` accepts any bytes.

**Verification:**
- `sync.go:151-152`:
  ```go
  // Store is the authoritative state store.
  // All data is stored as JSON bytes to avoid type ambiguity.
  ```
- `sync.go:158`: `Set(ctx context.Context, scope, entity, id string, data []byte) error`

**Finding:** CORRECT. No JSON validation occurs.

**Recommendation:** Either:
1. Add optional JSON validation in Store or Engine
2. Soften comment to "Data is typically JSON bytes" or "Data should be JSON bytes"

---

### 7. Error constants and codes: too many are unused or split-brain

**Review Claim:** `codeOK` is unused. Inconsistent error response format.

**Verification:**
- `sync.go:79`: `codeOK = ""` - **Unused** (grep confirms no references)
- Error responses vary:
  - `handlePush` returns `{"error": "..."}` for bad request (no code)
  - `handlePull` returns `{"code": "...", "error": "..."}` for cursor-too-old
  - Per-mutation results include `Code` field

**Finding:** CORRECT.

**Recommendation:**
1. Remove `codeOK` if not needed
2. Standardize error response format: always `{"code": "...", "error": "..."}`

---

### 8. Default scope constant should not be a magic string

**Review Claim:** `"_default"` appears in multiple places as a magic string.

**Verification:**
- `sync.go:286`, `307`, `381`, `404`: All use literal `"_default"`
- `sync_test.go:325`: Test expects `"_default"`

**Finding:** CORRECT. The string appears 4 times in sync.go.

**Recommendation:**
```go
const DefaultScope = "_default"
```
And use consistently throughout.

---

### 9. Limit handling should be bounded

**Review Claim:** Pull allows any limit value. Client could request millions.

**Verification:**
- `sync.go:383-385`:
  ```go
  if limit <= 0 {
      limit = 100
  }
  // No upper bound check
  ```
- Push body limited to 1MB but no mutation count limit

**Finding:** CORRECT.

**Recommendation:**
```go
if limit <= 0 { limit = 100 }
if limit > 1000 { limit = 1000 }
```
Also add max mutation count for Push.

---

### 10. Time source should be injectable for testing and determinism

**Review Claim:** `time.Now()` is hardcoded, preventing deterministic tests.

**Verification:**
- `sync.go:326`: `now := time.Now()`

**Finding:** CORRECT.

**Recommendation:**
```go
type Options struct {
    // ...
    Now func() time.Time // defaults to time.Now if nil
}
```

---

### 11. Missing authentication/authorization hook

**Review Claim:** Scope comes from mutation body, which is easy to spoof.

**Verification:**
- `sync.go:97-115`: `Mutation.Scope` is a JSON field from client
- `sync.go:305-308`: Engine uses scope directly without validation
- No hook in `Options` for auth/scope derivation

**Finding:** CORRECT. Any client can claim any scope.

**Recommendation:** Add a scope resolver hook:
```go
type Options struct {
    // ...
    ScopeFunc func(ctx context.Context, claimed string) (string, error)
}
```
Server can override scope based on JWT claims, session, etc.

---

### 12. HTTP routing and method choices

**Review Claim:** Using POST for pull/snapshot is fine but GET could be useful later.

**Finding:** Informational note, not an issue. Current design is valid.

---

## Implementation Priority

### Phase 1: Critical Correctness (High Priority)
1. Fix `errCursorTooOld` - either implement or remove dead code
2. Handle `Applied.Put` errors - maintain idempotency guarantee
3. Document cursor assignment contract in `Log.Append`

### Phase 2: DX and Wire Format (Medium Priority)
4. Switch `[]byte` to `json.RawMessage` for JSON payloads
5. Add max limits for Pull and Push batch size
6. Add `ScopeFunc` option for authorization

### Phase 3: Cleanup (Low Priority)
7. Add `DefaultScope` constant
8. Remove unused `codeOK`
9. Standardize error response format
10. Add injectable time source
11. Fix Store contract comment

---

## Code Changes Summary

```go
// 1. Export cursor error
var ErrCursorTooOld = errors.New("sync: cursor too old")

// 2. Add DefaultScope constant
const DefaultScope = "_default"

// 3. Update Options struct
type Options struct {
    Store    Store
    Log      Log
    Applied  Applied
    Mutator  Mutator
    Notify   Notifier
    Now      func() time.Time           // Injectable time source
    ScopeFunc func(context.Context, string) (string, error) // Auth hook
    MaxPullLimit int                    // Default 1000
    MaxPushBatch int                    // Default 100
}

// 4. Update Change struct
type Change struct {
    Cursor uint64          `json:"cursor"`
    Scope  string          `json:"scope"`
    Entity string          `json:"entity"`
    ID     string          `json:"id"`
    Op     Op              `json:"op"`
    Data   json.RawMessage `json:"data,omitempty"`  // Changed from []byte
    Time   time.Time       `json:"time"`
}

// 5. Handle Applied.Put error
if e.applied != nil && mut.ID != "" {
    if err := e.applied.Put(ctx, scope, mut.ID, result); err != nil {
        return Result{OK: false, Code: codeInternal, Error: "failed to store idempotency key"}
    }
}

// 6. Add limit bounds
if limit <= 0 { limit = 100 }
if e.maxPullLimit > 0 && limit > e.maxPullLimit { limit = e.maxPullLimit }
```
