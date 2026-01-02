package handler

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/answers"
	"github.com/go-mizu/mizu/blueprints/qa/feature/badges"
	"github.com/go-mizu/mizu/blueprints/qa/feature/comments"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/feature/search"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
)

// Page handles HTML page endpoints.
type Page struct {
	templates    map[string]*template.Template
	accounts     accounts.API
	questions    questions.API
	answers      answers.API
	comments     comments.API
	tags         tags.API
	badges       badges.API
	search       search.API
	getAccountID func(*mizu.Ctx) string
}

// NewPage creates a new page handler.
func NewPage(
	templates map[string]*template.Template,
	accounts accounts.API,
	questions questions.API,
	answers answers.API,
	comments comments.API,
	tags tags.API,
	badges badges.API,
	search search.API,
	getAccountID func(*mizu.Ctx) string,
) *Page {
	return &Page{
		templates:    templates,
		accounts:     accounts,
		questions:    questions,
		answers:      answers,
		comments:     comments,
		tags:         tags,
		badges:       badges,
		search:       search,
		getAccountID: getAccountID,
	}
}

// PageData is the data passed to page templates.
type PageData struct {
	Title       string
	ActiveNav   string
	Query       string
	CurrentUser *accounts.Account
	Data        any
}

func (h *Page) render(c *mizu.Ctx, name string, data PageData) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")

	accountID := h.getAccountID(c)
	if accountID != "" {
		data.CurrentUser, _ = h.accounts.GetByID(c.Request().Context(), accountID)
	}
	data.Query = c.Query("q")

	return tmpl.ExecuteTemplate(c.Writer(), name, data)
}

// Home renders the home page.
func (h *Page) Home(c *mizu.Ctx) error {
	sort := questions.SortBy(c.Query("sort"))
	if sort == "" {
		sort = questions.SortNewest
	}
	list, _ := h.questions.List(c.Request().Context(), questions.ListOpts{Limit: 30, SortBy: sort})
	h.enrichQuestions(c, list)

	return h.render(c, "home.html", PageData{
		Title:     "Top Questions",
		ActiveNav: "home",
		Data: map[string]any{
			"Questions": list,
			"Sort":      sort,
		},
	})
}

// Questions renders the questions page.
func (h *Page) Questions(c *mizu.Ctx) error {
	sort := questions.SortBy(c.Query("sort"))
	if sort == "" {
		sort = questions.SortNewest
	}
	list, _ := h.questions.List(c.Request().Context(), questions.ListOpts{Limit: 30, SortBy: sort})
	h.enrichQuestions(c, list)

	return h.render(c, "questions.html", PageData{
		Title:     "All Questions",
		ActiveNav: "questions",
		Data: map[string]any{
			"Questions": list,
			"Sort":      sort,
		},
	})
}

// Question renders question detail.
func (h *Page) Question(c *mizu.Ctx) error {
	id := c.Param("id")

	question, err := h.questions.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.Text(404, "Question not found")
	}
	_ = h.questions.IncrementViews(c.Request().Context(), id)

	answersList, _ := h.answers.ListByQuestion(c.Request().Context(), id, answers.ListOpts{Limit: 100})

	commentsOnQuestion, _ := h.comments.ListByTarget(c.Request().Context(), comments.TargetQuestion, id, comments.ListOpts{Limit: 50})

	related, _ := h.questions.List(c.Request().Context(), questions.ListOpts{Limit: 5, SortBy: questions.SortScore})
	h.enrichQuestions(c, related)
	answerViews := h.attachAnswerComments(c, answersList)

	return h.render(c, "question.html", PageData{
		Title:     question.Title,
		ActiveNav: "questions",
		Data: map[string]any{
			"Question":         question,
			"QuestionComments": commentsOnQuestion,
			"Answers":          answerViews,
			"AnswerCount":      int64(len(answersList)),
			"RelatedQuestions": related,
		},
	})
}

// Ask renders ask page.
func (h *Page) Ask(c *mizu.Ctx) error {
	return h.render(c, "ask.html", PageData{
		Title:     "Ask Question",
		ActiveNav: "questions",
		Data:      map[string]any{},
	})
}

