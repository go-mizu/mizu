package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/app/web/middleware"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/oklog/ulid/v2"
)

// Supabase-compatible CORS headers for Edge Functions
var functionsCORSHeaders = map[string]string{
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type, accept, accept-language, x-authorization, x-region",
	"Access-Control-Allow-Methods": "POST, GET, OPTIONS, PUT, DELETE, PATCH",
}

// Default region for edge functions
const defaultRegion = "us-east-1"

// Valid regions for edge functions
var validRegions = map[string]bool{
	"us-east-1":      true,
	"us-west-1":      true,
	"us-west-2":      true,
	"ca-central-1":   true,
	"eu-west-1":      true,
	"eu-west-2":      true,
	"eu-west-3":      true,
	"eu-central-1":   true,
	"ap-northeast-1": true,
	"ap-northeast-2": true,
	"ap-south-1":     true,
	"ap-southeast-1": true,
	"ap-southeast-2": true,
	"sa-east-1":      true,
}

// FunctionsHandler handles edge functions endpoints.
type FunctionsHandler struct {
	store *postgres.Store
}

// NewFunctionsHandler creates a new functions handler.
func NewFunctionsHandler(store *postgres.Store) *FunctionsHandler {
	return &FunctionsHandler{store: store}
}

// ListFunctions lists all functions.
func (h *FunctionsHandler) ListFunctions(c *mizu.Ctx) error {
	functions, err := h.store.Functions().ListFunctions(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list functions"})
	}

	// Ensure we return an empty array instead of null
	if functions == nil {
		functions = []*store.Function{}
	}

	return c.JSON(200, functions)
}

// CreateFunction creates a new function.
func (h *FunctionsHandler) CreateFunction(c *mizu.Ctx) error {
	var req struct {
		Name       string `json:"name"`
		Entrypoint string `json:"entrypoint"`
		ImportMap  string `json:"import_map"`
		VerifyJWT  bool   `json:"verify_jwt"`
		SourceCode string `json:"source_code"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}

	entrypoint := req.Entrypoint
	if entrypoint == "" {
		entrypoint = "index.ts"
	}

	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))

	fn := &store.Function{
		ID:         ulid.Make().String(),
		Name:       req.Name,
		Slug:       slug,
		Version:    1,
		Status:     "active",
		Entrypoint: entrypoint,
		ImportMap:  req.ImportMap,
		VerifyJWT:  req.VerifyJWT,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.store.Functions().CreateFunction(c.Context(), fn); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return c.JSON(400, map[string]string{"error": "function already exists"})
		}
		return c.JSON(500, map[string]string{"error": "failed to create function"})
	}

	// Create initial deployment if source code provided
	if req.SourceCode != "" {
		deployment := &store.Deployment{
			ID:         ulid.Make().String(),
			FunctionID: fn.ID,
			Version:    1,
			SourceCode: req.SourceCode,
			Status:     "deployed",
			DeployedAt: time.Now(),
		}
		_ = h.store.Functions().CreateDeployment(c.Context(), deployment)
	}

	return c.JSON(201, fn)
}

// GetFunction gets a function by ID.
func (h *FunctionsHandler) GetFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	fn, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	return c.JSON(200, fn)
}

// UpdateFunction updates a function.
func (h *FunctionsHandler) UpdateFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	fn, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	var req struct {
		Name       string `json:"name"`
		Entrypoint string `json:"entrypoint"`
		ImportMap  string `json:"import_map"`
		VerifyJWT  *bool  `json:"verify_jwt"`
		Status     string `json:"status"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name != "" {
		fn.Name = req.Name
		fn.Slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	}
	if req.Entrypoint != "" {
		fn.Entrypoint = req.Entrypoint
	}
	if req.ImportMap != "" {
		fn.ImportMap = req.ImportMap
	}
	if req.VerifyJWT != nil {
		fn.VerifyJWT = *req.VerifyJWT
	}
	if req.Status != "" {
		fn.Status = req.Status
	}
	fn.UpdatedAt = time.Now()

	if err := h.store.Functions().UpdateFunction(c.Context(), fn); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update function"})
	}

	return c.JSON(200, fn)
}

// DeleteFunction deletes a function.
func (h *FunctionsHandler) DeleteFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.Functions().DeleteFunction(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete function"})
	}

	return c.NoContent()
}

