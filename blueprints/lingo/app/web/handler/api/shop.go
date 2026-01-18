package api

import (
	"net/http"
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// ShopHandler handles shop endpoints
type ShopHandler struct {
	store store.Store
}

// NewShopHandler creates a new shop handler
func NewShopHandler(st store.Store) *ShopHandler {
	return &ShopHandler{store: st}
}

// ShopItem represents an item in the shop
type ShopItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	GemCost     int    `json:"gem_cost"`
	IconURL     string `json:"icon_url,omitempty"`
}

// GetItems returns available shop items
func (h *ShopHandler) GetItems(c *mizu.Ctx) error {
	items := []ShopItem{
		{
			ID:          "streak_freeze",
			Name:        "Streak Freeze",
			Description: "Protect your streak for one day",
			Type:        "consumable",
			GemCost:     200,
		},
		{
			ID:          "heart_refill",
			Name:        "Heart Refill",
			Description: "Refill all hearts instantly",
			Type:        "consumable",
			GemCost:     350,
		},
		{
			ID:          "double_or_nothing",
			Name:        "Double or Nothing",
			Description: "Risk your streak for double gems",
			Type:        "challenge",
			GemCost:     50,
		},
		{
			ID:          "xp_boost",
			Name:        "XP Boost",
			Description: "Earn double XP for 15 minutes",
			Type:        "consumable",
			GemCost:     100,
		},
		{
			ID:          "super_lingo_monthly",
			Name:        "Super Lingo (Monthly)",
			Description: "Unlimited hearts, no ads, and more",
			Type:        "subscription",
			GemCost:     0, // Real money purchase
		},
	}

	return c.JSON(http.StatusOK, items)
}

// PurchaseRequest represents a purchase request
type PurchaseRequest struct {
	ItemID string `json:"item_id"`
}

// Purchase handles item purchases
func (h *ShopHandler) Purchase(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	var req PurchaseRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	user, err := h.store.Users().GetByID(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	// Item prices
	prices := map[string]int{
		"streak_freeze":    200,
		"heart_refill":     350,
		"double_or_nothing": 50,
		"xp_boost":         100,
	}

	price, ok := prices[req.ItemID]
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid item"})
	}

	if user.Gems < price {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "not enough gems"})
	}

	// Deduct gems
	newGems := user.Gems - price
	if err := h.store.Users().UpdateGems(c.Context(), uid, newGems); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to process purchase"})
	}

	// Apply item effect
	switch req.ItemID {
	case "streak_freeze":
		// In production, add to user's inventory
	case "heart_refill":
		h.store.Users().UpdateHearts(c.Context(), uid, 5)
	case "xp_boost":
		// In production, activate XP boost
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "purchase successful",
		"gems":    newGems,
		"item_id": req.ItemID,
	})
}
