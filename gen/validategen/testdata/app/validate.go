// Package app is a set of request structs for the generator's tests.
//
// Between them they use every shape the generator writes code for, because the
// test beside them checks the generated method and validate.Struct against the
// same values and the two only agree if both modes read the tags the same way.
package app

import "time"

// A Sort is a column to order by, which is a string with a name of its own so
// that a check on it has to convert.
type Sort string

// A Listing is the query behind a list of posts.
//
//mizu:validate
type Listing struct {
	Q      string            `json:"q" validate:"omitempty,min=2,max=64"`
	Page   int               `json:"page" validate:"min=1"`
	Limit  int               `json:"limit" validate:"between=1 100"`
	Sort   Sort              `json:"sort" validate:"omitempty,min=2"`
	Tags   []string          `json:"tags" validate:"max=5,dive,required,min=2"`
	Attrs  map[string]string `json:"attrs" validate:"max=4"`
	Cursor []byte            `json:"cursor" validate:"omitempty,max=64"`
	Since  time.Time         `json:"since" validate:"required"`
	Wait   time.Duration     `json:"wait" validate:"omitempty,between=1s 30s"`
	Score  float64           `json:"score" validate:"min=0,max=1"`
	Draft  *bool             `json:"draft" validate:"required"`

	// A bound that is not a whole number, on a field that is not a float64, so
	// the comparison has to go through one to give the answer the tag
	// interpreter gives.
	Weight float32 `json:"weight" validate:"omitempty,between=0.5 1.5"`

	// A dive with no rules behind it, which is a tag somebody wrote one rule
	// too few of. There is nothing to check each element against, so the loop
	// that would have checked them is not written.
	Nums []int `json:"nums" validate:"omitempty,max=3,dive"`

	// A struct with a slice in it cannot be compared against its zero value, so
	// required on one asks reflect the same question the tag interpreter asks.
	Filter Filter `json:"filter" validate:"required"`

	// Nothing is asked of this one, since omitempty is not a check, and a
	// chain that comes to nothing is left out rather than written as an if
	// with an empty body.
	Group string `json:"group" validate:"omitempty"`

	// A field the request does not carry is not a field to check.
	Internal string `json:"-" validate:"required"`
	notes    string
}

// A Filter is a set of names to narrow a listing by. It carries no rules of its
// own, so nothing is checked under it, and it holds a slice, which is what
// makes it a struct that cannot be compared.
type Filter struct {
	Names []string `json:"names"`
}

// An Address is part of an order rather than a request of its own, so it is
// checked where it sits instead of through a function of its own.
type Address struct {
	City string `json:"city" validate:"required,max=64"`
	Zone string `json:"zone" validate:"omitempty,min=2"`
}

// A Line is one thing being bought. It is checked through a function because
// an order holds a list of them.
type Line struct {
	SKU      string `json:"sku" validate:"required,size=8"`
	Quantity int    `json:"quantity" validate:"between=1 99"`
}

// An Order is a checkout.
//
//mizu:validate
type Order struct {
	ID      int64     `path:"id" validate:"min=1"`
	Ref     string    `json:"ref" validate:"required,min=6,max=32"`
	Email   string    `json:"email" validate:"required,email"`
	Address Address   `json:"address"`
	Ship    *Address  `json:"ship"`
	Lines   []Line    `json:"lines" validate:"required,max=20,dive"`
	Codes   []*string `json:"codes" validate:"omitempty,dive,required,min=3"`
	Note    *string   `json:"note" validate:"omitempty,max=140"`

	// A list of pointers to a struct, so the function that checks one is called
	// only for the elements that are there.
	Extras []*Line `json:"extras" validate:"omitempty,max=4,dive"`

	// More than one pointer to the same struct, because the guards that read
	// through them are written in a loop and a loop that only ever runs once is
	// a loop nobody has checked.
	Origin **Address `json:"origin"`
}

// A Meta is what every hook carries, embedded rather than written out again,
// so its fields are named at the top level and not under meta.
type Meta struct {
	RequestID string `json:"request_id" validate:"required,uuid"`
}

// A Secret is behind a pointer, so the fields under it are checked only when
// there is something there to check.
type Secret struct {
	Value string `json:"value" validate:"required,min=16"`
	Host  string `json:"host" validate:"omitempty,hostname"`
}

// A Webhook is a delivery somebody registered.
//
//mizu:validate
type Webhook struct {
	Meta

	Event  string  `json:"event" validate:"required,min=3"`
	URL    string  `json:"url" validate:"required,url"`
	Secret *Secret `json:"secret"`
}

// A Tree is a category with categories under it, which is what makes the
// function that checks one call itself.
//
//mizu:validate
type Tree struct {
	Name     string `json:"name" validate:"required,max=32"`
	Children []Tree `json:"children" validate:"max=8,dive"`
}
