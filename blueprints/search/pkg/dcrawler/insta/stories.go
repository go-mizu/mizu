package insta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// GetStories fetches the current stories for a user by their user ID.
// Requires authentication. Uses the iPhone API for higher quality media.
func (c *Client) GetStories(ctx context.Context, userID string) (*Story, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("stories endpoint requires authentication")
	}

	if err := c.doSleep(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("reel_ids", userID)

	data, err := c.doGetIPhone(ctx, "api/v1/feed/reels_media/", params)
	if err != nil {
		return nil, fmt.Errorf("fetch stories for user %s: %w", userID, err)
	}

	var resp storiesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse stories response: %w", err)
	}

	story := &Story{UserID: userID}

	// Try reels_media format first
	for _, reel := range resp.ReelsMedia {
		if fmt.Sprintf("%d", reel.ID) == userID || fmt.Sprintf("%d", reel.User.PK) == userID {
			story.Username = reel.User.Username
			for _, item := range reel.Items {
				story.Items = append(story.Items, storyItemFromAPI(item))
			}
			return story, nil
		}
	}

	// Try reels map format
	if reelData, ok := resp.Reels[userID]; ok {
		story.Username = reelData.User.Username
		for _, item := range reelData.Items {
			story.Items = append(story.Items, storyItemFromAPI(item))
		}
		return story, nil
	}

	return story, nil // empty story (user may not have active stories)
}

// GetStoriesByUsername fetches stories by username (resolves to user ID first).
func (c *Client) GetStoriesByUsername(ctx context.Context, username string) (*Story, error) {
	profile, err := c.GetProfile(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("resolve username: %w", err)
	}
	story, err := c.GetStories(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	story.Username = username
	return story, nil
}

// GetHighlights fetches the list of highlights for a user.
// Requires authentication.
func (c *Client) GetHighlights(ctx context.Context, userID string) ([]Highlight, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("highlights endpoint requires authentication")
	}

	if err := c.doSleep(ctx); err != nil {
		return nil, err
	}

	vars := map[string]any{
		"user_id":          userID,
		"include_chaining": false,
		"include_reel":     false,
		"include_suggested_users":  false,
		"include_logged_out_extras": false,
		"include_highlight_reels":   true,
	}

	data, err := c.graphQL(ctx, HashHighlights, vars)
	if err != nil {
		return nil, fmt.Errorf("fetch highlights for user %s: %w", userID, err)
	}

	var resp highlightsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse highlights response: %w", err)
	}

	if resp.Data.User == nil || resp.Data.User.EdgeHighlightReels == nil {
		return nil, nil
	}

	var highlights []Highlight
	for _, edge := range resp.Data.User.EdgeHighlightReels.Edges {
		h := Highlight{
			ID:        edge.Node.ID,
			Title:     edge.Node.Title,
			CoverURL:  edge.Node.Cover.CroppedURL,
			ItemCount: edge.Node.EdgeHighlightItems.Count,
		}
		highlights = append(highlights, h)
	}

	return highlights, nil
}

// GetHighlightItems fetches the media items for a specific highlight reel.
// Requires authentication. Uses the iPhone API.
func (c *Client) GetHighlightItems(ctx context.Context, highlightID string) ([]StoryItem, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("highlight items endpoint requires authentication")
	}

	if err := c.doSleep(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("reel_ids", "highlight:"+highlightID)

	data, err := c.doGetIPhone(ctx, "api/v1/feed/reels_media/", params)
	if err != nil {
		return nil, fmt.Errorf("fetch highlight %s: %w", highlightID, err)
	}

	var resp storiesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse highlight response: %w", err)
	}

	var items []StoryItem
	for _, reel := range resp.ReelsMedia {
		for _, item := range reel.Items {
			items = append(items, storyItemFromAPI(item))
		}
	}

	// Also check reels map
	key := "highlight:" + highlightID
	if reelData, ok := resp.Reels[key]; ok {
		for _, item := range reelData.Items {
			items = append(items, storyItemFromAPI(item))
		}
	}

	return items, nil
}

func storyItemFromAPI(item storyMediaItem) StoryItem {
	si := StoryItem{
		ID:        item.ID,
		TakenAt:   time.Unix(item.TakenAt, 0),
		ExpiresAt: time.Unix(item.ExpiringAt, 0),
		OwnerID:   fmt.Sprintf("%d", item.User.PK),
		OwnerName: item.User.Username,
		Width:     item.OriginalWidth,
		Height:    item.OriginalHeight,
	}

	switch item.MediaType {
	case 1:
		si.TypeName = "StoryImage"
		if item.ImageVersions2 != nil && len(item.ImageVersions2.Candidates) > 0 {
			si.DisplayURL = item.ImageVersions2.Candidates[0].URL
			if si.Width == 0 {
				si.Width = item.ImageVersions2.Candidates[0].Width
				si.Height = item.ImageVersions2.Candidates[0].Height
			}
		}
	case 2:
		si.TypeName = "StoryVideo"
		si.IsVideo = true
		if len(item.VideoVersions) > 0 {
			si.VideoURL = item.VideoVersions[0].URL
			if si.Width == 0 {
				si.Width = item.VideoVersions[0].Width
				si.Height = item.VideoVersions[0].Height
			}
		}
		if item.ImageVersions2 != nil && len(item.ImageVersions2.Candidates) > 0 {
			si.DisplayURL = item.ImageVersions2.Candidates[0].URL
		}
	}

	return si
}
