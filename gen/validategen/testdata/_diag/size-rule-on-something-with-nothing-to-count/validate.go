package broken

//mizu:validate
type Listing struct {
	Draft bool `json:"draft" validate:"min=3"`
}
