package broken

//mizu:validate
type Listing struct {
	Q string `json:"q" validate:"min=3=4"`
}
