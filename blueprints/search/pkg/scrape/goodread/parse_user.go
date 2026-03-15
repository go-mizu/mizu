package goodread

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var reUserID = regexp.MustCompile(`/user/show/(\d+)`)
var reDate = regexp.MustCompile(`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{4}\b`)

// ParseUser parses a Goodreads user profile page.
func ParseUser(doc *goquery.Document, userID, pageURL string) (*User, error) {
	u := &User{
		UserID:    userID,
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// Name
	u.Name = strings.TrimSpace(doc.Find("h1.userProfileName, [data-testid='name'], h1").First().Text())

	// Username from URL
	if pageURL != "" {
		parts := strings.Split(strings.TrimRight(pageURL, "/"), "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if !strings.ContainsAny(last, "?#") {
				u.Username = last
			}
		}
	}

	// Avatar
	u.AvatarURL, _ = doc.Find("img.userPhoto, img[itemprop='image'], img[data-testid='avatar']").First().Attr("src")

	// Bio
	u.Bio = strings.TrimSpace(doc.Find("[data-testid='aboutMe'], .aboutAuthorInfo, #aboutAuthor").First().Text())

	// Location
	u.Location = strings.TrimSpace(doc.Find("[data-testid='userLocation'], .userLocation").First().Text())

	// Website
	doc.Find("a[href*='http']").Each(func(_ int, sel *goquery.Selection) {
		if u.Website != "" {
			return
		}
		href, _ := sel.Attr("href")
		if !strings.Contains(href, "goodreads.com") && strings.HasPrefix(href, "http") {
			u.Website = href
		}
	})

	// Joined date
	doc.Find("[class*='joined'], .memberSince").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if m := reDate.FindString(text); m != "" {
			u.JoinedDate, _ = time.Parse("Jan 2006", m)
		}
	})

	// Friends / books / ratings counts
	doc.Find("[class*='statsCount'], .statsPanelCount, [data-testid='statsCount']").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		n, _ := strconv.Atoi(strings.ReplaceAll(text, ",", ""))
		parent := strings.ToLower(strings.TrimSpace(sel.Parent().Text()))
		switch {
		case strings.Contains(parent, "friend"):
			u.FriendsCount = n
		case strings.Contains(parent, "book"):
			u.BooksReadCount = n
		case strings.Contains(parent, "rating"):
			u.RatingsCount = n
		case strings.Contains(parent, "review"):
			u.ReviewsCount = n
		}
	})

	// Favorite books
	doc.Find(".favoriteBooks a[href*='/book/show/']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		if id := extractIDFromPath(href, "/book/show/"); id != "" && !contains(u.FavoriteBookIDs, id) {
			u.FavoriteBookIDs = append(u.FavoriteBookIDs, id)
		}
	})

	return u, nil
}
