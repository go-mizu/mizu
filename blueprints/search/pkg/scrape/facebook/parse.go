package facebook

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var countPattern = regexp.MustCompile(`(?i)([\d.,]+)\s*([kmb])?`)

func NormalizeURL(raw string, preferMBasic bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		if strings.HasPrefix(raw, "/") {
			raw = BaseURL + raw
		} else if strings.HasPrefix(raw, "groups/") || strings.HasPrefix(raw, "profile.php") {
			raw = BaseURL + "/" + raw
		} else {
			raw = BaseURL + "/" + strings.TrimPrefix(raw, "/")
		}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := strings.ToLower(u.Host)
	if strings.Contains(host, "facebook.com") {
		if preferMBasic {
			u.Scheme = "https"
			u.Host = "mbasic.facebook.com"
		} else {
			u.Scheme = "https"
			u.Host = "www.facebook.com"
		}
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

func CanonicalURL(raw string) string {
	u, err := url.Parse(NormalizeURL(raw, false))
	if err != nil {
		return raw
	}
	u.Host = "www.facebook.com"
	return u.String()
}

func NormalizePostURL(raw string) string {
	return NormalizeURL(raw, true)
}

func NormalizePageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return NormalizeURL(raw, true)
	}
	if strings.HasPrefix(raw, "pages/") {
		return NormalizeURL(raw, true)
	}
	return NormalizeURL("/"+strings.Trim(raw, "/"), true)
}

func NormalizeProfileURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return NormalizeURL(raw, true)
	}
	if _, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return NormalizeURL("/profile.php?id="+raw, true)
	}
	return NormalizeURL("/"+strings.Trim(raw, "/"), true)
}

func NormalizeGroupURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return NormalizeURL(raw, true)
	}
	return NormalizeURL("/groups/"+strings.Trim(raw, "/"), true)
}

func BuildSearchURL(query, searchType string, page int) string {
	if page <= 0 {
		page = 1
	}
	path := "/search/top/"
	switch searchType {
	case "posts":
		path = "/search/posts/"
	case "pages":
		path = "/search/pages/"
	case "people":
		path = "/search/people/"
	case "groups":
		path = "/search/groups/"
	}
	v := url.Values{}
	v.Set("q", query)
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	return MBasicURL + path + "?" + v.Encode()
}

func ExtractFacebookID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil {
		if id := u.Query().Get("id"); id != "" {
			return id
		}
		if id := u.Query().Get("story_fbid"); id != "" {
			return id
		}
	}
	return ""
}

func InferEntityType(rawURL string) string {
	rawURL = NormalizeURL(rawURL, true)
	if strings.Contains(rawURL, "/groups/") {
		if strings.Contains(rawURL, "/posts/") {
			return EntityPost
		}
		return EntityGroup
	}
	if strings.Contains(rawURL, "story.php") || strings.Contains(rawURL, "/posts/") || strings.Contains(rawURL, "/permalink/") {
		return EntityPost
	}
	if strings.Contains(rawURL, "/search/") {
		return EntitySearch
	}
	if strings.Contains(rawURL, "profile.php?id=") {
		return EntityProfile
	}
	path := mustURLPath(rawURL)
	parts := pathParts(path)
	if len(parts) == 1 && parts[0] != "" {
		return EntityPage
	}
	return EntityPage
}

func ParsePage(doc *goquery.Document, rawURL string) *Page {
	name := firstNonEmpty(
		cleanText(doc.Find("title").First().Text()),
		cleanText(doc.Find("h1").First().Text()),
		cleanText(doc.Find("strong").First().Text()),
	)
	path := pathParts(mustURLPath(rawURL))
	slug := ""
	if len(path) > 0 {
		slug = path[0]
	}
	pageID := ExtractFacebookID(rawURL)
	if pageID == "" {
		pageID = slug
	}
	body := cleanText(doc.Find("body").Text())
	return &Page{
		PageID:         defaultString(pageID, slug),
		Slug:           slug,
		Name:           stripTitleSuffix(name),
		Category:       findLabeledValue(body, "Page"),
		About:          truncateString(body, 500),
		LikesCount:     findCountNear(body, "likes"),
		FollowersCount: findCountNear(body, "followers"),
		Verified:       strings.Contains(strings.ToLower(body), "verified"),
		Website:        findFirstExternalLink(doc),
		Phone:          findPhone(body),
		Address:        "",
		URL:            CanonicalURL(rawURL),
		FetchedAt:      time.Now(),
	}
}

