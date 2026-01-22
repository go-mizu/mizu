package api

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/bi/pkg/password"
	"github.com/go-mizu/blueprints/bi/store"
	"github.com/go-mizu/blueprints/bi/store/sqlite"
)

// Dashboards handles dashboard API endpoints.
type Dashboards struct {
	store *sqlite.Store
}

func NewDashboards(store *sqlite.Store) *Dashboards {
	return &Dashboards{store: store}
}

func (h *Dashboards) List(c *mizu.Ctx) error {
	dashboards, err := h.store.Dashboards().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, dashboards)
}

func (h *Dashboards) Create(c *mizu.Ctx) error {
	var d store.Dashboard
	if err := c.BindJSON(&d, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	if err := h.store.Dashboards().Create(c.Request().Context(), &d); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, d)
}

func (h *Dashboards) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	d, err := h.store.Dashboards().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if d == nil {
		return c.JSON(404, map[string]string{"error": "Dashboard not found"})
	}
	return c.JSON(200, d)
}

func (h *Dashboards) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var d store.Dashboard
	if err := c.BindJSON(&d, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	d.ID = id
	if err := h.store.Dashboards().Update(c.Request().Context(), &d); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, d)
}

func (h *Dashboards) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Dashboards().Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

func (h *Dashboards) ListCards(c *mizu.Ctx) error {
	id := c.Param("id")
	cards, err := h.store.Dashboards().ListCards(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, cards)
}

func (h *Dashboards) AddCard(c *mizu.Ctx) error {
	id := c.Param("id")
	var card store.DashboardCard
	if err := c.BindJSON(&card, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	card.DashboardID = id
	if err := h.store.Dashboards().CreateCard(c.Request().Context(), &card); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, card)
}

func (h *Dashboards) UpdateCard(c *mizu.Ctx) error {
	cardID := c.Param("card")
	var card store.DashboardCard
	if err := c.BindJSON(&card, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	card.ID = cardID
	if err := h.store.Dashboards().UpdateCard(c.Request().Context(), &card); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, card)
}

func (h *Dashboards) RemoveCard(c *mizu.Ctx) error {
	cardID := c.Param("card")
	if err := h.store.Dashboards().DeleteCard(c.Request().Context(), cardID); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// Collections handles collection API endpoints.
type Collections struct {
	store *sqlite.Store
}

func NewCollections(store *sqlite.Store) *Collections {
	return &Collections{store: store}
}

func (h *Collections) List(c *mizu.Ctx) error {
	collections, err := h.store.Collections().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, collections)
}

func (h *Collections) Create(c *mizu.Ctx) error {
	var col store.Collection
	if err := c.BindJSON(&col, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	if err := h.store.Collections().Create(c.Request().Context(), &col); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, col)
}

func (h *Collections) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	col, err := h.store.Collections().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if col == nil {
		return c.JSON(404, map[string]string{"error": "Collection not found"})
	}
	return c.JSON(200, col)
}

func (h *Collections) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var col store.Collection
	if err := c.BindJSON(&col, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	col.ID = id
	if err := h.store.Collections().Update(c.Request().Context(), &col); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, col)
}

func (h *Collections) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Collections().Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

func (h *Collections) ListItems(c *mizu.Ctx) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	questions, _ := h.store.Questions().ListByCollection(ctx, id)
	dashboards, _ := h.store.Dashboards().ListByCollection(ctx, id)
	subcollections, _ := h.store.Collections().ListByParent(ctx, id)

	return c.JSON(200, map[string]interface{}{
		"questions":      questions,
		"dashboards":     dashboards,
		"subcollections": subcollections,
	})
}

// Models handles model API endpoints.
type Models struct {
	store *sqlite.Store
}

func NewModels(store *sqlite.Store) *Models {
	return &Models{store: store}
}

func (h *Models) List(c *mizu.Ctx) error {
	models, err := h.store.Models().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, models)
}

func (h *Models) Create(c *mizu.Ctx) error {
	var m store.Model
	if err := c.BindJSON(&m, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	if err := h.store.Models().Create(c.Request().Context(), &m); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, m)
}

func (h *Models) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	m, err := h.store.Models().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if m == nil {
		return c.JSON(404, map[string]string{"error": "Model not found"})
	}
	return c.JSON(200, m)
}

func (h *Models) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var m store.Model
	if err := c.BindJSON(&m, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	m.ID = id
	if err := h.store.Models().Update(c.Request().Context(), &m); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, m)
}

func (h *Models) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Models().Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// Metrics handles metric API endpoints.
type Metrics struct {
	store *sqlite.Store
}

func NewMetrics(store *sqlite.Store) *Metrics {
	return &Metrics{store: store}
}

func (h *Metrics) List(c *mizu.Ctx) error {
	metrics, err := h.store.Metrics().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, metrics)
}

func (h *Metrics) Create(c *mizu.Ctx) error {
	var m store.Metric
	if err := c.BindJSON(&m, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	if err := h.store.Metrics().Create(c.Request().Context(), &m); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, m)
}

func (h *Metrics) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	m, err := h.store.Metrics().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if m == nil {
		return c.JSON(404, map[string]string{"error": "Metric not found"})
	}
	return c.JSON(200, m)
}

func (h *Metrics) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var m store.Metric
	if err := c.BindJSON(&m, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	m.ID = id
	if err := h.store.Metrics().Update(c.Request().Context(), &m); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, m)
}

func (h *Metrics) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Metrics().Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// Alerts handles alert API endpoints.
type Alerts struct {
	store *sqlite.Store
}

