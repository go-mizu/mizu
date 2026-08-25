package broken

import "time"

//mizu:validate
type Listing struct {
	Wait time.Duration `json:"wait" validate:"min=soon"`
}