// DeployFunction deploys a new version of a function.
func (h *FunctionsHandler) DeployFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	fn, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	var req struct {
		SourceCode string `json:"source_code"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.SourceCode == "" {
		return c.JSON(400, map[string]string{"error": "source_code required"})
	}

	// Increment version
	fn.Version++
	fn.UpdatedAt = time.Now()
	_ = h.store.Functions().UpdateFunction(c.Context(), fn)

	// Create deployment
	deployment := &store.Deployment{
		ID:         ulid.Make().String(),
		FunctionID: fn.ID,
		Version:    fn.Version,
		SourceCode: req.SourceCode,
		Status:     "deployed",
		DeployedAt: time.Now(),
	}

	if err := h.store.Functions().CreateDeployment(c.Context(), deployment); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create deployment"})
	}

	return c.JSON(201, deployment)
}

// ListDeployments lists deployments for a function.
func (h *FunctionsHandler) ListDeployments(c *mizu.Ctx) error {
	id := c.Param("id")
	limit := queryInt(c, "limit", 10)

	deployments, err := h.store.Functions().ListDeployments(c.Context(), id, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list deployments"})
	}

	// Ensure we return an empty array instead of null
	if deployments == nil {
		deployments = []*store.Deployment{}
	}

	return c.JSON(200, deployments)
}

// InvokeFunctionOptions handles OPTIONS preflight for function invocation.
func (h *FunctionsHandler) InvokeFunctionOptions(c *mizu.Ctx) error {
	// Set CORS headers for preflight
	for k, v := range functionsCORSHeaders {
		c.Header().Set(k, v)
	}
	return c.NoContent()
}

// InvokeFunction invokes a function (supports all HTTP methods).
func (h *FunctionsHandler) InvokeFunction(c *mizu.Ctx) error {
	name := c.Param("name")

	// Set CORS headers on all responses
	for k, v := range functionsCORSHeaders {
		c.Header().Set(k, v)
	}

	// Handle OPTIONS preflight
	if c.Request().Method == http.MethodOptions {
		return c.NoContent()
	}

	// Determine region from request
	region := h.getRequestRegion(c)
	c.Header().Set("x-sb-edge-region", region)

	// Look up function by slug or name
	fn, err := h.store.Functions().GetFunctionByName(c.Context(), name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error":   "Not Found",
			"message": "function not found",
		})
	}

	// Check if function is active
	if fn.Status != "active" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error":   "Service Unavailable",
			"message": "function is not active",
		})
	}

	// JWT verification if required
	if fn.VerifyJWT {
		role := middleware.GetRole(c)
		// Service role always allowed
		if role != "service_role" && role != "authenticated" {
			// Check if there's a valid JWT
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "Unauthorized",
					"message": "authorization required",
				})
			}
		}
	}

	// Get latest deployment
	deployment, err := h.store.Functions().GetLatestDeployment(c.Context(), fn.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "Internal Server Error",
			"message": "no deployment found",
		})
	}

	// Extract subpath for routing (path after function name)
	subpath := c.Param("subpath")

	// In production, we'd execute the function using Deno/Bun runtime
	// For now, return a placeholder response that mimics function execution
	response := map[string]any{
		"message":     "Function executed",
		"function":    fn.Name,
		"version":     deployment.Version,
		"method":      c.Request().Method,
		"region":      region,
		"executed_at": time.Now().Format(time.RFC3339),
	}

	// Include subpath if present
	if subpath != "" {
		response["path"] = "/" + subpath
	}

	return c.JSON(http.StatusOK, response)
}

// InvokeFunctionWithPath invokes a function with a subpath (supports all HTTP methods).
func (h *FunctionsHandler) InvokeFunctionWithPath(c *mizu.Ctx) error {
	return h.InvokeFunction(c)
}

// getRequestRegion determines the region from the request.
func (h *FunctionsHandler) getRequestRegion(c *mizu.Ctx) string {
	// Check x-region header first
	region := c.Request().Header.Get("x-region")
	if region != "" && validRegions[region] {
		return region
	}

	// Check forceFunctionRegion query parameter
	region = c.Query("forceFunctionRegion")
	if region != "" && validRegions[region] {
		return region
	}

	// Default region
	return defaultRegion
}

// ListSecrets lists all secrets (names only).
func (h *FunctionsHandler) ListSecrets(c *mizu.Ctx) error {
	secrets, err := h.store.Functions().ListSecrets(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list secrets"})
	}

	// Ensure we return an empty array instead of null
	if secrets == nil {
		secrets = []*store.Secret{}
	}

	return c.JSON(200, secrets)
}

// CreateSecret creates or updates a secret.
func (h *FunctionsHandler) CreateSecret(c *mizu.Ctx) error {
	var req struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" || req.Value == "" {
		return c.JSON(400, map[string]string{"error": "name and value required"})
	}

	secret := &store.Secret{
		ID:        ulid.Make().String(),
		Name:      req.Name,
		Value:     req.Value, // In production, encrypt this
		CreatedAt: time.Now(),
	}

	if err := h.store.Functions().CreateSecret(c.Context(), secret); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create secret"})
	}

	return c.JSON(201, map[string]string{
		"name":       secret.Name,
		"created_at": secret.CreatedAt.Format(time.RFC3339),
	})
}

// DeleteSecret deletes a secret.
func (h *FunctionsHandler) DeleteSecret(c *mizu.Ctx) error {
	name := c.Param("name")

	if err := h.store.Functions().DeleteSecret(c.Context(), name); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete secret"})
	}

	return c.NoContent()
}

// BulkUpdateSecrets creates or updates multiple secrets at once.
func (h *FunctionsHandler) BulkUpdateSecrets(c *mizu.Ctx) error {
	var req struct {
		Secrets []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"secrets"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	secrets := make([]*store.Secret, 0, len(req.Secrets))
	for _, s := range req.Secrets {
		if s.Name == "" || s.Value == "" {
			continue
		}
		secrets = append(secrets, &store.Secret{
			ID:        ulid.Make().String(),
			Name:      s.Name,
			Value:     s.Value,
			CreatedAt: time.Now(),
		})
	}

	created, updated, err := h.store.Functions().BulkUpsertSecrets(c.Context(), secrets)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update secrets"})
	}

	return c.JSON(200, map[string]any{
		"created": created,
		"updated": updated,
		"total":   created + updated,
	})
}

// GetSource returns the current source code for a function.
func (h *FunctionsHandler) GetSource(c *mizu.Ctx) error {
	id := c.Param("id")

	fn, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	// Get the latest deployment for source code
	deployment, err := h.store.Functions().GetLatestDeployment(c.Context(), id)

	response := map[string]any{
		"function_id": fn.ID,
		"entrypoint":  fn.Entrypoint,
		"import_map":  fn.ImportMap,
		"version":     fn.Version,
		"is_draft":    fn.DraftSource != "",
	}

	// Use draft if available, otherwise use deployed source
	if fn.DraftSource != "" {
		response["source_code"] = fn.DraftSource
		if fn.DraftImportMap != "" {
			response["import_map"] = fn.DraftImportMap
		}
	} else if err == nil && deployment != nil {
		response["source_code"] = deployment.SourceCode
	} else {
		// Return default template
		response["source_code"] = getDefaultFunctionSource(fn.Name)
	}

	return c.JSON(200, response)
}

// UpdateSource saves source code as a draft without deploying.
func (h *FunctionsHandler) UpdateSource(c *mizu.Ctx) error {
	id := c.Param("id")

	_, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	var req struct {
		SourceCode string `json:"source_code"`
		ImportMap  string `json:"import_map"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if err := h.store.Functions().SaveDraftSource(c.Context(), id, req.SourceCode, req.ImportMap); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to save draft"})
	}

	return c.JSON(200, map[string]any{
		"saved":    true,
		"is_draft": true,
	})
}

