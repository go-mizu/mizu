package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type ReviewHandler struct{ st store.Store }

func NewReviewHandler(st store.Store) *ReviewHandler { return &ReviewHandler{st: st} }

func (h *ReviewHandler) GetByBook(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	rating, _ := strconv.Atoi(c.Query("rating"))
	hasTextRaw := c.Query("has_text")
	var hasText *bool
	if hasTextRaw == "true" {
		v := true
		hasText = &v
	} else if hasTextRaw == "false" {
		v := false
		hasText = &v
	}
	includeSpoilers := c.Query("include_spoilers") == "true"
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	query := types.ReviewQuery{
		Page:            page,
		Limit:           limit,
		Sort:            c.Query("sort"),
		Rating:          rating,
		Source:          c.Query("source"),
		Query:           c.Query("q"),
		HasText:         hasText,
		IncludeSpoilers: includeSpoilers,
	}
	if query.Source == "goodreads" {
		query.Source = "imported"
	}
	reviews, total, err := h.st.Review().GetByBookFiltered(c.Context(), bookID, query)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	for i := range reviews {
		if reviews[i].Source == "goodreads" {
			reviews[i].Source = "imported"
		}
	}
	return c.JSON(200, map[string]any{
		"reviews": reviews,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

func (h *ReviewHandler) Create(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Rating       int    `json:"rating"`
		Text         string `json:"text"`
		IsSpoiler    bool   `json:"is_spoiler"`
		StartedAt    string `json:"started_at"`
		FinishedAt   string `json:"finished_at"`
		ReviewerName string `json:"reviewer_name"`
	}
	if err := c.BindJSON(&payload, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	review := types.Review{
		BookID:       bookID,
		Rating:       payload.Rating,
		Text:         payload.Text,
		IsSpoiler:    payload.IsSpoiler,
		ReviewerName: payload.ReviewerName,
		Source:       "user",
	}
	if t, err := parseDate(payload.StartedAt); err == nil {
		review.StartedAt = t
	}
	if t, err := parseDate(payload.FinishedAt); err == nil {
		review.FinishedAt = t
	}
	if err := h.st.Review().Create(c.Context(), &review); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Add feed entry
	go func() {
		book, _ := h.st.Book().Get(c.Context(), bookID)
		title := ""
		if book != nil {
			title = book.Title
		}
		feedType := "rating"
		if review.Text != "" {
			feedType = "review"
		}
		data, _ := json.Marshal(map[string]any{"rating": review.Rating, "text": review.Text})
		h.st.Feed().Add(c.Context(), &types.FeedItem{
			Type:      feedType,
			BookID:    bookID,
			BookTitle: title,
			Data:      string(data),
		})
	}()

	return c.JSON(201, review)
}

func (h *ReviewHandler) Update(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Rating     int    `json:"rating"`
		Text       string `json:"text"`
		IsSpoiler  bool   `json:"is_spoiler"`
		StartedAt  string `json:"started_at"`
		FinishedAt string `json:"finished_at"`
	}
	if err := c.BindJSON(&payload, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	review, err := h.st.Review().Get(c.Context(), id)
	if err != nil || review == nil {
		return c.JSON(404, map[string]string{"error": "review not found"})
	}
	review.Rating = payload.Rating
	review.Text = payload.Text
	review.IsSpoiler = payload.IsSpoiler
	if t, err := parseDate(payload.StartedAt); err == nil {
		review.StartedAt = t
	}
	if t, err := parseDate(payload.FinishedAt); err == nil {
		review.FinishedAt = t
	}
	if err := h.st.Review().Update(c.Context(), review); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, review)
}

func (h *ReviewHandler) Delete(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.st.Review().Delete(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

func (h *ReviewHandler) Like(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	likes, err := h.st.Review().AddLike(c.Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{"likes_count": likes})
}

func (h *ReviewHandler) GetComments(c *mizu.Ctx) error {
	reviewID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	comments, total, err := h.st.ReviewComment().GetByReview(c.Context(), reviewID, page, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{"comments": comments, "total": total})
}

func (h *ReviewHandler) CreateComment(c *mizu.Ctx) error {
	reviewID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Text       string `json:"text"`
		AuthorName string `json:"author_name"`
	}
	if err := c.BindJSON(&payload, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	payload.Text = strings.TrimSpace(payload.Text)
	if payload.Text == "" {
		return c.JSON(400, map[string]string{"error": "comment text required"})
	}
	author := strings.TrimSpace(payload.AuthorName)
	if author == "" {
		author = "You"
	}
	comment := types.ReviewComment{
		ReviewID:   reviewID,
		AuthorName: author,
		Text:       payload.Text,
	}
	if err := h.st.ReviewComment().Create(c.Context(), &comment); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, comment)
}

func (h *ReviewHandler) DeleteComment(c *mizu.Ctx) error {
	commentID, _ := strconv.ParseInt(c.Param("commentId"), 10, 64)
	if err := h.st.ReviewComment().Delete(c.Context(), commentID); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

func (h *ReviewHandler) GetProgress(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	progress, err := h.st.Progress().GetByBook(c.Context(), bookID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, progress)
}

func (h *ReviewHandler) UpdateProgress(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var p types.ReadingProgress
	if err := c.BindJSON(&p, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	p.BookID = bookID
	if err := h.st.Progress().Create(c.Context(), &p); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Add feed entry
	go func() {
		book, _ := h.st.Book().Get(c.Context(), bookID)
		title := ""
		if book != nil {
			title = book.Title
		}
		data, _ := json.Marshal(map[string]any{"page": p.Page, "percent": p.Percent})
		h.st.Feed().Add(c.Context(), &types.FeedItem{
			Type:      "progress",
			BookID:    bookID,
			BookTitle: title,
			Data:      string(data),
		})
	}()

	return c.JSON(201, p)
}

func parseDate(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return &t, nil
	}
	return nil, nil
}
