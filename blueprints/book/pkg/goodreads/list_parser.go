package goodreads

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reListTitle       = regexp.MustCompile(`<h1[^>]*>([^<]+)</h1>`)
	reListDesc        = regexp.MustCompile(`(?i)(?:description|about)[^>]*>\s*(?:<[^>]*>)*\s*"?([^"<]{10,200})"?`)
	reListVoters      = regexp.MustCompile(`([\d,]+)\s*voters?`)
	reListBook        = regexp.MustCompile(`(?s)<tr[^>]*class="[^"]*bookalike[^"]*"[^>]*>([\s\S]*?)</tr>`)
	reListBookTitle   = regexp.MustCompile(`<a[^>]*class="[^"]*bookTitle[^"]*"[^>]*>[\s\S]*?<span[^>]*>([^<]+)</span>`)
	reListBookAuthor  = regexp.MustCompile(`<a[^>]*class="[^"]*authorName[^"]*"[^>]*>[\s\S]*?<span[^>]*>([^<]+)</span>`)
	reListBookCover   = regexp.MustCompile(`<img[^>]*src="([^"]+(?:books|compressed)[^"]+)"`)
	reListBookRating  = regexp.MustCompile(`([\d.]+)\s*avg\s*rating`)
	reListBookRatings = regexp.MustCompile(`([\d,]+)\s*ratings?`)

	// Browse page patterns
	reBrowseList      = regexp.MustCompile(`(?s)<a[^>]*href="(/list/show/(\d+)[^"]*)"[^>]*>([^<]+)</a>`)
	reBrowseListInfo  = regexp.MustCompile(`([\d,]+)\s*books.*?([\d,]+)\s*voters?`)
)

func parseListPage(body string) *GoodreadsList {
	list := &GoodreadsList{}

	// Title
	if m := reListTitle.FindStringSubmatch(body); m != nil {
		list.Title = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	// Description - try to find it near the title
	if m := reListDesc.FindStringSubmatch(body); m != nil {
		list.Description = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	// Voter count
	if m := reListVoters.FindStringSubmatch(body); m != nil {
		list.VoterCount = parseCommaInt(m[1])
	}

	// Parse book entries
	bookBlocks := reListBook.FindAllStringSubmatch(body, 100)
	for _, block := range bookBlocks {
		content := block[1]
		item := GoodreadsListItem{}

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

		if item.Title != "" {
			list.Books = append(list.Books, item)
		}
	}

	return list
}

func parseListsBrowse(body string) []GoodreadsListSummary {
	var lists []GoodreadsListSummary

	// Find list links with /list/show/ pattern
	matches := reBrowseList.FindAllStringSubmatch(body, -1)

	seen := make(map[string]bool)
	for _, m := range matches {
		url := m[1]
		title := strings.TrimSpace(html.UnescapeString(m[3]))

		if seen[url] || title == "" {
			continue
		}
		seen[url] = true

		entry := GoodreadsListSummary{
			Title: title,
			URL:   "https://www.goodreads.com" + url,
		}

		// Try to find book count and voter count near this link
		// Look for the pattern "N books — N voters" in the surrounding context
		idx := strings.Index(body, url)
		if idx >= 0 {
			// Search within ~500 chars after the URL
			end := min(idx+500, len(body))
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
