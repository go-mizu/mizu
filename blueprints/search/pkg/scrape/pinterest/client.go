package pinterest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Client handles all Pinterest Resource API requests with session management,
// CSRF token handling, and rate limiting.
type Client struct {
	http       *http.Client
	jar        http.CookieJar
	userAgents []string
	delay      time.Duration
	mu         sync.Mutex
	lastReq    time.Time
	csrfToken  string
}

// NewClient creates a new Pinterest client and warms up the session by visiting
// the homepage to collect session cookies (including csrftoken).
func NewClient(cfg Config) (*Client, error) {
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     cfg.Workers + 2,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	c := &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Jar:       jar,
			Transport: transport,
		},
		jar:        jar,
		userAgents: userAgents,
		delay:      cfg.Delay,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := c.warmup(ctx); err != nil {
		return nil, fmt.Errorf("pinterest session warmup: %w", err)
	}
	return c, nil
}

// warmup fetches the Pinterest homepage to collect session cookies and CSRF token.
func (c *Client) warmup(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.pinterest.com/", nil)
	if err != nil {
		return err
	}
	c.setBaseHeaders(req, "/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Extract CSRF token from cookie jar
	u, _ := url.Parse("https://www.pinterest.com/")
	for _, cookie := range c.jar.Cookies(u) {
		if cookie.Name == "csrftoken" {
			c.csrfToken = cookie.Value
			break
		}
	}
	return nil
}

// ── Public API methods ────────────────────────────────────────────────────────

// SearchPins fetches all pins matching the query (up to maxPins).
func (c *Client) SearchPins(ctx context.Context, query string, maxPins int) ([]Pin, error) {
	var allPins []Pin
	var bookmark string

	for page := 1; ; page++ {
		if ctx.Err() != nil {
			return allPins, ctx.Err()
		}

		pins, next, err := c.searchPage(ctx, query, bookmark)
		if err != nil {
			if len(allPins) > 0 {
				return allPins, nil // partial result on error
			}
			return nil, fmt.Errorf("search page %d: %w", page, err)
		}

		allPins = append(allPins, pins...)

		if maxPins > 0 && len(allPins) >= maxPins {
			return allPins[:maxPins], nil
		}
		if isEndBookmark(next) || len(pins) == 0 {
			break
		}
		bookmark = next
	}
	return allPins, nil
}

// FetchBoardPage fetches the board HTML page and extracts the board ID.
func (c *Client) FetchBoardPage(ctx context.Context, boardURL string) (string, error) {
	body, _, err := c.fetchPageHTML(ctx, boardURL)
	if err != nil {
		return "", err
	}

	boardID := extractBoardID(body)
	if boardID == "" {
		return "", fmt.Errorf("could not extract board_id from %s", boardURL)
	}
	return boardID, nil
}

// FetchBoardBootstrap fetches board metadata and the SSR bootstrap feed embedded
// in the public board page.
func (c *Client) FetchBoardBootstrap(ctx context.Context, boardURL string) (*Board, []Pin, error) {
	body, _, err := c.fetchPageHTML(ctx, boardURL)
	if err != nil {
		return nil, nil, err
	}
	return parseBoardBootstrap(body, boardURL)
}

// FetchBoardPins fetches one page of pins from a board.
// Returns (pins, nextBookmark, error). nextBookmark="" means no more pages.
func (c *Client) FetchBoardPins(ctx context.Context, boardID, sourceURL, bookmark string) ([]Pin, string, error) {
	options := map[string]any{
		"board_id":      boardID,
		"page_size":     25,
		"field_set_key": "react_grid_pin",
	}
	if bookmark != "" {
		options["bookmarks"] = []string{bookmark}
	}

	data := map[string]any{
		"options": options,
		"context": map[string]any{},
	}
	dataJSON, _ := json.Marshal(data)

	params := url.Values{}
	params.Set("source_url", sourceURL)
	params.Set("data", string(dataJSON))
	params.Set("_", fmt.Sprintf("%d", time.Now().UnixMilli()))

	apiURL := "https://www.pinterest.com/resource/BoardFeedResource/get/?" + params.Encode()

	c.rateLimit()
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, "", err
	}
	c.setAPIHeaders(req, sourceURL)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, "", err
	}

	return parseBoardFeedResponse(body)
}

// FetchUser fetches a Pinterest user profile by username.
func (c *Client) FetchUser(ctx context.Context, username string) (*User, error) {
	user, _, err := c.FetchUserBootstrap(ctx, username)
	return user, err
}

