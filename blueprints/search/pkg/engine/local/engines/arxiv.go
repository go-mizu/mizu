package engines

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ArXiv implements arXiv academic paper search.
type ArXiv struct {
	*BaseEngine
}

// NewArXiv creates a new ArXiv engine.
func NewArXiv() *ArXiv {
	a := &ArXiv{
		BaseEngine: NewBaseEngine("arxiv", "arx", []Category{CategoryScience}),
	}

	a.SetPaging(true).
		SetMaxPage(10).
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:         "https://arxiv.org",
			WikidataID:      "Q118398",
			OfficialAPIDocs: "https://info.arxiv.org/help/api/index.html",
			UseOfficialAPI:  true,
			Results:         "XML",
		})

	return a
}

func (a *ArXiv) Request(ctx context.Context, query string, params *RequestParams) error {
	maxResults := 10
	start := 0
	if params.PageNo > 1 {
		start = (params.PageNo - 1) * maxResults
	}

	queryParams := url.Values{}
	queryParams.Set("search_query", "all:"+query)
	queryParams.Set("start", fmt.Sprintf("%d", start))
	queryParams.Set("max_results", fmt.Sprintf("%d", maxResults))

	params.URL = "https://export.arxiv.org/api/query?" + queryParams.Encode()
	params.Headers.Set("Accept", "application/atom+xml")

	return nil
}

func (a *ArXiv) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	var feed arxivFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}

	for _, entry := range feed.Entries {
		result := Result{
			URL:      entry.ID,
			Title:    strings.TrimSpace(entry.Title),
			Content:  strings.TrimSpace(entry.Summary),
			Template: "paper",
		}

		// Extract authors
		authors := make([]string, 0, len(entry.Authors))
		for _, author := range entry.Authors {
			authors = append(authors, author.Name)
		}
		result.Authors = authors

		// Parse published date
		if entry.Published != "" {
			if t, err := time.Parse(time.RFC3339, entry.Published); err == nil {
				result.PublishedAt = t
			}
		}

		// Find PDF link
		for _, link := range entry.Links {
			if link.Title == "pdf" {
				// Store PDF URL (could use a custom field or content)
				result.Content = result.Content + " [PDF: " + link.Href + "]"
			}
		}

		// Extract DOI if present
		if entry.DOI != "" {
			result.DOI = entry.DOI
		}

		// Extract journal reference
		if entry.JournalRef != "" {
			result.Journal = entry.JournalRef
		}

		result.ParsedURL, _ = url.Parse(result.URL)
		results.Add(result)
	}

	return results, nil
}

type arxivFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	ID         string       `xml:"id"`
	Title      string       `xml:"title"`
	Summary    string       `xml:"summary"`
	Published  string       `xml:"published"`
	Updated    string       `xml:"updated"`
	Authors    []arxivAuthor `xml:"author"`
	Links      []arxivLink  `xml:"link"`
	DOI        string       `xml:"doi"`
	JournalRef string       `xml:"journal_ref"`
	Comment    string       `xml:"comment"`
	Categories []struct {
		Term string `xml:"term,attr"`
	} `xml:"category"`
}

type arxivAuthor struct {
	Name string `xml:"name"`
}

type arxivLink struct {
	Href  string `xml:"href,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}
