# Mizu

[![Go Reference](https://pkg.go.dev/badge/github.com/go-mizu/mizu.svg)](https://pkg.go.dev/github.com/go-mizu/mizu)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-mizu/mizu)](https://goreportcard.com/report/github.com/go-mizu/mizu)
[![Build Status](https://github.com/go-mizu/mizu/actions/workflows/ci.yml/badge.svg)](https://github.com/go-mizu/mizu/actions)

> A lightweight, composable web framework for Go.

<p align="center">
  <img src="./docs/logo.png" width="200" alt="Mizu logo">
</p>

Mizu [水] - means water💧 in Japanese - helps you build clear and reliable web applications using modern Go. It keeps things simple, close to the standard library, and gives you the control you expect as a Go developer.

### Overview

Mizu is designed to make building web servers in Go straightforward and enjoyable. It provides just enough structure to handle routing, middleware, templates, and streaming without hiding how `net/http` works underneath.

Everything in Mizu is built to feel natural to Go developers. You can start with a few lines of code, and as your project grows, the same patterns still apply. There are no custom DSLs or hidden global states—just plain Go that compiles fast and reads cleanly.

### Quickstart

Install Mizu and create a simple web app:

```bash
go get github.com/go-mizu/mizu@latest
```

Create `main.go`:

```go
package main

import "github.com/go-mizu/mizu"

func main() {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "Hello, Mizu!")
	})

	app.Listen(":3000")
}
```

Run it:

```bash
go run main.go
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

### Contributing

Mizu is an open project, and everyone is welcome.
If you enjoy Go and want to help build a clean, modern framework, you are encouraged to contribute.

You can:

-  Report issues or suggest improvements
-  Improve examples and documentation
-  Write middleware or helper packages
-  Share your ideas and feedback in GitHub discussions

Every contribution helps make Mizu better. Thoughtful code, good design, and clear documentation are all equally valuable.

### License

MIT License © 2025 Mizu Contributors

Mizu keeps web development in Go clear, explicit, and enjoyable from the first line of code to production.
