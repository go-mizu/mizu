// Command hash stores a password as an Argon2id digest and checks one against
// it. The context is there because hashing a password is meant to take a while.
package main

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/hash"
)

func main() {
	ctx := context.Background()
	h := hash.Default()

	encoded, err := h.Hash(ctx, "correct horse battery staple")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(h.Verify(ctx, "correct horse battery staple", encoded))
	fmt.Println(h.NeedsRehash(encoded))
}
