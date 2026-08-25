package broken

//mizu:validate
type Listing struct {
	Tags []string `json:"tags" validate:"dive,dive,required"`
}
