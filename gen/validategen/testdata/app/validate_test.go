package app

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/validate"
)

// TestBothModesAgree is what the generator is for.
//
// Every value here is checked twice, once by the method in validate_gen.go and
// once by validate.Struct reading the same tags through reflection, and the two
// have to come back with the same failures under the same names in the same
// order with the same sentences. A generator that is faster and says something
// else is not the same rules written out, and switching a struct to it by
// adding a marker would change what the API answers.
func TestBothModesAgree(t *testing.T) {
	ctx := context.Background()
	since := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	yes := true
	no := false
	code := "abcd"
	short := "ab"
	note := strings.Repeat("x", 200)
	filter := Filter{Names: []string{"go"}}
	kyoto := &Address{City: "Kyoto"}
	empty := &Address{}

	// Whether each value is one that passes is written down next to it, so
	// that a corpus which quietly stopped finding anything is a corpus that
	// fails rather than one that agrees with itself about nothing.
	cases := []struct {
		name  string
		pass  bool
		value validate.Validator
	}{
		{"a listing with nothing filled in", false, Listing{}},
		{"a listing that passes", true, Listing{
			Page: 1, Limit: 20, Since: since, Draft: &yes, Filter: filter,
		}},
		{"a listing that is wrong in every way", false, Listing{
			Q:      "x",
			Page:   0,
			Limit:  101,
			Sort:   "a",
			Tags:   []string{"a", "b", "c", "d", "e", "f"},
			Attrs:  map[string]string{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5"},
			Cursor: []byte(strings.Repeat("c", 65)),
			Wait:   31 * time.Second,
			Score:  2,
			Draft:  &no,
		}},
		{"a listing whose tags are wrong one at a time", false, Listing{
			Page: 1, Limit: 1, Since: since, Draft: &yes,
			Tags: []string{"ok", "", "x"},
		}},
		{"a listing with a q that is too long", false, Listing{
			Q: strings.Repeat("q", 65), Page: 1, Limit: 1, Since: since, Draft: &yes,
		}},
		{"a listing at both edges of between", true, Listing{
			Page: 1, Limit: 100, Since: since, Draft: &yes, Filter: filter,
			Wait: time.Second, Score: 1, Weight: 0.5,
		}},
		{"a listing whose weight is under the bound", false, Listing{
			Page: 1, Limit: 1, Since: since, Draft: &yes, Filter: filter,
			Weight: 0.25,
		}},
		{"a listing with more numbers than it may have", false, Listing{
			Page: 1, Limit: 1, Since: since, Draft: &yes, Filter: filter,
			Nums: []int{1, 2, 3, 4},
		}},

		{"an order with nothing filled in", false, Order{}},
		{"an order that passes", true, Order{
			ID: 1, Ref: "ref-001", Email: "sam@example.com",
			Address: Address{City: "Kyoto"},
			Lines:   []Line{{SKU: "SKU-0001", Quantity: 1}},
		}},
		{"an order whose lines are wrong", false, Order{
			ID: 1, Ref: "ref-001", Email: "sam@example.com",
			Address: Address{City: "Kyoto"},
			Lines: []Line{
				{SKU: "SKU-0001", Quantity: 1},
				{SKU: "short", Quantity: 0},
				{Quantity: 100},
			},
		}},
		{"an order with a ship address that is wrong", false, Order{
			ID: 1, Ref: "ref-001", Email: "sam@example.com",
			Address: Address{City: "Kyoto"},
			Ship:    &Address{Zone: "a"},
			Lines:   []Line{{SKU: "SKU-0001", Quantity: 1}},
		}},
		{"an order with a nil code in the list", false, Order{
			ID: 1, Ref: "ref-001", Email: "sam@example.com",
			Address: Address{City: "Kyoto"},
			Lines:   []Line{{SKU: "SKU-0001", Quantity: 1}},
			Codes:   []*string{&code, nil, &short},
		}},
		{"an order whose extras are wrong", false, Order{
			ID: 1, Ref: "ref-001", Email: "sam@example.com",
			Address: Address{City: "Kyoto"},
			Lines:   []Line{{SKU: "SKU-0001", Quantity: 1}},
			Extras:  []*Line{{SKU: "SKU-0002", Quantity: 2}, nil, {SKU: "no", Quantity: 0}},
		}},
		{"an order whose origin is wrong two pointers down", false, Order{
			ID: 1, Ref: "ref-001", Email: "sam@example.com",
			Address: Address{City: "Kyoto"},
			Lines:   []Line{{SKU: "SKU-0001", Quantity: 1}},
			Origin:  &empty,
		}},
		{"an order whose origin is filled in", true, Order{
			ID: 1, Ref: "ref-001", Email: "sam@example.com",
			Address: Address{City: "Kyoto"},
			Lines:   []Line{{SKU: "SKU-0001", Quantity: 1}},
			Origin:  &kyoto,
		}},
		{"an order with a note that is too long", false, Order{
			ID: 1, Ref: "ref-001", Email: "sam@example.com",
			Address: Address{City: "Kyoto"},
			Lines:   []Line{{SKU: "SKU-0001", Quantity: 1}},
			Note:    &note,
		}},

		{"a webhook with nothing filled in", false, Webhook{}},
		{"a webhook that passes", true, Webhook{
			Meta:  Meta{RequestID: "0b0a4a4a-9f5f-4f5a-9d1a-6a2f0a5a1b2c"},
			Event: "post.created",
			URL:   "https://example.com/hooks/1",
		}},
		{"a webhook whose secret is too short", false, Webhook{
			Meta:   Meta{RequestID: "0b0a4a4a-9f5f-4f5a-9d1a-6a2f0a5a1b2c"},
			Event:  "post.created",
			URL:    "https://example.com/hooks/1",
			Secret: &Secret{Value: "tooshort", Host: "not a host"},
		}},

		{"a tree with nothing filled in", false, Tree{}},
		{"a tree three levels deep", false, Tree{
			Name: "root",
			Children: []Tree{
				{Name: "one"},
				{Children: []Tree{{Name: strings.Repeat("d", 40)}}},
			},
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.value.Validate(ctx)
			got, want := report(err), report(validate.Struct(ctx, c.value))
			if got != want {
				t.Errorf("the generated method said\n%s\nand validate.Struct said\n%s", got, want)
			}
			if c.pass != (err == nil) {
				t.Errorf("this value was written down as one that passes: %v, and it came back\n%s", c.pass, got)
			}
		})
	}
}

// report is everything a caller can see about the answer, written out so that
// a difference between the two modes reads as a diff rather than as two
// pointers that are not equal.
func report(err error) string {
	if err == nil {
		return "\tnothing failed"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\t%v", errs.KindOf(err))
	for _, f := range errs.Fields(err) {
		fmt.Fprintf(&b, "\n\t%s\t%s\t%s", f.Name, f.Code, f.Msg)
	}
	return b.String()
}
