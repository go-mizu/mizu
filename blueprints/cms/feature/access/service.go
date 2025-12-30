package access

import (
	"github.com/go-mizu/blueprints/cms/config"
)

// Service implements the access Checker interface.
type Service struct{}

// NewService creates a new access control service.
func NewService() *Service {
	return &Service{}
}

// toConfigAccessContext converts our AccessContext to config.AccessContext.
func toConfigAccessContext(ctx *AccessContext) *config.AccessContext {
	return &config.AccessContext{
		Req:        ctx.Req,
		User:       ctx.User,
		ID:         ctx.ID,
		Data:       ctx.Data,
		Collection: ctx.Collection,
	}
}

// fromConfigAccessResult converts config.AccessResult to our Result.
func fromConfigAccessResult(r *config.AccessResult) *Result {
	if r == nil {
		return &Result{Allowed: false}
	}
	return &Result{
		Allowed: r.Allowed,
		Where:   r.Where,
	}
}

// defaultAccessResult returns default access (allow if user exists).
func defaultAccessResult(ctx *AccessContext) *Result {
	// Default: allow if user is authenticated
	if ctx.User != nil {
		return &Result{Allowed: true}
	}
	return &Result{Allowed: false}
}

// CanCreate checks if the user can create documents.
func (s *Service) CanCreate(ctx *AccessContext, access *config.AccessConfig) (*Result, error) {
	if access == nil || access.Create == nil {
		return defaultAccessResult(ctx), nil
	}

	configCtx := toConfigAccessContext(ctx)
	result, err := access.Create(configCtx)
	if err != nil {
		return nil, err
	}

	return fromConfigAccessResult(result), nil
}

// CanRead checks if the user can read documents.
func (s *Service) CanRead(ctx *AccessContext, access *config.AccessConfig) (*Result, error) {
	if access == nil || access.Read == nil {
		return defaultAccessResult(ctx), nil
	}

	configCtx := toConfigAccessContext(ctx)
	result, err := access.Read(configCtx)
	if err != nil {
		return nil, err
	}

	return fromConfigAccessResult(result), nil
}

// CanUpdate checks if the user can update documents.
func (s *Service) CanUpdate(ctx *AccessContext, access *config.AccessConfig) (*Result, error) {
	if access == nil || access.Update == nil {
		return defaultAccessResult(ctx), nil
	}

	configCtx := toConfigAccessContext(ctx)
	result, err := access.Update(configCtx)
	if err != nil {
		return nil, err
	}

	return fromConfigAccessResult(result), nil
}

// CanDelete checks if the user can delete documents.
func (s *Service) CanDelete(ctx *AccessContext, access *config.AccessConfig) (*Result, error) {
	if access == nil || access.Delete == nil {
		return defaultAccessResult(ctx), nil
	}

	configCtx := toConfigAccessContext(ctx)
	result, err := access.Delete(configCtx)
	if err != nil {
		return nil, err
	}

	return fromConfigAccessResult(result), nil
}

// CanAdmin checks if the user has admin access.
func (s *Service) CanAdmin(ctx *AccessContext, access *config.AccessConfig) (*Result, error) {
	if access == nil || access.Admin == nil {
		// Default: admin access requires explicit permission
		return &Result{Allowed: false}, nil
	}

	configCtx := toConfigAccessContext(ctx)
	result, err := access.Admin(configCtx)
	if err != nil {
		return nil, err
	}

	return fromConfigAccessResult(result), nil
}

// ApplyAccessFilter merges access control WHERE constraints with existing query.
func (s *Service) ApplyAccessFilter(where map[string]any, accessWhere map[string]any) map[string]any {
	if accessWhere == nil || len(accessWhere) == 0 {
		return where
	}

	if where == nil || len(where) == 0 {
		return accessWhere
	}

	// Combine with AND logic
	return map[string]any{
		"and": []map[string]any{where, accessWhere},
	}
}
