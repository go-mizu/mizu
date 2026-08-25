// Package macro measures a whole server over a real socket.
//
// A micro benchmark answers whether a call got slower. It cannot answer whether
// a request got slower, because a request is a few hundred calls plus a
// database, a connection, a pool, a garbage collector deciding when to run, and
// a scheduler deciding what runs next. Those are where the surprises are, and
// none of them shows up in a loop calling one function.
//
// So a scenario here starts a real server, points a separate load generator at
// it, and reports a full latency histogram rather than an average. A release
// that improves the mean and worsens p99.9 is a regression, and only a
// histogram shows that.
//
// # What goes here
//
// The scenarios, in the order they become possible:
//
//	plaintext          static string response
//	json               encode a small object
//	db-single          one indexed primary-key read
//	db-multi           20 reads plus an update
//	fortunes           read, render, escape, the TechEmpower shape
//	session-auth       session load, auth, gate check, render
//	queue-throughput   enqueue and drain 100k jobs
//	websocket-fanout   50k connections, one broadcast a second
//	rpc-unary          gRPC unary, one indexed read
//	rpc-stream         gRPC server stream, 1000 items a call
//	connect-json       Connect JSON over POST from a browser-shaped client
//	api-generated      the generated service path, compared against json
//	mixed              70 percent reads, 20 percent writes, 10 percent renders
//
// mixed is the one a release note quotes, because a single endpoint is a
// benchmark and a mixed workload is an application. rpc-unary is the second
// one, because it is the shape of an internal service and it is where a Go
// team's latency budget is usually tightest.
//
// # When
//
// plaintext and json arrive with M1, which is when there is a server to point
// at. The database scenarios follow the database in M2, and the rest follow the
// subsystems they name. The package is here now so that the first one has
// somewhere to land and so the layout is not an argument later.
package macro
