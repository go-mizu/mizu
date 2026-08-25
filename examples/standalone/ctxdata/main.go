// Command ctxdata puts a request id in a context once and gets it back out
// with its type, rather than an any and a cast.
package main

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/ctxdata"
)

var requestID = ctxdata.NewKey[string]("request_id", ctxdata.Logged())

func main() {
	ctx := ctxdata.With(context.Background(), requestID, "r_7")

	id, ok := ctxdata.Get(ctx, requestID)
	fmt.Println(id, ok)
	fmt.Println(ctxdata.Attrs(ctx))
}