// FetchUserBoards fetches one page of boards for a user.
// Returns (boards, nextBookmark, error).
func (c *Client) FetchUserBoards(ctx context.Context, username, bookmark string) ([]Board, string, error) {
	_, boards, err := c.FetchUserBootstrap(ctx, username)
	return boards, "", err
}

// FetchUserBootstrap fetches a user's public profile page and extracts the
// embedded SSR bootstrap data, including the visible board list.
func (c *Client) FetchUserBootstrap(ctx context.Context, username string) (*User, []Board, error) {
	userURL := NormalizeUserURL(username)
	body, _, err := c.fetchPageHTML(ctx, userURL)
	if err != nil {
		return nil, nil, err
	}
	return parseUserBootstrap(body, username)
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (c *Client) searchPage(ctx context.Context, query, bookmark string) ([]Pin, string, error) {
	sourceURL := fmt.Sprintf("/search/pins/?q=%s&rs=typed", url.QueryEscape(query))

	options := map[string]any{
		"query":     query,
		"scope":     "pins",
		"rs":        "typed",
		"page_size": 25,
	}
	if bookmark != "" {
		options["bookmarks"] = []string{bookmark}
	}

	data := map[string]any{
		"options": options,
		"context": map[string]any{},
	}
	dataJSON, _ := json.Marshal(data)

	params := url.Values{}
	params.Set("source_url", sourceURL)
	params.Set("data", string(dataJSON))
	params.Set("_", fmt.Sprintf("%d", time.Now().UnixMilli()))

	apiURL := "https://www.pinterest.com/resource/BaseSearchResource/get/?" + params.Encode()

	c.rateLimit()
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, "", err
	}
	c.setAPIHeaders(req, sourceURL)
	req.Header.Set("X-Pinterest-Pws-Handler", "www/search/[scope].js")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, "", err
	}

	return parseSearchResponse(body)
}

func (c *Client) fetchPageHTML(ctx context.Context, pageURL string) ([]byte, int, error) {
	c.rateLimit()
	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return nil, 0, err
	}
	c.setBaseHeaders(req, pageURL)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode == 404 {
		return nil, resp.StatusCode, fmt.Errorf("not found (HTTP 404)")
	}
	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, pageURL)
	}
	return body, resp.StatusCode, nil
}

func (c *Client) setBaseHeaders(req *http.Request, path string) {
	c.mu.Lock()
	ua := c.userAgents[rand.Intn(len(c.userAgents))]
	c.mu.Unlock()
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
}

func (c *Client) setAPIHeaders(req *http.Request, sourceURL string) {
	c.setBaseHeaders(req, sourceURL)
	req.Header.Set("Accept", "application/json, text/javascript, */*, q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-Pinterest-Appstate", "active")
	req.Header.Set("X-Pinterest-Source-Url", sourceURL)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Referer", "https://www.pinterest.com"+sourceURL)
	if c.csrfToken != "" {
		req.Header.Set("X-CSRFToken", c.csrfToken)
	}
}

func (c *Client) rateLimit() {
	if c.delay <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	since := time.Since(c.lastReq)
	if since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}

// ── JSON response parsers ─────────────────────────────────────────────────────

