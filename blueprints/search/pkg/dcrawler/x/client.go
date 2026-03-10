package x

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client wraps the X/Twitter GraphQL API with cookie-based auth.
type Client struct {
	gql        *graphqlClient
	cfg        Config
	authToken  string
	ct0        string
	searchMode string
	userCache  map[string]string // username → rest_id
}

// NewClient creates a new X/Twitter client.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		searchMode: SearchTop,
		userCache:  make(map[string]string),
	}
}

// Login returns an error — password login is not supported.
// Use import-session with cookie-based auth instead.
func (c *Client) Login(username, password string) error {
	return fmt.Errorf("password login is not supported; use import-session with X_AUTH_TOKEN and X_CSRF_TOKEN")
}

// Activate validates the session by attempting a lightweight API call.
// Returns true if the session appears valid.
func (c *Client) Activate() bool {
	if c.gql == nil {
		return false
	}
	// Try a lightweight request to check auth
	_, err := c.gql.doGraphQL(gqlUserByScreenName, map[string]any{
		"screen_name":                 "X",
		"withSafetyModeUserFields":    true,
		"withSuperFollowsUserFields":  true,
	}, userFieldToggles)
	return err == nil
}

// SetAuthToken sets auth_token and ct0 for cookie-based authentication.
func (c *Client) SetAuthToken(authToken, csrfToken string) {
	c.authToken = authToken
	c.ct0 = csrfToken
	c.gql = newGraphQLClient(authToken, csrfToken, c.cfg.Timeout)
}

// SetCookies extracts auth_token and ct0 from cookies and initializes the client.
func (c *Client) SetCookies(cookies []*http.Cookie) {
	var authToken, ct0 string
	for _, cookie := range cookies {
		switch cookie.Name {
		case "auth_token":
			authToken = cookie.Value
		case "ct0":
			ct0 = cookie.Value
		}
	}
	if authToken != "" && ct0 != "" {
		c.SetAuthToken(authToken, ct0)
	}
}

// AuthToken returns the current auth_token value.
func (c *Client) AuthToken() string { return c.authToken }

// CT0 returns the current ct0 (CSRF token) value.
func (c *Client) CT0() string { return c.ct0 }

// GetCookies returns the current auth as cookies (for backward compatibility).
func (c *Client) GetCookies() []*http.Cookie {
	if c.authToken == "" {
		return nil
	}
	return []*http.Cookie{
		{Name: "auth_token", Value: c.authToken, Domain: ".x.com", Path: "/"},
		{Name: "ct0", Value: c.ct0, Domain: ".x.com", Path: "/"},
	}
}

// SaveSessionFile saves the current session to disk.
func (c *Client) SaveSessionFile(path, username string) error {
	return SaveSession(path, username, c.authToken, c.ct0, c.GetCookies())
}

// LoadSessionFile loads a session from disk.
func (c *Client) LoadSessionFile(path string) (*Session, error) {
	sess, err := LoadSession(path)
	if err != nil {
		return nil, err
	}
	// Prefer explicit AuthToken/CT0 fields, fall back to extracting from cookies
	authToken := sess.AuthToken
	ct0 := sess.CT0
	if authToken == "" || ct0 == "" {
		for _, cookie := range sess.Cookies {
			switch cookie.Name {
			case "auth_token":
				authToken = cookie.Value
			case "ct0":
				ct0 = cookie.Value
			}
		}
	}
	if authToken == "" || ct0 == "" {
		return nil, fmt.Errorf("session missing auth_token or ct0")
	}
	c.SetAuthToken(authToken, ct0)
	return sess, nil
}

// SearchMode returns the current search mode.
func (c *Client) SearchMode() string { return c.searchMode }

// SetSearchMode sets the search mode (Top, Latest, Photos, Videos, People).
func (c *Client) SetSearchMode(mode string) {
	c.searchMode = mode
}

// resolveUserID resolves a username to a user rest_id, using cache.
func (c *Client) resolveUserID(username string) (string, error) {
	if id, ok := c.userCache[username]; ok {
		return id, nil
	}
	p, err := c.GetProfile(username)
	if err != nil {
		return "", err
	}
	c.userCache[username] = p.ID
	return p.ID, nil
}

