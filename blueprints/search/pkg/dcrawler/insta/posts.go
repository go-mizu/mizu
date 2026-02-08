package insta

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	// FeedUserURL is the mobile API endpoint for user feed (first page works unauthenticated).
	FeedUserURL = "https://i.instagram.com/api/v1/feed/user/"
)

// feedResponse is the response from the feed/user/ endpoint.
type feedResponse struct {
	Items         []feedItem `json:"items"`
	NumResults    int        `json:"num_results"`
	MoreAvailable bool       `json:"more_available"`
	NextMaxID     string     `json:"next_max_id"`
	Status        string     `json:"status"`
	Message       string     `json:"message"`
}

type feedItem struct {
	ID        string `json:"id"`
	PK        int64  `json:"pk"`
	Code      string `json:"code"`
	MediaType int    `json:"media_type"` // 1=image, 2=video, 8=carousel
	LikeCount int64  `json:"like_count"`
	CommentCount int64 `json:"comment_count"`
	ViewCount    int64 `json:"view_count"`
	PlayCount    int64 `json:"play_count"`
	TakenAt      int64 `json:"taken_at"`
	Caption   *struct {
		Text string `json:"text"`
	} `json:"caption"`
	ImageVersions2 *imageVersions `json:"image_versions2"`
	VideoVersions  []videoVersion `json:"video_versions"`
	CarouselMedia  []carouselItem `json:"carousel_media"`
	User           struct {
		PK       int64  `json:"pk"`
		Username string `json:"username"`
	} `json:"user"`
	Location *struct {
		PK   int64  `json:"pk"`
		Name string `json:"name"`
	} `json:"location"`
}

type imageVersions struct {
	Candidates []imageCandidate `json:"candidates"`
}

type imageCandidate struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type videoVersion struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type carouselItem struct {
	ID             string         `json:"id"`
	MediaType      int            `json:"media_type"`
	ImageVersions2 *imageVersions `json:"image_versions2"`
	VideoVersions  []videoVersion `json:"video_versions"`
}

// GetUserPosts fetches posts for a user using the profile endpoint (first 12 posts)
// and optionally the feed API for additional posts.
//
// Without authentication, typically only the first 12 posts (from profile) are available.
// The feed API may return additional posts but rate-limits quickly.
func (c *Client) GetUserPosts(ctx context.Context, username string, maxPosts int, cb ProgressCallback) ([]Post, error) {
	// Step 1: Get profile + first 12 posts from web_profile_info
	result, err := c.GetProfileWithPosts(ctx, username)
	if err != nil {
		return nil, err
	}

	if result.Profile.IsPrivate {
		return nil, fmt.Errorf("@%s is a private account", username)
	}

	allPosts := result.Posts
	total := result.Profile.PostCount
	if maxPosts > 0 && int64(maxPosts) < total {
		total = int64(maxPosts)
	}

	if cb != nil {
		cb(Progress{Phase: "posts", Total: total, Current: int64(len(allPosts))})
	}

	// If we have enough or no more available, return
	if (maxPosts > 0 && len(allPosts) >= maxPosts) || !result.HasMore {
		if maxPosts > 0 && len(allPosts) > maxPosts {
			allPosts = allPosts[:maxPosts]
		}
		if cb != nil {
			cb(Progress{Phase: "posts", Total: int64(len(allPosts)), Current: int64(len(allPosts)), Done: true})
		}
		return allPosts, nil
	}

	// Step 2: Try feed API for more posts (may fail with auth requirement)
	if err := c.delay(ctx); err != nil {
		return allPosts, err
	}

	feedPosts, feedErr := c.fetchFeedPage(ctx, result.Profile.ID)
	if feedErr == nil && len(feedPosts) > 0 {
		// Merge, avoiding duplicates
		seen := make(map[string]bool, len(allPosts))
		for _, p := range allPosts {
			seen[p.ID] = true
		}
		for _, p := range feedPosts {
			if !seen[p.ID] {
				allPosts = append(allPosts, p)
				seen[p.ID] = true
			}
		}
	}

	if maxPosts > 0 && len(allPosts) > maxPosts {
		allPosts = allPosts[:maxPosts]
	}

	if cb != nil {
		cb(Progress{Phase: "posts", Total: int64(len(allPosts)), Current: int64(len(allPosts)), Done: true})
	}

	return allPosts, nil
}

