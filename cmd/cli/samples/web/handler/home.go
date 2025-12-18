package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/view"
)

func Home(c *mizu.Ctx) error {
	return view.Render(c, "home", view.Data{
		"Title":   "Welcome to web",
		"Message": "A modern web application built with Mizu",
		"Features": []string{
			"Server-rendered HTML with Go templates",
			"Component-based architecture",
			"Tailwind CSS for styling",
			"Embedded static assets",
		},
	})
}
