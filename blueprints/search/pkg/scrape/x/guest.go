package x

// guest.go — fetch tweets and profiles via X's guest token API.
//
// X allows anonymous access to some GraphQL endpoints using a "guest token"
// obtained by posting to /1.1/guest/activate.json with the public bearer token.
// Guest tokens have their own rate limit pool, separate from cookie-auth sessions.
//
// Supported without auth:
//   - GetProfileGuest(username) — fetch public user profile
//   - GetTweetGuest(id) — fetch a single tweet by ID

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const guestActivateURL = "https://api.twitter.com/1.1/guest/activate.json"

// guestTokenCache caches the last fetched guest token to avoid re-activating on every call.
var (
	guestMu        sync.Mutex
	cachedToken    string
	cachedTokenExp time.Time
)

// fetchGuestToken obtains a fresh guest token from X's activate endpoint.
// Guest tokens are cached for 45 minutes (X typically issues them with a ~1h lifetime).
func fetchGuestToken() (string, error) {
	guestMu.Lock()
	defer guestMu.Unlock()

	if cachedToken != "" && time.Now().Before(cachedTokenExp) {
		return cachedToken, nil
	}

	req, err := http.NewRequest("POST", guestActivateURL, nil)
	if err != nil {
		return "", fmt.Errorf("guest activate request: %w", err)
	}
	req.Header.Set("Authorization", bearerToken)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("guest activate fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("guest activate read: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("guest activate HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body))[:min(200, len(body))])
	}

	var result struct {
		GuestToken string `json:"guest_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("guest activate parse: %w", err)
	}
	if result.GuestToken == "" {
		return "", fmt.Errorf("guest activate: empty token in response")
	}

	cachedToken = result.GuestToken
	cachedTokenExp = time.Now().Add(45 * time.Minute)
	return cachedToken, nil
}

// invalidateGuestToken clears the cached guest token (called when a 403/401 is returned).
func invalidateGuestToken() {
	guestMu.Lock()
	cachedToken = ""
	guestMu.Unlock()
}

// doGuestGraphQL makes a GraphQL GET request using a guest token instead of cookie auth.
func doGuestGraphQL(guestToken, endpoint string, variables map[string]any, fieldToggles string) (map[string]any, error) {
	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("marshal variables: %w", err)
	}

	params := url.Values{}
	params.Set("variables", string(varsJSON))
	params.Set("features", gqlFeatures)
	if fieldToggles != "" {
		params.Set("fieldToggles", fieldToggles)
	}

	fullURL := graphqlBaseURL + endpoint + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("guest graphql request: %w", err)
	}

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "gzip")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("authorization", bearerToken)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://x.com")
	req.Header.Set("referer", "https://x.com/")
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("x-guest-token", guestToken)
	req.Header.Set("x-twitter-active-user", "yes")
	req.Header.Set("x-twitter-client-language", "en")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="142", "Chromium";v="142", "Not A(Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")
	req.Header.Set("priority", "u=1, i")

	// Generate x-client-transaction-id (same as authenticated requests)
	parsedURL, _ := url.Parse(fullURL)
	if tid, err := generateTID(parsedURL.Path); err == nil && tid != "" {
		req.Header.Set("x-client-transaction-id", tid)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("guest graphql fetch: %w", err)
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("guest graphql gzip: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("guest graphql read: %w", err)
	}

	if resp.StatusCode == 429 {
		return nil, &RateLimitError{Wait: 15 * time.Minute}
	}
	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		// Guest token expired or invalid
		invalidateGuestToken()
		return nil, fmt.Errorf("guest token rejected (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("guest graphql HTTP %d: %s", resp.StatusCode, snippet)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("guest graphql parse: %w", err)
	}

	// Check for critical API errors
	if errs := asSlice(result["errors"]); len(errs) > 0 {
		if first := asMap(errs[0]); first != nil {
			code := asInt(first["code"])
			msg := asStr(first["message"])
			switch code {
			case 88:
				return nil, &RateLimitError{Wait: 15 * time.Minute}
			case 239:
				// Bad guest token — invalidate and signal retry
				invalidateGuestToken()
				return nil, fmt.Errorf("bad guest token: %s", msg)
			default:
				if result["data"] != nil {
					return result, nil
				}
				return nil, fmt.Errorf("API error %d: %s", code, msg)
			}
		}
	}

	return result, nil
}

// GetProfileGuest fetches a public user profile using a guest token (no cookie auth needed).
func GetProfileGuest(username string) (*Profile, error) {
	token, err := fetchGuestToken()
	if err != nil {
		return nil, fmt.Errorf("get guest token: %w", err)
	}

	data, err := doGuestGraphQL(token, gqlUserByScreenName, map[string]any{
		"screen_name":              username,
		"withSafetyModeUserFields": true,
	}, userFieldToggles)
	if err != nil {
		// If token was invalidated, retry once with a fresh token
		if strings.Contains(err.Error(), "guest token") {
			token, err = fetchGuestToken()
			if err != nil {
				return nil, fmt.Errorf("get profile @%s (guest retry): %w", username, err)
			}
			data, err = doGuestGraphQL(token, gqlUserByScreenName, map[string]any{
				"screen_name":              username,
				"withSafetyModeUserFields": true,
			}, userFieldToggles)
		}
		if err != nil {
			return nil, fmt.Errorf("get profile @%s (guest): %w", username, err)
		}
	}

	p := parseUserResult(data)
	if p == nil {
		return nil, fmt.Errorf("get profile @%s (guest): user not found", username)
	}
	return p, nil
}

// GetTweetGuest fetches a single tweet by ID using a guest token (no cookie auth needed).
func GetTweetGuest(id string) (*Tweet, error) {
	token, err := fetchGuestToken()
	if err != nil {
		return nil, fmt.Errorf("get guest token: %w", err)
	}

	data, err := doGuestGraphQL(token, gqlConversationTimeline, map[string]any{
		"focalTweetId":                           id,
		"count":                                  1,
		"includePromotedContent":                 false,
		"withCommunity":                          false,
		"withQuickPromoteEligibilityTweetFields": false,
		"withBirdwatchNotes":                     false,
		"withVoice":                              false,
	}, tweetDetailFieldToggles)
	if err != nil {
		// Retry once on bad token
		if strings.Contains(err.Error(), "guest token") {
			token, err = fetchGuestToken()
			if err != nil {
				return nil, fmt.Errorf("get tweet %s (guest retry): %w", id, err)
			}
			data, err = doGuestGraphQL(token, gqlConversationTimeline, map[string]any{
				"focalTweetId":                           id,
				"count":                                  1,
				"includePromotedContent":                 false,
				"withCommunity":                          false,
				"withQuickPromoteEligibilityTweetFields": false,
				"withBirdwatchNotes":                     false,
				"withVoice":                              false,
			}, tweetDetailFieldToggles)
		}
		if err != nil {
			return nil, fmt.Errorf("get tweet %s (guest): %w", id, err)
		}
	}

	mainTweet, _, _ := parseConversation(data, id)
	if mainTweet == nil {
		return nil, fmt.Errorf("tweet %s not found", id)
	}
	return mainTweet, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