// fetchFeedPage fetches one page of posts from the feed/user/ endpoint.
func (c *Client) fetchFeedPage(ctx context.Context, userID string) ([]Post, error) {
	rawURL := fmt.Sprintf("%s%s/?count=12", FeedUserURL, userID)

	data, err := c.doGet(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch feed: %w", err)
	}

	var resp feedResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("feed API: %s", resp.Message)
	}

	var posts []Post
	for _, item := range resp.Items {
		post := feedItemToPost(item)
		posts = append(posts, post)
	}

	return posts, nil
}

// GetPost fetches a single post by shortcode.
// Without authentication, this only works if the post data was already fetched via profile/feed.
func (c *Client) GetPost(ctx context.Context, shortcode string) (*Post, error) {
	// Try the __a=1 endpoint (may not work without auth)
	rawURL := fmt.Sprintf("https://www.instagram.com/p/%s/?__a=1&__d=dis", shortcode)
	data, err := c.doGet(ctx, rawURL)
	if err == nil {
		var resp postDetailResponse
		if err := json.Unmarshal(data, &resp); err == nil {
			if resp.GraphQL.ShortcodeMedia.ID != "" {
				post := nodeToPost(resp.GraphQL.ShortcodeMedia)
				return &post, nil
			}
			if len(resp.Items) > 0 {
				post := detailItemToPost(resp.Items[0])
				return &post, nil
			}
		}
	}

	return nil, fmt.Errorf("post %q: endpoint requires authentication", shortcode)
}

// feedItemToPost converts a feed API item to a Post.
func feedItemToPost(item feedItem) Post {
	post := Post{
		ID:           item.ID,
		Shortcode:    item.Code,
		LikeCount:    item.LikeCount,
		CommentCount: item.CommentCount,
		ViewCount:    item.ViewCount,
		TakenAt:      time.Unix(item.TakenAt, 0),
		OwnerID:      fmt.Sprintf("%d", item.User.PK),
		OwnerName:    item.User.Username,
		FetchedAt:    time.Now(),
	}

	if item.PlayCount > post.ViewCount {
		post.ViewCount = item.PlayCount
	}
	if item.Caption != nil {
		post.Caption = item.Caption.Text
	}
	if item.Location != nil {
		post.LocationID = fmt.Sprintf("%d", item.Location.PK)
		post.LocationName = item.Location.Name
	}

	switch item.MediaType {
	case 1:
		post.TypeName = "GraphImage"
		if item.ImageVersions2 != nil && len(item.ImageVersions2.Candidates) > 0 {
			best := item.ImageVersions2.Candidates[0]
			post.DisplayURL = best.URL
			post.Width = best.Width
			post.Height = best.Height
		}
	case 2:
		post.TypeName = "GraphVideo"
		post.IsVideo = true
		if len(item.VideoVersions) > 0 {
			post.VideoURL = item.VideoVersions[0].URL
			post.Width = item.VideoVersions[0].Width
			post.Height = item.VideoVersions[0].Height
		}
		if item.ImageVersions2 != nil && len(item.ImageVersions2.Candidates) > 0 {
			post.DisplayURL = item.ImageVersions2.Candidates[0].URL
		}
	case 8:
		post.TypeName = "GraphSidecar"
		for _, cm := range item.CarouselMedia {
			child := Post{ID: cm.ID, FetchedAt: time.Now()}
			if cm.MediaType == 2 && len(cm.VideoVersions) > 0 {
				child.TypeName = "GraphVideo"
				child.IsVideo = true
				child.VideoURL = cm.VideoVersions[0].URL
				child.Width = cm.VideoVersions[0].Width
				child.Height = cm.VideoVersions[0].Height
			}
			if cm.ImageVersions2 != nil && len(cm.ImageVersions2.Candidates) > 0 {
				child.DisplayURL = cm.ImageVersions2.Candidates[0].URL
				if !child.IsVideo {
					child.TypeName = "GraphImage"
					child.Width = cm.ImageVersions2.Candidates[0].Width
					child.Height = cm.ImageVersions2.Candidates[0].Height
				}
			}
			post.Children = append(post.Children, child)
		}
		if item.ImageVersions2 != nil && len(item.ImageVersions2.Candidates) > 0 {
			post.DisplayURL = item.ImageVersions2.Candidates[0].URL
			post.Width = item.ImageVersions2.Candidates[0].Width
			post.Height = item.ImageVersions2.Candidates[0].Height
		}
	}

	return post
}