// Tags renders tags page.
func (h *Page) Tags(c *mizu.Ctx) error {
	query := c.Query("q")
	list, _ := h.tags.List(c.Request().Context(), tags.ListOpts{Limit: 100, Query: query})
	return h.render(c, "tags.html", PageData{
		Title:     "Tags",
		ActiveNav: "tags",
		Data: map[string]any{
			"Tags":     list,
			"TagCount": int64(len(list)),
			"Query":    query,
		},
	})
}

// Tag renders tag detail.
func (h *Page) Tag(c *mizu.Ctx) error {
	name := strings.ToLower(c.Param("name"))

	tag, err := h.tags.GetByName(c.Request().Context(), name)
	if err != nil {
		return c.Text(404, "Tag not found")
	}

	list, _ := h.questions.ListByTag(c.Request().Context(), name, questions.ListOpts{Limit: 30})
	h.enrichQuestions(c, list)

	return h.render(c, "tag.html", PageData{
		Title:     "Tag: " + name,
		ActiveNav: "tags",
		Data: map[string]any{
			"Tag":       tag,
			"Questions": list,
		},
	})
}

// Users renders users page.
func (h *Page) Users(c *mizu.Ctx) error {
	query := c.Query("q")
	var list []*accounts.Account
	if query != "" {
		list, _ = h.accounts.Search(c.Request().Context(), query, 60)
	} else {
		list, _ = h.accounts.List(c.Request().Context(), accounts.ListOpts{Limit: 60, OrderBy: "reputation"})
	}
	return h.render(c, "users.html", PageData{
		Title:     "Users",
		ActiveNav: "users",
		Data: map[string]any{
			"Users": list,
			"Query": query,
		},
	})
}

// User renders user profile.
func (h *Page) User(c *mizu.Ctx) error {
	id := c.Param("id")

	user, err := h.accounts.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.Text(404, "User not found")
	}
	qs, _ := h.questions.ListByAuthor(c.Request().Context(), id, questions.ListOpts{Limit: 10})
	h.enrichQuestions(c, qs)

	tagsList, _ := h.tags.List(c.Request().Context(), tags.ListOpts{Limit: 6})

	return h.render(c, "user.html", PageData{
		Title:     user.Username,
		ActiveNav: "users",
		Data: map[string]any{
			"User":      user,
			"Questions": qs,
			"TopTags":   tagsList,
		},
	})
}

// Badges renders badges page.
func (h *Page) Badges(c *mizu.Ctx) error {
	list, _ := h.badges.List(c.Request().Context(), 100)
	return h.render(c, "badges.html", PageData{
		Title:     "Badges",
		ActiveNav: "badges",
		Data: map[string]any{
			"Badges": list,
		},
	})
}

// Search renders search results.
func (h *Page) Search(c *mizu.Ctx) error {
	q := c.Query("q")
	result, _ := h.search.Search(c.Request().Context(), q, 30)
	qs := result.Questions
	h.enrichQuestions(c, qs)

	return h.render(c, "search.html", PageData{
		Title:     "Search",
		ActiveNav: "questions",
		Data: map[string]any{
			"Query":     q,
			"Questions": qs,
		},
	})
}

// Login renders login page.
func (h *Page) Login(c *mizu.Ctx) error {
	return h.render(c, "login.html", PageData{
		Title:     "Log in",
		ActiveNav: "",
		Data:      map[string]any{},
	})
}

// Register renders register page.
func (h *Page) Register(c *mizu.Ctx) error {
	return h.render(c, "register.html", PageData{
		Title:     "Sign up",
		ActiveNav: "",
		Data:      map[string]any{},
	})
}

func (h *Page) enrichQuestions(c *mizu.Ctx, list []*questions.Question) {
	_ = h.questions.EnrichQuestions(c.Request().Context(), list)
}

func (h *Page) attachAnswerComments(c *mizu.Ctx, list []*answers.Answer) []AnswerView {
	views := make([]AnswerView, 0, len(list))

	// Collect answer IDs
	answerIDs := make([]string, len(list))
	for i, answer := range list {
		answerIDs[i] = answer.ID
	}

	// Batch load all comments for all answers
	commentsMap, _ := h.comments.ListByTargets(c.Request().Context(), comments.TargetAnswer, answerIDs, comments.ListOpts{Limit: 20})

	for _, answer := range list {
		views = append(views, AnswerView{
			Answer:   answer,
			Comments: commentsMap[answer.ID],
		})
	}
	return views
}

// AnswerView enriches answers with comments for templates.
type AnswerView struct {
	*answers.Answer
	Comments []*comments.Comment
}
