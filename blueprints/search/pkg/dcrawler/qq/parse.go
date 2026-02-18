package qq

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// windowDataRe extracts the window.DATA = {...} JSON from article HTML.
	windowDataRe = regexp.MustCompile(`window\.DATA\s*=\s*(\{.+?\})\s*;?\s*(?:</script>|window\.)`)

	// originContentRe extracts originContent from the page HTML.
	// It appears as a separate assignment: originContent: { text: "..." }
	// or sometimes inline in window.DATA.
	originContentRe = regexp.MustCompile(`(?:originContent|window\.originContent)\s*(?:=|:)\s*(\{.+?\})\s*;?\s*(?:</script>|$)`)
)

// ParseArticlePage extracts article data from a news.qq.com article page HTML.
func ParseArticlePage(html string, articleID string) (*Article, error) {
	// Extract window.DATA JSON
	wd, err := extractWindowData(html)
	if err != nil {
		return nil, fmt.Errorf("extract window.DATA: %w", err)
	}

	article := &Article{
		ArticleID: articleID,
		Title:     wd.Title,
		Abstract:  wd.Desc,
		Channel:   wd.Catalog1,
		Source:    wd.Media,
		SourceID:  wd.MediaID,
		URL:       ArticleBaseURL + articleID,
		ImageURL:  wd.ShareImg,
		CommentID: wd.CommentID,
		CrawledAt: time.Now(),
	}

	// Parse article type
	if wd.ArticleType != "" {
		article.ArticleType, _ = strconv.Atoi(wd.ArticleType)
	} else if wd.AType != "" {
		article.ArticleType, _ = strconv.Atoi(wd.AType)
	}

	// Parse publish time
	if wd.PubTime != "" {
		article.PublishTime = parseQQTime(wd.PubTime)
	}

	// Extract content from originContent
	if wd.OriginContent != nil && wd.OriginContent.Text != "" {
		article.Content = decodeUnicodeEscapes(wd.OriginContent.Text)
	} else {
		// Try to extract originContent separately (it's often outside window.DATA)
		content := extractOriginContent(html)
		if content != "" {
			article.Content = decodeUnicodeEscapes(content)
		}
	}

	return article, nil
}

func extractWindowData(html string) (*WindowData, error) {
	matches := windowDataRe.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("window.DATA not found in page")
	}

	var wd WindowData
	if err := json.Unmarshal([]byte(matches[1]), &wd); err != nil {
		return nil, fmt.Errorf("parse window.DATA JSON: %w", err)
	}

	return &wd, nil
}

func extractOriginContent(html string) string {
	matches := originContentRe.FindStringSubmatch(html)
	if len(matches) < 2 {
		return ""
	}

	var oc OriginContent
	if err := json.Unmarshal([]byte(matches[1]), &oc); err != nil {
		return ""
	}

	return oc.Text
}

// parseQQTime parses QQ's time format: "2025-10-29 17:53:02"
func parseQQTime(s string) time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc)
	if err != nil {
		return time.Time{}
	}
	return t
}

// decodeUnicodeEscapes converts \u003c style escapes to actual characters.
func decodeUnicodeEscapes(s string) string {
	// If it looks like it has unicode escapes, try JSON unquoting
	if strings.Contains(s, `\u`) {
		// Wrap in quotes for JSON string parsing
		var decoded string
		if err := json.Unmarshal([]byte(`"`+s+`"`), &decoded); err == nil {
			return decoded
		}
	}
	return s
}

// ExtractArticleID extracts the article ID from a news.qq.com URL.
// e.g., "https://news.qq.com/rain/a/20260217A02A7D00" -> "20260217A02A7D00"
func ExtractArticleID(rawURL string) string {
	// Handle /rain/a/ pattern
	if idx := strings.Index(rawURL, "/rain/a/"); idx != -1 {
		id := rawURL[idx+8:]
		// Trim trailing slash or query params
		if q := strings.IndexAny(id, "?#/"); q != -1 {
			id = id[:q]
		}
		return id
	}
	// Handle /a/ pattern (view.inews.qq.com/a/...)
	if idx := strings.Index(rawURL, "/a/"); idx != -1 {
		id := rawURL[idx+3:]
		if q := strings.IndexAny(id, "?#/"); q != -1 {
			id = id[:q]
		}
		return id
	}
	return ""
}
