package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/answers"
	"github.com/go-mizu/mizu/blueprints/qa/feature/badges"
	"github.com/go-mizu/mizu/blueprints/qa/feature/comments"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
	"github.com/go-mizu/mizu/blueprints/qa/store/duckdb"
)

// NewSeed creates the seed command.
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long:  "Populates the QA database with sample users, tags, questions, answers, and badges.",
		RunE:  runSeed,
	}
	cmd.AddCommand(NewSeedSE())
	cmd.AddCommand(NewSeedFixTags())
	return cmd
}

func runSeed(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconDatabase, "Seeding QA Database")
	ui.Blank()

	start := time.Now()
	ui.StartSpinner("Opening database...")

	store, err := duckdb.Open(dataDir)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return err
	}
	defer store.Close()

	ui.StopSpinner("Database opened", time.Since(start))

	ctx := context.Background()

	accountsSvc := accounts.NewService(store.Accounts())
	tagsSvc := tags.NewService(store.Tags())
	questionsSvc := questions.NewService(store.Questions(), accountsSvc, tagsSvc)
	answersSvc := answers.NewService(store.Answers(), accountsSvc, questionsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, questionsSvc, answersSvc)
	badgesSvc := badges.NewService(store.Badges())

	ui.Header(iconUser, "Creating Users")
	users := []struct {
		username string
		email    string
	}{
		{"admin", "admin@example.com"},
		{"alice", "alice@example.com"},
		{"bob", "bob@example.com"},
		{"charlie", "charlie@example.com"},
		{"dana", "dana@example.com"},
	}

	userIDs := make(map[string]string)
	for _, u := range users {
		acct, err := accountsSvc.Create(ctx, accounts.CreateIn{
			Username: u.username,
			Email:    u.email,
			Password: "password123",
		})
		if err != nil {
			existing, _ := accountsSvc.GetByUsername(ctx, u.username)
			if existing != nil {
				userIDs[u.username] = existing.ID
				ui.Warn(fmt.Sprintf("User @%s already exists", u.username))
			}
			continue
		}
		userIDs[u.username] = acct.ID
		ui.Step("@" + u.username)
	}

	ui.Header(iconTag, "Creating Tags")
	tagNames := []string{"go", "python", "javascript", "html", "css", "sql", "docker", "kubernetes", "react", "postgresql", "aws", "linux"}
	for _, name := range tagNames {
		_, _ = tagsSvc.Create(ctx, name, "")
		ui.Step(name)
	}

	ui.Header(iconQuestion, "Creating Questions")
	questionsSeed := []struct {
		author string
		title  string
		body   string
		tags   []string
	}{
		{"alice", "How do I center a div with CSS?", "I'm trying to center a div both vertically and horizontally. What is the most reliable approach?", []string{"css", "html"}},
		{"bob", "What is the idiomatic way to handle errors in Go?", "I'm new to Go and want to learn best practices for error handling.", []string{"go"}},
		{"charlie", "How can I optimize a SQL query with multiple joins?", "My query is slow. What are common strategies for optimization?", []string{"sql", "postgresql"}},
		{"dana", "Best way to structure React state for large forms?", "I have a large multi-step form. Looking for patterns to keep state manageable.", []string{"javascript", "react"}},
	}

	questionIDs := make([]string, 0, len(questionsSeed))
	for _, q := range questionsSeed {
		qid := userIDs[q.author]
		if qid == "" {
			continue
		}
		question, err := questionsSvc.Create(ctx, qid, questions.CreateIn{
			Title: q.title,
			Body:  q.body,
			Tags:  q.tags,
		})
		if err != nil {
			ui.Warn("Failed to create question: " + q.title)
			continue
		}
		questionIDs = append(questionIDs, question.ID)
		ui.Step(q.title)
	}

	ui.Header(iconAnswer, "Creating Answers")
	answerSeeds := []struct {
		questionIndex int
		author        string
		body          string
	}{
		{0, "bob", "Use flexbox: `display: flex; align-items: center; justify-content: center;`"},
		{1, "alice", "Return errors and handle them at the call site. Wrap with context when needed."},
		{2, "admin", "Check indexes, reduce joins, and analyze the query plan."},
		{3, "charlie", "Use a form library like React Hook Form and keep state localized."},
	}

	for _, a := range answerSeeds {
		if a.questionIndex >= len(questionIDs) {
			continue
		}
		qid := questionIDs[a.questionIndex]
		authorID := userIDs[a.author]
		if authorID == "" {
			continue
		}
		_, _ = answersSvc.Create(ctx, authorID, answers.CreateIn{QuestionID: qid, Body: a.body})
		ui.Step("Answer by @" + a.author)
	}

	ui.Header(iconComment, "Creating Comments")
	for _, qid := range questionIDs {
		_, _ = commentsSvc.Create(ctx, userIDs["alice"], comments.CreateIn{
			TargetType: comments.TargetQuestion,
			TargetID:   qid,
			Body:       "Can you add more details?",
		})
	}

	ui.Header(iconInfo, "Creating Badges")
	badgesSeed := []badges.Badge{
		{Name: "Enthusiast", Tier: "bronze", Description: "Visited 10 days in a row"},
		{Name: "Teacher", Tier: "silver", Description: "Answered a question with score of 1 or more"},
		{Name: "Legendary", Tier: "gold", Description: "Earned 1000 reputation in a day"},
	}
	for _, b := range badgesSeed {
		_, _ = badgesSvc.Create(ctx, b)
		ui.Step(b.Name)
	}

	ui.Success("Sample data seeded successfully!")
	ui.Blank()
	ui.Hint("Next: run 'qa serve' to start the server")

	return nil
}
