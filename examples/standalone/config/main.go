// Command config reads one setting from an environment that is passed in
// rather than read from the process, which is why the same code runs in a test.
package main

import (
	"fmt"

	"github.com/go-mizu/mizu/config"
)

func main() {
	l, err := config.Open(config.Sources{Env: "production", Environ: []string{"PORT=8080"}})
	if err != nil {
		fmt.Println(err)
		return
	}

	var port int
	config.Get(l, &port, config.Field{Name: "Port", Env: "PORT", Default: "3000"}, config.Int)
	fmt.Println(port, l.Err())
}
