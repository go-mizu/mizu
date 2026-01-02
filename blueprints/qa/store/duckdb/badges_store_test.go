package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/badges"
)

func TestBadgesStore(t *testing.T) {
	db := newTestDB(t)
	accountsStore := NewAccountsStore(db)
	badgesStore := NewBadgesStore(db)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           "acc-1",
		Username:     "user",
		Email:        "user@example.com",
		PasswordHash: "hash",
		CreatedAt:    mustTime(2024, time.January, 1, 10, 0, 0),
		UpdatedAt:    mustTime(2024, time.January, 1, 10, 0, 0),
	}
	if err := accountsStore.Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	badgeGold := &badges.Badge{ID: "badge-1", Name: "Nice Answer", Tier: "gold", Description: "Gold badge"}
	badgeBronze := &badges.Badge{ID: "badge-2", Name: "Supporter", Tier: "bronze", Description: "Bronze badge"}
	if err := badgesStore.Create(ctx, badgeGold); err != nil {
		t.Fatalf("create badge gold: %v", err)
	}
	if err := badgesStore.Create(ctx, badgeBronze); err != nil {
		t.Fatalf("create badge bronze: %v", err)
	}

	listed, err := badgesStore.List(ctx, 10)
	if err != nil {
		t.Fatalf("list badges: %v", err)
	}
	if len(listed) != 2 || listed[0].Tier != "gold" {
		t.Fatalf("unexpected badge order: %#v", listed)
	}

	award := &badges.Award{
		ID:        "award-1",
		AccountID: account.ID,
		BadgeID:   badgeGold.ID,
		CreatedAt: mustTime(2024, time.January, 2, 10, 0, 0),
	}
	if err := badgesStore.CreateAward(ctx, award); err != nil {
		t.Fatalf("create award: %v", err)
	}

	awards, err := badgesStore.ListAwards(ctx, account.ID)
	if err != nil {
		t.Fatalf("list awards: %v", err)
	}
	if len(awards) != 1 || awards[0].ID != award.ID {
		t.Fatalf("unexpected awards: %#v", awards)
	}
}
