package broken

//mizu:bind
type Listing struct {
	Q      string `bind:"-"`
	Secret string `json:"-"`
	page   int
}