func ParseProfile(doc *goquery.Document, rawURL string) *Profile {
	name := firstNonEmpty(
		cleanText(doc.Find("title").First().Text()),
		cleanText(doc.Find("h1").First().Text()),
		cleanText(doc.Find("strong").First().Text()),
	)
	u, _ := url.Parse(rawURL)
	username := ""
	if u != nil && u.Path != "" && !strings.Contains(u.Path, "profile.php") {
		parts := pathParts(u.Path)
		if len(parts) > 0 {
			username = parts[0]
		}
	}
	profileID := ExtractFacebookID(rawURL)
	if profileID == "" {
		profileID = username
	}
	body := cleanText(doc.Find("body").Text())
	return &Profile{
		ProfileID:      defaultString(profileID, username),
		Username:       username,
		Name:           stripTitleSuffix(name),
		Intro:          truncateString(body, 240),
		Bio:            truncateString(body, 600),
		FollowersCount: findCountNear(body, "followers"),
		FriendsCount:   findCountNear(body, "friends"),
		Verified:       strings.Contains(strings.ToLower(body), "verified"),
		Hometown:       findLabeledValue(body, "Lives in"),
		CurrentCity:    findLabeledValue(body, "From"),
		Work:           findLabeledValue(body, "Works at"),
		Education:      findLabeledValue(body, "Studied at"),
		URL:            CanonicalURL(rawURL),
		FetchedAt:      time.Now(),
	}
}

func ParseGroup(doc *goquery.Document, rawURL string) *Group {
	name := firstNonEmpty(
		cleanText(doc.Find("title").First().Text()),
		cleanText(doc.Find("h1").First().Text()),
		cleanText(doc.Find("strong").First().Text()),
	)
	path := pathParts(mustURLPath(rawURL))
	slug := ""
	groupID := ExtractFacebookID(rawURL)
	for i, part := range path {
		if part == "groups" && i+1 < len(path) {
			slug = path[i+1]
			if groupID == "" {
				groupID = slug
			}
		}
	}
	body := cleanText(doc.Find("body").Text())
	return &Group{
		GroupID:      defaultString(groupID, slug),
		Slug:         slug,
		Name:         stripTitleSuffix(name),
		Description:  truncateString(body, 700),
		Privacy:      firstMatching(body, []string{"Public group", "Private group", "Visible", "Hidden"}),
		MembersCount: findCountNear(body, "members"),
		URL:          CanonicalURL(rawURL),
		FetchedAt:    time.Now(),
	}
}

func ParsePosts(doc *goquery.Document, rawURL, ownerID, ownerName, ownerType string) []Post {
	seen := map[string]bool{}
	var posts []Post

	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if !looksLikePostLink(href) {
			return
		}
		permalink := NormalizeURL(resolveRelative(rawURL, href), true)
		postID := extractPostID(permalink)
		if postID == "" || seen[postID] {
			return
		}
		seen[postID] = true

		container := nearestRichContainer(a)
		text := cleanText(container.Text())
		mediaURLs, external := extractLinks(container, rawURL)

		posts = append(posts, Post{
			PostID:        postID,
			OwnerID:       ownerID,
			OwnerName:     ownerName,
			OwnerType:     ownerType,
			Text:          truncateString(removeEcho(text, ownerName), 4000),
			CreatedAtText: cleanText(a.Text()),
			LikeCount:     findCountNear(text, "like"),
			CommentCount:  findCountNear(text, "comment"),
			ShareCount:    findCountNear(text, "share"),
			Permalink:     CanonicalURL(permalink),
			MediaURLs:     mediaURLs,
			ExternalLinks: external,
			FetchedAt:     time.Now(),
		})
	})
	return posts
}

