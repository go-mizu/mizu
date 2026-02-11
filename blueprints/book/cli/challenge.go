package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/book/types"
	"github.com/spf13/cobra"
)

func NewChallenge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge [year] [goal]",
		Short: "Set or view reading challenge",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			store, err := sqlite.New(GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.Ensure(ctx); err != nil {
				return err
			}

			year := time.Now().Year()
			if len(args) >= 1 {
				year, _ = strconv.Atoi(args[0])
			}

			if len(args) >= 2 {
				goal, _ := strconv.Atoi(args[1])
				if goal <= 0 {
					return fmt.Errorf("goal must be positive")
				}
				ch := &types.ReadingChallenge{Year: year, Goal: goal}
				if err := store.Challenge().Set(ctx, ch); err != nil {
					return err
				}
				fmt.Println(successStyle.Render(fmt.Sprintf("Set %d reading challenge: %d books", year, goal)))
				return nil
			}

			// View challenge
			ch, err := store.Challenge().Get(ctx, year)
			if err != nil {
				return err
			}
			if ch == nil {
				fmt.Printf("No reading challenge set for %d\n", year)
				fmt.Println(dimStyle.Render("  Set one: book challenge " + strconv.Itoa(year) + " 52"))
				return nil
			}

			progress, _ := store.Challenge().GetProgress(ctx, year)
			pct := 0
			if ch.Goal > 0 {
				pct = progress * 100 / ch.Goal
			}

			fmt.Println(titleStyle.Render(fmt.Sprintf("%d Reading Challenge", year)))
			fmt.Printf("  Goal:     %d books\n", ch.Goal)
			fmt.Printf("  Read:     %d books\n", progress)
			fmt.Printf("  Progress: %d%%\n", pct)

			// Progress bar
			barLen := 30
			filled := barLen * pct / 100
			if filled > barLen {
				filled = barLen
			}
			bar := ""
			for i := 0; i < filled; i++ {
				bar += "█"
			}
			for i := filled; i < barLen; i++ {
				bar += "░"
			}
			fmt.Printf("  [%s]\n", successStyle.Render(bar))

			return nil
		},
	}
	return cmd
}
