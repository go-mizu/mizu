// Package scenarios holds the reference applications.
//
// They are here because performance work needs something bigger than a handler
// to point at, and they stay here because four other things need the same
// thing. Each application is the load for the macro benchmarks, the corpus the
// integration tests run against, the source the documentation examples are
// taken from, the subject of the eject tests, and the thing an upgrade is
// rehearsed on before a release goes out. Carrying one application for five
// purposes is the only way any of them stays current.
//
// # The applications
//
//	blog        server-rendered, sessions, 12 models, an admin area
//	api         JSON only, token auth, no templates, 30 endpoints
//	realtime    WebSocket, presence, queue-heavy
//	rpc         gRPC and Connect, 20 methods, one streaming, a generated client
//
// blog is the shape a new project starts in. api measures the framework with
// the view layer entirely absent, which is what an incremental adoption looks
// like. realtime measures the parts a request benchmark never reaches. rpc
// measures the API critical path and carries the wire compatibility corpus.
//
// Each one has a fixed seeded dataset and a scripted user journey, so whether a
// release made the blog slower is a question with a number behind it.
//
// # When
//
// api arrives with M1 and blog with M4, since a server-rendered application
// needs the view layer. realtime follows the queue and the broadcaster in M6
// and M7, and rpc follows the RPC layer. The package is here now so the first
// one has somewhere to land.
package scenarios