// GetLogs returns execution logs for a function.
func (h *FunctionsHandler) GetLogs(c *mizu.Ctx) error {
	id := c.Param("id")
	limit := queryInt(c, "limit", 100)
	level := c.Query("level")

	var since *time.Time
	if sinceStr := c.Query("since"); sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			since = &t
		}
	}

	logs, err := h.store.Functions().ListFunctionLogs(c.Context(), id, limit, level, since)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to get logs"})
	}

	return c.JSON(200, map[string]any{"logs": logs})
}

// GetMetrics returns execution metrics for a function.
func (h *FunctionsHandler) GetMetrics(c *mizu.Ctx) error {
	id := c.Param("id")
	period := c.Query("period")
	if period == "" {
		period = "24h"
	}

	// Parse period
	var duration time.Duration
	switch period {
	case "1h":
		duration = time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		duration = 24 * time.Hour
	}

	to := time.Now().UTC()
	from := to.Add(-duration)

	metrics, err := h.store.Functions().GetFunctionMetrics(c.Context(), id, from, to)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to get metrics"})
	}

	// Aggregate the metrics
	var totalInvocations, totalSuccesses, totalErrors int
	var totalDuration int64
	byHour := make([]map[string]any, 0)

	for _, m := range metrics {
		totalInvocations += m.Invocations
		totalSuccesses += m.Successes
		totalErrors += m.Errors
		totalDuration += m.TotalDurationMs

		byHour = append(byHour, map[string]any{
			"hour":  m.Hour.Format(time.RFC3339),
			"count": m.Invocations,
		})
	}

	// Calculate average latency
	avgLatency := 0
	if totalInvocations > 0 {
		avgLatency = int(totalDuration / int64(totalInvocations))
	}

	return c.JSON(200, map[string]any{
		"function_id": id,
		"period":      period,
		"invocations": map[string]any{
			"total":   totalInvocations,
			"success": totalSuccesses,
			"error":   totalErrors,
			"by_hour": byHour,
		},
		"latency": map[string]any{
			"avg": avgLatency,
		},
	})
}

