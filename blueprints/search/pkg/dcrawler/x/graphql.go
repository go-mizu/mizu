package x

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// graphqlClient handles raw HTTP requests to X's GraphQL API.
type graphqlClient struct {
	http      *http.Client
	authToken string
	ct0       string
}

func newGraphQLClient(authToken, ct0 string, timeout time.Duration) *graphqlClient {
	return &graphqlClient{
		http: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Detect login redirect (session expired)
				if strings.Contains(req.URL.Path, "/accounts/login") ||
					strings.Contains(req.URL.Path, "/login") {
					return fmt.Errorf("redirected to login: session expired")
				}
				return nil
			},
		},
		authToken: authToken,
		ct0:       ct0,
	}
}

// doGraphQL makes a GET request to a GraphQL endpoint and returns parsed JSON.
func (g *graphqlClient) doGraphQL(endpoint string, variables map[string]any, fieldToggles string) (map[string]any, error) {
	// Build URL
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
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Parse URL to get path for TID generation
	parsedURL, _ := url.Parse(fullURL)
	apiPath := parsedURL.Path

	// Generate x-client-transaction-id
	tid, err := generateTID(apiPath)
	if err != nil {
		// Continue without TID — some endpoints may still work
		tid = ""
	}

	// Set headers (from Nitter's genHeaders)
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "gzip")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://x.com")
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("x-twitter-active-user", "yes")
	req.Header.Set("x-twitter-client-language", "en")
	req.Header.Set("priority", "u=1, i")

	// Auth headers
	req.Header.Set("authorization", bearerToken)
	req.Header.Set("x-twitter-auth-type", "OAuth2Session")
	req.Header.Set("x-csrf-token", g.ct0)
	req.Header.Set("cookie", "auth_token="+g.authToken+"; ct0="+g.ct0)

	// Security headers
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="142", "Chromium";v="142", "Not A(Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")

	if tid != "" {
		req.Header.Set("x-client-transaction-id", tid)
	}

	resp, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	// Handle gzip
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == 429 {
		// Rate limited — check reset header
		resetStr := resp.Header.Get("x-rate-limit-reset")
		if resetStr != "" {
			resetTime, _ := fmt.Sscanf(resetStr, "%d", new(int))
			_ = resetTime
		}
		return nil, fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != 200 {
		// Truncate body for error message
		errBody := string(body)
		if len(errBody) > 200 {
			errBody = errBody[:200]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, errBody)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	// Check for API errors — only fail on critical codes.
	// Twitter often returns non-critical errors alongside valid data.
	if errs := asSlice(result["errors"]); len(errs) > 0 {
		if first := asMap(errs[0]); first != nil {
			msg := asStr(first["message"])
			code := asInt(first["code"])
			switch code {
			case 88:
				return nil, fmt.Errorf("rate limited: %s", msg)
			case 89:
				return nil, fmt.Errorf("expired token: %s", msg)
			case 239:
				return nil, fmt.Errorf("bad token: %s", msg)
			case 326:
				return nil, fmt.Errorf("account locked: %s", msg)
			case 37:
				return nil, fmt.Errorf("user suspended: %s", msg)
			default:
				// Non-critical error (e.g. code 0 "Query: Unspecified").
				// Return data if present — caller will use it.
				if result["data"] != nil {
					return result, nil
				}
				return nil, fmt.Errorf("API error %d: %s", code, msg)
			}
		}
	}

	return result, nil
}
