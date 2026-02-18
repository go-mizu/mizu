package qq

import "time"

// Article represents a news article from news.qq.com.
type Article struct {
	ArticleID   string    `json:"article_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`      // HTML body from originContent.text
	Abstract    string    `json:"abstract"`      // desc
	PublishTime time.Time `json:"publish_time"`
	Channel     string    `json:"channel"`       // catalog1
	Source      string    `json:"source"`        // media name
	SourceID    string    `json:"source_id"`     // media_id
	ArticleType int       `json:"article_type"`  // 0=article, 4=video
	URL         string    `json:"url"`
	ImageURL    string    `json:"image_url"`     // shareImg
	CommentID   string    `json:"comment_id"`
	CrawledAt   time.Time `json:"crawled_at"`
	StatusCode  int       `json:"status_code"`
	Error       string    `json:"error,omitempty"`
}

// WindowData represents the window.DATA object embedded in article pages.
type WindowData struct {
	URL            string `json:"url"`
	ArticleID      string `json:"article_id"`
	Title          string `json:"title"`
	Desc           string `json:"desc"`
	Catalog1       string `json:"catalog1"`
	Media          string `json:"media"`
	MediaID        string `json:"media_id"`
	PubTime        string `json:"pubtime"`
	CommentID      string `json:"comment_id"`
	CMSID          string `json:"cmsId"`
	ShareImg       string `json:"shareImg"`
	ArticleType    string `json:"article_type"`
	AType          string `json:"atype"`
	OriginContent  *OriginContent `json:"originContent"`
}

// OriginContent holds the actual article body.
type OriginContent struct {
	Text string `json:"text"`
}

// ── Feed API response types ──

// HotRankingResponse is the response from hot_ranking_list.
type HotRankingResponse struct {
	Ret    int `json:"ret"`
	IDList []struct {
		NewsList []HotNewsItem `json:"newslist"`
	} `json:"idlist"`
}

// HotNewsItem represents a single item in the hot ranking.
type HotNewsItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Abstract    string `json:"abstract"`
	Time        string `json:"time"`
	ArticleType string `json:"articletype"`
	Source      string `json:"source"`
	MediaID     string `json:"media_id"`
	ChlID       string `json:"chlid"`
	ChlName     string `json:"chlname"`
	ShareURL    string `json:"shareUrl"`
}

// FeedRequest is the request body for getPCList.
type FeedRequest struct {
	QImei36    string          `json:"qimei36"`
	Forward    string          `json:"forward"`
	BaseReq    FeedBaseReq     `json:"base_req"`
	FlushNum   *int            `json:"flush_num"`
	ChannelID  string          `json:"channel_id"`
	DeviceID   string          `json:"device_id"`
	IsLocalCh  string          `json:"is_local_chlid"`
}

// FeedBaseReq is the base_req field in feed requests.
type FeedBaseReq struct {
	From string `json:"from"`
}

// FeedResponse is the response from getPCList.
type FeedResponse struct {
	Data []FeedDataItem `json:"data"`
}

// FeedDataItem is a top-level item in the feed (may contain sub_items).
type FeedDataItem struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	ArticleType string         `json:"articletype"`
	SubItems    []FeedSubItem  `json:"sub_item"`
}

// FeedSubItem represents a nested article in a feed data item.
type FeedSubItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ArticleType string `json:"articletype"`
	PublishTime string `json:"publish_time"`
}

// FeedNewsItem represents a flattened news item extracted from feed.
type FeedNewsItem struct {
	ID          string
	Title       string
	ArticleType string
}

// ── Sitemap types ──

// SitemapIndex represents a sitemap index XML.
type SitemapIndex struct {
	Sitemaps []SitemapEntry `xml:"sitemap"`
}

// SitemapEntry is a single sitemap reference in the index.
type SitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// URLSet represents an individual sitemap XML.
type URLSet struct {
	URLs []SitemapURL `xml:"url"`
}

// SitemapURL is a single URL entry in a sitemap.
type SitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}
