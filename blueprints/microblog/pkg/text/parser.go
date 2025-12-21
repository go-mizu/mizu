// Package text provides parsing for mentions, hashtags, and URLs in post content.
package text

import (
	"regexp"
	"strings"
)

var (
	// mentionRegex matches @username mentions
	mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]+)`)

	// hashtagRegex matches #hashtag tags
	hashtagRegex = regexp.MustCompile(`#([a-zA-Z0-9_]+)`)

	// urlRegex matches URLs
	urlRegex = regexp.MustCompile(`https?://[^\s<>\[\]()]+`)
)

// Entities contains parsed entities from post content.
type Entities struct {
	Mentions []Mention
	Hashtags []Hashtag
	URLs     []URL
}

// Mention represents an @mention in content.
type Mention struct {
	Username string
	Start    int
	End      int
}

// Hashtag represents a #hashtag in content.
type Hashtag struct {
	Tag   string // Lowercase
	Raw   string // Original case
	Start int
	End   int
}

// URL represents a URL in content.
type URL struct {
	URL   string
	Start int
	End   int
}

// Parse extracts entities from post content.
func Parse(content string) Entities {
	var entities Entities

	// Extract mentions
	for _, match := range mentionRegex.FindAllStringSubmatchIndex(content, -1) {
		if len(match) >= 4 {
			entities.Mentions = append(entities.Mentions, Mention{
				Username: content[match[2]:match[3]],
				Start:    match[0],
				End:      match[1],
			})
		}
	}

	// Extract hashtags
	for _, match := range hashtagRegex.FindAllStringSubmatchIndex(content, -1) {
		if len(match) >= 4 {
			raw := content[match[2]:match[3]]
			entities.Hashtags = append(entities.Hashtags, Hashtag{
				Tag:   strings.ToLower(raw),
				Raw:   raw,
				Start: match[0],
				End:   match[1],
			})
		}
	}

	// Extract URLs
	for _, match := range urlRegex.FindAllStringIndex(content, -1) {
		entities.URLs = append(entities.URLs, URL{
			URL:   content[match[0]:match[1]],
			Start: match[0],
			End:   match[1],
		})
	}

	return entities
}

// ExtractMentions returns all usernames mentioned in content.
func ExtractMentions(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	usernames := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 && !seen[match[1]] {
			usernames = append(usernames, match[1])
			seen[match[1]] = true
		}
	}
	return usernames
}

// ExtractHashtags returns all hashtags in content (lowercase, deduplicated).
func ExtractHashtags(content string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(content, -1)
	tags := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			tag := strings.ToLower(match[1])
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}
	return tags
}

// ToHTML converts content with entities to HTML with links.
func ToHTML(content string) string {
	entities := Parse(content)

	// Sort entities by position (reverse order to replace from end)
	type entity struct {
		start int
		end   int
		html  string
	}
	var all []entity

	for _, m := range entities.Mentions {
		all = append(all, entity{
			start: m.Start,
			end:   m.End,
			html:  `<a href="/@` + m.Username + `" class="mention">@` + m.Username + `</a>`,
		})
	}

	for _, h := range entities.Hashtags {
		all = append(all, entity{
			start: h.Start,
			end:   h.End,
			html:  `<a href="/tags/` + h.Tag + `" class="hashtag">#` + h.Raw + `</a>`,
		})
	}

	for _, u := range entities.URLs {
		all = append(all, entity{
			start: u.Start,
			end:   u.End,
			html:  `<a href="` + u.URL + `" target="_blank" rel="noopener">` + u.URL + `</a>`,
		})
	}

	// Sort by start position descending
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].start > all[i].start {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	// Replace from end to start to preserve positions
	result := content
	for _, e := range all {
		result = result[:e.start] + e.html + result[e.end:]
	}

	return result
}

// Truncate truncates content to maxLen, adding ellipsis if needed.
func Truncate(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen-1]) + "…"
}

// CharCount returns the character count of content (Unicode-aware).
func CharCount(content string) int {
	return len([]rune(content))
}
