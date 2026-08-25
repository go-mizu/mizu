package broken

//mizu:validate
type Listing struct {
	Limit int `json:"limit" validate:"between=few 10"`
}
