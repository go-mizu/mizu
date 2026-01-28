package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// YouTube implements YouTube video search (no API key required).
type YouTube struct {
	*BaseEngine
	timeRangeMap map[TimeRange]string
}

// NewYouTube creates a new YouTube engine.
func NewYouTube() *YouTube {
	y := &YouTube{
		BaseEngine: NewBaseEngine("youtube", "yt", []Category{CategoryVideos, CategoryMusic}),
		timeRangeMap: map[TimeRange]string{
			TimeRangeDay:   "Ag",
			TimeRangeWeek:  "Aw",
			TimeRangeMonth: "BA",
			TimeRangeYear:  "BQ",
		},
	}

	y.SetPaging(true).
		SetTimeRangeSupport(true).
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://www.youtube.com",
			WikidataID: "Q866",
			Results:    "HTML+JSON",
		})

	return y
}

func (y *YouTube) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("search_query", query)

	// Time range filter
	if params.TimeRange != "" {
		if sp, ok := y.timeRangeMap[params.TimeRange]; ok {
			queryParams.Set("sp", "EgIIA"+sp+"%3D%3D")
		}
	}

	params.URL = "https://www.youtube.com/results?" + queryParams.Encode()
	params.Headers.Set("Accept", "text/html")
	params.Headers.Set("Accept-Language", "en-US,en;q=0.9")
	params.Cookies = append(params.Cookies, &http.Cookie{Name: "CONSENT", Value: "YES+"})

	return nil
}

func (y *YouTube) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	// Extract ytInitialData JSON from HTML
	re := regexp.MustCompile(`var ytInitialData = ({.+?});</script>`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		return results, nil
	}

	var data ytInitialData
	if err := json.Unmarshal(matches[1], &data); err != nil {
		return results, nil
	}

	// Navigate to video results
	for _, content := range data.Contents.TwoColumnSearchResultsRenderer.PrimaryContents.SectionListRenderer.Contents {
		for _, item := range content.ItemSectionRenderer.Contents {
			if item.VideoRenderer.VideoID != "" {
				vr := item.VideoRenderer

				// Build title from runs
				var title strings.Builder
				for _, run := range vr.Title.Runs {
					title.WriteString(run.Text)
				}

				// Build description from runs
				var desc strings.Builder
				for _, run := range vr.DescriptionSnippet.Runs {
					desc.WriteString(run.Text)
				}

				result := Result{
					URL:       fmt.Sprintf("https://www.youtube.com/watch?v=%s", vr.VideoID),
					Title:     title.String(),
					Content:   desc.String(),
					Template:  "videos",
					Duration:  vr.LengthText.SimpleText,
					EmbedURL:  fmt.Sprintf("https://www.youtube-nocookie.com/embed/%s", vr.VideoID),
					IFrameSrc: fmt.Sprintf("https://www.youtube-nocookie.com/embed/%s", vr.VideoID),
				}

				// Get author/channel
				if len(vr.OwnerText.Runs) > 0 {
					result.Artist = vr.OwnerText.Runs[0].Text
				}

				// Get thumbnail (last one is usually highest quality)
				if len(vr.Thumbnail.Thumbnails) > 0 {
					result.ThumbnailURL = vr.Thumbnail.Thumbnails[len(vr.Thumbnail.Thumbnails)-1].URL
				}

				result.ParsedURL, _ = url.Parse(result.URL)
				results.Add(result)
			}
		}
	}

	return results, nil
}

// YouTube JSON structures
type ytInitialData struct {
	Contents struct {
		TwoColumnSearchResultsRenderer struct {
			PrimaryContents struct {
				SectionListRenderer struct {
					Contents []struct {
						ItemSectionRenderer struct {
							Contents []struct {
								VideoRenderer ytVideoRenderer `json:"videoRenderer"`
							} `json:"contents"`
						} `json:"itemSectionRenderer"`
					} `json:"contents"`
				} `json:"sectionListRenderer"`
			} `json:"primaryContents"`
		} `json:"twoColumnSearchResultsRenderer"`
	} `json:"contents"`
}

type ytVideoRenderer struct {
	VideoID   string `json:"videoId"`
	Thumbnail struct {
		Thumbnails []struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"thumbnails"`
	} `json:"thumbnail"`
	Title struct {
		Runs []struct {
			Text string `json:"text"`
		} `json:"runs"`
	} `json:"title"`
	DescriptionSnippet struct {
		Runs []struct {
			Text string `json:"text"`
		} `json:"runs"`
	} `json:"descriptionSnippet"`
	LengthText struct {
		SimpleText string `json:"simpleText"`
	} `json:"lengthText"`
	OwnerText struct {
		Runs []struct {
			Text string `json:"text"`
		} `json:"runs"`
	} `json:"ownerText"`
	ViewCountText struct {
		SimpleText string `json:"simpleText"`
	} `json:"viewCountText"`
	PublishedTimeText struct {
		SimpleText string `json:"simpleText"`
	} `json:"publishedTimeText"`
}
