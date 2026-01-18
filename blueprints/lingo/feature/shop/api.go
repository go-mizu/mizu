package shop

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for shop
type Handler struct {
	svc *Service
}

// NewHandler creates a new shop handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers shop routes
func (h *Handler) RegisterRoutes(r *mizu.Router) {
	r.Get("/shop/items", h.GetItems)
	r.Get("/shop/items/{id}", h.GetItem)
	r.Post("/shop/purchase", h.Purchase)
	r.Get("/shop/categories", h.GetCategories)
}

// GetItems handles GET /shop/items
func (h *Handler) GetItems(c *mizu.Ctx) error {
	category := c.Query("category")

	items, err := h.svc.GetShopItems(c.Context(), category)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get items"})
	}

	return c.JSON(http.StatusOK, items)
}

// GetItem handles GET /shop/items/{id}
func (h *Handler) GetItem(c *mizu.Ctx) error {
	itemID := c.Param("id")

	item, err := h.svc.GetItem(c.Context(), itemID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "item not found"})
	}

	return c.JSON(http.StatusOK, item)
}

// PurchaseRequest represents a purchase request
type PurchaseRequest struct {
	ItemID string `json:"item_id"`
}

// Purchase handles POST /shop/purchase
func (h *Handler) Purchase(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req PurchaseRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.ItemID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "item_id is required"})
	}

	result, err := h.svc.Purchase(c.Context(), userID, req.ItemID)
	if err != nil {
		switch err {
		case ErrItemNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "item not found"})
		case ErrInsufficientGems:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "insufficient gems"})
		case ErrItemNotPurchasable:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "item not purchasable"})
		case ErrAlreadyOwned:
			return c.JSON(http.StatusConflict, map[string]string{"error": "item already owned"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to purchase"})
		}
	}

	return c.JSON(http.StatusOK, result)
}

// GetCategories handles GET /shop/categories
func (h *Handler) GetCategories(c *mizu.Ctx) error {
	return c.JSON(http.StatusOK, h.svc.Categories())
}

// getUserID extracts the user ID from the request context
func getUserID(c *mizu.Ctx) uuid.UUID {
	if userIDStr := c.Header().Get("X-User-ID"); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			return id
		}
	}
	return uuid.Nil
}
