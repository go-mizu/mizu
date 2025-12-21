// app/web/handlers.go
package web

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
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
			"IsHome":    true,
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

	// If exactly one result, redirect directly to that page
	if len(results) == 1 {
		r := results[0]
		return c.Redirect(302, fmt.Sprintf("/page?id=%s", r.ID))
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

func (s *Server) randomPage(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	id, err := s.view.RandomID(ctx)
	if err != nil {
		return c.Text(500, "No pages available")
	}
	return c.Redirect(302, fmt.Sprintf("/page?id=%s", id))
}

func (s *Server) page(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	id := strings.TrimSpace(c.Query("id"))
	wikiname := strings.TrimSpace(c.Query("wiki"))
	title := strings.TrimSpace(c.Query("title"))

	var p *view.Page
	var err error

	switch {
	case id != "":
		p, err = s.view.ByID(ctx, id)
	case wikiname != "" && title != "":
		p, err = s.view.ByTitle(ctx, wikiname, title)
	default:
		return c.Text(400, "missing id or (wiki,title)")
	}

	if err != nil {
		return c.Text(404, err.Error())
	}

	// Parse infoboxes, format dates, and compute read stats for display
	_ = p.ParseInfoboxes()
	p.FormatDates()
	p.ComputeReadStats()

	// Render page content (uses WikiText if available for preserved links)
	var htmlContent template.HTML
	rendered, err := view.RenderPage(p)
	if err == nil {
		htmlContent = template.HTML(rendered)
	}

	s.render(c, "page/view.html", map[string]any{
		"PageTitle": fmt.Sprintf("%s - FineWiki", p.Title),
		"Query":     "",
		"Page":      p,
		"Infoboxes": p.Infoboxes,
		"TOC":       nil,
		"HTML":      htmlContent,
		"Theme":     "",
	})
	return nil
}
