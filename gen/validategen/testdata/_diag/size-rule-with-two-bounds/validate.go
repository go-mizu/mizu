package broken

//mizu:validate
type Listing struct {
	Limit int `json:"limit" validate:"min=1 10"`
}
