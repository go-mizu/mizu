package api

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	mizu "github.com/go-mizu/mizu"
)

var (
	jsHashOnce sync.Once
	jsHashMap  map[string]string
)

func buildJSHashes(staticFS fs.FS) {
	jsHashMap = make(map[string]string)
	entries, err := fs.ReadDir(staticFS, "static/js")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		data, err := fs.ReadFile(staticFS, "static/js/"+e.Name())
		if err != nil {
			continue
		}
		h := sha256.Sum256(data)
		jsHashMap[e.Name()] = fmt.Sprintf("%x", h[:4])
	}
}

func handleIndex(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		data, err := fs.ReadFile(d.StaticFS, "static/index.html")
		if err != nil {
			return c.Text(500, "internal error")
		}
		mode := "search"
		if d.Hub != nil {
			mode = "dashboard"
		}
		data = bytes.Replace(data, []byte(`"__SERVER_MODE__"`), []byte(`"`+mode+`"`), 1)
		data = bytes.Replace(data, []byte(`"__DEFAULT_ENGINE__"`), []byte(`"`+d.EngineName+`"`), 1)

		jsHashOnce.Do(func() { buildJSHashes(d.StaticFS) })
		for name, hash := range jsHashMap {
			old := fmt.Sprintf(`src="/static/js/%s"`, name)
			neu := fmt.Sprintf(`src="/static/js/%s?v=%s"`, name, hash)
			data = bytes.ReplaceAll(data, []byte(old), []byte(neu))
		}

		c.Header().Set("Content-Type", "text/html; charset=utf-8")
		c.Header().Set("Cache-Control", "no-cache")
		_, err = c.Writer().Write(data)
		return err
	}
}
