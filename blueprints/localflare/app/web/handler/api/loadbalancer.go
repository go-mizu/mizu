package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// LoadBalancer handles load balancer requests.
type LoadBalancer struct {
	store store.LoadBalancerStore
}

// NewLoadBalancer creates a new LoadBalancer handler.
func NewLoadBalancer(store store.LoadBalancerStore) *LoadBalancer {
	return &LoadBalancer{store: store}
}

// List lists all load balancers for a zone.
func (h *LoadBalancer) List(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	lbs, err := h.store.List(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  lbs,
	})
}

// Get retrieves a load balancer by ID.
func (h *LoadBalancer) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	lb, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Load balancer not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  lb,
	})
}

// CreateLoadBalancerInput is the input for creating a load balancer.
type CreateLoadBalancerInput struct {
	Name            string   `json:"name"`
	FallbackPool    string   `json:"fallback_pool"`
	DefaultPools    []string `json:"default_pools"`
	SessionAffinity string   `json:"session_affinity"`
	SteeringPolicy  string   `json:"steering_policy"`
	Enabled         bool     `json:"enabled"`
}

// Create creates a new load balancer.
func (h *LoadBalancer) Create(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input CreateLoadBalancerInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "Name is required"})
	}

	lb := &store.LoadBalancer{
		ID:              ulid.Make().String(),
		ZoneID:          zoneID,
		Name:            input.Name,
		Fallback:        input.FallbackPool,
		DefaultPools:    input.DefaultPools,
		SessionAffinity: input.SessionAffinity,
		SteeringPolicy:  input.SteeringPolicy,
		Enabled:         input.Enabled,
		CreatedAt:       time.Now(),
	}

	if lb.SessionAffinity == "" {
		lb.SessionAffinity = "none"
	}
	if lb.SteeringPolicy == "" {
		lb.SteeringPolicy = "off"
	}

	if err := h.store.Create(c.Request().Context(), lb); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  lb,
	})
}

// Update updates a load balancer.
func (h *LoadBalancer) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	lb, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Load balancer not found"})
	}

	var input CreateLoadBalancerInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name != "" {
		lb.Name = input.Name
	}
	if input.FallbackPool != "" {
		lb.Fallback = input.FallbackPool
	}
	if input.DefaultPools != nil {
		lb.DefaultPools = input.DefaultPools
	}
	if input.SessionAffinity != "" {
		lb.SessionAffinity = input.SessionAffinity
	}
	if input.SteeringPolicy != "" {
		lb.SteeringPolicy = input.SteeringPolicy
	}
	lb.Enabled = input.Enabled

	if err := h.store.Update(c.Request().Context(), lb); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  lb,
	})
}

// Delete deletes a load balancer.
func (h *LoadBalancer) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListPools lists all origin pools.
func (h *LoadBalancer) ListPools(c *mizu.Ctx) error {
	pools, err := h.store.ListPools(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  pools,
	})
}

// GetPool retrieves an origin pool by ID.
func (h *LoadBalancer) GetPool(c *mizu.Ctx) error {
	id := c.Param("id")
	pool, err := h.store.GetPool(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Pool not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  pool,
	})
}

// CreatePoolInput is the input for creating an origin pool.
type CreatePoolInput struct {
	Name              string         `json:"name"`
	Origins           []store.Origin `json:"origins"`
	CheckRegions      []string       `json:"check_regions"`
	Description       string         `json:"description"`
	Enabled           bool           `json:"enabled"`
	MinimumOrigins    int            `json:"minimum_origins"`
	Monitor           string         `json:"monitor"`
	NotificationEmail string         `json:"notification_email"`
}

// CreatePool creates a new origin pool.
func (h *LoadBalancer) CreatePool(c *mizu.Ctx) error {
	var input CreatePoolInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "Name is required"})
	}

	pool := &store.OriginPool{
		ID:           ulid.Make().String(),
		Name:         input.Name,
		Origins:      input.Origins,
		CheckRegions: input.CheckRegions,
		Description:  input.Description,
		Enabled:      input.Enabled,
		MinOrigins:   input.MinimumOrigins,
		Monitor:      input.Monitor,
		NotifyEmail:  input.NotificationEmail,
		CreatedAt:    time.Now(),
	}

	if pool.MinOrigins == 0 {
		pool.MinOrigins = 1
	}

	if err := h.store.CreatePool(c.Request().Context(), pool); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  pool,
	})
}

// UpdatePool updates an origin pool.
func (h *LoadBalancer) UpdatePool(c *mizu.Ctx) error {
	id := c.Param("id")
	pool, err := h.store.GetPool(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Pool not found"})
	}

	var input CreatePoolInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name != "" {
		pool.Name = input.Name
	}
	if input.Origins != nil {
		pool.Origins = input.Origins
	}
	if input.CheckRegions != nil {
		pool.CheckRegions = input.CheckRegions
	}
	pool.Description = input.Description
	pool.Enabled = input.Enabled
	if input.MinimumOrigins != 0 {
		pool.MinOrigins = input.MinimumOrigins
	}
	pool.Monitor = input.Monitor
	pool.NotifyEmail = input.NotificationEmail

	if err := h.store.UpdatePool(c.Request().Context(), pool); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  pool,
	})
}

// DeletePool deletes an origin pool.
func (h *LoadBalancer) DeletePool(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeletePool(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListHealthChecks lists all health checks.
func (h *LoadBalancer) ListHealthChecks(c *mizu.Ctx) error {
	checks, err := h.store.ListHealthChecks(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  checks,
	})
}

// CreateHealthCheckInput is the input for creating a health check.
type CreateHealthCheckInput struct {
	Description     string              `json:"description"`
	Type            string              `json:"type"`
	Method          string              `json:"method"`
	Path            string              `json:"path"`
	Header          map[string][]string `json:"header"`
	Port            int                 `json:"port"`
	Timeout         int                 `json:"timeout"`
	Retries         int                 `json:"retries"`
	Interval        int                 `json:"interval"`
	ExpectedBody    string              `json:"expected_body"`
	ExpectedCodes   string              `json:"expected_codes"`
	FollowRedirects bool                `json:"follow_redirects"`
	AllowInsecure   bool                `json:"allow_insecure"`
}

// CreateHealthCheck creates a new health check.
func (h *LoadBalancer) CreateHealthCheck(c *mizu.Ctx) error {
	var input CreateHealthCheckInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	hc := &store.HealthCheck{
		ID:              ulid.Make().String(),
		Description:     input.Description,
		Type:            input.Type,
		Method:          input.Method,
		Path:            input.Path,
		Header:          input.Header,
		Port:            input.Port,
		Timeout:         input.Timeout,
		Retries:         input.Retries,
		Interval:        input.Interval,
		ExpectedBody:    input.ExpectedBody,
		ExpectedCodes:   input.ExpectedCodes,
		FollowRedirects: input.FollowRedirects,
		AllowInsecure:   input.AllowInsecure,
	}

	// Set defaults
	if hc.Type == "" {
		hc.Type = "http"
	}
	if hc.Method == "" {
		hc.Method = "GET"
	}
	if hc.Path == "" {
		hc.Path = "/"
	}
	if hc.Port == 0 {
		hc.Port = 80
	}
	if hc.Timeout == 0 {
		hc.Timeout = 5
	}
	if hc.Retries == 0 {
		hc.Retries = 2
	}
	if hc.Interval == 0 {
		hc.Interval = 60
	}
	if hc.ExpectedCodes == "" {
		hc.ExpectedCodes = "200"
	}

	if err := h.store.CreateHealthCheck(c.Request().Context(), hc); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  hc,
	})
}
