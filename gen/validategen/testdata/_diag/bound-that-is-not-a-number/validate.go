package broken

//mizu:validate
type Listing struct {
	Q string `json:"q" validate:"min=lots"`
}
