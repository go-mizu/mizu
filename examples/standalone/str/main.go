// Command str does the string work that ends up hand written in every project.
package main

import (
	"fmt"

	"github.com/go-mizu/mizu/str"
)

func main() {
	fmt.Println(str.Slug("Hello, World! 3rd edition"))
	fmt.Println(str.Snake("HTTPServerAddr"))
	fmt.Println(str.Limit("the quick brown fox", 9, "..."))
	fmt.Println(str.Mask("4111111111111111", '*', 4))
	fmt.Println(str.PluralN("category", 2))
}
