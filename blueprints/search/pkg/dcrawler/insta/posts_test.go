package insta

import (
	"testing"
	"time"
)

func TestNodeToPost(t *testing.T) {
	node := mediaNode{
		ID:        "12345",
		Shortcode: "AbCdEf",
		TypeName:  "GraphImage",
		DisplayURL: "https://example.com/img.jpg",
		IsVideo:   false,
		Dimensions: dimensions{Width: 1080, Height: 1080},
		EdgeMediaPreviewLike: countField{Count: 500},
		EdgeMediaToComment:   countField{Count: 20},
		TakenAtTimestamp:     1700000000,
		Owner: ownerNode{ID: "999", Username: "testuser"},
		EdgeMediaToCaption: captionEdge{
			Edges: []captionNode{
				{Node: struct {
					Text string `json:"text"`
				}{Text: "Hello world"}},
			},
		},
	}

	post := nodeToPost(node)

	if post.ID != "12345" {
		t.Errorf("ID = %q, want %q", post.ID, "12345")
	}
	if post.Shortcode != "AbCdEf" {
		t.Errorf("Shortcode = %q, want %q", post.Shortcode, "AbCdEf")
	}
	if post.TypeName != "GraphImage" {
		t.Errorf("TypeName = %q, want %q", post.TypeName, "GraphImage")
	}
	if post.DisplayURL != "https://example.com/img.jpg" {
		t.Errorf("DisplayURL = %q", post.DisplayURL)
	}
	if post.IsVideo {
		t.Error("IsVideo should be false")
	}
	if post.Width != 1080 || post.Height != 1080 {
		t.Errorf("Dimensions = %dx%d, want 1080x1080", post.Width, post.Height)
	}
	if post.LikeCount != 500 {
		t.Errorf("LikeCount = %d, want 500", post.LikeCount)
	}
	if post.CommentCount != 20 {
		t.Errorf("CommentCount = %d, want 20", post.CommentCount)
	}
	if post.Caption != "Hello world" {
		t.Errorf("Caption = %q, want %q", post.Caption, "Hello world")
	}
	if post.OwnerID != "999" {
		t.Errorf("OwnerID = %q, want %q", post.OwnerID, "999")
	}
	if post.OwnerName != "testuser" {
		t.Errorf("OwnerName = %q, want %q", post.OwnerName, "testuser")
	}
}

func TestNodeToPost_Video(t *testing.T) {
	node := mediaNode{
		ID:        "67890",
		Shortcode: "VidTest",
		TypeName:  "GraphVideo",
		DisplayURL: "https://example.com/thumb.jpg",
		VideoURL:  "https://example.com/video.mp4",
		IsVideo:   true,
		Dimensions:     dimensions{Width: 1920, Height: 1080},
		VideoViewCount: 10000,
	}

	post := nodeToPost(node)

	if !post.IsVideo {
		t.Error("IsVideo should be true")
	}
	if post.VideoURL != "https://example.com/video.mp4" {
		t.Errorf("VideoURL = %q", post.VideoURL)
	}
	if post.ViewCount != 10000 {
		t.Errorf("ViewCount = %d, want 10000", post.ViewCount)
	}
}

func TestNodeToPost_Carousel(t *testing.T) {
	node := mediaNode{
		ID:        "carousel1",
		Shortcode: "CarTest",
		TypeName:  "GraphSidecar",
		DisplayURL: "https://example.com/cover.jpg",
		Dimensions: dimensions{Width: 1080, Height: 1080},
		EdgeSidecarToChildren: &mediaConnection{
			Edges: []edge{
				{Node: mediaNode{ID: "child1", TypeName: "GraphImage", DisplayURL: "https://example.com/1.jpg", Dimensions: dimensions{Width: 1080, Height: 1080}}},
				{Node: mediaNode{ID: "child2", TypeName: "GraphVideo", DisplayURL: "https://example.com/2.jpg", VideoURL: "https://example.com/2.mp4", IsVideo: true, Dimensions: dimensions{Width: 1080, Height: 1920}}},
			},
		},
	}

	post := nodeToPost(node)

	if len(post.Children) != 2 {
		t.Fatalf("Children = %d, want 2", len(post.Children))
	}
	if post.Children[0].TypeName != "GraphImage" {
		t.Errorf("Child 0 type = %q, want GraphImage", post.Children[0].TypeName)
	}
	if post.Children[1].IsVideo != true {
		t.Error("Child 1 should be video")
	}
	if post.Children[1].VideoURL != "https://example.com/2.mp4" {
		t.Errorf("Child 1 VideoURL = %q", post.Children[1].VideoURL)
	}
}

