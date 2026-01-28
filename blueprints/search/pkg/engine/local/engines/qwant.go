package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Qwant implements Qwant web search.
type Qwant struct {
	*BaseEngine
}

// NewQwant creates a new Qwant engine.
// Note: Qwant has aggressive bot protection (CAPTCHA).
func NewQwant() *Qwant {
	q := &Qwant{
		BaseEngine: NewBaseEngine("qwant", "qw", []Category{CategoryGeneral, CategoryWeb}),
	}

	q.SetPaging(true).
		SetSafeSearch(true).
		SetTimeout(5 * time.Second).
		SetDisabled(true). // Bot protection (CAPTCHA)
		SetAbout(EngineAbout{
			Website:    "https://www.qwant.com",
			WikidataID: "Q14657870",
			Results:    "JSON",
		})

	return q
}

func (q *Qwant) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("count", "10")
	queryParams.Set("locale", "en_US")
	queryParams.Set("tgp", "3")
	queryParams.Set("llm", "false")

	// Safe search
	safeValue := "1"
	switch params.SafeSearch {
	case SafeSearchOff:
		safeValue = "0"
	case SafeSearchStrict:
		safeValue = "2"
	}
	queryParams.Set("safesearch", safeValue)

	// Pagination
	if params.PageNo > 1 {
		queryParams.Set("offset", fmt.Sprintf("%d", (params.PageNo-1)*10))
	}

	params.URL = "https://api.qwant.com/v3/search/web?" + queryParams.Encode()
	params.Headers.Set("Accept", "application/json")
	params.Headers.Set("User-Agent", "Mozilla/5.0 (compatible; SearXNG)")

	return nil
}

func (q *Qwant) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	var apiResp qwantResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Status != "success" {
		return results, nil
	}

	// Navigate through mainline items
	for _, mainline := range apiResp.Data.Result.Items.Mainline {
		for _, item := range mainline.Items {
			if item.Type == "web" {
				result := Result{
					URL:     item.URL,
					Title:   item.Title,
					Content: item.Desc,
				}
				result.ParsedURL, _ = url.Parse(result.URL)
				results.Add(result)
			}
		}
	}

	return results, nil
}

type qwantResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result struct {
			Items struct {
				Mainline []struct {
					Type  string `json:"type"`
					Items []struct {
						Type  string `json:"type"`
						URL   string `json:"url"`
						Title string `json:"title"`
						Desc  string `json:"desc"`
					} `json:"items"`
				} `json:"mainline"`
			} `json:"items"`
		} `json:"result"`
	} `json:"data"`
}

// QwantImages implements Qwant image search.
type QwantImages struct {
	*BaseEngine
}

// NewQwantImages creates a new Qwant Images engine.
// Note: Qwant has aggressive bot protection (CAPTCHA).
func NewQwantImages() *QwantImages {
	q := &QwantImages{
		BaseEngine: NewBaseEngine("qwant images", "qwi", []Category{CategoryImages}),
	}

	q.SetPaging(true).
		SetSafeSearch(true).
		SetTimeout(5 * time.Second).
		SetDisabled(true). // Bot protection (CAPTCHA)
		SetAbout(EngineAbout{
			Website:    "https://www.qwant.com",
			WikidataID: "Q14657870",
			Results:    "JSON",
		})

	return q
}

func (q *QwantImages) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("count", "50")
	queryParams.Set("locale", "en_US")
	queryParams.Set("tgp", "3")

	// Safe search
	safeValue := "1"
	switch params.SafeSearch {
	case SafeSearchOff:
		safeValue = "0"
	case SafeSearchStrict:
		safeValue = "2"
	}
	queryParams.Set("safesearch", safeValue)

	// Pagination
	if params.PageNo > 1 {
		queryParams.Set("offset", fmt.Sprintf("%d", (params.PageNo-1)*50))
	}

	params.URL = "https://api.qwant.com/v3/search/images?" + queryParams.Encode()
	params.Headers.Set("Accept", "application/json")
	params.Headers.Set("User-Agent", "Mozilla/5.0 (compatible; SearXNG)")

	return nil
}

func (q *QwantImages) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	var apiResp qwantImagesResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Status != "success" {
		return results, nil
	}

	for _, mainline := range apiResp.Data.Result.Items.Mainline {
		for _, item := range mainline.Items {
			result := Result{
				URL:          item.URL,
				Title:        item.Title,
				Template:     "images",
				ThumbnailURL: item.Thumbnail,
				ImageURL:     item.Media,
				Resolution:   fmt.Sprintf("%dx%d", item.Width, item.Height),
			}
			result.ParsedURL, _ = url.Parse(result.URL)
			results.Add(result)
		}
	}

	return results, nil
}

type qwantImagesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result struct {
			Items struct {
				Mainline []struct {
					Type  string `json:"type"`
					Items []struct {
						URL       string `json:"url"`
						Title     string `json:"title"`
						Thumbnail string `json:"thumbnail"`
						Media     string `json:"media"`
						Width     int    `json:"width"`
						Height    int    `json:"height"`
					} `json:"items"`
				} `json:"mainline"`
			} `json:"items"`
		} `json:"result"`
	} `json:"data"`
}

// QwantNews implements Qwant news search.
type QwantNews struct {
	*BaseEngine
}

// NewQwantNews creates a new Qwant News engine.
// Note: Qwant has aggressive bot protection (CAPTCHA).
func NewQwantNews() *QwantNews {
	q := &QwantNews{
		BaseEngine: NewBaseEngine("qwant news", "qwn", []Category{CategoryNews}),
	}

	q.SetPaging(true).
		SetTimeout(5 * time.Second).
		SetDisabled(true). // Bot protection (CAPTCHA)
		SetAbout(EngineAbout{
			Website:    "https://www.qwant.com",
			WikidataID: "Q14657870",
			Results:    "JSON",
		})

	return q
}

func (q *QwantNews) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("count", "10")
	queryParams.Set("locale", "en_US")
	queryParams.Set("tgp", "3")

	if params.PageNo > 1 {
		queryParams.Set("offset", fmt.Sprintf("%d", (params.PageNo-1)*10))
	}

	params.URL = "https://api.qwant.com/v3/search/news?" + queryParams.Encode()
	params.Headers.Set("Accept", "application/json")
	params.Headers.Set("User-Agent", "Mozilla/5.0 (compatible; SearXNG)")

	return nil
}

func (q *QwantNews) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	var apiResp qwantNewsResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Status != "success" {
		return results, nil
	}

	for _, mainline := range apiResp.Data.Result.Items.Mainline {
		for _, item := range mainline.Items {
			result := Result{
				URL:          item.URL,
				Title:        item.Title,
				Content:      item.Desc,
				Template:     "news",
				ThumbnailURL: item.Thumbnail,
			}

			// Parse date
			if item.Date > 0 {
				result.PublishedAt = time.Unix(item.Date, 0)
			}

			result.ParsedURL, _ = url.Parse(result.URL)
			results.Add(result)
		}
	}

	return results, nil
}

type qwantNewsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result struct {
			Items struct {
				Mainline []struct {
					Type  string `json:"type"`
					Items []struct {
						URL       string `json:"url"`
						Title     string `json:"title"`
						Desc      string `json:"desc"`
						Thumbnail string `json:"thumbnail"`
						Date      int64  `json:"date"`
					} `json:"items"`
				} `json:"mainline"`
			} `json:"items"`
		} `json:"result"`
	} `json:"data"`
}