// DoGraphQL exposes the GraphQL client for testing.
func (c *Client) DoGraphQL(endpoint string, variables map[string]any, fieldToggles string) (map[string]any, error) {
	return c.gql.doGraphQL(endpoint, variables, fieldToggles)
}

// ── Profile ──────────────────────────────────────────────

// GetProfile fetches a user profile by username.
// Tries guest token first (no session rate limit consumed), falls back to cookie auth.
func (c *Client) GetProfile(username string) (*Profile, error) {
	data, err := c.doGuestFirst(gqlUserByScreenName, map[string]any{
		"screen_name":              username,
		"withSafetyModeUserFields": true,
	}, userFieldToggles)
	if err != nil {
		return nil, fmt.Errorf("get profile @%s: %w", username, err)
	}

	p := parseUserResult(data)
	if p == nil {
		return nil, fmt.Errorf("get profile @%s: user not found", username)
	}

	c.userCache[p.Username] = p.ID
	return p, nil
}

// SearchProfiles searches for user profiles matching a query.
func (c *Client) SearchProfiles(ctx context.Context, query string, maxUsers int, cb ProgressCallback) ([]FollowUser, error) {
	var users []FollowUser
	cursor := ""

	for {
		if ctx.Err() != nil {
			break
		}

		vars := map[string]any{
			"rawQuery":    query,
			"count":       40,
			"querySource": "typed_query",
			"product":     "People",
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		data, err := c.gql.doGraphQL(gqlSearchTimeline, vars, "")
		if err != nil {
			if len(users) > 0 {
				return users, err
			}
			return nil, fmt.Errorf("search users: %w", err)
		}

		pageUsers, nextCursor := parseSearchUsers(data)
		users = append(users, pageUsers...)

		if cb != nil {
			cb(Progress{Phase: "search_users", Current: int64(len(users))})
		}

		if maxUsers > 0 && len(users) >= maxUsers {
			users = users[:maxUsers]
			break
		}
		if nextCursor == "" || len(pageUsers) == 0 {
			break
		}
		cursor = nextCursor

		time.Sleep(c.gql.PacedDelay(c.cfg.Delay))
	}

	return users, nil
}

// ── Tweets ───────────────────────────────────────────────

// GetTweets fetches user timeline tweets.
func (c *Client) GetTweets(ctx context.Context, username string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	return c.getUserTimeline(ctx, username, gqlUserTweetsV2, "tweets", maxTweets, cb, nil)
}

// GetTweetsWithBatch fetches user timeline tweets, calling batchCb with each page for incremental saving.
func (c *Client) GetTweetsWithBatch(ctx context.Context, username string, maxTweets int, cb ProgressCallback, batchCb BatchCallback) ([]Tweet, error) {
	return c.getUserTimeline(ctx, username, gqlUserTweetsV2, "tweets", maxTweets, cb, batchCb)
}

// GetTweetsAndReplies fetches user timeline tweets including replies.
func (c *Client) GetTweetsAndReplies(ctx context.Context, username string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	return c.getUserTimeline(ctx, username, gqlUserTweetsAndRepliesV2, "tweets+replies", maxTweets, cb, nil)
}

// GetMediaTweets fetches user media timeline (photos/videos only).
// Uses legacy UserMedia endpoint with userId (cookie auth compatibility).
func (c *Client) GetMediaTweets(ctx context.Context, username string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	return c.getUserTimeline(ctx, username, gqlUserMedia, "media", maxTweets, cb, nil)
}

func (c *Client) getUserTimeline(ctx context.Context, username, endpoint, phase string, maxTweets int, cb ProgressCallback, batchCb BatchCallback) ([]Tweet, error) {
	userID, err := c.resolveUserID(username)
	if err != nil {
		return nil, err
	}

	var tweets []Tweet
	cursor := ""
	emptyPages := 0
	maxEmpty := 3
	if maxTweets == 0 {
		maxEmpty = 10 // more tolerance for unlimited fetches
	}

	// All endpoints now use "userId". UserMedia has different field toggles.
	isMedia := endpoint == gqlUserMedia
	toggles := userTweetsFieldToggles
	if isMedia {
		toggles = ""
	}

	for {
		if ctx.Err() != nil {
			break
		}

		vars := map[string]any{
			"userId":                            userID,
			"count":                             40,
			"includePromotedContent":            false,
			"withQuickPromoteEligibilityTweetFields": true,
			"withVoice":                         true,
		}
		if isMedia {
			vars["withClientEventToken"] = false
			vars["withBirdwatchNotes"] = false
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		data, err := c.doGraphQLRetry(ctx, endpoint, vars, toggles, cb, phase, int64(len(tweets)))
		if err != nil {
			if len(tweets) > 0 {
				return tweets, err
			}
			return nil, fmt.Errorf("get %s @%s: %w", phase, username, err)
		}

		result := parseTimeline(data)
		newTweets := result.Tweets
		tweets = append(tweets, newTweets...)

		// Incremental save via batch callback
		if batchCb != nil && len(newTweets) > 0 {
			batchCb(newTweets)
		}

		if cb != nil {
			cb(Progress{Phase: phase, Current: int64(len(tweets))})
		}

		if maxTweets > 0 && len(tweets) >= maxTweets {
			tweets = tweets[:maxTweets]
			break
		}

		// Stop if no cursor or too many empty pages
		if result.Cursor == "" {
			if cb != nil {
				cb(Progress{Phase: phase, Current: int64(len(tweets)), Message: "no more pages (cursor empty)"})
			}
			break
		}
		if len(newTweets) == 0 {
			emptyPages++
			if cb != nil {
				cb(Progress{Phase: phase, Current: int64(len(tweets)), Message: fmt.Sprintf("empty page %d/%d", emptyPages, maxEmpty)})
			}
			if emptyPages >= maxEmpty {
				break
			}
		} else {
			emptyPages = 0
		}
		cursor = result.Cursor

		time.Sleep(c.gql.PacedDelay(c.cfg.Delay))
	}

	if cb != nil {
		cb(Progress{Phase: phase, Current: int64(len(tweets)), Done: true})
	}
	return tweets, nil
}

// asRateLimitError extracts a RateLimitError from an error.
func asRateLimitError(err error) *RateLimitError {
	if err == nil {
		return nil
	}
	if rle, ok := err.(*RateLimitError); ok {
		return rle
	}
	// Fallback: check message for older error formats
	msg := err.Error()
	if strings.Contains(msg, "rate limit") || strings.Contains(msg, "429") {
		return &RateLimitError{Wait: 15 * time.Minute}
	}
	return nil
}

// doGuestFirst attempts the GraphQL call via guest token first (no session rate limit consumed).
// Falls back to cookie auth if:
//   - guest token fetch fails
//   - guest returns rate limit (token rotated once, then auth fallback)
//   - guest returns 401/403 (auth-only endpoint)
func (c *Client) doGuestFirst(endpoint string, vars map[string]any, toggles string) (map[string]any, error) {
	token, tokenErr := fetchGuestToken()
	if tokenErr == nil {
		data, guestErr := doGuestGraphQL(token, endpoint, vars, toggles)
		if guestErr == nil {
			return data, nil
		}
		// On rate limit: rotate token and retry once; then try proxy pool before auth fallback
		if rle := asRateLimitError(guestErr); rle != nil {
			_ = rle
			invalidateGuestToken()
			if token2, err2 := fetchGuestToken(); err2 == nil {
				if data2, guestErr2 := doGuestGraphQL(token2, endpoint, vars, toggles); guestErr2 == nil {
					return data2, nil
				}
			}
			// Proxy pool: try a guest token from a different IP's rate-limit bucket
			if poolToken, poolErr := FetchGuestTokenFromPool(); poolErr == nil {
				if data3, err3 := doGuestGraphQL(poolToken, endpoint, vars, toggles); err3 == nil {
					return data3, nil
				}
			}
		}
		// 401/403 (auth-only endpoint) or other error — fall through to cookie auth
	}
	// Cookie auth fallback
	if c.gql == nil {
		return nil, fmt.Errorf("guest token failed (%v) and no session configured", tokenErr)
	}
	return c.gql.doGraphQL(endpoint, vars, toggles)
}

// doGraphQLRetry calls doGuestFirst then retries on rate limit.
// On guest rate limit: tries cookie auth immediately before waiting.
// On cookie rate limit: waits for reset (up to 3 retries).
func (c *Client) doGraphQLRetry(ctx context.Context, endpoint string, vars map[string]any, toggles string, cb ProgressCallback, phase string, current int64) (map[string]any, error) {
	data, err := c.doGuestFirst(endpoint, vars, toggles)
	if err == nil {
		return data, nil
	}
	rle := asRateLimitError(err)
	if rle == nil {
		return nil, err
	}
	// Rate limited — wait for reset, retry up to 3 times
	for retry := 0; retry < 3; retry++ {
		wait := rle.Wait
		if wait < 10*time.Second {
			wait = 10 * time.Second
		}
		if wait > 16*time.Minute {
			wait = 16 * time.Minute // cap at 16 min
		}
		if cb != nil {
			msg := fmt.Sprintf("rate limited, waiting %s", wait.Truncate(time.Second))
			if !rle.ResetAt.IsZero() {
				msg += fmt.Sprintf(" (resets %s)", rle.ResetAt.Format("15:04:05"))
			}
			cb(Progress{Phase: phase, Current: current, Message: msg})
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		data, err = c.doGuestFirst(endpoint, vars, toggles)
		if err == nil {
			return data, nil
		}
		rle = asRateLimitError(err)
		if rle == nil {
			return nil, err // different error
		}
	}
	return nil, err // exhausted retries
}

// GetTweet fetches a single tweet by ID.
// Strategy: syndication API (no auth) → guest token GraphQL → cookie auth.
func (c *Client) GetTweet(id string) (*Tweet, error) {
	// 1. Syndication API — fastest, no auth, generous limits
	if t, err := GetTweetSyndication(id); err == nil {
		return t, nil
	}

	// 2. Guest token + TweetDetail GraphQL — full data including views
	vars := map[string]any{
		"focalTweetId":                           id,
		"referrer":                               "tweet",
		"with_rux_injections":                    false,
		"rankingMode":                            "Relevance",
		"includePromotedContent":                 true,
		"withCommunity":                          true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withBirdwatchNotes":                     true,
		"withVoice":                              true,
		"withV2Timeline":                         true,
	}

	data, err := c.doGuestFirst(gqlConversationTimeline, vars, tweetDetailFieldToggles)
	if err != nil {
		return nil, fmt.Errorf("get tweet %s: %w", id, err)
	}

	mainTweet, _, _ := parseConversation(data, id)
	if mainTweet == nil {
		return nil, fmt.Errorf("get tweet %s: not found", id)
	}
	return mainTweet, nil
}

// FetchURL performs an authenticated HTTP GET with the current session cookies.
// Useful for fetching X Article pages or other X content that requires auth.
func (c *Client) FetchURL(rawURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	// Standard browser headers + session cookies
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	if c.authToken != "" {
		req.AddCookie(&http.Cookie{Name: "auth_token", Value: c.authToken})
		req.AddCookie(&http.Cookie{Name: "ct0", Value: c.ct0})
	}

	httpClient := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Forward cookies on redirect
			if c.authToken != "" {
				req.AddCookie(&http.Cookie{Name: "auth_token", Value: c.authToken})
				req.AddCookie(&http.Cookie{Name: "ct0", Value: c.ct0})
			}
			if len(via) > 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
}

// GetTweetByRestID fetches a tweet (or X Article) by ID using TweetResultByRestId endpoint.
// This works for X Articles where the ConversationTimeline endpoint returns "not found".
func (c *Client) GetTweetByRestID(id string) (*Tweet, error) {
	vars := map[string]any{
		"tweetId":                id,
		"withCommunity":          true,
		"includePromotedContent": false,
		"withBirdwatchNotes":     true,
		"withVoice":              true,
	}

	data, err := c.gql.doGraphQL(gqlUserById, vars, tweetDetailFieldToggles)
	if err != nil {
		return nil, fmt.Errorf("get tweet by rest id %s: %w", id, err)
	}

	t, debugMsg := parseTweetResultByIDDebug(data)
	if t == nil {
		// Fallback: try ConversationTimeline (some articles are accessible this way)
		if t2, err2 := c.GetTweet(id); err2 == nil && t2 != nil {
			return t2, nil
		}
		return nil, fmt.Errorf("get tweet by rest id %s: not found (%s)", id, debugMsg)
	}
	return t, nil
}

// ── No-auth fetch (syndication + guest token) ────────────

// GetTweetNoAuth fetches a single tweet without cookie authentication.
// Strategy (in order):
//  1. Syndication API (cdn.syndication.twimg.com) — no auth, embed endpoint
//  2. Guest token GraphQL — anonymous session, same GraphQL API
//
// Use this when no session is configured or as a rate-limit bypass.
// Note: returns less data than cookie-auth (no replies, no home timeline).
func (c *Client) GetTweetNoAuth(id string) (*Tweet, error) {
	return GetTweetNoAuth(id)
}

// GetTweetNoAuth fetches a single tweet without any authentication.
// Tries the syndication/embed API first, then falls back to guest token GraphQL.
func GetTweetNoAuth(id string) (*Tweet, error) {
	// Try syndication first (fastest, no token rotation needed)
	t, err := GetTweetSyndication(id)
	if err == nil {
		return t, nil
	}
	syndErr := err

	// Fall back to guest token GraphQL
	t, err = GetTweetGuest(id)
	if err == nil {
		return t, nil
	}

	return nil, fmt.Errorf("no-auth tweet fetch failed — syndication: %v; guest: %v", syndErr, err)
}

// GetProfileNoAuth fetches a public user profile without cookie authentication.
// Uses the guest token GraphQL endpoint — returns the same fields as cookie auth
// for public profiles (followers, bio, stats, etc.).
func (c *Client) GetProfileNoAuth(username string) (*Profile, error) {
	return GetProfileGuest(username)
}

// HasAuth reports whether this client has cookie credentials configured.
func (c *Client) HasAuth() bool {
	return c.authToken != "" && c.ct0 != ""
}

// GetTweetReplies fetches replies to a tweet.
func (c *Client) GetTweetReplies(id string) ([]Tweet, error) {
	var allReplies []Tweet
	cursor := ""

	for {
		vars := map[string]any{
			"focalTweetId":                        id,
			"referrer":                            "tweet",
			"with_rux_injections":                 false,
			"rankingMode":                         "Relevance",
			"includePromotedContent":              true,
			"withCommunity":                       true,
			"withQuickPromoteEligibilityTweetFields": true,
			"withBirdwatchNotes":                  true,
			"withVoice":                           true,
			"withV2Timeline":                      true,
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		data, err := c.doGuestFirst(gqlConversationTimeline, vars, tweetDetailFieldToggles)
		if err != nil {
			if len(allReplies) > 0 {
				return allReplies, err
			}
			return nil, fmt.Errorf("get replies for %s: %w", id, err)
		}

		_, replies, nextCursor := parseConversation(data, id)
		allReplies = append(allReplies, replies...)

		if nextCursor == "" || len(replies) == 0 {
			break
		}
		cursor = nextCursor

		time.Sleep(c.cfg.Delay)
	}

	return allReplies, nil
}

// GetRetweeters fetches users who retweeted a tweet.
func (c *Client) GetRetweeters(tweetID string, maxUsers int) ([]FollowUser, error) {
	vars := map[string]any{
		"tweetId": tweetID,
		"count":   maxUsers,
	}

	data, err := c.gql.doGraphQL(gqlRetweeters, vars, "")
	if err != nil {
		return nil, fmt.Errorf("get retweeters %s: %w", tweetID, err)
	}

	users, _ := parseFollowList(data)
	return users, nil
}

// GetFavoriters fetches users who liked (favorited) a tweet.
func (c *Client) GetFavoriters(tweetID string, maxUsers int) ([]FollowUser, error) {
	vars := map[string]any{
		"tweetId": tweetID,
		"count":   maxUsers,
	}

	data, err := c.gql.doGraphQL(gqlFavoriters, vars, "")
	if err != nil {
		return nil, fmt.Errorf("get favoriters %s: %w", tweetID, err)
	}

	users, _ := parseFollowList(data)
	return users, nil
}

// ── Search ───────────────────────────────────────────────

// SearchTweets searches for tweets matching a query.
func (c *Client) SearchTweets(ctx context.Context, query string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	return c.searchTweetsInternal(ctx, query, maxTweets, cb, nil)
}

// SearchTweetsWithBatch searches for tweets, calling batchCb with each page for incremental saving.
func (c *Client) SearchTweetsWithBatch(ctx context.Context, query string, maxTweets int, cb ProgressCallback, batchCb BatchCallback) ([]Tweet, error) {
	return c.searchTweetsInternal(ctx, query, maxTweets, cb, batchCb)
}

func (c *Client) searchTweetsInternal(ctx context.Context, query string, maxTweets int, cb ProgressCallback, batchCb BatchCallback) ([]Tweet, error) {
	var tweets []Tweet
	cursor := ""
	emptyPages := 0
	maxEmpty := 3
	if maxTweets == 0 {
		maxEmpty = 5
	}

	for {
		if ctx.Err() != nil {
			break
		}

		vars := map[string]any{
			"rawQuery":    query,
			"count":       40,
			"querySource": "typed_query",
			"product":     c.searchMode,
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		data, err := c.doGraphQLRetry(ctx, gqlSearchTimeline, vars, "", cb, "search", int64(len(tweets)))
		if err != nil {
			if len(tweets) > 0 {
				return tweets, err
			}
			return nil, fmt.Errorf("search tweets: %w", err)
		}

		result := parseSearchTweets(data)
		newTweets := result.Tweets
		tweets = append(tweets, newTweets...)

		if batchCb != nil && len(newTweets) > 0 {
			batchCb(newTweets)
		}

		if cb != nil {
			cb(Progress{Phase: "search", Current: int64(len(tweets))})
		}

		if maxTweets > 0 && len(tweets) >= maxTweets {
			tweets = tweets[:maxTweets]
			break
		}

		if result.Cursor == "" {
			break
		}
		if len(newTweets) == 0 {
			emptyPages++
			if emptyPages >= maxEmpty {
				break
			}
		} else {
			emptyPages = 0
		}
		cursor = result.Cursor

		time.Sleep(c.gql.PacedDelay(c.cfg.Delay))
	}

	if cb != nil {
		cb(Progress{Phase: "search", Current: int64(len(tweets)), Done: true})
	}
	return tweets, nil
}

// ── Followers/Following ──────────────────────────────────

// GetFollowers fetches a user's followers list.
func (c *Client) GetFollowers(ctx context.Context, username string, maxUsers int, cb ProgressCallback) ([]FollowUser, error) {
	return c.getFollowList(ctx, username, gqlFollowers, "followers", maxUsers, cb)
}

// GetFollowing fetches a user's following list.
func (c *Client) GetFollowing(ctx context.Context, username string, maxUsers int, cb ProgressCallback) ([]FollowUser, error) {
	return c.getFollowList(ctx, username, gqlFollowing, "following", maxUsers, cb)
}

func (c *Client) getFollowList(ctx context.Context, username, endpoint, phase string, maxUsers int, cb ProgressCallback) ([]FollowUser, error) {
	userID, err := c.resolveUserID(username)
	if err != nil {
		return nil, err
	}

	var users []FollowUser
	cursor := ""

	for {
		if ctx.Err() != nil {
			break
		}

		vars := map[string]any{
			"userId":               userID,
			"count":                200,
			"includePromotedContent": false,
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		data, err := c.gql.doGraphQL(endpoint, vars, "")
		if err != nil {
			if len(users) > 0 {
				return users, err
			}
			return nil, fmt.Errorf("get %s @%s: %w", phase, username, err)
		}

		pageUsers, nextCursor := parseFollowList(data)
		users = append(users, pageUsers...)

		if cb != nil {
			cb(Progress{Phase: phase, Current: int64(len(users))})
		}

		if maxUsers > 0 && len(users) >= maxUsers {
			users = users[:maxUsers]
			break
		}
		if nextCursor == "" || len(pageUsers) == 0 {
			break
		}
		cursor = nextCursor

		time.Sleep(c.gql.PacedDelay(c.cfg.Delay))
	}

	return users, nil
}

// ── Bookmarks ────────────────────────────────────────────

// GetBookmarks fetches the authenticated user's bookmarked tweets.
func (c *Client) GetBookmarks(ctx context.Context, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	var tweets []Tweet
	cursor := ""
	emptyPages := 0

	for {
		if ctx.Err() != nil {
			break
		}

		vars := map[string]any{
			"rawQuery":               "",
			"count":                  40,
			"includePromotedContent": false,
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		data, err := c.gql.doGraphQL(gqlBookmarks, vars, "")
		if err != nil {
			if len(tweets) > 0 {
				return tweets, err
			}
			return nil, fmt.Errorf("get bookmarks: %w", err)
		}

		result := parseTimeline(data)
		tweets = append(tweets, result.Tweets...)

		if cb != nil {
			cb(Progress{Phase: "bookmarks", Current: int64(len(tweets))})
		}

		if maxTweets > 0 && len(tweets) >= maxTweets {
			tweets = tweets[:maxTweets]
			break
		}

		if result.Cursor == "" {
			break
		}
		if len(result.Tweets) == 0 {
			emptyPages++
			if emptyPages >= 3 {
				break
			}
		} else {
			emptyPages = 0
		}
		cursor = result.Cursor

		time.Sleep(c.gql.PacedDelay(c.cfg.Delay))
	}

	if cb != nil {
		cb(Progress{Phase: "bookmarks", Current: int64(len(tweets)), Done: true})
	}
	return tweets, nil
}

// ── Home/ForYou Timelines ────────────────────────────────

// GetHomeTweets fetches the authenticated user's home timeline.
func (c *Client) GetHomeTweets(ctx context.Context, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	return c.getTimeline(ctx, gqlHomeLatestTimeline, "home", maxTweets, cb)
}

// GetForYouTweets fetches the authenticated user's "For You" timeline.
func (c *Client) GetForYouTweets(ctx context.Context, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	return c.getTimeline(ctx, gqlHomeTimeline, "foryou", maxTweets, cb)
}

func (c *Client) getTimeline(ctx context.Context, endpoint, phase string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	var tweets []Tweet
	cursor := ""
	emptyPages := 0

	for {
		if ctx.Err() != nil {
			break
		}

		vars := map[string]any{
			"count":                  40,
			"includePromotedContent": false,
			"latestControlAvailable": true,
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		data, err := c.gql.doGraphQL(endpoint, vars, "")
		if err != nil {
			if len(tweets) > 0 {
				return tweets, err
			}
			return nil, fmt.Errorf("get %s: %w", phase, err)
		}

		result := parseTimeline(data)
		tweets = append(tweets, result.Tweets...)

		if cb != nil {
			cb(Progress{Phase: phase, Current: int64(len(tweets))})
		}

		if maxTweets > 0 && len(tweets) >= maxTweets {
			tweets = tweets[:maxTweets]
			break
		}

		if result.Cursor == "" {
			break
		}
		if len(result.Tweets) == 0 {
			emptyPages++
			if emptyPages >= 3 {
				break
			}
		} else {
			emptyPages = 0
		}
		cursor = result.Cursor

		time.Sleep(c.gql.PacedDelay(c.cfg.Delay))
	}

	if cb != nil {
		cb(Progress{Phase: phase, Current: int64(len(tweets)), Done: true})
	}
	return tweets, nil
}

// ── Lists ────────────────────────────────────────────────

// GetListByID fetches list metadata by list ID.
func (c *Client) GetListByID(id string) (*List, error) {
	data, err := c.gql.doGraphQL(gqlListById, map[string]any{
		"listId": id,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("get list %s: %w", id, err)
	}
	l := parseGraphList(data)
	if l == nil {
		return nil, fmt.Errorf("get list %s: not found", id)
	}
	return l, nil
}

// GetListBySlug fetches list metadata by owner username and slug.
func (c *Client) GetListBySlug(ownerUsername, slug string) (*List, error) {
	data, err := c.gql.doGraphQL(gqlListBySlug, map[string]any{
		"screenName": ownerUsername,
		"listSlug":   slug,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("get list %s/%s: %w", ownerUsername, slug, err)
	}
	l := parseGraphList(data)
	if l == nil {
		return nil, fmt.Errorf("get list %s/%s: not found", ownerUsername, slug)
	}
	return l, nil
}

// GetListTweets fetches tweets from a list timeline.
func (c *Client) GetListTweets(ctx context.Context, listID string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	var tweets []Tweet
	cursor := ""
	emptyPages := 0

	for {
		if ctx.Err() != nil {
			break
		}

		vars := map[string]any{
			"rest_id": listID,
			"count":   40,
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		data, err := c.gql.doGraphQL(gqlListTweets, vars, "")
		if err != nil {
			if len(tweets) > 0 {
				return tweets, err
			}
			return nil, fmt.Errorf("get list tweets %s: %w", listID, err)
		}

		result := parseListTimeline(data)
		tweets = append(tweets, result.Tweets...)

		if cb != nil {
			cb(Progress{Phase: "list_tweets", Current: int64(len(tweets))})
		}

		if maxTweets > 0 && len(tweets) >= maxTweets {
			tweets = tweets[:maxTweets]
			break
		}

		if result.Cursor == "" {
			break
		}
		if len(result.Tweets) == 0 {
			emptyPages++
			if emptyPages >= 3 {
				break
			}
		} else {
			emptyPages = 0
		}
		cursor = result.Cursor

		time.Sleep(c.gql.PacedDelay(c.cfg.Delay))
	}

	if cb != nil {
		cb(Progress{Phase: "list_tweets", Current: int64(len(tweets)), Done: true})
	}
	return tweets, nil
}

// GetListMembers fetches members of a list.
func (c *Client) GetListMembers(ctx context.Context, listID string, maxUsers int, cb ProgressCallback) ([]FollowUser, error) {
	var users []FollowUser
	cursor := ""

	for {
		if ctx.Err() != nil {
			break
		}

		vars := map[string]any{
			"listId": listID,
			"count":  200,
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		data, err := c.gql.doGraphQL(gqlListMembers, vars, "")
		if err != nil {
			if len(users) > 0 {
				return users, err
			}
			return nil, fmt.Errorf("get list members %s: %w", listID, err)
		}

		pageUsers, nextCursor := parseListMembers(data)
		users = append(users, pageUsers...)

		if cb != nil {
			cb(Progress{Phase: "list_members", Current: int64(len(users))})
		}

		if maxUsers > 0 && len(users) >= maxUsers {
			users = users[:maxUsers]
			break
		}
		if nextCursor == "" || len(pageUsers) == 0 {
			break
		}
		cursor = nextCursor

		time.Sleep(c.gql.PacedDelay(c.cfg.Delay))
	}

	return users, nil
}

// ── Spaces ───────────────────────────────────────────────

// GetSpace fetches audio space details by ID.
// Note: Spaces API requires a different endpoint not available in GraphQL.
// Returns a basic stub — full implementation would need the Spaces API.
func (c *Client) GetSpace(id string) (*Space, error) {
	return nil, fmt.Errorf("get space %s: spaces API not supported in GraphQL mode", id)
}

// ── Trends ───────────────────────────────────────────────

// GetTrends fetches current trending topics via GraphQL ExplorePage.
func (c *Client) GetTrends() ([]string, error) {
	data, err := c.gql.doGraphQL(gqlExplorePage, map[string]any{}, "")
	if err != nil {
		return nil, fmt.Errorf("get trends: %w", err)
	}

	// Recursively find all trend names in the response
	var trends []string
	findTrends(data, &trends)
	return trends, nil
}

// findTrends recursively finds all trend names in a GraphQL response.
func findTrends(node any, trends *[]string) {
	switch v := node.(type) {
	case map[string]any:
		// Check: {"trend": {"name": "..."}} (guide.json format)
		if trend := asMap(v["trend"]); trend != nil {
			if name := asStr(trend["name"]); name != "" {
				*trends = append(*trends, name)
				return
			}
		}
		// Check: {"__typename": "TimelineTrend", "name": "..."} (ExplorePage format)
		if asStr(v["__typename"]) == "TimelineTrend" {
			if name := asStr(v["name"]); name != "" {
				*trends = append(*trends, name)
				return
			}
		}
		for _, val := range v {
			findTrends(val, trends)
		}
	case []any:
		for _, item := range v {
			findTrends(item, trends)
		}
	}
}
