// Package app is a set of request structs for the generator's tests. Between
// them they hold one field of every shape the generator claims to read, so
// that regenerating the file and building the result is a check on all of them
// at once.
package app

import (
	"net/netip"
	"time"

	"github.com/go-mizu/mizu/web"
)

// Sort is a named string, so that a defined type over a basic one is covered.
type Sort string

// Paging is embedded, and an embedded struct adds nothing to the names of the
// fields inside it.
type Paging struct {
	Page    int `query:"page"`
	PerPage int `query:"per_page"`
}

// Listing is what a search endpoint reads off the query string.
//
//mizu:bind
type Listing struct {
	Paging

	Q      string    `query:"q"`
	Sort   Sort      `query:"sort"`
	Tags   []string  `query:"tags"`
	Cursor []byte    `query:"cursor"`
	Limit  uint32    `query:"limit"`
	Score  float32   `query:"score"`
	Draft  *bool     `query:"draft"`
	Since  time.Time `query:"since"`

	// Internal is filled in by the handler and never by the request.
	Internal string `bind:"-"`

	// Secret is out of a JSON body as well, so nothing reads it here either.
	Secret string `json:"-"`
}

// Address is nested twice below, once by value and once behind a pointer, and
// the fields inside it are named under whatever it is called.
type Address struct {
	City    string      `form:"city"`
	Country string      `form:"country"`
	Zone    *netip.Addr `form:"zone"`
}

// Order is a checkout, and reads from every place a request carries something.
//
//mizu:bind
type Order struct {
	ID     int64  `path:"id"`
	Ref    string `header:"X-Request-Id"`
	Locale string `cookie:"locale"`

	Traces []string `header:"X-Trace"`

	// Seen is a list with one place to read it from, which is the shape a
	// cookie and a route parameter both have.
	Seen []string `cookie:"seen"`

	Note   *string       `form:"note"`
	Wait   time.Duration `form:"wait"`
	Codes  []int         `form:"codes"`
	Origin netip.Addr    `form:"origin"`

	// Labels is a second list off the form, so that a struct with more than one
	// accumulator in it is covered.
	Labels []string `form:"labels"`

	// Sizes and Names are lists of pointers, which is the one shape where an
	// empty value has to be left out of the list rather than appended as a zero.
	Sizes []*int    `form:"sizes"`
	Names []*string `form:"names"`

	Address Address  `form:"address"`
	Ship    *Address `form:"ship"`

	// CouponCode carries no tag, so it takes its own name in snake case.
	CouponCode string

	// hidden is unexported, which is a field nothing outside this file can
	// write and the generator does not try to.
	hidden string
}

// Album is a named list of files, so that a slice type of its own is covered
// alongside the plain one.
type Album []*web.Upload

// Profile is a form with files in it.
//
//mizu:bind
type Profile struct {
	Name   string        `form:"name"`
	Avatar *web.Upload   `form:"avatar"`
	Photos []*web.Upload `form:"photos"`
	Extra  Album         `form:"extra"`
}

// Webhook takes a body whose shape is not fixed, and embedding AllowUnknown is
// how a struct says a member it has never heard of is not an error.
//
//mizu:bind
type Webhook struct {
	web.AllowUnknown

	Event string `json:"event"`
	Kind  string `json:"kind"`
}

// Tree is a body a query string has no way to carry, which is where the walk
// stops rather than following the type round.
//
//mizu:bind
type Tree struct {
	Name     string  `json:"name"`
	Parent   *Tree   `json:"parent"`
	Children []*Tree `json:"children"`
}
