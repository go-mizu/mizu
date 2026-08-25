package web

import (
	"testing"

	"github.com/go-mizu/mizu/errs"
)

// The two builds of the JSON decoder are held to the same behaviour rather than
// to the same code, and this is where that is written down. Everything about a
// body in body_test.go runs both ways, so the shared half is already covered
// there. What is here is the half they disagree on.

func TestAMemberSentTwiceIsWhateverThisBuildSaysItIs(t *testing.T) {
	in, err := bind[post](t, jsonRequest(`{"title":"first","title":"second"}`))

	if !jsonRefusesDuplicates {
		// The older decoder takes the last one and says nothing, which is what
		// it has always done and what it cannot be talked out of without
		// reading the body twice.
		if err != nil {
			t.Fatal(err)
		}
		if in.Title != "second" {
			t.Errorf("the title is %q, want the last of the two", in.Title)
		}
		return
	}

	f := firstField(t, err)
	if f.Name != "title" || f.Code != "duplicate_field" {
		t.Errorf("the field error is %+v, want title and duplicate_field", f)
	}
	if errs.KindOf(err) != errs.Invalid {
		t.Errorf("the error is of kind %v, want Invalid", errs.KindOf(err))
	}
}
