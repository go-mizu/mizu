package web

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
	"github.com/go-mizu/blueprints/microblog/feature/search"
)

// Auth handlers

func (s *Server) handleRegister(c *mizu.Ctx) error {
	var in accounts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, errorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	account, err := s.accounts.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(400, errorResponse("REGISTRATION_FAILED", err.Error()))
	}

	session, err := s.accounts.CreateSession(c.Request().Context(), account.ID)
	if err != nil {
		return c.JSON(500, errorResponse("SESSION_FAILED", "Failed to create session"))
	}

	return c.JSON(200, map[string]any{
		"data": map[string]any{
			"account": account,
			"token":   session.Token,
		},
	})
}

func (s *Server) handleLogin(c *mizu.Ctx) error {
	var in accounts.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, errorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	session, err := s.accounts.Login(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(401, errorResponse("LOGIN_FAILED", err.Error()))
	}

	account, _ := s.accounts.GetByID(c.Request().Context(), session.AccountID)

	return c.JSON(200, map[string]any{
		"data": map[string]any{
			"account": account,
			"token":   session.Token,
		},
	})
}

func (s *Server) handleLogout(c *mizu.Ctx) error {
	token := c.Request().Header.Get("Authorization")
	if len(token) > 7 {
		token = token[7:] // Remove "Bearer "
		_ = s.accounts.DeleteSession(c.Request().Context(), token)
	}
	return c.JSON(200, map[string]any{"data": map[string]any{"success": true}})
}

// Account handlers

func (s *Server) handleVerifyCredentials(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	account, err := s.accounts.GetByID(c.Request().Context(), accountID)
	if err != nil {
		return c.JSON(404, errorResponse("NOT_FOUND", "Account not found"))
	}
	return c.JSON(200, map[string]any{"data": account})
}

func (s *Server) handleUpdateCredentials(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	var in accounts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, errorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	account, err := s.accounts.Update(c.Request().Context(), accountID, &in)
	if err != nil {
		return c.JSON(400, errorResponse("UPDATE_FAILED", err.Error()))
	}
	return c.JSON(200, map[string]any{"data": account})
}

func (s *Server) handleGetAccount(c *mizu.Ctx) error {
	id := c.Param("id")

	// Check if it's a username
	var account *accounts.Account
	var err error

	if len(id) < 26 { // ULIDs are 26 chars
		account, err = s.accounts.GetByUsername(c.Request().Context(), id)
	} else {
		account, err = s.accounts.GetByID(c.Request().Context(), id)
	}

	if err != nil {
		return c.JSON(404, errorResponse("NOT_FOUND", "Account not found"))
	}

	// Load follower/following counts
	account.FollowersCount, _ = s.relationships.CountFollowers(c.Request().Context(), account.ID)
	account.FollowingCount, _ = s.relationships.CountFollowing(c.Request().Context(), account.ID)

	return c.JSON(200, map[string]any{"data": account})
}

func (s *Server) handleAccountPosts(c *mizu.Ctx) error {
	accountID := c.Param("id")
	viewerID := s.optionalAuth(c)
	limit := intQuery(c, "limit", 20)
	maxID := c.Query("max_id")

	postList, err := s.timelines.Account(c.Request().Context(), accountID, viewerID, limit, maxID, false, false)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}

func (s *Server) handleAccountFollowers(c *mizu.Ctx) error {
	accountID := c.Param("id")
	limit := intQuery(c, "limit", 40)
	offset := intQuery(c, "offset", 0)

	ids, err := s.relationships.GetFollowers(c.Request().Context(), accountID, limit, offset)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	// Load accounts
	var accts []*accounts.Account
	for _, id := range ids {
		if a, err := s.accounts.GetByID(c.Request().Context(), id); err == nil {
			accts = append(accts, a)
		}
	}

	return c.JSON(200, map[string]any{"data": accts})
}

