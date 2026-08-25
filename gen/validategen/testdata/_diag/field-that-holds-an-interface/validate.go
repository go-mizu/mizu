package broken

//mizu:validate
type Listing struct {
	Extra any `json:"extra" validate:"required"`
}
