package broken

import "context"

//mizu:validate
type Listing struct {
	Q string `json:"q" validate:"required"`
}

func (v Listing) Validate(ctx context.Context) error { return nil }