// TestFunction tests a function with a simulated request.
func (h *FunctionsHandler) TestFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	fn, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	var req struct {
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Headers map[string]string `json:"headers"`
		Body    any               `json:"body"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	// Simulate function execution
	startTime := time.Now()

	// In a real implementation, we would execute the function
	// For now, return a simulated response
	response := map[string]any{
		"status": 200,
		"headers": map[string]string{
			"Content-Type":       "application/json",
			"X-Function-Version": string(rune(fn.Version + '0')),
		},
		"body": map[string]any{
			"message":  "Function executed successfully",
			"function": fn.Name,
			"method":   req.Method,
			"path":     req.Path,
		},
		"duration_ms": int(time.Since(startTime).Milliseconds()) + 50, // Add simulated latency
		"logs": []map[string]any{
			{
				"level":     "info",
				"message":   "Function invoked",
				"timestamp": startTime.Format(time.RFC3339),
			},
			{
				"level":     "info",
				"message":   "Response sent",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	return c.JSON(200, response)
}

// ListTemplates returns available function templates.
func (h *FunctionsHandler) ListTemplates(c *mizu.Ctx) error {
	templates := []map[string]any{
		{
			"id":          "hello-world",
			"name":        "Hello World",
			"description": "Basic HTTP response function",
			"category":    "starter",
			"icon":        "wave",
		},
		{
			"id":          "stripe-webhook",
			"name":        "Stripe Webhook",
			"description": "Handle Stripe payment events",
			"category":    "integration",
			"icon":        "credit-card",
		},
		{
			"id":          "openai-proxy",
			"name":        "OpenAI Proxy",
			"description": "Proxy requests to OpenAI API",
			"category":    "integration",
			"icon":        "robot",
		},
		{
			"id":          "send-email",
			"name":        "Send Email",
			"description": "Send emails using Resend or SendGrid",
			"category":    "communication",
			"icon":        "mail",
		},
		{
			"id":          "file-upload",
			"name":        "File Upload",
			"description": "Handle file uploads to storage",
			"category":    "storage",
			"icon":        "upload",
		},
		{
			"id":          "auth-hook",
			"name":        "Auth Hook",
			"description": "Custom authentication logic",
			"category":    "auth",
			"icon":        "shield",
		},
		{
			"id":          "scheduled-task",
			"name":        "Scheduled Task",
			"description": "Cron-style background job",
			"category":    "background",
			"icon":        "clock",
		},
		{
			"id":          "database-trigger",
			"name":        "Database Trigger",
			"description": "Respond to database changes",
			"category":    "database",
			"icon":        "database",
		},
	}

	return c.JSON(200, map[string]any{"templates": templates})
}

// GetTemplate returns a specific function template with source code.
func (h *FunctionsHandler) GetTemplate(c *mizu.Ctx) error {
	templateID := c.Param("templateId")

	source, importMap := getTemplateSource(templateID)
	if source == "" {
		return c.JSON(404, map[string]string{"error": "template not found"})
	}

	return c.JSON(200, map[string]any{
		"id":          templateID,
		"source_code": source,
		"import_map":  importMap,
	})
}

// DownloadFunction returns the function source as a downloadable file.
func (h *FunctionsHandler) DownloadFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	fn, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	deployment, err := h.store.Functions().GetLatestDeployment(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "no deployment found"})
	}

	// Set download headers
	c.Header().Set("Content-Type", "application/typescript")
	c.Header().Set("Content-Disposition", "attachment; filename="+fn.Slug+".ts")

	return c.Text(200, deployment.SourceCode)
}

// getDefaultFunctionSource returns a default source template for new functions.
func getDefaultFunctionSource(name string) string {
	return `// Edge function: ` + name + `
import { serve } from "https://deno.land/std@0.208.0/http/server.ts";

serve(async (req: Request) => {
  const { method, url } = req;

  // Parse request body for POST/PUT/PATCH
  let body = null;
  if (["POST", "PUT", "PATCH"].includes(method)) {
    try {
      body = await req.json();
    } catch {
      // Not JSON body
    }
  }

  // Your function logic here
  const response = {
    message: "Hello from ` + name + `!",
    method,
    url,
    body,
    timestamp: new Date().toISOString(),
  };

  return new Response(JSON.stringify(response), {
    headers: { "Content-Type": "application/json" },
  });
});
`
}

// getTemplateSource returns source code for a template.
func getTemplateSource(templateID string) (source, importMap string) {
	templates := map[string]struct {
		source    string
		importMap string
	}{
		"hello-world": {
			source: `// Hello World Edge Function
import { serve } from "https://deno.land/std@0.208.0/http/server.ts";

serve(async (req: Request) => {
  const { name } = await req.json().catch(() => ({ name: "World" }));

  return new Response(
    JSON.stringify({ message: ` + "`Hello ${name}!`" + ` }),
    { headers: { "Content-Type": "application/json" } }
  );
});
`,
		},
		"stripe-webhook": {
			source: `// Stripe Webhook Handler
import { serve } from "https://deno.land/std@0.208.0/http/server.ts";
import Stripe from "https://esm.sh/stripe@14.10.0?target=deno";

const stripe = new Stripe(Deno.env.get("STRIPE_SECRET_KEY")!, {
  apiVersion: "2023-10-16",
});

serve(async (req: Request) => {
  const signature = req.headers.get("stripe-signature")!;
  const body = await req.text();

  let event: Stripe.Event;
  try {
    event = stripe.webhooks.constructEvent(
      body,
      signature,
      Deno.env.get("STRIPE_WEBHOOK_SECRET")!
    );
  } catch (err) {
    return new Response(JSON.stringify({ error: "Invalid signature" }), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });
  }

  // Handle the event
  switch (event.type) {
    case "checkout.session.completed":
      const session = event.data.object;
      console.log("Payment successful:", session.id);
      break;
    case "customer.subscription.updated":
      const subscription = event.data.object;
      console.log("Subscription updated:", subscription.id);
      break;
    default:
      console.log("Unhandled event type:", event.type);
  }

  return new Response(JSON.stringify({ received: true }), {
    headers: { "Content-Type": "application/json" },
  });
});
`,
		},
		"openai-proxy": {
			source: `// OpenAI API Proxy
