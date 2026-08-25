package budget

import "time"

// The budget itself.
//
// The order here is the order the performance document prints, so a row that
// moves in one moves in the other. Adding a row means adding a benchmark or
// naming the milestone that will add one, because benchrun check does not let
// the two drift apart.
var rows = []Row{
	// The HTTP path. The design is doc 07 and doc 08, which arrive with M1, so
	// the router rows are measured and the rest of it is waiting on the package
	// that will do the work.
	{
		ID: "router/match", Group: httpPath, Doc: "07",
		Op:   "Match a 3-segment path with 2 parameters in a 400-route table",
		Time: 300 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "router/miss", Group: httpPath, Doc: "07",
		Op:   "Non-matching path, 400-route table",
		Time: 250 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "router/url", Group: httpPath, Doc: "07", Since: "M1",
		Op:   "Generated typed URL function, 2 parameters",
		Time: 150 * time.Nanosecond, Allocs: 2,
	},
	{
		ID: "mw/chain", Group: httpPath, Doc: "07",
		Op:   "8 middleware with no work, entry to handler",
		Time: 40 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "mw/requestid", Group: httpPath, Doc: "07",
		Op:   "Make a ULID, put it in the context and on the response",
		Time: 400 * time.Nanosecond, Allocs: 7,
	},
	{
		ID: "mw/lifecycle", Group: httpPath, Doc: "07",
		Op:   "Recover, RealIP, RequestID, Logger, MaxBody and Timeout, entry to handler",
		Time: 3 * time.Microsecond, Allocs: 22,
	},
	{
		ID: "mw/compress", Group: httpPath, Doc: "07",
		Op:   "gzip an 8 KB HTML page, pooled writer",
		Time: 15 * time.Microsecond, Allocs: 8,
	},
	{
		ID: "mw/etag", Group: httpPath, Doc: "07",
		Op:   "Hold an 8 KB page, hash it and answer a matching If-None-Match",
		Time: 5 * time.Microsecond, Allocs: 9,
	},
	{
		ID: "ctx/acquire", Group: httpPath, Doc: "08",
		Op:   "web.H around a handler that does nothing: acquire, fill, release",
		Time: 200 * time.Nanosecond, Allocs: 2,
	},
	{
		ID: "bind/form", Group: httpPath, Doc: "08",
		Op:   "12-field struct from a urlencoded form, reflection binder",
		Time: 3 * time.Microsecond, Allocs: 70,
	},
	{
		ID: "bind/json", Group: httpPath, Doc: "08",
		Op:   "The same struct from a JSON body, reflection binder",
		Time: 600 * time.Nanosecond, Allocs: 10,
	},
	{
		ID: "bind/upload", Group: httpPath, Doc: "08",
		Op:   "The same struct plus a 64 KB file from a multipart form",
		Time: 45 * time.Microsecond, Allocs: 320,
	},
	{
		ID: "bind/gen/form", Group: httpPath, Doc: "08",
		Op:   "The same form through a generated binder",
		Time: 300 * time.Nanosecond, Allocs: 3,
	},
	{
		ID: "bind/gen/json", Group: httpPath, Doc: "08",
		Op:   "The same JSON body through a generated binder",
		Time: 300 * time.Nanosecond, Allocs: 3,
	},
	{
		ID: "validate/build", Group: httpPath, Doc: "08",
		Op:   "4 fields, 10 rules, all passing, programmatic builder",
		Time: 500 * time.Nanosecond, Allocs: 3,
	},
	{
		ID: "validate/format", Group: httpPath, Doc: "08",
		Op:   "An email, a URL, a UUID and an IP address checked, all passing",
		Time: 600 * time.Nanosecond, Allocs: 2,
	},
	{
		ID: "validate/report", Group: httpPath, Doc: "08",
		Op:   "Three failed rules turned into a 422 with a message each",
		Time: 1500 * time.Nanosecond, Allocs: 24,
	},
	{
		ID: "validate/gen", Group: httpPath, Doc: "08", Since: "M1",
		Op:   "12 fields, 20 rules, all passing, generated validator",
		Time: 900 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "validate/reflect", Group: httpPath, Doc: "08", Since: "M1",
		Op:   "Same via the reflective interpreter",
		Time: 36 * time.Microsecond, Allocs: 140,
	},
	{
		ID: "respond/json", Group: httpPath, Doc: "08", Since: "M1",
		Op:   "web.JSON of a 12-field struct",
		Time: 2500 * time.Nanosecond, Allocs: 4,
	},
	{
		ID: "respond/problem", Group: httpPath, Doc: "06", Since: "M1",
		Op:   "RFC 9457 error body with 3 field errors",
		Time: 3 * time.Microsecond, Allocs: 9,
	},
	{
		ID: "http/e2e/hello", Group: httpPath, Doc: "07", Since: "M1",
		Op:   "Full stack, no database, plaintext response, p50",
		Time: 18 * time.Microsecond, Allocs: 22,
	},
	{
		ID: "http/e2e/json", Group: httpPath, Doc: "08", Since: "M1",
		Op:   "Full stack, JSON in and out, validation, no database, p50",
		Time: 30 * time.Microsecond, Allocs: 40,
	},

	// Templates. Doc 09, which arrives with M4.
	{
		ID: "view/simple", Group: templates, Doc: "09", Since: "M4",
		Op:   "20-node page, no components",
		Time: 12 * time.Microsecond, Allocs: 4,
	},
	{
		ID: "view/components", Group: templates, Doc: "09", Since: "M4",
		Op:   "50 components plus a 200-row table",
		Time: 300 * time.Microsecond, Allocs: 30,
	},
	{
		ID: "view/escape", Group: templates, Doc: "09", Since: "M4",
		Op:   "1 KB of interpolated untrusted text, HTML context",
		Time: 2200 * time.Nanosecond, Allocs: 1,
	},
	{
		ID: "view/fragment", Group: templates, Doc: "09", Since: "M4",
		Op:   "Render one fragment from a large page",
		Time: 20 * time.Microsecond, Allocs: 3,
	},
	{
		ID: "view/compile", Group: templates, Doc: "09", Since: "M4",
		Op:   "Compile one 200-line template, generator time",
		Time: 4 * time.Millisecond, Allocs: NoBudget,
	},

	// Database, the query builder and the ORM. Times exclude the round trip,
	// because the round trip dominates and hides the framework.
	{
		ID: "query/compile/cold", Group: database, Doc: "12", Since: "M2",
		Op:   "Build and compile a 5-predicate select, cold fingerprint cache",
		Time: 4 * time.Microsecond, Allocs: 40,
	},
	{
		ID: "query/compile/warm", Group: database, Doc: "12", Since: "M2",
		Op:   "Same, cached",
		Time: 300 * time.Nanosecond, Allocs: 3,
	},
	{
		ID: "query/build/join", Group: database, Doc: "12", Since: "M2",
		Op:   "2 joins, 8 predicates, 3 order clauses, warm",
		Time: 700 * time.Nanosecond, Allocs: 6,
	},
	{
		ID: "db/get", Group: database, Doc: "11", Since: "M2",
		Op:   "db.Get of a 10-field row, in-memory driver",
		Time: 1100 * time.Nanosecond, Allocs: 11,
	},
	{
		ID: "db/select/1000", Group: database, Doc: "11", Since: "M2",
		Op:   "Scan 1000 10-field rows, budgeted at 2 allocations a row",
		Time: 900 * time.Microsecond, Allocs: 2000,
	},
	{
		ID: "db/iterate/1000", Group: database, Doc: "11", Since: "M2",
		Op:   "Same via iter.Seq2, streaming, budgeted at 1 allocation a row",
		Time: 850 * time.Microsecond, Allocs: 1000,
	},
	{
		ID: "orm/find", Group: database, Doc: "13", Since: "M3",
		Op:   "Repo.Find by primary key",
		Time: 1500 * time.Nanosecond, Allocs: 12,
	},
	{
		ID: "orm/save/clean", Group: database, Doc: "13", Since: "M3",
		Op:   "Save with no dirty fields, snapshot present",
		Time: 200 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "orm/save/dirty", Group: database, Doc: "13", Since: "M3",
		Op:   "Save with 3 changed fields of 12",
		Time: 2200 * time.Nanosecond, Allocs: 14,
	},
	{
		ID: "orm/eager/3", Group: database, Doc: "13", Since: "M3",
		Op:   "Eager load 3 relations for 100 parents, query assembly only",
		Time: 40 * time.Microsecond, Allocs: 900,
	},
	{
		ID: "orm/hydrate/1000", Group: database, Doc: "13", Since: "M3",
		Op:   "Hydrate 1000 models with generated scanners, budgeted at 3 allocations a row",
		Time: 1100 * time.Microsecond, Allocs: 3000,
	},
	{
		ID: "tx/begin", Group: database, Doc: "11", Since: "M2",
		Op:   "Transaction begin and commit, in-memory driver",
		Time: 400 * time.Nanosecond, Allocs: 2,
	},

	// Cache, queue and events. M6.
	//
	// cache/mem/hit/bytes is cache/mem/hit in doc 15. It had to move, because
	// go test matches a -bench pattern part by part and a pattern with fewer
	// parts than the name matches anyway, so asking for cache/mem/hit ran
	// cache/mem/hit/typed as well. i18n/t/plain moved for the same reason.
	{
		ID: "cache/mem/hit/bytes", Group: cacheQueue, Doc: "15", Since: "M6",
		Op:   "Memory driver hit, byte slice value",
		Time: 80 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "cache/mem/hit/typed", Group: cacheQueue, Doc: "15", Since: "M6",
		Op:   "cache.Remember hit, 6-field struct, JSON codec",
		Time: 700 * time.Nanosecond, Allocs: 5,
	},
	{
		ID: "cache/key", Group: cacheQueue, Doc: "15", Since: "M6",
		Op:   "cache.Key construction and validation, 4 segments",
		Time: 90 * time.Nanosecond, Allocs: 1,
	},
	{
		ID: "cache/tiered/hit", Group: cacheQueue, Doc: "15", Since: "M6",
		Op:   "Tiered driver, front hit",
		Time: 110 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "lock/local", Group: cacheQueue, Doc: "15", Since: "M6",
		Op:   "lock.Run uncontended, memory driver",
		Time: 150 * time.Nanosecond, Allocs: 1,
	},
	{
		ID: "queue/encode", Group: cacheQueue, Doc: "16", Since: "M6",
		Op:   "Envelope encode of a 6-field job",
		Time: 1400 * time.Nanosecond, Allocs: 12,
	},
	{
		ID: "queue/dispatch/sql", Group: cacheQueue, Doc: "16", Since: "M6",
		Op:   "Enqueue on the SQL driver, batched, per job",
		Time: 25 * time.Microsecond, Allocs: 30,
	},
	{
		ID: "queue/reserve/sql", Group: cacheQueue, Doc: "16", Since: "M6",
		Op:   "Reserve one job with SKIP LOCKED",
		Time: 180 * time.Microsecond, Allocs: 40,
	},
	{
		ID: "events/emit/3", Group: cacheQueue, Doc: "18", Since: "M6",
		Op:   "Emit to 3 synchronous listeners, generated dispatch",
		Time: 200 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "events/emit/queued", Group: cacheQueue, Doc: "18", Since: "M6",
		Op:   "Emit to 1 queued listener, encode plus enqueue",
		Time: 27 * time.Microsecond, Allocs: 45,
	},
	{
		ID: "broadcast/fanout/1000", Group: cacheQueue, Doc: "18", Since: "M6",
		Op:   "Serialise once, write to 1000 subscribers",
		Time: 900 * time.Microsecond, Allocs: 1100,
	},

	// Auth, crypto and session. The hashes and the keyring are here already,
	// so this is the group with something to measure.
	{
		ID: "hash/argon2id/verify", Group: authCrypto, Doc: "22",
		Op:   "Verify against the default parameters, 19 MiB, 2 passes, 1 lane",
		Time: 40 * time.Millisecond, Allocs: 20,
	},
	{
		ID: "hash/bcrypt/verify", Group: authCrypto, Doc: "22",
		Op:   "Verify a cost 12 hash, the legacy path",
		Time: 250 * time.Millisecond, Allocs: 6,
	},
	{
		ID: "crypt/seal/1kb", Group: authCrypto, Doc: "22",
		Op:   "AEAD seal of 1 KB of plaintext",
		Time: 1600 * time.Nanosecond, Allocs: 3,
	},
	{
		ID: "crypt/open/1kb", Group: authCrypto, Doc: "22",
		Op:   "Open, including the header parse and the associated data check",
		Time: 1700 * time.Nanosecond, Allocs: 3,
	},
	{
		ID: "sign/hmac", Group: authCrypto, Doc: "22",
		Op:   "Signed cookie or URL MAC",
		Time: 700 * time.Nanosecond, Allocs: 2,
	},
	{
		ID: "session/load", Group: authCrypto, Doc: "10", Since: "M5",
		Op:   "Load and decode a 2 KB session, excluding the driver",
		Time: 6 * time.Microsecond, Allocs: 25,
	},
	{
		ID: "session/save/clean", Group: authCrypto, Doc: "10", Since: "M5",
		Op:   "No changes, no write",
		Time: 40 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "csrf/verify", Group: authCrypto, Doc: "10", Since: "M5",
		Op:   "Unmask and constant-time compare",
		Time: 350 * time.Nanosecond, Allocs: 1,
	},
	{
		ID: "gate/check", Group: authCrypto, Doc: "21", Since: "M5",
		Op:   "gate.Allows with a policy method, no database",
		Time: 250 * time.Nanosecond, Allocs: 1,
	},
	{
		ID: "auth/token/verify", Group: authCrypto, Doc: "21", Since: "M5",
		Op:   "Token lookup by ID prefix plus hash compare, cache hit",
		Time: 900 * time.Nanosecond, Allocs: 4,
	},

	// Everything else. The logger, the errors and the sequence helpers are
	// here already.
	{
		ID: "config/get", Group: everythingElse, Doc: "05", Since: "M1",
		Op:   "Typed generated config accessor",
		Time: 2 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "log/info/off", Group: everythingElse, Doc: "06",
		Op:   "Log call with 2 attributes, below the configured level",
		Time: 8 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "log/info/json", Group: everythingElse, Doc: "06",
		Op:   "One line, 6 attributes, JSON handler, to a discard writer",
		Time: 900 * time.Nanosecond, Allocs: 3,
	},
	{
		ID: "errs/wrap", Group: everythingElse, Doc: "06",
		Op:   "Wrap with a kind, a code and 2 pieces of metadata",
		Time: 180 * time.Nanosecond, Allocs: 3,
	},
	{
		ID: "diag/text", Group: everythingElse, Doc: "36",
		Op:   "One diagnostic with a code, a source snippet and a suggestion, rendered for a terminal",
		Time: 3 * time.Microsecond, Allocs: 24,
	},
	{
		ID: "diag/json", Group: everythingElse, Doc: "37",
		Op:   "The same diagnostic as a mizu.diag/1 document",
		Time: 10 * time.Microsecond, Allocs: 16,
	},
	{
		ID: "diag/suggest", Group: everythingElse, Doc: "36",
		Op:   "Did you mean, over the seventy settings of a middling application",
		Time: 150 * time.Microsecond, Allocs: 32,
	},
	{
		ID: "trace/span/off", Group: everythingElse, Doc: "26", Since: "M8",
		Op:   "Span start and end, sampled out",
		Time: 60 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "trace/span/on", Group: everythingElse, Doc: "26", Since: "M8",
		Op:   "Span start and end, sampled in, 6 attributes",
		Time: 1500 * time.Nanosecond, Allocs: 12,
	},
	{
		ID: "metric/record", Group: everythingElse, Doc: "26", Since: "M8",
		Op:   "Histogram observation with 3 labels",
		Time: 100 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "pulse/record", Group: everythingElse, Doc: "26", Since: "M8",
		Op:   "One Pulse aggregation update",
		Time: 1 * time.Microsecond, Allocs: 0,
	},
	{
		ID: "telescope/record", Group: everythingElse, Doc: "26", Since: "M8",
		Op:   "Full request recording, queued",
		Time: 400 * time.Microsecond, Allocs: NoBudget,
	},
	{
		ID: "i18n/t/plain", Group: everythingElse, Doc: "27", Since: "M8",
		Op:   "Generated typed accessor, no placeholders",
		Time: 45 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "i18n/t/icu", Group: everythingElse, Doc: "27", Since: "M8",
		Op:   "ICU message with a plural and 2 placeholders",
		Time: 1200 * time.Nanosecond, Allocs: 4,
	},
	{
		ID: "flag/active", Group: everythingElse, Doc: "27", Since: "M8",
		Op:   "Cached per-request flag check",
		Time: 25 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "xs/map/1000", Group: everythingElse, Doc: "24",
		Op:   "xs.Map over a 1000-element slice, collected",
		Time: 3 * time.Microsecond, Allocs: 1,
	},
	{
		ID: "di/resolve", Group: everythingElse, Doc: "04", Since: "M1",
		Op:   "Generated container access",
		Time: 0, Allocs: 0,
	},

	// The RPC path, doc 38, which arrives with M1. rpc/errmap is the one row
	// here that is measurable now, because the kind to status mapping is part
	// of errs rather than of the transport.
	{
		ID: "rpc/http/decode", Group: rpcPath, Doc: "38", Since: "M1",
		Op:   "Generated decode plus validate, 8-field request",
		Time: 900 * time.Nanosecond, Allocs: 3,
	},
	{
		ID: "rpc/http/e2e", Group: rpcPath, Doc: "38", Since: "M1",
		Op:   "Generated HTTP handler end to end, in process, trivial method",
		Time: 4 * time.Microsecond, Allocs: 12,
	},
	{
		ID: "rpc/grpc/dispatch", Group: rpcPath, Doc: "38", Since: "M1",
		Op:   "gRPC unary dispatch, the mizu layer only",
		Time: 350 * time.Nanosecond, Allocs: 2,
	},
	{
		ID: "rpc/grpc/e2e", Group: rpcPath, Doc: "38", Since: "M1",
		Op:   "gRPC unary end to end over loopback, trivial method",
		Time: 42 * time.Microsecond, Allocs: 68,
	},
	{
		ID: "rpc/connect/e2e", Group: rpcPath, Doc: "38", Since: "M1",
		Op:   "Connect JSON POST end to end over loopback",
		Time: 38 * time.Microsecond, Allocs: 60,
	},
	{
		ID: "rpc/grpc/convert", Group: rpcPath, Doc: "38", Since: "M1",
		Op:   "Go struct to protobuf message, 8 fields",
		Time: 200 * time.Nanosecond, Allocs: 1,
	},
	{
		ID: "rpc/errmap", Group: rpcPath, Doc: "06",
		Op:   "errs.Kind to an HTTP status and an RPC code",
		Time: 5 * time.Nanosecond, Allocs: 0,
	},
	{
		ID: "rpc/grpc/stream/send", Group: rpcPath, Doc: "38", Since: "M1",
		Op:   "Server-stream send, per item",
		Time: 700 * time.Nanosecond, Allocs: 2,
	},
	{
		ID: "rpc/idem/hit", Group: rpcPath, Doc: "38", Since: "M1",
		Op:   "Idempotency lookup, cache hit, excluding the driver",
		Time: 90 * time.Nanosecond, Allocs: 1,
	},
	{
		ID: "rpc/reflect/decode", Group: rpcPath, Doc: "38", Since: "M1",
		Op:   "The reflective fallback for rpc/http/decode",
		Time: 14 * time.Microsecond, Allocs: 92,
	},
}

// The group headings, named so that a typo in one is a compile error rather
// than a second group with almost the same name.
const (
	httpPath       = "HTTP path"
	templates      = "Templates"
	database       = "Database, query builder, ORM"
	cacheQueue     = "Cache, queue, events"
	authCrypto     = "Auth, crypto, session"
	everythingElse = "Everything else"
	rpcPath        = "RPC and gRPC path"
)