func TestNodeToPost_Location(t *testing.T) {
	node := mediaNode{
		ID:        "loc1",
		Shortcode: "LocTest",
		TypeName:  "GraphImage",
		DisplayURL: "https://example.com/img.jpg",
		Location:  &locationNode{ID: "123456", Name: "Central Park"},
	}

	post := nodeToPost(node)

	if post.LocationID != "123456" {
		t.Errorf("LocationID = %q, want %q", post.LocationID, "123456")
	}
	if post.LocationName != "Central Park" {
		t.Errorf("LocationName = %q, want %q", post.LocationName, "Central Park")
	}
}

func TestNodeToPost_EdgeLikedByPreferred(t *testing.T) {
	node := mediaNode{
		ID:                   "like1",
		Shortcode:            "LikeTest",
		TypeName:             "GraphImage",
		DisplayURL:           "https://example.com/img.jpg",
		EdgeLikedBy:          countField{Count: 1000},
		EdgeMediaPreviewLike: countField{Count: 500},
	}

	post := nodeToPost(node)

	// EdgeLikedBy (1000) should win over EdgeMediaPreviewLike (500)
	if post.LikeCount != 1000 {
		t.Errorf("LikeCount = %d, want 1000 (edge_liked_by)", post.LikeCount)
	}
}

func TestFeedItemToPost_Image(t *testing.T) {
	item := feedItem{
		ID:        "feed1",
		Code:      "FeedImg",
		MediaType: 1,
		LikeCount: 100,
		CommentCount: 10,
		TakenAt:   1700000000,
		Caption:   &struct{ Text string `json:"text"` }{Text: "Feed caption"},
		ImageVersions2: &imageVersions{
			Candidates: []imageCandidate{
				{URL: "https://example.com/best.jpg", Width: 1080, Height: 1350},
			},
		},
		User: struct {
			PK       int64  `json:"pk"`
			Username string `json:"username"`
		}{PK: 42, Username: "feeduser"},
	}

	post := feedItemToPost(item)

	if post.Shortcode != "FeedImg" {
		t.Errorf("Shortcode = %q, want %q", post.Shortcode, "FeedImg")
	}
	if post.TypeName != "GraphImage" {
		t.Errorf("TypeName = %q, want %q", post.TypeName, "GraphImage")
	}
	if post.DisplayURL != "https://example.com/best.jpg" {
		t.Errorf("DisplayURL = %q", post.DisplayURL)
	}
	if post.Width != 1080 || post.Height != 1350 {
		t.Errorf("Dimensions = %dx%d, want 1080x1350", post.Width, post.Height)
	}
	if post.Caption != "Feed caption" {
		t.Errorf("Caption = %q", post.Caption)
	}
}

