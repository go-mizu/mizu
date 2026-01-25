// Package bang provides bang shortcut parsing and handling.
package bang

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// Service handles bang parsing and resolution.
type Service struct {
	store store.BangStore
}

// NewService creates a new bang service.
func NewService(st store.BangStore) *Service {
	return &Service{store: st}
}

// bangPattern matches bang shortcuts in queries.
// Supports: !bang query, bang! query, query !bang, query bang!
var bangPattern = regexp.MustCompile(`(?:^!(\w+)\s+|^(\w+)!\s+|\s+!(\w+)$|\s+(\w+)!$)`)

// Parse extracts bang from a query and returns the result.
func (s *Service) Parse(ctx context.Context, query string) (*types.BangResult, error) {
	result := &types.BangResult{
		OrigQuery: query,
		Query:     query,
	}

	// Check for "feeling lucky" pattern: "! query" or "query !"
	if strings.HasPrefix(query, "! ") {
		result.Query = strings.TrimPrefix(query, "! ")
		result.Internal = true
		result.Category = "lucky"
		return result, nil
	}
	if strings.HasSuffix(query, " !") {
		result.Query = strings.TrimSuffix(query, " !")
		result.Internal = true
		result.Category = "lucky"
		return result, nil
	}

	// Extract bang trigger from query
	trigger := extractBang(query)
	if trigger == "" {
		return result, nil
	}

	// Clean query (remove bang)
	result.Query = removeBang(query, trigger)

	// Check for internal bangs (images, news, videos, maps)
	if category, ok := types.InternalBangs[trigger]; ok {
		result.Internal = true
		result.Category = category
		return result, nil
	}

	// Check for AI bangs
	for _, aiBang := range types.AIBangs {
		if trigger == aiBang {
			result.Internal = true
			result.Category = "ai"
			return result, nil
		}
	}

	// Check for summarizer bangs
	for _, sumBang := range types.SummarizerBangs {
		if trigger == sumBang {
			result.Internal = true
			result.Category = "summarize"
			return result, nil
		}
	}

	// Check for time filter bangs
	if timeRange, ok := types.TimeFilterBangs[trigger]; ok {
		result.Internal = true
		result.Category = "time:" + timeRange
		return result, nil
	}

	// Look up custom bang in database
	bang, err := s.store.GetBang(ctx, trigger)
	if err != nil {
		return nil, err
	}
	if bang != nil {
		result.Bang = bang
		result.RedirectURL = buildRedirectURL(bang.URLTemplate, result.Query)
		return result, nil
	}

	// Check built-in external bangs
	if builtIn, ok := types.ExternalBangs[trigger]; ok {
		result.Bang = &types.Bang{
			Trigger:     builtIn.Trigger,
			Name:        builtIn.Name,
			URLTemplate: builtIn.URLTemplate,
			Category:    builtIn.Category,
			IsBuiltin:   true,
		}
		result.RedirectURL = buildRedirectURL(builtIn.URLTemplate, result.Query)
		return result, nil
	}

	// No bang found, restore original query
	result.Query = query
	return result, nil
}

// extractBang extracts the bang trigger from a query.
func extractBang(query string) string {
	query = strings.TrimSpace(query)

	// Check prefix: !bang query
	if strings.HasPrefix(query, "!") {
		parts := strings.SplitN(query, " ", 2)
		if len(parts) >= 1 {
			return strings.TrimPrefix(parts[0], "!")
		}
	}

	// Check prefix: bang! query
	if idx := strings.Index(query, "! "); idx > 0 && idx < 20 {
		prefix := query[:idx]
		if !strings.Contains(prefix, " ") {
			return prefix
		}
	}

	// Check suffix: query !bang
	if idx := strings.LastIndex(query, " !"); idx > 0 {
		suffix := query[idx+2:]
		if !strings.Contains(suffix, " ") {
			return suffix
		}
	}

	// Check suffix: query bang!
	if strings.HasSuffix(query, "!") {
		parts := strings.Split(query, " ")
		last := parts[len(parts)-1]
		if strings.HasSuffix(last, "!") {
			return strings.TrimSuffix(last, "!")
		}
	}

	return ""
}

// removeBang removes the bang from a query.
func removeBang(query, trigger string) string {
	query = strings.TrimSpace(query)

	// Remove !bang from prefix
	query = strings.TrimPrefix(query, "!"+trigger+" ")
	query = strings.TrimPrefix(query, "!"+trigger)

	// Remove bang! from prefix
	query = strings.TrimPrefix(query, trigger+"! ")
	query = strings.TrimPrefix(query, trigger+"!")

	// Remove !bang from suffix
	query = strings.TrimSuffix(query, " !"+trigger)
	query = strings.TrimSuffix(query, "!"+trigger)

	// Remove bang! from suffix
	query = strings.TrimSuffix(query, " "+trigger+"!")
	query = strings.TrimSuffix(query, trigger+"!")

	return strings.TrimSpace(query)
}

// buildRedirectURL builds the redirect URL from a template.
func buildRedirectURL(template, query string) string {
	encoded := url.QueryEscape(query)
	return strings.ReplaceAll(template, "{query}", encoded)
}

// List returns all available bangs.
func (s *Service) List(ctx context.Context) ([]*types.Bang, error) {
	bangs, err := s.store.ListBangs(ctx)
	if err != nil {
		return nil, err
	}

	// Add built-in bangs
	for _, b := range types.ExternalBangs {
		bangs = append(bangs, &types.Bang{
			Trigger:     b.Trigger,
			Name:        b.Name,
			URLTemplate: b.URLTemplate,
			Category:    b.Category,
			IsBuiltin:   true,
		})
	}

	return bangs, nil
}

// Create creates a new custom bang.
func (s *Service) Create(ctx context.Context, bang *types.Bang) error {
	return s.store.CreateBang(ctx, bang)
}

// Delete removes a custom bang.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.DeleteBang(ctx, id)
}

// SeedBuiltins seeds the database with built-in bangs.
func (s *Service) SeedBuiltins(ctx context.Context) error {
	return s.store.SeedBuiltinBangs(ctx)
}
