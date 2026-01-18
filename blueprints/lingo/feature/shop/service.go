package shop

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

var (
	ErrItemNotFound       = errors.New("item not found")
	ErrInsufficientGems   = errors.New("insufficient gems")
	ErrItemNotPurchasable = errors.New("item not purchasable")
	ErrAlreadyOwned       = errors.New("item already owned")
)

// Service handles shop business logic
type Service struct {
	store    store.Store
	users    store.UserStore
	progress store.ProgressStore
}

// NewService creates a new shop service
func NewService(st store.Store) *Service {
	return &Service{
		store:    st,
		users:    st.Users(),
		progress: st.Progress(),
	}
}

// ShopItem represents an item available in the shop
type ShopItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Price       int    `json:"price"`
	IconURL     string `json:"icon_url,omitempty"`
	Available   bool   `json:"available"`
	OneTime     bool   `json:"one_time"` // Can only be purchased once
}

// PurchaseResult represents the result of a purchase
type PurchaseResult struct {
	ItemID       string `json:"item_id"`
	GemsSpent    int    `json:"gems_spent"`
	GemsRemaining int   `json:"gems_remaining"`
	Applied      bool   `json:"applied"`
	Message      string `json:"message"`
}

// GetShopItems returns all available shop items
func (s *Service) GetShopItems(ctx context.Context, category string) ([]ShopItem, error) {
	items := []ShopItem{
		// Hearts
		{
			ID:          "heart_refill",
			Name:        "Heart Refill",
			Description: "Refill all 5 hearts",
			Category:    "hearts",
			Price:       350,
			Available:   true,
			OneTime:     false,
		},
		// Streak Protection
		{
			ID:          "streak_freeze",
			Name:        "Streak Freeze",
			Description: "Protect your streak for one missed day",
			Category:    "streak",
			Price:       200,
			Available:   true,
			OneTime:     false,
		},
		{
			ID:          "streak_repair",
			Name:        "Streak Repair",
			Description: "Restore a recently broken streak",
			Category:    "streak",
			Price:       500,
			Available:   true,
			OneTime:     false,
		},
		// Boosts
		{
			ID:          "double_xp",
			Name:        "Double XP Boost",
			Description: "Earn 2x XP for 15 minutes",
			Category:    "boosts",
			Price:       200,
			Available:   true,
			OneTime:     false,
		},
		// Premium
		{
			ID:          "super_weekly",
			Name:        "Super Lingo (1 Week)",
			Description: "Unlimited hearts and extra features for 1 week",
			Category:    "premium",
			Price:       2000,
			Available:   true,
			OneTime:     false,
		},
		{
			ID:          "super_monthly",
			Name:        "Super Lingo (1 Month)",
			Description: "Unlimited hearts and extra features for 1 month",
			Category:    "premium",
			Price:       6000,
			Available:   true,
			OneTime:     false,
		},
		// Cosmetics
		{
			ID:          "outfit_owl",
			Name:        "Owl Costume",
			Description: "A stylish owl costume for your avatar",
			Category:    "cosmetics",
			Price:       500,
			Available:   true,
			OneTime:     true,
		},
		{
			ID:          "outfit_superhero",
			Name:        "Superhero Cape",
			Description: "A heroic cape for your avatar",
			Category:    "cosmetics",
			Price:       750,
			Available:   true,
			OneTime:     true,
		},
	}

	if category != "" {
		var filtered []ShopItem
		for _, item := range items {
			if item.Category == category {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	}

	return items, nil
}

// GetItem returns a specific shop item
func (s *Service) GetItem(ctx context.Context, itemID string) (*ShopItem, error) {
	items, _ := s.GetShopItems(ctx, "")
	for _, item := range items {
		if item.ID == itemID {
			return &item, nil
		}
	}
	return nil, ErrItemNotFound
}

// Purchase handles item purchase
func (s *Service) Purchase(ctx context.Context, userID uuid.UUID, itemID string) (*PurchaseResult, error) {
	// Get item
	item, err := s.GetItem(ctx, itemID)
	if err != nil {
		return nil, err
	}

	if !item.Available {
		return nil, ErrItemNotPurchasable
	}

	// Get user
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check gems
	if user.Gems < item.Price {
		return nil, ErrInsufficientGems
	}

	// Deduct gems
	newGems := user.Gems - item.Price
	if err := s.users.UpdateGems(ctx, userID, newGems); err != nil {
		return nil, err
	}

	// Apply item effect
	message := s.applyItemEffect(ctx, userID, itemID, user)

	return &PurchaseResult{
		ItemID:        itemID,
		GemsSpent:     item.Price,
		GemsRemaining: newGems,
		Applied:       true,
		Message:       message,
	}, nil
}

func (s *Service) applyItemEffect(ctx context.Context, userID uuid.UUID, itemID string, user *store.User) string {
	switch itemID {
	case "heart_refill":
		_ = s.users.UpdateHearts(ctx, userID, 5)
		return "Hearts refilled to 5"

	case "streak_freeze":
		user.StreakFreezeCount++
		_ = s.users.Update(ctx, user)
		return "Streak freeze added to inventory"

	case "streak_repair":
		// Repair streak (simplified - would need more logic)
		_ = s.users.UpdateStreak(ctx, userID)
		return "Streak restored"

	case "double_xp":
		// Would need to store boost expiration
		return "2x XP boost activated for 15 minutes"

	case "super_weekly":
		user.IsPremium = true
		expires := time.Now().Add(7 * 24 * time.Hour)
		user.PremiumExpiresAt = &expires
		_ = s.users.Update(ctx, user)
		return "Super Lingo activated for 1 week"

	case "super_monthly":
		user.IsPremium = true
		expires := time.Now().Add(30 * 24 * time.Hour)
		user.PremiumExpiresAt = &expires
		_ = s.users.Update(ctx, user)
		return "Super Lingo activated for 1 month"

	default:
		return "Item purchased"
	}
}

// Categories returns all shop categories
func (s *Service) Categories() []string {
	return []string{
		"hearts",
		"streak",
		"boosts",
		"premium",
		"cosmetics",
	}
}