func TestFeedItemToPost_Video(t *testing.T) {
	item := feedItem{
		ID:        "feed2",
		Code:      "FeedVid",
		MediaType: 2,
		ViewCount: 5000,
		PlayCount: 8000,
		VideoVersions: []videoVersion{
			{URL: "https://example.com/vid.mp4", Width: 1080, Height: 1920},
		},
		ImageVersions2: &imageVersions{
			Candidates: []imageCandidate{
				{URL: "https://example.com/thumb.jpg", Width: 1080, Height: 1920},
			},
		},
	}

	post := feedItemToPost(item)

	if post.TypeName != "GraphVideo" {
		t.Errorf("TypeName = %q, want %q", post.TypeName, "GraphVideo")
	}
	if !post.IsVideo {
		t.Error("IsVideo should be true")
	}
	// PlayCount (8000) > ViewCount (5000), so ViewCount should be 8000
	if post.ViewCount != 8000 {
		t.Errorf("ViewCount = %d, want 8000 (play_count)", post.ViewCount)
	}
	if post.VideoURL != "https://example.com/vid.mp4" {
		t.Errorf("VideoURL = %q", post.VideoURL)
	}
	if post.DisplayURL != "https://example.com/thumb.jpg" {
		t.Errorf("DisplayURL = %q", post.DisplayURL)
	}
}

func TestFeedItemToPost_Carousel(t *testing.T) {
	item := feedItem{
		ID:        "feed3",
		Code:      "FeedCar",
		MediaType: 8,
		CarouselMedia: []carouselItem{
			{
				ID: "c1", MediaType: 1,
				ImageVersions2: &imageVersions{Candidates: []imageCandidate{{URL: "https://example.com/1.jpg", Width: 1080, Height: 1080}}},
			},
			{
				ID: "c2", MediaType: 2,
				VideoVersions:  []videoVersion{{URL: "https://example.com/2.mp4", Width: 1080, Height: 1920}},
				ImageVersions2: &imageVersions{Candidates: []imageCandidate{{URL: "https://example.com/2.jpg", Width: 1080, Height: 1920}}},
			},
		},
		ImageVersions2: &imageVersions{Candidates: []imageCandidate{{URL: "https://example.com/cover.jpg", Width: 1080, Height: 1080}}},
	}

	post := feedItemToPost(item)

	if post.TypeName != "GraphSidecar" {
		t.Errorf("TypeName = %q, want %q", post.TypeName, "GraphSidecar")
	}
	if len(post.Children) != 2 {
		t.Fatalf("Children = %d, want 2", len(post.Children))
	}
	if post.Children[0].TypeName != "GraphImage" {
		t.Errorf("Child 0 type = %q, want GraphImage", post.Children[0].TypeName)
	}
	if !post.Children[1].IsVideo {
		t.Error("Child 1 should be video")
	}
	if post.Children[1].VideoURL != "https://example.com/2.mp4" {
		t.Errorf("Child 1 VideoURL = %q", post.Children[1].VideoURL)
	}
}

func TestCollectMediaItems_SingleImage(t *testing.T) {
	posts := []Post{
		{ID: "1", Shortcode: "ABC", DisplayURL: "https://example.com/1.jpg", Width: 1080, Height: 1080},
	}

	items := CollectMediaItems(posts)

	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].URL != "https://example.com/1.jpg" {
		t.Errorf("URL = %q", items[0].URL)
	}
	if items[0].Type != "image" {
		t.Errorf("Type = %q, want image", items[0].Type)
	}
	if items[0].Shortcode != "ABC" {
		t.Errorf("Shortcode = %q", items[0].Shortcode)
	}
	if items[0].Index != 0 {
		t.Errorf("Index = %d, want 0", items[0].Index)
	}
}

func TestCollectMediaItems_Video(t *testing.T) {
	posts := []Post{
		{ID: "2", Shortcode: "VID", IsVideo: true, VideoURL: "https://example.com/v.mp4", DisplayURL: "https://example.com/thumb.jpg"},
	}

	items := CollectMediaItems(posts)

	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].URL != "https://example.com/v.mp4" {
		t.Errorf("URL = %q, want video URL", items[0].URL)
	}
	if items[0].Type != "video" {
		t.Errorf("Type = %q, want video", items[0].Type)
	}
}

