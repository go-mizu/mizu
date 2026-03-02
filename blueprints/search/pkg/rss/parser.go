package rss

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gilliek/go-opml/opml"
	"github.com/mmcdole/gofeed"
)

// ParseURL fetches and parses a feed (RSS/Atom) or OPML file from a URL.
func ParseURL(url string) (interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try OPML if it looks like OPML
	if bytes.Contains(body, []byte("<opml")) {
		doc, err := opml.NewOPML(body)
		if err == nil {
			return toGenericOPML(doc), nil
		}
	}

	// Try gofeed
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(string(body))
	if err != nil {
		return nil, err
	}
	return toGenericFeed(feed), nil
}

func toGenericFeed(f *gofeed.Feed) *Feed {
	items := make([]*Item, len(f.Items))
	for i, item := range f.Items {
		var published *time.Time
		if item.PublishedParsed != nil {
			published = item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			published = item.UpdatedParsed
		}
		items[i] = &Item{
			Title:       item.Title,
			Description: item.Description,
			Content:     item.Content,
			Link:        item.Link,
			Published:   published,
			GUID:        item.GUID,
		}
	}

	return &Feed{
		Title:       f.Title,
		Description: f.Description,
		Link:        f.Link,
		FeedLink:    f.FeedLink,
		Items:       items,
	}
}

func toGenericOPML(doc *opml.OPML) *OPML {
	return &OPML{
		Title:    doc.Head.Title,
		Outlines: toGenericOutlines(doc.Body.Outlines),
	}
}

func toGenericOutlines(outlines []opml.Outline) []Outline {
	genericOutlines := make([]Outline, len(outlines))
	for i, o := range outlines {
		genericOutlines[i] = Outline{
			Text:     o.Text,
			Title:    o.Title,
			Type:     o.Type,
			XMLURL:   o.XMLURL,
			HTMLURL:  o.HTMLURL,
			Outlines: toGenericOutlines(o.Outlines),
		}
	}
	return genericOutlines
}