import { serve } from "https://deno.land/std@0.208.0/http/server.ts";

const OPENAI_API_KEY = Deno.env.get("OPENAI_API_KEY");

serve(async (req: Request) => {
  const { messages, model = "gpt-4" } = await req.json();

  const response = await fetch("https://api.openai.com/v1/chat/completions", {
    method: "POST",
    headers: {
      "Authorization": ` + "`Bearer ${OPENAI_API_KEY}`" + `,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      model,
      messages,
      stream: false,
    }),
  });

  const data = await response.json();

  return new Response(JSON.stringify(data), {
    headers: { "Content-Type": "application/json" },
  });
});
`,
		},
		"send-email": {
			source: `// Send Email using Resend
import { serve } from "https://deno.land/std@0.208.0/http/server.ts";

const RESEND_API_KEY = Deno.env.get("RESEND_API_KEY");

serve(async (req: Request) => {
  const { to, subject, html } = await req.json();

  const response = await fetch("https://api.resend.com/emails", {
    method: "POST",
    headers: {
      "Authorization": ` + "`Bearer ${RESEND_API_KEY}`" + `,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      from: "noreply@example.com",
      to,
      subject,
      html,
    }),
  });

  const data = await response.json();

  return new Response(JSON.stringify(data), {
    status: response.ok ? 200 : 400,
    headers: { "Content-Type": "application/json" },
  });
});
`,
		},
		"file-upload": {
			source: `// File Upload Handler
