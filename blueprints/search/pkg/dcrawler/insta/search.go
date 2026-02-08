package insta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Search performs a top search for users, hashtags, and places.
func (c *Client) Search(ctx context.Context, query string, count int) (*SearchResult, error) {
	if count <= 0 {
		count = 50
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("count", strconv.Itoa(count))
	rawURL := TopSearchURL + "?" + params.Encode()

	data, err := c.doGet(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("search %q: %w", query, err)
	}

	var resp topSearchResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	result := &SearchResult{}

	for _, uw := range resp.Users {
		result.Users = append(result.Users, SearchUser{
			ID:         uw.User.PK,
			Username:   uw.User.Username,
			FullName:   uw.User.FullName,
			IsPrivate:  uw.User.IsPrivate,
			IsVerified: uw.User.IsVerified,
			PicURL:     uw.User.ProfilePicURL,
			Followers:  uw.User.FollowerCount,
		})
	}

	for _, hw := range resp.Hashtags {
		result.Hashtags = append(result.Hashtags, SearchHashtag{
			ID:         hw.Hashtag.ID,
			Name:       hw.Hashtag.Name,
			MediaCount: hw.Hashtag.MediaCount,
		})
	}

	for _, pw := range resp.Places {
		result.Places = append(result.Places, SearchPlace{
			LocationID: pw.Place.Location.PK,
			Title:      pw.Place.Title,
			Address:    pw.Place.Location.Address,
			City:       pw.Place.Location.City,
			Lat:        pw.Place.Location.Lat,
			Lng:        pw.Place.Location.Lng,
		})
	}

	return result, nil
}