func (s *Server) handleAccountFollowing(c *mizu.Ctx) error {
	accountID := c.Param("id")
	limit := intQuery(c, "limit", 40)
	offset := intQuery(c, "offset", 0)

	ids, err := s.relationships.GetFollowing(c.Request().Context(), accountID, limit, offset)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	var accts []*accounts.Account
	for _, id := range ids {
		if a, err := s.accounts.GetByID(c.Request().Context(), id); err == nil {
			accts = append(accts, a)
		}
	}

	return c.JSON(200, map[string]any{"data": accts})
}

// Relationship handlers

func (s *Server) handleFollow(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	targetID := c.Param("id")

	if err := s.relationships.Follow(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, errorResponse("FOLLOW_FAILED", err.Error()))
	}

	rel, _ := s.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

func (s *Server) handleUnfollow(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	targetID := c.Param("id")

	if err := s.relationships.Unfollow(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, errorResponse("UNFOLLOW_FAILED", err.Error()))
	}

	rel, _ := s.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

func (s *Server) handleBlock(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	targetID := c.Param("id")

	if err := s.relationships.Block(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, errorResponse("BLOCK_FAILED", err.Error()))
	}

	rel, _ := s.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

func (s *Server) handleUnblock(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	targetID := c.Param("id")

	if err := s.relationships.Unblock(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, errorResponse("UNBLOCK_FAILED", err.Error()))
	}

	rel, _ := s.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

