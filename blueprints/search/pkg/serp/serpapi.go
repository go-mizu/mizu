package serp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const serpAPIBase = "https://serpapi.com"

var verificationLinkRe = regexp.MustCompile(`https://serpapi\.com/users/confirmation\?confirmation_token=[^\s"<']+`)

type SerpAPIClient struct {
	hc *http.Client
}

func NewSerpAPIClient() *SerpAPIClient {
	return &SerpAPIClient{hc: &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

// RegisterHTTP attempts to register an account via HTTP POST.
// Returns nil on success (2xx), or error if blocked/failed.
func (c *SerpAPIClient) RegisterHTTP(email, password string) error {
	body, _ := json.Marshal(map[string]any{
		"user": map[string]string{
			"email":                 email,
			"password":              password,
			"password_confirmation": password,
		},
	})
	req, _ := http.NewRequest("POST", serpAPIBase+"/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("serpapi register HTTP %d", resp.StatusCode)
	}
	return nil
}

// VerifyEmail follows the verification link from the email body.
func (c *SerpAPIClient) VerifyEmail(emailBody string) error {
	link := verificationLinkRe.FindString(emailBody)
	if link == "" {
		return fmt.Errorf("no verification link found in email body")
	}
	hc := &http.Client{Timeout: 20 * time.Second}
	resp, err := hc.Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("verification GET returned %d", resp.StatusCode)
	}
	return nil
}

// SignIn logs in and returns the API key.
func (c *SerpAPIClient) SignIn(email, password string) (apiKey string, err error) {
	body, _ := json.Marshal(map[string]any{
		"user": map[string]string{"email": email, "password": password},
	})
	hc := &http.Client{Timeout: 20 * time.Second}
	req, _ := http.NewRequest("POST", serpAPIBase+"/users/sign_in.json", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.APIKey == "" {
		return "", fmt.Errorf("sign_in response missing api_key")
	}
	return result.APIKey, nil
}

type AccountInfo struct {
	APIKey            string `json:"api_key"`
	PlanSearchesLeft  int    `json:"plan_searches_left"`
	TotalSearchesLeft int    `json:"total_searches_left"`
	ThisMonthUsage    int    `json:"this_month_usage"`
}

// GetAccount fetches account info (doesn't count toward quota).
func (c *SerpAPIClient) GetAccount(apiKey string) (*AccountInfo, error) {
	hc := &http.Client{Timeout: 15 * time.Second}
	resp, err := hc.Get(fmt.Sprintf("%s/account.json?api_key=%s", serpAPIBase, url.QueryEscape(apiKey)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("account API returned %d", resp.StatusCode)
	}
	var info AccountInfo
	return &info, json.NewDecoder(resp.Body).Decode(&info)
}

type SearchResult struct {
	SearchMetadata map[string]any   `json:"search_metadata"`
	OrganicResults []map[string]any `json:"organic_results"`
	Error          string           `json:"error"`
}

// Search executes a Google search via SerpAPI.
func (c *SerpAPIClient) Search(apiKey, query string) (*SearchResult, error) {
	hc := &http.Client{Timeout: 30 * time.Second}
	u := fmt.Sprintf("%s/search.json?q=%s&api_key=%s&engine=google",
		serpAPIBase, url.QueryEscape(query), url.QueryEscape(apiKey))
	resp, err := hc.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("serpapi: %s", result.Error)
	}
	return &result, nil
}
