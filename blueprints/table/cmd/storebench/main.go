package main

import (
	"os"

	"github.com/go-mizu/blueprints/table/storebench"
)

func main() {
	storebench.Main(os.Args[1:])
}
