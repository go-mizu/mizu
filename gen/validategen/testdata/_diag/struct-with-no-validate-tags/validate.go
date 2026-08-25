package broken

//mizu:validate
type Listing struct {
	Q      string `json:"q"`
	Page   int    `json:"page" validate:"-"`
	Secret string `json:"-" validate:"required"`
	limit  int    `validate:"min=1"`
}
