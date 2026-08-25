package broken

//mizu:validate
type Listing struct {
	Page int `json:"page" validate:"email"`
}
