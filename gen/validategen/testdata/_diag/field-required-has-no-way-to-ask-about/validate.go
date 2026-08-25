package broken

import "unsafe"

//mizu:validate
type Listing struct {
	At unsafe.Pointer `json:"at" validate:"required"`
}
