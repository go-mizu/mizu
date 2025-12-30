package hooks

import (
	"net/http"

	"github.com/go-mizu/blueprints/cms/config"
)

// Service implements the hook Executor interface.
type Service struct{}

// NewService creates a new hooks service.
func NewService() *Service {
	return &Service{}
}

// toConfigHookContext converts our HookContext to config.HookContext.
func toConfigHookContext(ctx *HookContext) *config.HookContext {
	var req *http.Request
	if ctx.Req != nil {
		req = ctx.Req
	}

	return &config.HookContext{
		Req:         req,
		Collection:  ctx.Collection,
		Operation:   string(ctx.Operation),
		ID:          ctx.ID,
		Data:        ctx.Data,
		OriginalDoc: ctx.OriginalDoc,
		User:        ctx.User,
		FindArgs:    toConfigFindArgs(ctx.FindArgs),
	}
}

func toConfigFindArgs(args *FindArgs) *config.FindArgs {
	if args == nil {
		return nil
	}
	return &config.FindArgs{
		Where:          args.Where,
		Sort:           args.Sort,
		Limit:          args.Limit,
		Page:           args.Page,
		Depth:          args.Depth,
		Locale:         args.Locale,
		FallbackLocale: args.FallbackLocale,
	}
}

// ExecuteBeforeOperation executes beforeOperation hooks.
func (s *Service) ExecuteBeforeOperation(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.BeforeOperation) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.BeforeOperation {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	// Sync any mutations back
	ctx.Data = configCtx.Data
	return nil
}

// ExecuteBeforeValidate executes beforeValidate hooks.
func (s *Service) ExecuteBeforeValidate(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.BeforeValidate) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.BeforeValidate {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	ctx.Data = configCtx.Data
	return nil
}

// ExecuteBeforeChange executes beforeChange hooks.
func (s *Service) ExecuteBeforeChange(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.BeforeChange) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.BeforeChange {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	ctx.Data = configCtx.Data
	return nil
}

// ExecuteAfterChange executes afterChange hooks.
func (s *Service) ExecuteAfterChange(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.AfterChange) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	configCtx.Data = ctx.Doc // AfterChange receives the result document
	for _, hook := range hooks.AfterChange {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteBeforeRead executes beforeRead hooks.
func (s *Service) ExecuteBeforeRead(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.BeforeRead) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.BeforeRead {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAfterRead executes afterRead hooks.
func (s *Service) ExecuteAfterRead(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.AfterRead) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	configCtx.Data = ctx.Doc
	for _, hook := range hooks.AfterRead {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	// Sync mutations
	ctx.Doc = configCtx.Data
	return nil
}

// ExecuteBeforeDelete executes beforeDelete hooks.
func (s *Service) ExecuteBeforeDelete(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.BeforeDelete) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.BeforeDelete {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAfterDelete executes afterDelete hooks.
func (s *Service) ExecuteAfterDelete(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.AfterDelete) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.AfterDelete {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAfterOperation executes afterOperation hooks.
func (s *Service) ExecuteAfterOperation(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.AfterOperation) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.AfterOperation {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAfterError executes afterError hooks.
func (s *Service) ExecuteAfterError(ctx *HookContext, hooks *config.CollectionHooks, opErr error) error {
	if hooks == nil || len(hooks.AfterError) == 0 {
		return opErr
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.AfterError {
		if err := hook(configCtx, opErr); err != nil {
			return err
		}
	}

	return opErr
}

// ExecuteBeforeLogin executes beforeLogin hooks.
func (s *Service) ExecuteBeforeLogin(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.BeforeLogin) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.BeforeLogin {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	ctx.Data = configCtx.Data
	return nil
}

// ExecuteAfterLogin executes afterLogin hooks.
func (s *Service) ExecuteAfterLogin(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.AfterLogin) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.AfterLogin {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAfterLogout executes afterLogout hooks.
func (s *Service) ExecuteAfterLogout(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.AfterLogout) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.AfterLogout {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAfterMe executes afterMe hooks.
func (s *Service) ExecuteAfterMe(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.AfterMe) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.AfterMe {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAfterRefresh executes afterRefresh hooks.
func (s *Service) ExecuteAfterRefresh(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.AfterRefresh) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.AfterRefresh {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAfterForgotPassword executes afterForgotPassword hooks.
func (s *Service) ExecuteAfterForgotPassword(ctx *HookContext, hooks *config.CollectionHooks) error {
	if hooks == nil || len(hooks.AfterForgotPassword) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.AfterForgotPassword {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteGlobalBeforeChange executes global beforeChange hooks.
func (s *Service) ExecuteGlobalBeforeChange(ctx *HookContext, hooks *config.GlobalHooks) error {
	if hooks == nil || len(hooks.BeforeChange) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.BeforeChange {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	ctx.Data = configCtx.Data
	return nil
}

// ExecuteGlobalAfterChange executes global afterChange hooks.
func (s *Service) ExecuteGlobalAfterChange(ctx *HookContext, hooks *config.GlobalHooks) error {
	if hooks == nil || len(hooks.AfterChange) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	configCtx.Data = ctx.Doc
	for _, hook := range hooks.AfterChange {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteGlobalBeforeRead executes global beforeRead hooks.
func (s *Service) ExecuteGlobalBeforeRead(ctx *HookContext, hooks *config.GlobalHooks) error {
	if hooks == nil || len(hooks.BeforeRead) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	for _, hook := range hooks.BeforeRead {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteGlobalAfterRead executes global afterRead hooks.
func (s *Service) ExecuteGlobalAfterRead(ctx *HookContext, hooks *config.GlobalHooks) error {
	if hooks == nil || len(hooks.AfterRead) == 0 {
		return nil
	}

	configCtx := toConfigHookContext(ctx)
	configCtx.Data = ctx.Doc
	for _, hook := range hooks.AfterRead {
		if err := hook(configCtx); err != nil {
			return err
		}
	}

	ctx.Doc = configCtx.Data
	return nil
}
