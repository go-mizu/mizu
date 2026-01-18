package storetest

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
)

func TestGamificationStore_GetLeagues(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		gamification := s.Gamification()

		leagues, err := gamification.GetLeagues(ctx)
		assertNoError(t, err, "get leagues")

		if len(leagues) < 1 {
			t.Fatal("expected at least 1 league")
		}

		// Verify league fields
		for _, league := range leagues {
			if league.Name == "" {
				t.Fatal("expected league name")
			}
		}
	})
}

func TestGamificationStore_GetCurrentSeason(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		gamification := s.Gamification()

		// Get leagues first
		leagues, _ := gamification.GetLeagues(ctx)
		if len(leagues) < 1 {
			t.Skip("no leagues available")
		}

		leagueID := leagues[0].ID

		// Get current season
		season, err := gamification.GetCurrentSeason(ctx, leagueID)
		assertNoError(t, err, "get current season")

		if season == nil {
			t.Fatal("expected season to not be nil")
		}
		assertEqual(t, leagueID, season.LeagueID, "league id")
	})
}

func TestGamificationStore_JoinLeague(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		gamification := s.Gamification()

		// Get a league and its season
		leagues, _ := gamification.GetLeagues(ctx)
		if len(leagues) < 1 {
			t.Skip("no leagues available")
		}
		leagueID := leagues[0].ID

		season, err := gamification.GetCurrentSeason(ctx, leagueID)
		if err != nil || season == nil {
			t.Skip("no season available")
		}

		// Join league
		err = gamification.JoinLeague(ctx, user.ID, season.ID)
		assertNoError(t, err, "join league")

		// Get user league
		userLeague, err := gamification.GetUserLeague(ctx, user.ID)
		assertNoError(t, err, "get user league")
		assertEqual(t, season.ID, userLeague.SeasonID, "season id")
	})
}

func TestGamificationStore_UpdateLeagueXP(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		gamification := s.Gamification()

		// Get a league and join
		leagues, _ := gamification.GetLeagues(ctx)
		if len(leagues) < 1 {
			t.Skip("no leagues available")
		}
		leagueID := leagues[0].ID

		season, err := gamification.GetCurrentSeason(ctx, leagueID)
		if err != nil || season == nil {
			t.Skip("no season available")
		}

		_ = gamification.JoinLeague(ctx, user.ID, season.ID)

		// Update XP
		err = gamification.UpdateLeagueXP(ctx, user.ID, season.ID, 150)
		assertNoError(t, err, "update league xp")

		// Verify update
		userLeague, _ := gamification.GetUserLeague(ctx, user.ID)
		assertEqual(t, 150, userLeague.XPEarned, "xp earned")
	})
}

func TestGamificationStore_GetLeaderboard(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		gamification := s.Gamification()

		// Get a league
		leagues, _ := gamification.GetLeagues(ctx)
		if len(leagues) < 1 {
			t.Skip("no leagues available")
		}
		leagueID := leagues[0].ID

		season, err := gamification.GetCurrentSeason(ctx, leagueID)
		if err != nil || season == nil {
			t.Skip("no season available")
		}

		// Create users and add to league with different XP
		for i := 0; i < 5; i++ {
			user := createTestUser(t, s)
			_ = gamification.JoinLeague(ctx, user.ID, season.ID)
			_ = gamification.UpdateLeagueXP(ctx, user.ID, season.ID, (5-i)*100)
		}

		// Get leaderboard
		leaderboard, err := gamification.GetLeaderboard(ctx, season.ID, 10)
		assertNoError(t, err, "get leaderboard")

		if len(leaderboard) < 5 {
			t.Fatalf("expected at least 5 entries, got %d", len(leaderboard))
		}

		// Verify ordering (highest XP first)
		for i := 1; i < len(leaderboard); i++ {
			if leaderboard[i].XPEarned > leaderboard[i-1].XPEarned {
				t.Fatal("leaderboard should be ordered by XP descending")
			}
		}
	})
}
