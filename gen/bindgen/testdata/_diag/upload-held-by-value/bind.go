package broken

import "github.com/go-mizu/mizu/web"

//mizu:bind
type Profile struct {
	Avatar web.Upload `form:"avatar"`
}