func TestCollectMediaItems_Carousel(t *testing.T) {
	posts := []Post{
		{
			ID: "3", Shortcode: "CAR",
			Children: []Post{
				{ID: "c1", DisplayURL: "https://example.com/1.jpg", Width: 1080, Height: 1080},
				{ID: "c2", DisplayURL: "https://example.com/2.jpg", Width: 1080, Height: 1080},
				{ID: "c3", IsVideo: true, VideoURL: "https://example.com/3.mp4", Width: 1080, Height: 1920},
			},
		},
	}

	items := CollectMediaItems(posts)

	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	// Carousel items use parent's shortcode
	for _, item := range items {
		if item.Shortcode != "CAR" {
			t.Errorf("Shortcode = %q, want CAR", item.Shortcode)
		}
		if item.PostID != "3" {
			t.Errorf("PostID = %q, want 3", item.PostID)
		}
	}
	if items[0].Index != 0 {
		t.Errorf("items[0].Index = %d, want 0", items[0].Index)
	}
	if items[1].Index != 1 {
		t.Errorf("items[1].Index = %d, want 1", items[1].Index)
	}
	if items[2].Index != 2 {
		t.Errorf("items[2].Index = %d, want 2", items[2].Index)
	}
	if items[2].Type != "video" {
		t.Errorf("items[2].Type = %q, want video", items[2].Type)
	}
}

func TestCollectMediaItems_Empty(t *testing.T) {
	items := CollectMediaItems(nil)
	if items != nil {
		t.Errorf("items = %v, want nil", items)
	}
}

func TestCollectMediaItems_NoURL(t *testing.T) {
	posts := []Post{
		{ID: "nourl", Shortcode: "NOURL"}, // no DisplayURL or VideoURL
	}
	items := CollectMediaItems(posts)
	if len(items) != 0 {
		t.Errorf("items = %d, want 0 (no URL)", len(items))
	}
}

