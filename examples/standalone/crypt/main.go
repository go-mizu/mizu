// Command crypt encrypts a string with a key it generates, and reads it back.
// A real program keeps the key somewhere other than the process that made it.
package main

import (
	"fmt"

	"github.com/go-mizu/mizu/crypt"
)

func main() {
	c, err := crypt.New(crypt.GenerateKey())
	if err != nil {
		fmt.Println(err)
		return
	}

	sealed := c.EncryptString("4111 1111 1111 1111")
	plain, err := c.DecryptString(sealed)
	fmt.Println(plain, err)
}