import { serve } from "https://deno.land/std@0.208.0/http/server.ts";
import { createClient } from "https://esm.sh/@supabase/supabase-js@2";

serve(async (req: Request) => {
  const supabase = createClient(
    Deno.env.get("SUPABASE_URL")!,
    Deno.env.get("SUPABASE_SERVICE_ROLE_KEY")!
  );

  const formData = await req.formData();
  const file = formData.get("file") as File;

  if (!file) {
    return new Response(JSON.stringify({ error: "No file provided" }), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });
  }

  const { data, error } = await supabase.storage
    .from("uploads")
    .upload(file.name, file, {
      contentType: file.type,
    });

  if (error) {
    return new Response(JSON.stringify({ error: error.message }), {
      status: 500,
      headers: { "Content-Type": "application/json" },
    });
  }

  return new Response(JSON.stringify({ path: data.path }), {
    headers: { "Content-Type": "application/json" },
  });
});
`,
		},
		"auth-hook": {
			source: `// Custom Auth Hook
import { serve } from "https://deno.land/std@0.208.0/http/server.ts";

serve(async (req: Request) => {
  const { event, user } = await req.json();

  console.log("Auth event:", event);
  console.log("User:", user?.id);

  // Custom logic based on auth event
  switch (event) {
    case "SIGNED_IN":
      // User signed in - update last login, send notification, etc.
      break;
    case "SIGNED_UP":
      // New user - create profile, send welcome email, etc.
      break;
    case "PASSWORD_RECOVERY":
      // Password recovery requested
      break;
  }

  return new Response(JSON.stringify({ success: true }), {
    headers: { "Content-Type": "application/json" },
  });
});
`,
		},
		"scheduled-task": {
			source: `// Scheduled Task (Cron Job)
import { serve } from "https://deno.land/std@0.208.0/http/server.ts";
import { createClient } from "https://esm.sh/@supabase/supabase-js@2";

serve(async (req: Request) => {
  // Verify this is a scheduled invocation (optional)
  const authHeader = req.headers.get("Authorization");

  const supabase = createClient(
    Deno.env.get("SUPABASE_URL")!,
    Deno.env.get("SUPABASE_SERVICE_ROLE_KEY")!
  );

  // Example: Clean up old records
  const thirtyDaysAgo = new Date();
  thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);

  const { count, error } = await supabase
    .from("old_records")
    .delete()
    .lt("created_at", thirtyDaysAgo.toISOString());

  if (error) {
    console.error("Cleanup failed:", error);
    return new Response(JSON.stringify({ error: error.message }), {
      status: 500,
      headers: { "Content-Type": "application/json" },
    });
  }

  console.log("Cleaned up records:", count);

  return new Response(JSON.stringify({ deleted: count }), {
    headers: { "Content-Type": "application/json" },
  });
});
`,
		},
		"database-trigger": {
			source: `// Database Trigger Handler
import { serve } from "https://deno.land/std@0.208.0/http/server.ts";

interface DatabasePayload {
  type: "INSERT" | "UPDATE" | "DELETE";
  table: string;
  schema: string;
  record: Record<string, any>;
  old_record?: Record<string, any>;
}

serve(async (req: Request) => {
  const payload: DatabasePayload = await req.json();

  console.log("Database change:", payload.type, payload.table);

  switch (payload.type) {
    case "INSERT":
      // Handle new record
      console.log("New record:", payload.record);
      break;
    case "UPDATE":
      // Handle updated record
      console.log("Updated from:", payload.old_record);
      console.log("Updated to:", payload.record);
      break;
    case "DELETE":
      // Handle deleted record
      console.log("Deleted record:", payload.old_record);
      break;
  }

  return new Response(JSON.stringify({ processed: true }), {
    headers: { "Content-Type": "application/json" },
  });
});
`,
		},
	}

	if t, ok := templates[templateID]; ok {
		return t.source, t.importMap
	}
	return "", ""
}