// pinterestImage matches the image variant shape used across all API responses.
type pinterestImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type rawBoard struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	Description   string `json:"description"`
	PinCount      int    `json:"pin_count"`
	FollowerCount int    `json:"follower_count"`
	Privacy       string `json:"privacy"`
	Category      string `json:"category"`
	ImageCoverHD  string `json:"image_cover_hd_url"`
	Owner         struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"owner"`
	Header *struct {
		LargeImageURL  string `json:"large_image_url"`
		XLargeImageURL string `json:"xlarge_image_url"`
		SmallImageURL  string `json:"small_image_url"`
	} `json:"header"`
	CoverPin *struct {
		ImageURL string `json:"image_url"`
	} `json:"cover_pin"`
}

type rawUserProfile struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	FullName       string `json:"full_name"`
	About          string `json:"about"`
	WebsiteURL     string `json:"website_url"`
	FollowerCount  int    `json:"follower_count"`
	FollowingCount int    `json:"following_count"`
	BoardCount     int    `json:"board_count"`
	PinCount       int    `json:"pin_count"`
	MonthlyViews   int64  `json:"monthly_views"`
	ImageMediumURL string `json:"image_medium_url"`
}

// rawPin is the raw pin shape shared by search and board feed responses.
type rawPin struct {
	ID           string                    `json:"id"`
	Type         string                    `json:"type"`
	Title        string                    `json:"title"`
	GridTitle    string                    `json:"grid_title"`
	Description  string                    `json:"description"`
	AutoAltText  string                    `json:"auto_alt_text"`
	Link         string                    `json:"link"`
	SaveCount    int                       `json:"save_count"`
	CommentCount int                       `json:"comment_count"`
	CreatedAt    string                    `json:"created_at"`
	Images       map[string]pinterestImage `json:"images"`
	Board        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"board"`
	Pinner struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"pinner"`
}

// parseSearchResponse parses a BaseSearchResource response.
type searchResponse struct {
	ResourceResponse struct {
		Data struct {
			Results []rawPin `json:"results"`
		} `json:"data"`
		Bookmark string `json:"bookmark"`
	} `json:"resource_response"`
}

func parseSearchResponse(body []byte) ([]Pin, string, error) {
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("parse search response: %w", err)
	}
	pins := convertPins(resp.ResourceResponse.Data.Results)
	return pins, resp.ResourceResponse.Bookmark, nil
}

// parseBoardFeedResponse parses a BoardFeedResource response.
type boardFeedResponse struct {
	ResourceResponse struct {
		Data     []rawPin `json:"data"`
		Bookmark string   `json:"bookmark"`
	} `json:"resource_response"`
}

func parseBoardFeedResponse(body []byte) ([]Pin, string, error) {
	var resp boardFeedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("parse board feed response: %w", err)
	}
	pins := convertPins(resp.ResourceResponse.Data)
	return pins, resp.ResourceResponse.Bookmark, nil
}

// parseUserResponse parses a UserResource response.
type userResponse struct {
	ResourceResponse struct {
		Data struct {
			ID             string `json:"id"`
			Username       string `json:"username"`
			FullName       string `json:"full_name"`
			About          string `json:"about"`
			WebsiteURL     string `json:"website_url"`
			FollowerCount  int    `json:"follower_count"`
			FollowingCount int    `json:"following_count"`
			BoardCount     int    `json:"board_count"`
			PinCount       int    `json:"pin_count"`
			MonthlyViews   int64  `json:"monthly_views"`
			ImageMediumURL string `json:"image_medium_url"`
		} `json:"data"`
	} `json:"resource_response"`
}

func parseUserResponse(body []byte, username string) (*User, error) {
	var resp userResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse user response: %w", err)
	}
	d := resp.ResourceResponse.Data
	if d.ID == "" {
		return nil, fmt.Errorf("user %q not found in API response", username)
	}
	return &User{
		UserID:         d.ID,
		Username:       d.Username,
		FullName:       d.FullName,
		Bio:            d.About,
		Website:        d.WebsiteURL,
		FollowerCount:  d.FollowerCount,
		FollowingCount: d.FollowingCount,
		BoardCount:     d.BoardCount,
		PinCount:       d.PinCount,
		MonthlyViews:   d.MonthlyViews,
		AvatarURL:      d.ImageMediumURL,
		URL:            BaseURL + "/" + d.Username + "/",
		FetchedAt:      time.Now(),
	}, nil
}

// parseUserBoardsResponse parses a BoardsResource response.
type userBoardsResponse struct {
	ResourceResponse struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			URL         string `json:"url"`
			Description string `json:"description"`
			PinCount    int    `json:"pin_count"`
			Followers   int    `json:"follower_count"`
			Privacy     string `json:"privacy"`
			Category    string `json:"category"`
			CoverPin    *struct {
				Images map[string]pinterestImage `json:"images"`
			} `json:"cover_pin"`
			Owner struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			} `json:"owner"`
		} `json:"data"`
		Bookmark string `json:"bookmark"`
	} `json:"resource_response"`
}

func parseUserBoardsResponse(body []byte, ownerUsername string) ([]Board, string, error) {
	var resp userBoardsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("parse user boards response: %w", err)
	}

	var boards []Board
	now := time.Now()
	for _, d := range resp.ResourceResponse.Data {
		if d.ID == "" {
			continue
		}
		coverURL := ""
		if d.CoverPin != nil {
			coverURL, _, _ = bestImage(d.CoverPin.Images)
		}
		boardURL := d.URL
		if !strings.HasPrefix(boardURL, "http") {
			boardURL = BaseURL + boardURL
		}
		// Derive slug from URL: /username/slug/
		slug := ""
		parts := strings.Split(strings.Trim(d.URL, "/"), "/")
		if len(parts) >= 2 {
			slug = parts[len(parts)-1]
		}
		username := d.Owner.Username
		if username == "" {
			username = ownerUsername
		}

		boards = append(boards, Board{
			BoardID:       d.ID,
			Name:          d.Name,
			Slug:          slug,
			Description:   d.Description,
			UserID:        d.Owner.ID,
			Username:      username,
			PinCount:      d.PinCount,
			FollowerCount: d.Followers,
			CoverURL:      coverURL,
			Category:      d.Category,
			IsSecret:      d.Privacy == "secret",
			URL:           boardURL,
			FetchedAt:     now,
		})
	}
	return boards, resp.ResourceResponse.Bookmark, nil
}

type boardBootstrap struct {
	InitialReduxState struct {
		Boards map[string]rawBoard `json:"boards"`
		Pins   map[string]rawPin   `json:"pins"`
	} `json:"initialReduxState"`
	BoardFeedResource map[string]struct {
		Data         []rawPin `json:"data"`
		NextBookmark string   `json:"nextBookmark"`
	} `json:"BoardFeedResource"`
}

func parseBoardBootstrap(body []byte, boardURL string) (*Board, []Pin, error) {
	payload, err := extractScriptJSON(body, "__PWS_INITIAL_PROPS__")
	if err != nil {
		return nil, nil, err
	}

	var bootstrap boardBootstrap
	if err := json.Unmarshal(payload, &bootstrap); err != nil {
		return nil, nil, fmt.Errorf("parse board bootstrap: %w", err)
	}

	var board *Board
	for _, rb := range bootstrap.InitialReduxState.Boards {
		b := convertBoard(rb)
		if board == nil || b.URL == boardURL {
			candidate := b
			board = &candidate
			if b.URL == boardURL {
				break
			}
		}
	}
	if board == nil {
		return nil, nil, fmt.Errorf("board metadata not found in bootstrap")
	}

	var pins []Pin
	for _, resource := range bootstrap.BoardFeedResource {
		pins = append(pins, convertPins(resource.Data)...)
	}
	if len(pins) == 0 && len(bootstrap.InitialReduxState.Pins) > 0 {
		rawPins := make([]rawPin, 0, len(bootstrap.InitialReduxState.Pins))
		for _, pin := range bootstrap.InitialReduxState.Pins {
			rawPins = append(rawPins, pin)
		}
		pins = convertPins(rawPins)
	}

	return board, dedupePins(pins), nil
}

type userBootstrap struct {
	InitialReduxState struct {
		Boards map[string]rawBoard       `json:"boards"`
		Users  map[string]rawUserProfile `json:"users"`
	} `json:"initialReduxState"`
	Resources struct {
		UserResource map[string]struct {
			Data rawUserProfile `json:"data"`
		} `json:"UserResource"`
	} `json:"resources"`
}

func parseUserBootstrap(body []byte, username string) (*User, []Board, error) {
	payload, err := extractScriptJSON(body, "__PWS_INITIAL_PROPS__")
	if err != nil {
		return nil, nil, err
	}

	var bootstrap userBootstrap
	if err := json.Unmarshal(payload, &bootstrap); err != nil {
		return nil, nil, fmt.Errorf("parse user bootstrap: %w", err)
	}

	var rawUser *rawUserProfile
	for _, u := range bootstrap.InitialReduxState.Users {
		if u.Username == username {
			u := u
			rawUser = &u
			break
		}
	}
	if rawUser == nil {
		for _, u := range bootstrap.Resources.UserResource {
			if u.Data.Username == username {
				user := u.Data
				rawUser = &user
				break
			}
		}
	}
	if rawUser == nil {
		return nil, nil, fmt.Errorf("user %q not found in bootstrap", username)
	}

	boards := make([]Board, 0, len(bootstrap.InitialReduxState.Boards))
	for _, rb := range bootstrap.InitialReduxState.Boards {
		board := convertBoard(rb)
		if board.Username == username {
			boards = append(boards, board)
		}
	}

	user := &User{
		UserID:         rawUser.ID,
		Username:       rawUser.Username,
		FullName:       rawUser.FullName,
		Bio:            rawUser.About,
		Website:        rawUser.WebsiteURL,
		FollowerCount:  rawUser.FollowerCount,
		FollowingCount: rawUser.FollowingCount,
		BoardCount:     rawUser.BoardCount,
		PinCount:       rawUser.PinCount,
		MonthlyViews:   rawUser.MonthlyViews,
		AvatarURL:      rawUser.ImageMediumURL,
		URL:            NormalizeUserURL(rawUser.Username),
		FetchedAt:      time.Now(),
	}

	return user, boards, nil
}

func convertBoard(rb rawBoard) Board {
	boardURL := rb.URL
	if boardURL != "" && !strings.HasPrefix(boardURL, "http") {
		boardURL = BaseURL + boardURL
	}
	username, slug := ExtractBoardSlug(boardURL)
	coverURL := rb.ImageCoverHD
	if coverURL == "" && rb.Header != nil {
		coverURL = rb.Header.XLargeImageURL
		if coverURL == "" {
			coverURL = rb.Header.LargeImageURL
		}
		if coverURL == "" {
			coverURL = rb.Header.SmallImageURL
		}
	}
	if coverURL == "" && rb.CoverPin != nil {
		coverURL = rb.CoverPin.ImageURL
	}
	return Board{
		BoardID:       rb.ID,
		Name:          rb.Name,
		Slug:          slug,
		Description:   rb.Description,
		UserID:        rb.Owner.ID,
		Username:      firstNonEmpty(rb.Owner.Username, username),
		PinCount:      rb.PinCount,
		FollowerCount: rb.FollowerCount,
		CoverURL:      coverURL,
		Category:      rb.Category,
		IsSecret:      rb.Privacy == "secret",
		URL:           boardURL,
		FetchedAt:     time.Now(),
	}
}

func extractScriptJSON(body []byte, scriptID string) ([]byte, error) {
	startMarker := `<script id="` + scriptID + `" type="application/json">`
	start := strings.Index(string(body), startMarker)
	if start < 0 {
		return nil, fmt.Errorf("%s not found in HTML", scriptID)
	}
	start += len(startMarker)
	end := strings.Index(string(body[start:]), "</script>")
	if end < 0 {
		return nil, fmt.Errorf("closing script tag not found for %s", scriptID)
	}
	return body[start : start+end], nil
}

func dedupePins(pins []Pin) []Pin {
	if len(pins) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(pins))
	out := make([]Pin, 0, len(pins))
	for _, pin := range pins {
		if pin.PinID == "" {
			continue
		}
		if _, ok := seen[pin.PinID]; ok {
			continue
		}
		seen[pin.PinID] = struct{}{}
		out = append(out, pin)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// convertPins converts rawPin slices to Pin domain objects.
func convertPins(raw []rawPin) []Pin {
	now := time.Now()
	var pins []Pin
	for _, p := range raw {
		if p.ID == "" || p.Type != "pin" {
			continue
		}
		imgURL, w, h := bestImage(p.Images)
		if imgURL == "" {
			continue
		}
		title := p.GridTitle
		if title == "" {
			title = p.Title
		}
		alt := p.AutoAltText
		if alt == "" {
			alt = title
		}
		var createdAt time.Time
		if p.CreatedAt != "" {
			createdAt, _ = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", p.CreatedAt)
		}
		pins = append(pins, Pin{
			PinID:        p.ID,
			Title:        title,
			Description:  p.Description,
			AltText:      alt,
			ImageURL:     imgURL,
			ImageWidth:   w,
			ImageHeight:  h,
			PinURL:       fmt.Sprintf("%s/pin/%s/", BaseURL, p.ID),
			SourceURL:    p.Link,
			BoardID:      p.Board.ID,
			BoardName:    p.Board.Name,
			UserID:       p.Pinner.ID,
			Username:     p.Pinner.Username,
			SavedCount:   p.SaveCount,
			CommentCount: p.CommentCount,
			CreatedAt:    createdAt,
			FetchedAt:    now,
		})
	}
	return pins
}

// bestImage picks the highest-resolution image from a Pinterest images map.
func bestImage(images map[string]pinterestImage) (string, int, int) {
	priority := []string{"orig", "736x", "474x", "236x"}
	for _, key := range priority {
		if img, ok := images[key]; ok && img.URL != "" {
			return img.URL, img.Width, img.Height
		}
	}
	// Fallback: largest by pixel count
	var bestURL string
	var bestW, bestH int
	for _, img := range images {
		if img.Width*img.Height > bestW*bestH {
			bestURL = img.URL
			bestW = img.Width
			bestH = img.Height
		}
	}
	return bestURL, bestW, bestH
}

// isEndBookmark returns true when the bookmark signals no more pages.
func isEndBookmark(b string) bool {
	return b == "" || b == "-end-" || strings.HasPrefix(b, "Y2JOb25l")
}

// boardIDPatterns matches board_id in embedded Pinterest HTML JSON state.
var boardIDPatterns = []*regexp.Regexp{
	regexp.MustCompile(`"board_id"\s*:\s*"(\d+)"`),
	regexp.MustCompile(`"board"\s*:\s*\{[^}]*"id"\s*:\s*"(\d+)"`),
	regexp.MustCompile(`"id"\s*:\s*"(\d+)"[^}]*"type"\s*:\s*"board"`),
}

// extractBoardID searches HTML body for the numeric Pinterest board ID.
func extractBoardID(body []byte) string {
	s := string(body)
	for _, re := range boardIDPatterns {
		if m := re.FindStringSubmatch(s); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}
