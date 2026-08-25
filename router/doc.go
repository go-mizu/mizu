// Package router matches an HTTP request to a handler.
//
// The patterns are [net/http.ServeMux] patterns, with one thing added:
//
//	GET /posts                a method and a literal path
//	GET /posts/{id}           one segment, read back under the name id
//	GET /files/{path...}      the rest of the path
//	GET /posts/{$}            this path and nothing under it
//	GET example.com/posts     only for that host
//	GET /posts/{id:uuid}      one segment that has to be a UUID
//
// Registering returns the route, so anything hung off it stays with it:
//
//	r := router.New()
//	r.Handle("GET /posts/{id:uuid}", show).Name("posts.show")
//
//	rt, params, ok := r.Lookup("GET", "", "/posts/018f6f7d-0000-7000-8000-000000000000")
//
// # Why this is not ServeMux
//
// [net/http.ServeMux] matches the same patterns and matches them well, and this
// package copies its precedence rules exactly rather than inventing better
// ones. What it does not do is hand back the thing that matched. A request goes
// in and a handler runs, and the route in between is gone.
//
// Six subsystems want that route. A metric labelled by route has bounded
// cardinality where one labelled by path does not. A span reads better named
// posts.show than named /posts/018f6f7d-0000-7000-8000-000000000000. A rate
// limit, a CSRF exemption, an OpenAPI document and a route listing are all
// per route rather than per request. Recovering it by looking the path up a
// second time inside middleware is slower than storing it on the node that
// matched, and it stops being right the moment two patterns can match one path.
//
// So the tree is written here, and the price of writing it is that it has to
// stay the same as the one in the standard library. That is what
// FuzzMatchesServeMux is for: it builds the same table in both, throws paths at
// them, and fails when they disagree.
//
// # Constraints
//
// A wildcard may say what it accepts, after a colon:
//
//	int     an optionally signed whole number that fits an int64
//	uint    a whole number that fits a uint64
//	uuid    the canonical 8-4-4-4-12 form, in either case
//	ulid    26 characters of Crockford base32
//	slug    lower case words and digits joined by single hyphens
//	alpha   letters
//	word    letters, digits and underscores
//	date    a calendar date written YYYY-MM-DD
//
// A segment the constraint turns down is not a match, so a request for
// /posts/latest walks past /posts/{id:int} and carries on down the tree. That is
// what lets /posts/{id:int} and /posts/{slug:slug} both be registered: without
// the constraints they are the same pattern written twice, and with them they
// are two branches that never answer the same request.
//
// The other half of what a constraint buys is where the error goes. A handler
// that has to parse its own parameter reports a bad one from inside the
// handler, after the middleware has run and usually as a 500. A constraint
// turns the same request into a 404 before anything starts, which is also the
// answer that tells an attacker the least.
//
// [Constrain] adds one under a name of your own, and [Regexp] is the way to
// write one when the shape is genuinely irregular.
//
// # Which pattern wins
//
// The rules are ServeMux's, in this order:
//
//  1. A pattern with a host beats one without.
//  2. Otherwise the more specific of the two wins, where one pattern is more
//     specific than another when the other matches every request it does and
//     more. Within a segment that reads as: a literal beats a constrained
//     wildcard, which beats a bare wildcard, which beats a trailing wildcard.
//  3. A pattern with a method beats one without, and a GET route answers HEAD.
//
// When two patterns can match the same request and neither is more specific,
// there is no answer, and registration panics naming both patterns and the
// place the other one was registered. /posts/{id}/edit and /{kind}/latest/edit
// are that pair: each matches something the other does not.
//
// Constraints are the one place mizu adds a rule. Two wildcards in the same
// position with different constraints are separate branches of the tree, tried
// in the order they were registered, so a request both accept goes to the one
// written first. Whether two constraints can accept the same segment is not a
// question this package can answer, since a constraint is a Go function, so it
// does not ask it and does not call the pair a conflict.
//
// A constraint against a literal is a question it can answer, so it does. When
// /posts/{id:int}/edit is registered next to /{kind}/latest/edit, int is run on
// latest, it says no, and the two are not a conflict. Without the constraint
// they are, since each matches a path the other does not.
//
// # Reading the parameters
//
// Inside a handler, [PathValue] and [Matched] read what the router found:
//
//	func show(w http.ResponseWriter, req *http.Request) {
//		id := router.PathValue(req, "id")
//		...
//	}
//
// [net/http.Request.PathValue] stays empty, because filling it in costs a map
// allocation on every request that has a parameter. A handler that wants the
// standard shape can call req.SetPathValue for itself.
//
// # What it costs
//
// Matching allocates nothing. The wildcard values of a match live in an array
// on the stack, up to eight of them, and the tree is read through one atomic
// load rather than a lock, so requests on different cores do not queue behind
// each other to look at the route table. The budget is in bench/budget: 300 ns
// for a three segment path with two parameters in a table of four hundred
// routes.
//
// Registering is where the work is. Every new pattern is compared against every
// pattern already there, so building a table of n routes is quadratic in n. Four
// hundred routes is a few milliseconds once, at startup, and buys the guarantee
// that no two of them are ambiguous.
//
// # What is not here yet
//
// Groups, per route middleware, the Get and Post shorthands and the resource
// controllers are all about handlers rather than about matching, and they
// arrive with the handler and middleware layer. Redirects are off unless
// [RedirectTrailingSlash] or [RedirectCleanPath] is asked for.
package router
