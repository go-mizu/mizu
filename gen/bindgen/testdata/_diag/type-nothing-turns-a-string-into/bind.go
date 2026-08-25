package broken

//mizu:bind
type Listing struct {
	Done  chan int   `query:"done"`
	Ratio complex128 `query:"ratio"`
}
