// Package markers holds one of every shape the marker scanner has to handle.
// It compiles, because a generator that only works on code that does not is
// not much of a generator.
//
//mizu:manual
package markers

import "time"

// A Post is an article.
//
//mizu:model table=posts
//mizu:searchable index=posts
type Post struct {
	ID    int64
	Title string

	// Body is the article text, which goes over the wire as a string.
	//
	//mizu:ts type=string
	Body []byte

	Created time.Time
}

// Publish makes a post visible.
//
//mizu:rpc method=POST path=/v1/posts/{id}/publish ability=post.publish
//mizu:response 409 ConflictBody
func (p *Post) Publish() error { return nil }

// A Status is how far along a post is.
//
//mizu:enum
type Status int

// DefaultQueue is where post jobs go.
//
//mizu:config key=queue
const DefaultQueue = "posts"

// Prune deletes old drafts.
//
//mizu:command name="posts:prune" desc="Delete drafts older than a year" standalone
func Prune() {}

// A Feed carries a marker on an embedded field, which has no name to resolve.
type Feed struct {
	//mizu:embed
	Post

	Items []string
}

// A Watcher streams changes.
type Watcher interface {
	// Watch sends every change as it happens.
	//
	//mizu:rpc stream=server transport=grpc,connect,http
	Watch() error
}

// Ordinary carries no markers, and the scanner should not mention it.
type Ordinary struct{}

// The comment on this group is not copied onto the names inside it, so the
// marker below belongs to A alone.
//
//mizu:api stable
const (
	// A is the first one.
	//
	//mizu:api experimental
	A = 1

	B = 2
)
