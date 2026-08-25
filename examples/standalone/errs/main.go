// Command errs returns an error that says what kind of failure it is, so that
// whatever is above it can answer without reading the message.
package main

import (
	"errors"
	"fmt"

	"github.com/go-mizu/mizu/errs"
)

func main() {
	err := errs.NotFoundf("no user with id %q", "u_42").WithMeta("user_id", "u_42")

	fmt.Println(err)
	fmt.Println(errs.KindOf(err), errs.Retryable(err))
	fmt.Println(errors.Is(err, errs.NotFound))
}