func TestMediaFilename(t *testing.T) {
	tests := []struct {
		name string
		item MediaItem
		want string
	}{
		{
			name: "simple image",
			item: MediaItem{Shortcode: "ABC", Type: "image", URL: "https://example.com/img.jpg"},
			want: "ABC.jpg",
		},
		{
			name: "video",
			item: MediaItem{Shortcode: "VID", Type: "video", URL: "https://example.com/vid.mp4"},
			want: "VID.mp4",
		},
		{
			name: "carousel index 1",
			item: MediaItem{Shortcode: "CAR", Type: "image", URL: "https://example.com/img.jpg", Index: 1},
			want: "CAR_1.jpg",
		},
		{
			name: "carousel index 0 (no suffix)",
			item: MediaItem{Shortcode: "CAR", Type: "image", URL: "https://example.com/img.jpg", Index: 0},
			want: "CAR.jpg",
		},
		{
			name: "png extension from URL",
			item: MediaItem{Shortcode: "PNG", Type: "image", URL: "https://example.com/img.png?quality=90"},
			want: "PNG.png",
		},
		{
			name: "webp extension from URL",
			item: MediaItem{Shortcode: "WBP", Type: "image", URL: "https://example.com/img.webp"},
			want: "WBP.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mediaFilename(tt.item)
			if got != tt.want {
				t.Errorf("mediaFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseDocIDPostsResponse_Empty(t *testing.T) {
	posts, cursor, hasMore := parseDocIDPostsResponse([]byte(`{}`))
	if len(posts) != 0 {
		t.Errorf("posts = %d, want 0", len(posts))
	}
	if cursor != "" {
		t.Errorf("cursor = %q, want empty", cursor)
	}
	if hasMore {
		t.Error("hasMore should be false")
	}
}

func TestParseDocIDPostsResponse_WithPosts(t *testing.T) {
	data := []byte(`{
		"data": {
			"user": {
				"edge_owner_to_timeline_media": {
					"count": 100,
					"page_info": {"has_next_page": true, "end_cursor": "QVFAbC=="},
					"edges": [
						{"node": {"id": "1", "shortcode": "AAA", "__typename": "GraphImage", "display_url": "https://example.com/1.jpg", "dimensions": {"width": 1080, "height": 1080}, "taken_at_timestamp": 1700000000}},
						{"node": {"id": "2", "shortcode": "BBB", "__typename": "GraphVideo", "display_url": "https://example.com/2.jpg", "is_video": true, "dimensions": {"width": 1080, "height": 1920}, "taken_at_timestamp": 1700000100}}
					]
				}
			}
		}
	}`)

	posts, cursor, hasMore := parseDocIDPostsResponse(data)

	if len(posts) != 2 {
		t.Fatalf("posts = %d, want 2", len(posts))
	}
	if cursor != "QVFAbC==" {
		t.Errorf("cursor = %q, want QVFAbC==", cursor)
	}
	if !hasMore {
		t.Error("hasMore should be true")
	}
	if posts[0].Shortcode != "AAA" {
		t.Errorf("posts[0].Shortcode = %q, want AAA", posts[0].Shortcode)
	}
	if posts[1].IsVideo != true {
		t.Error("posts[1].IsVideo should be true")
	}
}

func TestParseDocIDPostsResponse_Invalid(t *testing.T) {
	posts, _, _ := parseDocIDPostsResponse([]byte(`not json`))
	if len(posts) != 0 {
		t.Errorf("posts = %d, want 0 for invalid JSON", len(posts))
	}
}

func TestParseDocIDPostsResponse_XDTFormat(t *testing.T) {
	data := []byte(`{
		"data": {
			"xdt_api__v1__feed__user_timeline_graphql_connection": {
				"edges": [
					{"cursor": "", "node": {
						"id": "123_456", "pk": "123", "code": "ABC123",
						"media_type": 1, "like_count": 500, "comment_count": 10,
						"taken_at": 1700000000, "original_width": 1080, "original_height": 1080,
						"caption": {"text": "Hello world"},
						"image_versions2": {"candidates": [{"url": "https://example.com/1.jpg", "width": 1080, "height": 1080}]},
						"user": {"pk": "456", "username": "testuser"},
						"location": {"pk": "789", "name": "NYC"}
					}},
					{"cursor": "", "node": {
						"id": "200_456", "pk": "200", "code": "VID456",
						"media_type": 2, "like_count": 1000, "comment_count": 50,
						"taken_at": 1700000100, "original_width": 1080, "original_height": 1920,
						"video_versions": [{"url": "https://example.com/v.mp4", "width": 1080, "height": 1920}],
						"image_versions2": {"candidates": [{"url": "https://example.com/thumb.jpg", "width": 1080, "height": 1920}]},
						"user": {"pk": "456", "username": "testuser"}
					}}
				],
				"page_info": {"has_next_page": true, "end_cursor": "200_456"}
			}
		}
	}`)

	posts, cursor, hasMore := parseDocIDPostsResponse(data)

	if len(posts) != 2 {
		t.Fatalf("posts = %d, want 2", len(posts))
	}
	if cursor != "200_456" {
		t.Errorf("cursor = %q, want 200_456", cursor)
	}
	if !hasMore {
		t.Error("hasMore should be true")
	}

	// First post: image
	p := posts[0]
	if p.Shortcode != "ABC123" {
		t.Errorf("Shortcode = %q, want ABC123", p.Shortcode)
	}
	if p.TypeName != "GraphImage" {
		t.Errorf("TypeName = %q, want GraphImage", p.TypeName)
	}
	if p.LikeCount != 500 {
		t.Errorf("LikeCount = %d, want 500", p.LikeCount)
	}
	if p.Caption != "Hello world" {
		t.Errorf("Caption = %q, want Hello world", p.Caption)
	}
	if p.OwnerName != "testuser" {
		t.Errorf("OwnerName = %q, want testuser", p.OwnerName)
	}
	if p.OwnerID != "456" {
		t.Errorf("OwnerID = %q, want 456", p.OwnerID)
	}
	if p.LocationName != "NYC" {
		t.Errorf("LocationName = %q, want NYC", p.LocationName)
	}
	if p.DisplayURL != "https://example.com/1.jpg" {
		t.Errorf("DisplayURL = %q", p.DisplayURL)
	}
	if p.Width != 1080 {
		t.Errorf("Width = %d, want 1080", p.Width)
	}

	// Second post: video
	v := posts[1]
	if v.Shortcode != "VID456" {
		t.Errorf("Shortcode = %q, want VID456", v.Shortcode)
	}
	if v.TypeName != "GraphVideo" {
		t.Errorf("TypeName = %q, want GraphVideo", v.TypeName)
	}
	if !v.IsVideo {
		t.Error("IsVideo should be true")
	}
	if v.VideoURL != "https://example.com/v.mp4" {
		t.Errorf("VideoURL = %q", v.VideoURL)
	}
	if v.DisplayURL != "https://example.com/thumb.jpg" {
		t.Errorf("DisplayURL = %q (should be thumbnail)", v.DisplayURL)
	}
}

func TestParseDocIDPostsResponse_XDTCarousel(t *testing.T) {
	data := []byte(`{
		"data": {
			"xdt_api__v1__feed__user_timeline_graphql_connection": {
				"edges": [
					{"cursor": "", "node": {
						"id": "300_456", "pk": "300", "code": "CAR789",
						"media_type": 8, "like_count": 2000, "comment_count": 100,
						"taken_at": 1700000200, "original_width": 1080, "original_height": 1080,
						"image_versions2": {"candidates": [{"url": "https://example.com/cover.jpg", "width": 1080, "height": 1080}]},
						"carousel_media": [
							{"id": "c1", "media_type": 1, "original_width": 1080, "original_height": 1080, "image_versions2": {"candidates": [{"url": "https://example.com/c1.jpg", "width": 1080, "height": 1080}]}},
							{"id": "c2", "media_type": 2, "original_width": 1080, "original_height": 1920, "image_versions2": {"candidates": [{"url": "https://example.com/c2_thumb.jpg", "width": 1080, "height": 1920}]}, "video_versions": [{"url": "https://example.com/c2.mp4", "width": 1080, "height": 1920}]}
						],
						"user": {"pk": "456", "username": "testuser"}
					}}
				],
				"page_info": {"has_next_page": false, "end_cursor": ""}
			}
		}
	}`)

	posts, _, hasMore := parseDocIDPostsResponse(data)

	if len(posts) != 1 {
		t.Fatalf("posts = %d, want 1", len(posts))
	}
	if hasMore {
		t.Error("hasMore should be false")
	}

	p := posts[0]
	if p.TypeName != "GraphSidecar" {
		t.Errorf("TypeName = %q, want GraphSidecar", p.TypeName)
	}
	if len(p.Children) != 2 {
		t.Fatalf("Children = %d, want 2", len(p.Children))
	}
	if p.Children[0].TypeName != "GraphImage" {
		t.Errorf("Child 0 TypeName = %q, want GraphImage", p.Children[0].TypeName)
	}
	if p.Children[1].TypeName != "GraphVideo" {
		t.Errorf("Child 1 TypeName = %q, want GraphVideo", p.Children[1].TypeName)
	}
	if !p.Children[1].IsVideo {
		t.Error("Child 1 should be video")
	}
	if p.Children[1].VideoURL != "https://example.com/c2.mp4" {
		t.Errorf("Child 1 VideoURL = %q", p.Children[1].VideoURL)
	}
}

// Ensure time conversion works correctly
func TestNodeToPost_TimestampConversion(t *testing.T) {
	ts := int64(1700000000) // 2023-11-14 22:13:20 UTC
	node := mediaNode{
		ID:               "ts1",
		Shortcode:        "TSTest",
		TypeName:         "GraphImage",
		DisplayURL:       "https://example.com/img.jpg",
		TakenAtTimestamp:  ts,
	}

	post := nodeToPost(node)
	expected := time.Unix(ts, 0)

	if !post.TakenAt.Equal(expected) {
		t.Errorf("TakenAt = %v, want %v", post.TakenAt, expected)
	}
}
