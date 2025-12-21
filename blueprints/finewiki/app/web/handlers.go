// app/web/handlers.go
package web

import (
	"html/template"
	"strings"

	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/mizu"
)

func (s *Server) home(c *mizu.Ctx) {
	s.render(c, "page/home.html", map[string]any{
		"Query": "",
		"Theme": "",
	})
}

func (s *Server) searchPage(c *mizu.Ctx) {
	ctx := c.Request().Context()

	text := strings.TrimSpace(c.Query("q"))
	wikiname := strings.TrimSpace(c.Query("wiki"))
	lang := strings.TrimSpace(c.Query("lang"))

	results, err := s.search.Search(ctx, search.Query{
		Text:       text,
		WikiName:   wikiname,
		InLanguage: lang,
		Limit:      20,
		EnableFTS:  false,
	})
	if err != nil {
		c.Text(500, err.Error())
		return
	}

	s.render(c, "page/search.html", map[string]any{
		"Query":      text,
		"WikiName":   wikiname,
		"InLanguage": lang,
		"Results":    results,
		"Theme":      "",
	})
}

func (s *Server) page(c *mizu.Ctx) {
	ctx := c.Request().Context()

	id := strings.TrimSpace(c.Query("id"))
	wikiname := strings.TrimSpace(c.Query("wiki"))
	title := strings.TrimSpace(c.Query("title"))

	switch {
	case id != "":
		p, err := s.view.ByID(ctx, id)
		if err != nil {
			c.Text(404, err.Error())
			return
		}
		s.render(c, "page/view.html", map[string]any{
			"Query": "",
			"Page":  p,
			"TOC":   nil,
			"HTML":  template.HTML(""),
			"Theme": "",
		})
		return

	case wikiname != "" && title != "":
		p, err := s.view.ByTitle(ctx, wikiname, title)
		if err != nil {
			c.Text(404, err.Error())
			return
		}
		s.render(c, "page/view.html", map[string]any{
			"Query": "",
			"Page":  p,
			"TOC":   nil,
			"HTML":  template.HTML(""),
			"Theme": "",
		})
		return

	default:
		c.Text(400, "missing id or (wiki,title)")
	}
}