func ParseComments(doc *goquery.Document, postID, rawURL string, limit int) []Comment {
	if limit <= 0 {
		limit = 100
	}
	var out []Comment
	seen := map[string]bool{}
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		if len(out) >= limit {
			return
		}
		href, _ := a.Attr("href")
		if !looksLikeCommentLink(href) {
			return
		}
		permalink := NormalizeURL(resolveRelative(rawURL, href), true)
		commentID := extractCommentID(permalink)
		if commentID == "" || seen[commentID] {
			return
		}
		seen[commentID] = true

		container := nearestRichContainer(a)
		text := cleanText(container.Text())
		authorName := cleanText(a.Text())

		out = append(out, Comment{
			CommentID:     commentID,
			PostID:        postID,
			AuthorID:      extractProfileOrPageID(permalink),
			AuthorName:    authorName,
			Text:          truncateString(text, 2000),
			CreatedAtText: "",
			LikeCount:     findCountNear(text, "like"),
			Permalink:     CanonicalURL(permalink),
			FetchedAt:     time.Now(),
		})
	})
	return out
}

func ParseSearchResults(doc *goquery.Document, query, rawURL string) []SearchResult {
	seen := map[string]bool{}
	var out []SearchResult
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		full := NormalizeURL(resolveRelative(rawURL, href), true)
		if !strings.Contains(full, "facebook.com") {
			return
		}
		entityType := InferEntityType(full)
		if entityType == EntitySearch || seen[full] {
			return
		}
		seen[full] = true
		out = append(out, SearchResult{
			Query:      query,
			ResultURL:  CanonicalURL(full),
			EntityType: entityType,
			Title:      truncateString(cleanText(a.Text()), 200),
			Snippet:    truncateString(cleanText(nearestRichContainer(a).Text()), 500),
			FetchedAt:  time.Now(),
		})
	})
	return out
}

func DiscoverLinks(doc *goquery.Document, rawURL string) []QueueItem {
	seen := map[string]bool{}
	var out []QueueItem
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		full := NormalizeURL(resolveRelative(rawURL, href), true)
		if !strings.Contains(full, "facebook.com") {
			return
		}
		entityType := InferEntityType(full)
		if entityType == EntitySearch || seen[full] {
			return
		}
		seen[full] = true
		priority := 1
		if entityType == EntityPost {
			priority = 5
		}
		out = append(out, QueueItem{URL: full, EntityType: entityType, Priority: priority})
	})
	return out
}

func ParseNextPage(doc *goquery.Document, rawURL string) string {
	var next string
	doc.Find("a[href]").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		txt := strings.ToLower(cleanText(a.Text()))
		if strings.Contains(txt, "see more") || strings.Contains(txt, "more stories") || strings.Contains(txt, "more results") || txt == "more" {
			href, _ := a.Attr("href")
			next = NormalizeURL(resolveRelative(rawURL, href), true)
			return false
		}
		return true
	})
	return next
}

func findFirstExternalLink(doc *goquery.Document) string {
	var link string
	doc.Find("a[href]").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		href, _ := a.Attr("href")
		if href == "" {
			return true
		}
		full := resolveRelative(BaseURL, href)
		if !strings.Contains(full, "facebook.com") && strings.HasPrefix(full, "http") {
			link = full
			return false
		}
		return true
	})
	return link
}

func extractLinks(sel *goquery.Selection, rawURL string) ([]string, []string) {
	var media, external []string
	seenMedia := map[string]bool{}
	seenExt := map[string]bool{}
	sel.Find("a[href],img[src]").Each(func(_ int, n *goquery.Selection) {
		if src, ok := n.Attr("src"); ok {
			full := resolveRelative(rawURL, src)
			if !seenMedia[full] {
				seenMedia[full] = true
				media = append(media, full)
			}
		}
		if href, ok := n.Attr("href"); ok {
			full := resolveRelative(rawURL, href)
			if strings.Contains(full, "facebook.com/photo") || strings.Contains(full, "facebook.com/watch") || strings.HasSuffix(strings.ToLower(full), ".jpg") || strings.HasSuffix(strings.ToLower(full), ".mp4") {
				if !seenMedia[full] {
					seenMedia[full] = true
					media = append(media, full)
				}
			} else if strings.HasPrefix(full, "http") && !strings.Contains(full, "facebook.com") && !seenExt[full] {
				seenExt[full] = true
				external = append(external, full)
			}
		}
	})
	return media, external
}

func nearestRichContainer(sel *goquery.Selection) *goquery.Selection {
	for _, node := range []*goquery.Selection{
		sel.ParentsFiltered("article").First(),
		sel.ParentsFiltered("div").First(),
		sel.Parent(),
	} {
		if node != nil && cleanText(node.Text()) != "" {
			return node
		}
	}
	return sel
}

