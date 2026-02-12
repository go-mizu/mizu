package goodreads

import (
	"html"
	"regexp"
	"strings"
)

var (
	reAuthorName      = regexp.MustCompile(`<h1[^>]*class="[^"]*authorName[^"]*"[^>]*>\s*<span[^>]*>([^<]+)</span>`)
	reAuthorNameAlt   = regexp.MustCompile(`<title>([^(<]+?)(?:\s*\(Author)`)
	reAuthorBio       = regexp.MustCompile(`(?s)<div[^>]*class="[^"]*aboutAuthorInfo[^"]*"[^>]*>([\s\S]*?)</div>`)
	reAuthorBioSpan   = regexp.MustCompile(`(?s)<span[^>]*>([\s\S]*?)</span>`)
	reAuthorPhoto     = regexp.MustCompile(`<img[^>]*(?:itemprop="image"|class="[^"]*authorPhoto[^"]*")[^>]*src="([^"]+)"`)
	reAuthorBorn      = regexp.MustCompile(`(?i)(?:born|Born)\s*(?:in\s+)?([A-Z][a-z]+\s+\d{1,2},\s+\d{4})`)
	reAuthorBornData  = regexp.MustCompile(`(?i)Born\s*</dt>\s*<dd[^>]*>([^<]+)`)
	reAuthorDied      = regexp.MustCompile(`(?i)(?:died|Died)\s*(?:in\s+)?([A-Z][a-z]+\s+\d{1,2},\s+\d{4})`)
	reAuthorDiedData  = regexp.MustCompile(`(?i)Died\s*</dt>\s*<dd[^>]*>([^<]+)`)
	reAuthorWorks     = regexp.MustCompile(`(\d[\d,]*)\s*(?:distinct\s+)?works?`)
	reAuthorFollowers = regexp.MustCompile(`(\d[\d,]*)\s*followers?`)
	reAuthorGenre     = regexp.MustCompile(`<a[^>]*href="/genres/[^"]*"[^>]*>([^<]+)</a>`)
	reAuthorWebsite   = regexp.MustCompile(`(?is)Website\s*</dt>\s*<dd[^>]*>[\s\S]*?href="([^"]+)"`)
	reAuthorInfluence = regexp.MustCompile(`(?is)Influences?\s*</dt>\s*<dd[^>]*>([\s\S]*?)</dd>`)
	reAuthorLinkName  = regexp.MustCompile(`>([^<]+)</a>`)
)

func parseAuthorPage(body string) *GoodreadsAuthor {
	a := &GoodreadsAuthor{}

	// Name
	if m := reAuthorName.FindStringSubmatch(body); m != nil {
		a.Name = strings.TrimSpace(html.UnescapeString(m[1]))
	} else if m := reAuthorNameAlt.FindStringSubmatch(body); m != nil {
		a.Name = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	// Bio
	if m := reAuthorBio.FindStringSubmatch(body); m != nil {
		content := m[1]
		spans := reAuthorBioSpan.FindAllStringSubmatch(content, -1)
		longest := ""
		for _, s := range spans {
			text := stripTags(s[1])
			if len(text) > len(longest) {
				longest = text
			}
		}
		if longest != "" {
			a.Bio = strings.TrimSpace(html.UnescapeString(longest))
		} else {
			a.Bio = strings.TrimSpace(html.UnescapeString(stripTags(content)))
		}
	}

	// Photo
	if m := reAuthorPhoto.FindStringSubmatch(body); m != nil {
		a.PhotoURL = m[1]
	}

	// Born date
	if m := reAuthorBornData.FindStringSubmatch(body); m != nil {
		a.BornDate = strings.TrimSpace(m[1])
	} else if m := reAuthorBorn.FindStringSubmatch(body); m != nil {
		a.BornDate = strings.TrimSpace(m[1])
	}

	// Died date
	if m := reAuthorDiedData.FindStringSubmatch(body); m != nil {
		a.DiedDate = strings.TrimSpace(m[1])
	} else if m := reAuthorDied.FindStringSubmatch(body); m != nil {
		a.DiedDate = strings.TrimSpace(m[1])
	}

	// Works count
	if m := reAuthorWorks.FindStringSubmatch(body); m != nil {
		a.WorksCount = parseCommaInt(m[1])
	}

	// Followers
	if m := reAuthorFollowers.FindStringSubmatch(body); m != nil {
		a.Followers = parseCommaInt(m[1])
	}

	// Genres
	matches := reAuthorGenre.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool)
	var genres []string
	for _, m := range matches {
		genre := strings.TrimSpace(m[1])
		if genre == "" || seen[genre] {
			continue
		}
		seen[genre] = true
		genres = append(genres, genre)
	}
	a.Genres = strings.Join(genres, ", ")

	// Website
	if m := reAuthorWebsite.FindStringSubmatch(body); m != nil {
		a.Website = strings.TrimSpace(m[1])
	}

	// Influences
	if m := reAuthorInfluence.FindStringSubmatch(body); m != nil {
		var influences []string
		seenInfluence := make(map[string]bool)
		for _, link := range reAuthorLinkName.FindAllStringSubmatch(m[1], -1) {
			name := strings.TrimSpace(html.UnescapeString(link[1]))
			if name == "" || seenInfluence[name] {
				continue
			}
			seenInfluence[name] = true
			influences = append(influences, name)
		}
		a.Influences = strings.Join(influences, ", ")
	}

	return a
}
