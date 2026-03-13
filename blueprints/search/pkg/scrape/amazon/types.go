package amazon

import "time"

type Product struct {
	Query        string
	ASIN         string
	Title        string
	URL          string
	ImageURL     string
	PriceText    string
	PriceValue   float64
	Currency     string
	Rating       float64
	ReviewCount  int
	IsPrime      bool
	IsSponsored  bool
	Badge        string
	Position     int
	ResultPage   int
	ScrapedAt    time.Time
	RawContainer string
}

type CrawlStats struct {
	Query      string
	Pages      int
	Products   int
	UniqueASIN int
}
