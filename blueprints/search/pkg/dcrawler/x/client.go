package x

import (
	"context"
	"fmt"
	"net/http"
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
func (c *Client) GetProfile(username string) (*Profile, error) {
	data, err := c.gql.doGraphQL(gqlUserByScreenName, map[string]any{
		"screen_name":                 username,
		"withSafetyModeUserFields":    true,
		"withSuperFollowsUserFields":  true,
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

		time.Sleep(c.cfg.Delay)
	}

	return users, nil
}

// ── Tweets ───────────────────────────────────────────────

// GetTweets fetches user timeline tweets.
func (c *Client) GetTweets(ctx context.Context, username string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	return c.getUserTimeline(ctx, username, gqlUserTweetsV2, "tweets", maxTweets, cb)
}

// GetTweetsAndReplies fetches user timeline tweets including replies.
func (c *Client) GetTweetsAndReplies(ctx context.Context, username string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	return c.getUserTimeline(ctx, username, gqlUserTweetsAndRepliesV2, "tweets+replies", maxTweets, cb)
}

// GetMediaTweets fetches user media timeline (photos/videos only).
// Uses legacy UserMedia endpoint with userId (cookie auth compatibility).
func (c *Client) GetMediaTweets(ctx context.Context, username string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	return c.getUserTimeline(ctx, username, gqlUserMedia, "media", maxTweets, cb)
}

func (c *Client) getUserTimeline(ctx context.Context, username, endpoint, phase string, maxTweets int, cb ProgressCallback) ([]Tweet, error) {
	userID, err := c.resolveUserID(username)
	if err != nil {
		return nil, err
	}

	var tweets []Tweet
	cursor := ""
	emptyPages := 0

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

		data, err := c.gql.doGraphQL(endpoint, vars, toggles)
		if err != nil {
			// Fatal error (rate limit, expired token, etc.) — stop
			if len(tweets) > 0 {
				return tweets, err
			}
			return nil, fmt.Errorf("get %s @%s: %w", phase, username, err)
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

		// Stop if no cursor or too many empty pages
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

		time.Sleep(c.cfg.Delay)
	}

	if cb != nil {
		cb(Progress{Phase: phase, Current: int64(len(tweets)), Done: true})
	}
	return tweets, nil
}

// GetTweet fetches a single tweet by ID.
// Uses ConversationTimeline endpoint (same as Nitter for cookie auth).
func (c *Client) GetTweet(id string) (*Tweet, error) {
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

	data, err := c.gql.doGraphQL(gqlConversationTimeline, vars, tweetDetailFieldToggles)
	if err != nil {
		return nil, fmt.Errorf("get tweet %s: %w", id, err)
	}

	mainTweet, _, _ := parseConversation(data, id)
	if mainTweet == nil {
		return nil, fmt.Errorf("get tweet %s: not found", id)
	}
	return mainTweet, nil
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

		data, err := c.gql.doGraphQL(gqlConversationTimeline, vars, tweetDetailFieldToggles)
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
	var tweets []Tweet
	cursor := ""
	emptyPages := 0

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

		data, err := c.gql.doGraphQL(gqlSearchTimeline, vars, "")
		if err != nil {
			if len(tweets) > 0 {
				return tweets, err
			}
			return nil, fmt.Errorf("search tweets: %w", err)
		}

		result := parseSearchTweets(data)
		tweets = append(tweets, result.Tweets...)

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
		if len(result.Tweets) == 0 {
			emptyPages++
			if emptyPages >= 3 {
				break
			}
		} else {
			emptyPages = 0
		}
		cursor = result.Cursor

		time.Sleep(c.cfg.Delay)
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

		time.Sleep(c.cfg.Delay)
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

		time.Sleep(c.cfg.Delay)
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

		time.Sleep(c.cfg.Delay)
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

		time.Sleep(c.cfg.Delay)
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

		time.Sleep(c.cfg.Delay)
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
