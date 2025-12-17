Below is a targeted review focused on correctness, DX, and long term maintainability.

## What is strong already

* Clear package contract: authoritative, offline first, transport agnostic, pull by cursor.
* The “mutation pipeline” is explicit and maps well to typical offline sync systems.
* Interfaces are minimal and composable (Store, Log, Applied, Mutator, Notifier).
* The engine API (`Push`, `Pull`, `Snapshot`, `Mount`) is straightforward for app code.

## Correctness gaps to address

### 1) `Change.Cursor` is never assigned

`Change` includes `Cursor uint64`, but `Log.Append` returns only the final cursor. Nothing populates `changes[i].Cursor`, so clients will receive cursor zero for each change unless the Log implementation mutates the slice (which is not stated, and is a surprising side effect).

Fix options (pick one, but document it explicitly):

**Option A (recommended): Log.Append assigns cursor fields in-place**
Change the Log contract comment to explicitly require it:

```go
// Append assigns sequential Change.Cursor values (and may override Scope/Time if desired).
// It must write the assigned cursor back into changes[i].Cursor.
Append(ctx context.Context, scope string, changes []Change) (uint64, error)
```

**Option B: Engine assigns cursors**
Change `Log.Append` to return `(from, to uint64)` or `start uint64`, then engine can fill `Change.Cursor`. For example:

```go
Append(ctx context.Context, scope string, changes []Change) (start uint64, end uint64, err error)
```

If you do nothing here, your wire protocol is ambiguous: the client cannot advance its cursor reliably based on returned changes.

### 2) `errCursorTooOld` is defined but never produced

`handlePull()` checks `errors.Is(err, errCursorTooOld)` to return HTTP 410, but `Engine.Pull()` just returns whatever `Log.Since()` returns and never maps anything to `errCursorTooOld`.

You need a stable contract:

* Either require `Log.Since` to return `errCursorTooOld` when cursor is trimmed, and document that explicitly, or
* Introduce a `Log` sentinel error (exported) that `Log` implementations return, and map it in the engine.

For example:

```go
var ErrCursorTooOld = errors.New("sync: cursor too old") // exported

// Log.Since must return ErrCursorTooOld when cursor is below the retention window.
```

Then have HTTP map `ErrCursorTooOld` to 410 with code `cursor_too_old`.

### 3) Idempotency storage errors are ignored

In `processMutation`, this line discards errors:

```go
_ = e.applied.Put(ctx, scope, mut.ID, result)
```

If `Applied` is your correctness layer for idempotency, a failed write means a replay can reapply the mutation, which violates your principles.

Safer behavior:

* If `Applied.Put` fails, return `OK=false` (internal error) even if store/log succeeded, or
* Make Applied optional but when provided, treat failure as fatal.

If you want “best effort dedupe,” then the docs should not claim “strict idempotency.”

### 4) Push does not provide atomicity across a batch

`Push` processes mutations one by one. If a client sends a batch and mutation 3 fails, mutations 1 and 2 may already be committed. That can be fine, but it should be an explicit contract because clients will assume batch atomicity unless told otherwise.

If you want stronger semantics later, consider:

* `Push(ctx, mutations)` is “best effort per mutation”
* Add `PushOne` and encourage clients to send one at a time for simpler semantics
* Or add a `Tx` capability on Store/Log for atomic batches (optional interface)

## Data model and wire format issues

### 5) `[]byte` in JSON will be base64 encoded

In `Change.Data []byte` and snapshot `map[string]map[string][]byte`, JSON will base64 encode these values. That may be acceptable, but it is often a DX footgun in JS/TS clients.

If you are storing JSON bytes, consider using `json.RawMessage` instead of `[]byte` for both Store and wire types:

```go
Data json.RawMessage `json:"data,omitempty"`
```

and

```go
Snapshot(ctx, scope string) (map[string]map[string]json.RawMessage, error)
```

This keeps payloads as plain JSON values on the wire and avoids base64 overhead.

### 6) Store contract says “All data is stored as JSON bytes” but `Set` does not enforce JSON

Either enforce validity (optional), or soften the comment. Otherwise implementers will store arbitrary bytes and clients will break.

## API and naming consistency

### 7) Error constants and codes: too many are unused or split-brain

* `codeOK` is unused.
* You have both exported errors (`ErrNotFound`) and internal codes (`codeNotFound`). That is fine, but centralize the mapping and use it consistently across HTTP responses (Push and Snapshot currently return only `"error": ...` with no `"code"`).

Recommendation:

* Always return `{ code, error }` for non-200 responses from all endpoints.
* For Push, consider returning per-mutation `{ ok, code, error }` (you already do), and only use HTTP 500 for request-level failures.

### 8) Default scope constant should not be a magic string

`"_default"` appears in multiple places. Make it a constant:

```go
const DefaultScope = "_default"
```

Also consider whether empty scope should be allowed at all. If empty is common, the constant helps keep behavior stable.

## Transport and operational hardening

### 9) Limit handling should be bounded

`Pull` defaults to 100 but allows any `Limit` from the request. A client can request `Limit=10_000_000` and strain memory or the log backend.

Clamp it:

```go
if limit <= 0 { limit = 100 }
if limit > 1000 { limit = 1000 }
```

Do the same for Push batch size.

### 10) Time source should be injectable for testing and determinism

`time.Now()` is called inside `processMutation`. For deterministic tests and for systems that want a monotonic or database time, allow injection:

```go
type Options struct {
  ...
  Now func() time.Time
}
```

Default to `time.Now` if nil.

### 11) Missing authentication/authorization hook

Sync is almost always scoped by user, tenant, or session. Right now scope is a string in the mutation body. That is easy to spoof.

Even if auth is “out of scope,” add a hook to derive scope from request context in the HTTP transport, or a `ScopeFunc` in Engine options, so the server remains authoritative.

Example direction:

* Remove `Scope` from the client mutation (or treat it as a hint)
* Compute scope server-side from JWT claims or session

### 12) HTTP routing and method choices

Using POST for pull and snapshot is fine for JSON bodies, but consider allowing GET variants later for caching or debuggability. Not required, but worth noting.

## Smaller implementation notes

* `net/http` import is used, fine.
* `codeOK` and `codeNotFound` style is consistent; just ensure all are used or remove dead ones.
* `processMutation` returns `Result` and never returns an error. That makes sense for per-mutation results, but then `Push`’s `error` return becomes mostly redundant. You can simplify `Push` to return only `[]Result` unless you plan request-level failures later.

## Suggested minimal patch set (highest value per line)

1. Fix cursor assignment contract (most critical).
2. Make cursor-too-old error real and exported.
3. Stop ignoring `Applied.Put` errors (or downgrade the idempotency claim).
4. Switch snapshot and change data from `[]byte` to `json.RawMessage` for JS DX.
5. Add max limits for Pull and Push.
6. Add `DefaultScope` constant.

If you want, I can provide a revised `sync` file that applies the above changes while preserving your overall structure and Go-idiomatic naming.
