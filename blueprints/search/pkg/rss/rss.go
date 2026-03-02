package rss

import "time"

// Feed is a generic representation of an RSS or Atom feed.
type Feed struct {
	Title       string
	Description string
	Link        string
	FeedLink    string
	Items       []*Item
	Source      *Feed
}

// Item is a generic representation of an item in a feed.
type Item struct {
	Title       string
	Description string
	Content     string
	Link        string
	Published   *time.Time
	GUID        string
}

// OPML is a representation of an OPML file.
type OPML struct {
	Title    string
	Outlines []Outline
}

// Outline is an outline in an OPML file.
type Outline struct {
	Text     string
	Title    string
	Type     string
	XMLURL   string
	HTMLURL  string
	Outlines []Outline
}