func (s *Server) handleMute(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	targetID := c.Param("id")

	if err := s.relationships.Mute(c.Request().Context(), accountID, targetID, true, nil); err != nil {
		return c.JSON(400, errorResponse("MUTE_FAILED", err.Error()))
	}

	rel, _ := s.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

func (s *Server) handleUnmute(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	targetID := c.Param("id")

	if err := s.relationships.Unmute(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, errorResponse("UNMUTE_FAILED", err.Error()))
	}

	rel, _ := s.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

func (s *Server) handleRelationships(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	ids := c.Query("id[]")

	if ids == "" {
		return c.JSON(200, map[string]any{"data": []any{}})
	}

	// Simple implementation - just get one relationship
	rel, _ := s.relationships.Get(c.Request().Context(), accountID, ids)
	return c.JSON(200, map[string]any{"data": []any{rel}})
}

// Post handlers

func (s *Server) handleCreatePost(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	var in posts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, errorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	post, err := s.posts.Create(c.Request().Context(), accountID, &in)
	if err != nil {
		return c.JSON(400, errorResponse("CREATE_FAILED", err.Error()))
	}

	return c.JSON(201, map[string]any{"data": post})
}

func (s *Server) handleGetPost(c *mizu.Ctx) error {
	id := c.Param("id")
	viewerID := s.optionalAuth(c)

	post, err := s.posts.GetByID(c.Request().Context(), id, viewerID)
	if err != nil {
		return c.JSON(404, errorResponse("NOT_FOUND", "Post not found"))
	}

	return c.JSON(200, map[string]any{"data": post})
}

func (s *Server) handleUpdatePost(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := s.getAccountID(c)
	var in posts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, errorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	post, err := s.posts.Update(c.Request().Context(), id, accountID, &in)
	if err != nil {
		return c.JSON(400, errorResponse("UPDATE_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": post})
}

func (s *Server) handleDeletePost(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := s.getAccountID(c)

	if err := s.posts.Delete(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(400, errorResponse("DELETE_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": map[string]any{"success": true}})
}

func (s *Server) handlePostContext(c *mizu.Ctx) error {
	id := c.Param("id")
	viewerID := s.optionalAuth(c)

	ctx, err := s.posts.GetThread(c.Request().Context(), id, viewerID)
	if err != nil {
		return c.JSON(404, errorResponse("NOT_FOUND", "Post not found"))
	}

	return c.JSON(200, map[string]any{"data": ctx})
}

// Interaction handlers

func (s *Server) handleLike(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	postID := c.Param("id")

	if err := s.interactions.Like(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, errorResponse("LIKE_FAILED", err.Error()))
	}

	post, _ := s.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

func (s *Server) handleUnlike(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	postID := c.Param("id")

	if err := s.interactions.Unlike(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, errorResponse("UNLIKE_FAILED", err.Error()))
	}

	post, _ := s.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

func (s *Server) handleRepost(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	postID := c.Param("id")

	if err := s.interactions.Repost(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, errorResponse("REPOST_FAILED", err.Error()))
	}

	post, _ := s.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

func (s *Server) handleUnrepost(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	postID := c.Param("id")

	if err := s.interactions.Unrepost(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, errorResponse("UNREPOST_FAILED", err.Error()))
	}

	post, _ := s.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

func (s *Server) handleBookmark(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	postID := c.Param("id")

	if err := s.interactions.Bookmark(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, errorResponse("BOOKMARK_FAILED", err.Error()))
	}

	post, _ := s.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

func (s *Server) handleUnbookmark(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	postID := c.Param("id")

	if err := s.interactions.Unbookmark(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, errorResponse("UNBOOKMARK_FAILED", err.Error()))
	}

	post, _ := s.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

func (s *Server) handleLikedBy(c *mizu.Ctx) error {
	postID := c.Param("id")
	limit := intQuery(c, "limit", 40)

	ids, err := s.interactions.GetLikedBy(c.Request().Context(), postID, limit, 0)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	var accts []*accounts.Account
	for _, id := range ids {
		if a, err := s.accounts.GetByID(c.Request().Context(), id); err == nil {
			accts = append(accts, a)
		}
	}

	return c.JSON(200, map[string]any{"data": accts})
}

func (s *Server) handleRepostedBy(c *mizu.Ctx) error {
	postID := c.Param("id")
	limit := intQuery(c, "limit", 40)

	ids, err := s.interactions.GetRepostedBy(c.Request().Context(), postID, limit, 0)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	var accts []*accounts.Account
	for _, id := range ids {
		if a, err := s.accounts.GetByID(c.Request().Context(), id); err == nil {
			accts = append(accts, a)
		}
	}

	return c.JSON(200, map[string]any{"data": accts})
}

// Timeline handlers

func (s *Server) handleHomeTimeline(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	limit := intQuery(c, "limit", 20)
	maxID := c.Query("max_id")
	sinceID := c.Query("since_id")

	postList, err := s.timelines.Home(c.Request().Context(), accountID, limit, maxID, sinceID)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}

func (s *Server) handleLocalTimeline(c *mizu.Ctx) error {
	viewerID := s.optionalAuth(c)
	limit := intQuery(c, "limit", 20)
	maxID := c.Query("max_id")
	sinceID := c.Query("since_id")

	postList, err := s.timelines.Local(c.Request().Context(), viewerID, limit, maxID, sinceID)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}

func (s *Server) handleHashtagTimeline(c *mizu.Ctx) error {
	tag := c.Param("tag")
	viewerID := s.optionalAuth(c)
	limit := intQuery(c, "limit", 20)
	maxID := c.Query("max_id")
	sinceID := c.Query("since_id")

	postList, err := s.timelines.Hashtag(c.Request().Context(), tag, viewerID, limit, maxID, sinceID)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}

// Notification handlers

func (s *Server) handleNotifications(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	limit := intQuery(c, "limit", 30)
	maxID := c.Query("max_id")
	sinceID := c.Query("since_id")

	notifs, err := s.notifications.List(c.Request().Context(), accountID, nil, limit, maxID, sinceID, nil)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": notifs})
}

func (s *Server) handleClearNotifications(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	if err := s.notifications.MarkAllAsRead(c.Request().Context(), accountID); err != nil {
		return c.JSON(500, errorResponse("CLEAR_FAILED", err.Error()))
	}
	return c.JSON(200, map[string]any{"data": map[string]any{"success": true}})
}

func (s *Server) handleDismissNotification(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	id := c.Param("id")
	if err := s.notifications.Dismiss(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(500, errorResponse("DISMISS_FAILED", err.Error()))
	}
	return c.JSON(200, map[string]any{"data": map[string]any{"success": true}})
}

// Search handler

func (s *Server) handleSearch(c *mizu.Ctx) error {
	query := c.Query("q")
	limit := intQuery(c, "limit", 25)
	viewerID := s.optionalAuth(c)

	results, err := s.search.Search(c.Request().Context(), query, nil, limit, viewerID)
	if err != nil {
		return c.JSON(500, errorResponse("SEARCH_FAILED", err.Error()))
	}

	// Group results by type
	var accountResults, hashtagResults, postResults []*search.Result
	for _, r := range results {
		switch r.Type {
		case search.ResultTypeAccount:
			accountResults = append(accountResults, r)
		case search.ResultTypeHashtag:
			hashtagResults = append(hashtagResults, r)
		case search.ResultTypePost:
			postResults = append(postResults, r)
		}
	}

	return c.JSON(200, map[string]any{
		"data": map[string]any{
			"accounts": accountResults,
			"hashtags": hashtagResults,
			"posts":    postResults,
		},
	})
}

// Trends handlers

func (s *Server) handleTrendingTags(c *mizu.Ctx) error {
	limit := intQuery(c, "limit", 10)
	tags, err := s.trending.Tags(c.Request().Context(), limit)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}
	return c.JSON(200, map[string]any{"data": tags})
}

func (s *Server) handleTrendingPosts(c *mizu.Ctx) error {
	limit := intQuery(c, "limit", 20)
	viewerID := s.optionalAuth(c)

	ids, err := s.trending.Posts(c.Request().Context(), limit)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	var postList []*posts.Post
	for _, id := range ids {
		if p, err := s.posts.GetByID(c.Request().Context(), id, viewerID); err == nil {
			postList = append(postList, p)
		}
	}

	return c.JSON(200, map[string]any{"data": postList})
}

// Bookmarks handler

func (s *Server) handleBookmarks(c *mizu.Ctx) error {
	accountID := s.getAccountID(c)
	limit := intQuery(c, "limit", 20)
	maxID := c.Query("max_id")

	postList, err := s.timelines.Bookmarks(c.Request().Context(), accountID, limit, maxID)
	if err != nil {
		return c.JSON(500, errorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}

// Web page handlers (return HTML)

func (s *Server) handleHomePage(c *mizu.Ctx) error {
	return c.Text(200, "Home page - TODO: implement view")
}

func (s *Server) handleLoginPage(c *mizu.Ctx) error {
	return c.Text(200, "Login page - TODO: implement view")
}

func (s *Server) handleRegisterPage(c *mizu.Ctx) error {
	return c.Text(200, "Register page - TODO: implement view")
}

func (s *Server) handleProfilePage(c *mizu.Ctx) error {
	return c.Text(200, "Profile page - TODO: implement view")
}

func (s *Server) handlePostPage(c *mizu.Ctx) error {
	return c.Text(200, "Post page - TODO: implement view")
}

func (s *Server) handleTagPage(c *mizu.Ctx) error {
	return c.Text(200, "Tag page - TODO: implement view")
}

func (s *Server) handleExplorePage(c *mizu.Ctx) error {
	return c.Text(200, "Explore page - TODO: implement view")
}

func (s *Server) handleNotificationsPage(c *mizu.Ctx) error {
	return c.Text(200, "Notifications page - TODO: implement view")
}

func (s *Server) handleSettingsPage(c *mizu.Ctx) error {
	return c.Text(200, "Settings page - TODO: implement view")
}

// Helpers

func errorResponse(code, message string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

func intQuery(c *mizu.Ctx, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}
