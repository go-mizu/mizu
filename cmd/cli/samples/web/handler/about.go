package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/view"
)

func About(c *mizu.Ctx) error {
	return view.Render(c, "about", view.Data{
		"Title": "About",
	})
}