func looksLikePostLink(href string) bool {
	href = strings.ToLower(href)
	return strings.Contains(href, "story.php") ||
		strings.Contains(href, "/posts/") ||
		strings.Contains(href, "/permalink/") ||
		strings.Contains(href, "/groups/") && strings.Contains(href, "/posts/") ||
		strings.Contains(href, "photo.php") ||
		strings.Contains(href, "/watch/?v=")
}

func looksLikeCommentLink(href string) bool {
	href = strings.ToLower(href)
	return strings.Contains(href, "comment_id=") || strings.Contains(href, "/comment/replies/")
}

func extractPostID(rawURL string) string {
	if id := ExtractFacebookID(rawURL); id != "" {
		return id
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if v := u.Query().Get("fbid"); v != "" {
		return v
	}
	parts := pathParts(u.Path)
	for i, part := range parts {
		if part == "posts" || part == "videos" || part == "permalink" {
			if i+1 < len(parts) {
				return sanitizeID(parts[i+1])
			}
		}
	}
	if len(parts) > 0 {
		last := sanitizeID(parts[len(parts)-1])
		if last != "" {
			return last
		}
	}
	return ""
}

func extractCommentID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil {
		if v := u.Query().Get("comment_id"); v != "" {
			return v
		}
	}
	return sanitizeID(rawURL)
}

func extractProfileOrPageID(rawURL string) string {
	if id := ExtractFacebookID(rawURL); id != "" {
		return id
	}
	parts := pathParts(mustURLPath(rawURL))
	if len(parts) > 0 {
		return sanitizeID(parts[0])
	}
	return ""
}

func findCountNear(body, label string) int64 {
	label = strings.ToLower(label)
	bodyLower := strings.ToLower(body)
	idx := strings.Index(bodyLower, label)
	if idx < 0 {
		return 0
	}
	start := idx - 32
	if start < 0 {
		start = 0
	}
	end := idx + len(label) + 16
	if end > len(body) {
		end = len(body)
	}
	return parseCompactCount(body[start:end])
}

func parseCompactCount(s string) int64 {
	m := countPattern.FindStringSubmatch(strings.ReplaceAll(s, ",", ""))
	if len(m) == 0 {
		return 0
	}
	n, _ := strconv.ParseFloat(strings.ReplaceAll(m[1], ".", "."), 64)
	switch strings.ToLower(m[2]) {
	case "k":
		n *= 1_000
	case "m":
		n *= 1_000_000
	case "b":
		n *= 1_000_000_000
	}
	return int64(n)
}

func cleanText(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(htmlEntityCleanup(s))), " ")
}

func htmlEntityCleanup(s string) string {
	repl := strings.NewReplacer("\u00a0", " ", "·", " ", "\n", " ", "\t", " ")
	return repl.Replace(s)
}

func truncateString(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n])
}

func mustURLPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Path
}

func pathParts(path string) []string {
	var out []string
	for _, part := range strings.Split(path, "/") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func stripTitleSuffix(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " | Facebook")
	s = strings.TrimSuffix(s, " - Facebook")
	return strings.TrimSpace(s)
}

func findLabeledValue(body, label string) string {
	idx := strings.Index(strings.ToLower(body), strings.ToLower(label))
	if idx < 0 {
		return ""
	}
	val := body[idx:]
	if len(val) > 120 {
		val = val[:120]
	}
	return strings.TrimSpace(val)
}

func firstMatching(body string, options []string) string {
	lower := strings.ToLower(body)
	for _, option := range options {
		if strings.Contains(lower, strings.ToLower(option)) {
			return option
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func findPhone(body string) string {
	re := regexp.MustCompile(`\+?\d[\d\s().-]{7,}\d`)
	return re.FindString(body)
}

func sanitizeID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	if i := strings.IndexAny(s, "?#&"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

func resolveRelative(base, ref string) string {
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	if strings.HasPrefix(ref, "//") {
		return "https:" + ref
	}
	b, err := url.Parse(NormalizeURL(base, true))
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}

func removeEcho(text, ownerName string) string {
	text = strings.TrimSpace(text)
	if ownerName == "" {
		return text
	}
	text = strings.TrimPrefix(text, ownerName)
	return strings.TrimSpace(text)
}

func queueStatsJSON(done, failed int64, duration time.Duration) string {
	return fmt.Sprintf(`{"done":%d,"failed":%d,"duration_seconds":%.3f}`, done, failed, duration.Seconds())
}
