package broken

//mizu:bind
type Listing struct {
	Q    string `query:"q"`
	Also string `form:"q"`
}