func NewAlerts(store *sqlite.Store) *Alerts {
	return &Alerts{store: store}
}

func (h *Alerts) List(c *mizu.Ctx) error {
	alerts, err := h.store.Alerts().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, alerts)
}

func (h *Alerts) Create(c *mizu.Ctx) error {
	var a store.Alert
	if err := c.BindJSON(&a, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	if err := h.store.Alerts().Create(c.Request().Context(), &a); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, a)
}

func (h *Alerts) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	a, err := h.store.Alerts().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if a == nil {
		return c.JSON(404, map[string]string{"error": "Alert not found"})
	}
	return c.JSON(200, a)
}

func (h *Alerts) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var a store.Alert
	if err := c.BindJSON(&a, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	a.ID = id
	if err := h.store.Alerts().Update(c.Request().Context(), &a); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, a)
}

func (h *Alerts) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Alerts().Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// Subscriptions handles subscription API endpoints.
type Subscriptions struct {
	store *sqlite.Store
}

func NewSubscriptions(store *sqlite.Store) *Subscriptions {
	return &Subscriptions{store: store}
}

func (h *Subscriptions) List(c *mizu.Ctx) error {
	subs, err := h.store.Subscriptions().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, subs)
}

func (h *Subscriptions) Create(c *mizu.Ctx) error {
	var s store.Subscription
	if err := c.BindJSON(&s, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	if err := h.store.Subscriptions().Create(c.Request().Context(), &s); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, s)
}

func (h *Subscriptions) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	s, err := h.store.Subscriptions().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if s == nil {
		return c.JSON(404, map[string]string{"error": "Subscription not found"})
	}
	return c.JSON(200, s)
}

func (h *Subscriptions) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var s store.Subscription
	if err := c.BindJSON(&s, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	s.ID = id
	if err := h.store.Subscriptions().Update(c.Request().Context(), &s); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, s)
}

func (h *Subscriptions) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Subscriptions().Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// Users handles user API endpoints.
type Users struct {
	store *sqlite.Store
}

func NewUsers(store *sqlite.Store) *Users {
	return &Users{store: store}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *Users) Login(c *mizu.Ctx) error {
	var req LoginRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	user, err := h.store.Users().GetByEmail(c.Request().Context(), req.Email)
	if err != nil || user == nil {
		return c.JSON(401, map[string]string{"error": "Invalid credentials"})
	}

	// Verify password using Argon2id
	if err := password.Verify(req.Password, user.PasswordHash); err != nil {
		return c.JSON(401, map[string]string{"error": "Invalid credentials"})
	}

	// Check if password needs rehash with updated parameters
	if password.NeedsRehash(user.PasswordHash, nil) {
		if newHash, err := password.Hash(req.Password, nil); err == nil {
			user.PasswordHash = newHash
			h.store.Users().Update(c.Request().Context(), user)
		}
	}

	// Create session
	token := make([]byte, 32)
	rand.Read(token)
	tokenStr := hex.EncodeToString(token)

	session := &store.Session{
		UserID:    user.ID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	h.store.Users().CreateSession(c.Request().Context(), session)
	h.store.Users().UpdateLastLogin(c.Request().Context(), user.ID)

	return c.JSON(200, map[string]interface{}{
		"token": tokenStr,
		"user":  user,
	})
}

func (h *Users) Logout(c *mizu.Ctx) error {
	token := c.Request().Header.Get("Authorization")
	if token != "" {
		h.store.Users().DeleteSession(c.Request().Context(), token)
	}
	return c.JSON(200, map[string]string{"status": "logged out"})
}

func (h *Users) Me(c *mizu.Ctx) error {
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	session, err := h.store.Users().GetSession(c.Request().Context(), token)
	if err != nil || session == nil {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	user, err := h.store.Users().GetByID(c.Request().Context(), session.UserID)
	if err != nil || user == nil {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	return c.JSON(200, user)
}

func (h *Users) List(c *mizu.Ctx) error {
	users, err := h.store.Users().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, users)
}

func (h *Users) Create(c *mizu.Ctx) error {
	var req CreateUserRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return c.JSON(400, map[string]string{"error": "Email, password, and name are required"})
	}

	// Hash the password using Argon2id
	passwordHash, err := password.Hash(req.Password, nil)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "Failed to hash password"})
	}

	user := &store.User{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: passwordHash,
		Role:         req.Role,
	}
	if user.Role == "" {
		user.Role = "user"
	}

	if err := h.store.Users().Create(c.Request().Context(), user); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Don't return the password hash
	user.PasswordHash = ""
	return c.JSON(201, user)
}

func (h *Users) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var user store.User
	if err := c.BindJSON(&user, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	user.ID = id
	if err := h.store.Users().Update(c.Request().Context(), &user); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, user)
}

func (h *Users) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Users().Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// Settings handles settings API endpoints.
type Settings struct {
	store *sqlite.Store
}

func NewSettings(store *sqlite.Store) *Settings {
	return &Settings{store: store}
}

func (h *Settings) List(c *mizu.Ctx) error {
	settings, err := h.store.Settings().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, settings)
}

func (h *Settings) Update(c *mizu.Ctx) error {
	var settings map[string]string
	if err := c.BindJSON(&settings, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	for key, value := range settings {
		if err := h.store.Settings().Set(c.Request().Context(), key, value); err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
	}
	return c.JSON(200, map[string]string{"status": "updated"})
}

func (h *Settings) AuditLogs(c *mizu.Ctx) error {
	logs, err := h.store.Settings().ListAuditLogs(c.Request().Context(), 100, 0)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, logs)
}
