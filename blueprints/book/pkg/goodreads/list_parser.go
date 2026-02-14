package goodreads

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reListTitle       = regexp.MustCompile(`<h1[^>]*>([^<]+)</h1>`)
	reListHTMLTitle   = regexp.MustCompile(`(?is)<title>\s*([^<]+)\s*</title>`)
	reListDesc        = regexp.MustCompile(`(?i)(?:description|about)[^>]*>\s*(?:<[^>]*>)*\s*"?([^"<]{10,200})"?`)
	reListVoters      = regexp.MustCompile(`([\d,]+)\s*voters?`)
	reListBook        = regexp.MustCompile(`(?s)<tr[^>]*class="[^"]*bookalike[^"]*"[^>]*>([\s\S]*?)</tr>`)
	reListBookTable   = regexp.MustCompile(`(?s)<tr[^>]*itemscope[^>]*itemtype="http://schema.org/Book"[^>]*>([\s\S]*?)</tr>`)
	reListBookRank    = regexp.MustCompile(`(?s)<td[^>]*class="[^"]*number[^"]*"[^>]*>\s*([0-9]+)\s*</td>`)
	reListBookURL     = regexp.MustCompile(`<a[^>]*class="[^"]*bookTitle[^"]*"[^>]*href="([^"]+)"`)
	reListBookTitle   = regexp.MustCompile(`<a[^>]*class="[^"]*bookTitle[^"]*"[^>]*>[\s\S]*?<span[^>]*>([^<]+)</span>`)
	reListBookAuthor  = regexp.MustCompile(`<a[^>]*class="[^"]*authorName[^"]*"[^>]*>[\s\S]*?<span[^>]*>([^<]+)</span>`)
	reListBookCover   = regexp.MustCompile(`<img[^>]*src="([^"]+(?:books|compressed)[^"]+)"`)
	reListBookRating  = regexp.MustCompile(`([\d.]+)\s*avg\s*rating`)
	reListBookRatings = regexp.MustCompile(`([\d,]+)\s*ratings?`)
	reListBookScore   = regexp.MustCompile(`(?i)score:\s*([\d,]+)\s*,\s*and\s*([\d,]+)\s*people\s*voted`)

	// Browse page patterns
	reBrowseList     = regexp.MustCompile(`(?s)<a[^>]*href="(/list/show/(\d+)[^"]*)"[^>]*>([^<]+)</a>`)
	reBrowseListTitle = regexp.MustCompile(`(?s)<a[^>]*class="[^"]*listTitle[^"]*"[^>]*href="(/list/show/(\d+)[^"]*)"[^>]*>([^<]+)</a>`)
	reBrowseListInfo = regexp.MustCompile(`(?s)([\d,]+)\s*books.*?([\d,]+)\s*voters?`)
)

func parseListPage(body string) *GoodreadsList {
	list := &GoodreadsList{}

	// Title
	if m := reListTitle.FindStringSubmatch(body); m != nil {
		list.Title = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	// Some pages use a "Score" modal h1 before the real page title.
	if list.Title == "" || strings.EqualFold(list.Title, "score") {
		if m := reListHTMLTitle.FindStringSubmatch(body); m != nil {
			title := strings.TrimSpace(html.UnescapeString(m[1]))
			if idx := strings.Index(title, "("); idx > 0 {
				title = strings.TrimSpace(title[:idx])
			}
			if title != "" {
				list.Title = title
			}
		}
	}

	// Description - try to find it near the title
	if m := reListDesc.FindStringSubmatch(body); m != nil {
		list.Description = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	// Voter count
	if all := reListVoters.FindAllStringSubmatch(body, -1); len(all) > 0 {
		maxVotes := 0
		for _, m := range all {
			if v := parseCommaInt(m[1]); v > maxVotes {
				maxVotes = v
			}
		}
		list.VoterCount = maxVotes
	}

	// Parse book entries
	bookBlocks := reListBook.FindAllStringSubmatch(body, 100)
	if len(bookBlocks) == 0 {
		bookBlocks = reListBookTable.FindAllStringSubmatch(body, 100)
	}
	for i, block := range bookBlocks {
		content := block[1]
		item := GoodreadsListItem{Position: i + 1}

		if m := reListBookRank.FindStringSubmatch(content); m != nil {
			item.Position = parseCommaInt(m[1])
		}
		if m := reListBookURL.FindStringSubmatch(content); m != nil {
			url := strings.TrimSpace(m[1])
			if strings.HasPrefix(url, "/") {
				item.URL = "https://www.goodreads.com" + url
			} else {
				item.URL = url
			}
			if id := extractListBookID(url); id != "" {
				item.GoodreadsID = id
			}
		}

		if m := reListBookTitle.FindStringSubmatch(content); m != nil {
			item.Title = strings.TrimSpace(html.UnescapeString(m[1]))
		}
		if m := reListBookAuthor.FindStringSubmatch(content); m != nil {
			item.AuthorName = strings.TrimSpace(html.UnescapeString(m[1]))
		}
		if m := reListBookCover.FindStringSubmatch(content); m != nil {
			item.CoverURL = m[1]
		}
		if m := reListBookRating.FindStringSubmatch(content); m != nil {
			item.AverageRating, _ = strconv.ParseFloat(m[1], 64)
		}
		if m := reListBookRatings.FindStringSubmatch(content); m != nil {
			item.RatingsCount = parseCommaInt(m[1])
		}
		if m := reListBookScore.FindStringSubmatch(content); m != nil {
			item.Score = parseCommaInt(m[1])
			item.Voters = parseCommaInt(m[2])
		}

		if item.Title != "" {
			list.Books = append(list.Books, item)
		}
	}

	return list
}

func parseListsBrowse(body string) []GoodreadsListSummary {
	var lists []GoodreadsListSummary

	// Prefer explicit list title links used by server-rendered popular/tag pages.
	matches := reBrowseListTitle.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		// Fallback for older or alternate layouts.
		matches = reBrowseList.FindAllStringSubmatch(body, -1)
	}

	seen := make(map[string]bool)
	for _, m := range matches {
		anchorHTML := m[0]
		url := m[1]
		title := strings.TrimSpace(html.UnescapeString(m[3]))

		if seen[url] || title == "" {
			continue
		}
		seen[url] = true

		entry := GoodreadsListSummary{
			GoodreadsID: m[2],
			Title:       title,
			URL:         "https://www.goodreads.com" + url,
		}

		// Try to find book count and voter count near this link
		// Look for the pattern "N books — N voters" in the surrounding context
		idx := strings.Index(body, anchorHTML)
		if idx >= 0 {
			// Search near the title anchor where list metadata is rendered.
			end := min(idx+1200, len(body))
			snippet := body[idx:end]

			if info := reBrowseListInfo.FindStringSubmatch(snippet); info != nil {
				entry.BookCount = parseCommaInt(info[1])
				entry.VoterCount = parseCommaInt(info[2])
			}
		}

		lists = append(lists, entry)
	}

	return lists
}

func extractListBookID(rawURL string) string {
	if m := reSearchBookID.FindStringSubmatch(rawURL); m != nil {
		return m[1]
	}
	return ""
}
