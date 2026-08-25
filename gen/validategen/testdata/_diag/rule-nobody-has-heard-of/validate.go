package broken

//mizu:validate
type Listing struct {
	Email string `json:"email" validate:"required,emial"`
}
