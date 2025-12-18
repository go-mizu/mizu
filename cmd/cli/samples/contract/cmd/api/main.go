package main

import (
	"log"

	"example.com/contract/app/server"
	"example.com/contract/service/todo"
)

func main() {
	cfg := server.LoadConfig()

	// Create the plain Go service (no framework dependencies)
	todoSvc := &todo.Service{}

	// Create the mizu app with all transports
	app, err := server.New(cfg, todoSvc)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s", cfg.Addr)
	log.Printf("REST:     http://localhost%s/todos", cfg.Addr)
	log.Printf("JSON-RPC: http://localhost%s/rpc", cfg.Addr)
	log.Printf("OpenAPI:  http://localhost%s/openapi.json", cfg.Addr)

	if err := app.Listen(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