// detailItemToPost converts a postDetailResponse item to a Post.
func detailItemToPost(item postDetailItem) Post {
	fi := feedItem{
		ID:        item.ID,
		Code:      item.Code,
		MediaType: item.MediaType,
		LikeCount: item.LikeCount,
		CommentCount: item.CommentCount,
		ViewCount: item.ViewCount,
		TakenAt:   item.TakenAt,
		Caption:   item.Caption,
		User:      item.User,
		Location:  item.Location,
	}

	if item.ImageVersions2 != nil {
		iv := &imageVersions{}
		for _, c := range item.ImageVersions2.Candidates {
			iv.Candidates = append(iv.Candidates, imageCandidate{URL: c.URL, Width: c.Width, Height: c.Height})
		}
		fi.ImageVersions2 = iv
	}
	for _, v := range item.VideoVersions {
		fi.VideoVersions = append(fi.VideoVersions, videoVersion{URL: v.URL, Width: v.Width, Height: v.Height})
	}
	for _, cm := range item.CarouselMedia {
		ci := carouselItem{ID: cm.ID, MediaType: cm.MediaType}
		if cm.ImageVersions2 != nil {
			iv := &imageVersions{}
			for _, c := range cm.ImageVersions2.Candidates {
				iv.Candidates = append(iv.Candidates, imageCandidate{URL: c.URL, Width: c.Width, Height: c.Height})
			}
			ci.ImageVersions2 = iv
		}
		for _, v := range cm.VideoVersions {
			ci.VideoVersions = append(ci.VideoVersions, videoVersion{URL: v.URL, Width: v.Width, Height: v.Height})
		}
		fi.CarouselMedia = append(fi.CarouselMedia, ci)
	}

	return feedItemToPost(fi)
}

// nodeToPost converts a GraphQL media node to a Post.
func nodeToPost(n mediaNode) Post {
	post := Post{
		ID:           n.ID,
		Shortcode:    n.Shortcode,
		TypeName:     n.TypeName,
		DisplayURL:   n.DisplayURL,
		VideoURL:     n.VideoURL,
		IsVideo:      n.IsVideo,
		Width:        n.Dimensions.Width,
		Height:       n.Dimensions.Height,
		LikeCount:    n.EdgeMediaPreviewLike.Count,
		CommentCount: n.EdgeMediaToComment.Count,
		ViewCount:    n.VideoViewCount,
		TakenAt:      time.Unix(n.TakenAtTimestamp, 0),
		OwnerID:      n.Owner.ID,
		OwnerName:    n.Owner.Username,
		FetchedAt:    time.Now(),
	}

	if n.EdgeLikedBy.Count > post.LikeCount {
		post.LikeCount = n.EdgeLikedBy.Count
	}
	if len(n.EdgeMediaToCaption.Edges) > 0 {
		post.Caption = n.EdgeMediaToCaption.Edges[0].Node.Text
	}
	if n.Location != nil {
		post.LocationID = n.Location.ID
		post.LocationName = n.Location.Name
	}
	if n.EdgeSidecarToChildren != nil {
		for _, child := range n.EdgeSidecarToChildren.Edges {
			post.Children = append(post.Children, nodeToPost(child.Node))
		}
	}

	return post
}

// CollectMediaItems extracts all downloadable media URLs from posts.
func CollectMediaItems(posts []Post) []MediaItem {
	var items []MediaItem

	for _, p := range posts {
		if len(p.Children) > 0 {
			for i, child := range p.Children {
				item := MediaItem{
					PostID:    p.ID,
					Shortcode: p.Shortcode,
					Width:     child.Width,
					Height:    child.Height,
					Index:     i,
				}
				if child.IsVideo && child.VideoURL != "" {
					item.URL = child.VideoURL
					item.Type = "video"
				} else if child.DisplayURL != "" {
					item.URL = child.DisplayURL
					item.Type = "image"
				}
				if item.URL != "" {
					items = append(items, item)
				}
			}
		} else {
			item := MediaItem{
				PostID:    p.ID,
				Shortcode: p.Shortcode,
				Width:     p.Width,
				Height:    p.Height,
				Index:     0,
			}
			if p.IsVideo && p.VideoURL != "" {
				item.URL = p.VideoURL
				item.Type = "video"
			} else if p.DisplayURL != "" {
				item.URL = p.DisplayURL
				item.Type = "image"
			}
			if item.URL != "" {
				items = append(items, item)
			}
		}
	}

	return items
}
