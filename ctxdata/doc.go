// Package ctxdata carries a few named values with a request, so that a log
// record made anywhere under it says which tenant it was for.
//
// A key is declared once, at package level, with the type of its value and what
// should happen to it:
//
//	var TenantID = ctxdata.NewKey[string]("tenant_id", ctxdata.Logged(), ctxdata.Propagated())
//
// Middleware stores it, and everything under that context can read it back:
//
//	ctx = ctxdata.With(ctx, TenantID, t.ID)
//	id, ok := ctxdata.Get(ctx, TenantID)
//
// This is [context.WithValue] with three things added: the value has a type
// rather than being an any, the key cannot collide with another package's key,
// and the datum can say that it belongs in a log record without every logging
// call naming it.
//
// # What it is not for
//
// Arguments. A value a function needs to do its work is a parameter, and a
// context is not a way to avoid writing one down. What belongs here is the
// handful of things every layer wants and no layer wants to thread: the request
// id, the tenant, the user, the locale.
//
// # Reading it back out
//
// [Attrs] is the logged data as [log/slog] attributes, which is what a handler
// adds to every record. [All] is every datum with its flags, for a carrier that
// needs more than logging, such as a queue writing the propagated ones into a
// job envelope. Nothing propagates anything yet, and the packages that will are
// in later milestones.
//
// # Cost
//
// Every datum lives in one context slot, so reading them back is a single
// [context.Context.Value] lookup no matter how many there are, and ranging over
// [All] allocates nothing.
//
// Storing one costs three allocations: the entry array, the interface header
// the context slot holds, and the context node itself. Storing the same key
// twice replaces the value rather than shadowing it, so middleware that
// overwrites a datum does not make the context deeper.
package ctxdata
