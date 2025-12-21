// app/web/handlers.go
package web

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/mizu"
)

func (s *Server) searchPage(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	text := strings.TrimSpace(c.Query("q"))

	// If no query, render home page
	if text == "" {
		s.render(c, "page/home.html", map[string]any{
			"PageTitle": "FineWiki - Fast Wiki Viewer",
			"Query":     "",
			"Theme":     "",
		})
		return nil
	}

	// Otherwise, perform search
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
		return c.Text(500, err.Error())
	}

	s.render(c, "page/search.html", map[string]any{
		"PageTitle":  fmt.Sprintf("%s - Search - FineWiki", text),
		"Query":      text,
		"WikiName":   wikiname,
		"InLanguage": lang,
		"Results":    results,
		"Theme":      "",
	})
	return nil
}

func (s *Server) page(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	id := strings.TrimSpace(c.Query("id"))
	wikiname := strings.TrimSpace(c.Query("wiki"))
	title := strings.TrimSpace(c.Query("title"))

	switch {
	case id != "":
		p, err := s.view.ByID(ctx, id)
		if err != nil {
			return c.Text(404, err.Error())
		}
		s.render(c, "page/view.html", map[string]any{
			"PageTitle": fmt.Sprintf("%s - FineWiki", p.Title),
			"Query":     "",
			"Page":      p,
			"TOC":       nil,
			"HTML":      template.HTML(""),
			"Theme":     "",
		})
		return nil

	case wikiname != "" && title != "":
		p, err := s.view.ByTitle(ctx, wikiname, title)
		if err != nil {
			return c.Text(404, err.Error())
		}
		s.render(c, "page/view.html", map[string]any{
			"PageTitle": fmt.Sprintf("%s - FineWiki", p.Title),
			"Query":     "",
			"Page":      p,
			"TOC":       nil,
			"HTML":      template.HTML(""),
			"Theme":     "",
		})
		return nil

	default:
		return c.Text(400, "missing id or (wiki,title)")
	}
}
