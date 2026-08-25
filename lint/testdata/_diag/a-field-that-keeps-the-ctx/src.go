package app

import "github.com/go-mizu/mizu/web"

// An upload is a file somebody is part way through sending, kept between the
// request that started it and the one that finishes it.
type upload struct {
	id  string
	got int64
	req *web.Ctx
}

// Start begins one.
func Start(c *web.Ctx) error {
	uploads[c.Param("id")] = &upload{id: c.Param("id"), req: c}
	return c.NoContent()
}

var uploads = map[string]*upload{}
