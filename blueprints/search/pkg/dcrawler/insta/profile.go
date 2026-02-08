package insta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// ProfileWithPosts holds profile data plus the initial batch of posts from the profile response.
type ProfileWithPosts struct {
	Profile  *Profile
	Posts    []Post
	Cursor   string // pagination cursor for next page
	HasMore  bool
}

// GetProfileWithPosts fetches a user's profile and the first page of posts (up to 12).
// This is the most reliable unauthenticated endpoint.
func (c *Client) GetProfileWithPosts(ctx context.Context, username string) (*ProfileWithPosts, error) {
	params := url.Values{}
	params.Set("username", username)
	rawURL := WebProfileURL + "?" + params.Encode()

	data, err := c.doGet(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch profile %q: %w", username, err)
	}

	var resp profileAPIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse profile response: %w", err)
	}

	u := resp.Data.User
	if u.Username == "" {
		return nil, fmt.Errorf("user %q not found", username)
	}

	picURL := u.ProfilePicURLHD
	if picURL == "" {
		picURL = u.ProfilePicURL
	}

	profile := &Profile{
		ID:             u.ID,
		Username:       u.Username,
		FullName:       u.FullName,
		Biography:      u.Biography,
		ProfilePicURL:  picURL,
		ExternalURL:    u.ExternalURL,
		IsPrivate:      u.IsPrivate,
		IsVerified:     u.IsVerified,
		IsBusiness:     u.IsBusinessAccount || u.IsProfessionalAccount,
		CategoryName:   u.CategoryName,
		FollowerCount:  u.EdgeFollowedBy.Count,
		FollowingCount: u.EdgeFollow.Count,
		PostCount:      0,
		FetchedAt:      time.Now(),
	}

	result := &ProfileWithPosts{Profile: profile}

	if u.EdgeOwnerToTimelineMedia != nil {
		profile.PostCount = u.EdgeOwnerToTimelineMedia.Count

		// Extract initial posts
		for _, e := range u.EdgeOwnerToTimelineMedia.Edges {
			post := nodeToPost(e.Node)
			result.Posts = append(result.Posts, post)
		}

		result.Cursor = u.EdgeOwnerToTimelineMedia.PageInfo.EndCursor
		result.HasMore = u.EdgeOwnerToTimelineMedia.PageInfo.HasNextPage
	}

	return result, nil
}

// GetProfile fetches only profile information (no posts).
func (c *Client) GetProfile(ctx context.Context, username string) (*Profile, error) {
	result, err := c.GetProfileWithPosts(ctx, username)
	if err != nil {
		return nil, err
	}
	return result.Profile, nil
}

// SaveProfile saves a profile to a JSON file.
func (c *Client) SaveProfile(profile *Profile) error {
	dir := c.cfg.UserDir(profile.Username)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "profile.json"), data, 0o644)
}

// LoadProfile loads a previously saved profile from disk.
func LoadProfile(cfg Config, username string) (*Profile, error) {
	data, err := os.ReadFile(cfg.ProfilePath(username))
	if err != nil {
		return nil, err
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}
