package live

import (
	"html/template"
	"strings"
	"testing"
	"time"
)

func TestLiveOptions(t *testing.T) {
	t.Run("apply defaults", func(t *testing.T) {
		opts := Options{}
		opts.applyDefaults()

		if opts.SessionTimeout != DefaultSessionTimeout {
			t.Errorf("expected %v, got %v", DefaultSessionTimeout, opts.SessionTimeout)
		}
		if opts.HeartbeatInterval != DefaultHeartbeatInterval {
			t.Errorf("expected %v, got %v", DefaultHeartbeatInterval, opts.HeartbeatInterval)
		}
		if opts.MaxMessageSize != DefaultMaxMessageSize {
			t.Errorf("expected %d, got %d", DefaultMaxMessageSize, opts.MaxMessageSize)
		}
	})

	t.Run("preserve custom values", func(t *testing.T) {
		opts := Options{
			SessionTimeout:    5 * time.Minute,
			HeartbeatInterval: 10 * time.Second,
			MaxMessageSize:    1024,
		}
		opts.applyDefaults()

		if opts.SessionTimeout != 5*time.Minute {
			t.Error("should preserve custom session timeout")
		}
		if opts.HeartbeatInterval != 10*time.Second {
			t.Error("should preserve custom heartbeat interval")
		}
		if opts.MaxMessageSize != 1024 {
			t.Error("should preserve custom max message size")
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("creates live engine with defaults", func(t *testing.T) {
		lv := New(Options{})

		if lv.pubsub == nil {
			t.Error("expected default pubsub")
		}
		if lv.store == nil {
			t.Error("expected default store")
		}
		if lv.pages == nil {
			t.Error("expected pages map")
		}
	})

	t.Run("uses custom pubsub", func(t *testing.T) {
		ps := NewInmemPubSub()
		lv := New(Options{PubSub: ps})

		if lv.pubsub != ps {
			t.Error("should use custom pubsub")
		}
	})

	t.Run("uses custom store", func(t *testing.T) {
		store := NewMemoryStore()
		lv := New(Options{SessionStore: store})

		if lv.store != store {
			t.Error("should use custom store")
		}
	})
}

func TestTemplateFuncs(t *testing.T) {
	funcs := TemplateFuncs()

	t.Run("all funcs present", func(t *testing.T) {
		expected := []string{
			"lvClick", "lvSubmit", "lvChange",
			"lvKeydown", "lvKeyup", "lvFocus", "lvBlur",
			"lvVal", "lvDebounce", "lvThrottle",
			"lvLoading", "lvTarget",
		}

		for _, name := range expected {
			if _, ok := funcs[name]; !ok {
				t.Errorf("missing func: %s", name)
			}
		}
	})

	t.Run("lvClick", func(t *testing.T) {
		result := lvClick("save")
		expected := `data-lv-click="save"`
		if string(result) != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("lvClick with values", func(t *testing.T) {
		result := lvClick("delete", map[string]any{"id": 123})
		if !strings.Contains(string(result), `data-lv-click="delete"`) {
			t.Error("missing click attr")
		}
		if !strings.Contains(string(result), `data-lv-value-id="123"`) {
			t.Error("missing value attr")
		}
	})

	t.Run("lvSubmit", func(t *testing.T) {
		result := lvSubmit("create")
		expected := `data-lv-submit="create"`
		if string(result) != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("lvChange", func(t *testing.T) {
		result := lvChange("search")
		expected := `data-lv-change="search"`
		if string(result) != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("lvKeydown", func(t *testing.T) {
		result := lvKeydown("submit", "Enter")
		if !strings.Contains(string(result), `data-lv-keydown="submit"`) {
			t.Error("missing keydown attr")
		}
		if !strings.Contains(string(result), `data-lv-key="Enter"`) {
			t.Error("missing key attr")
		}
	})

	t.Run("lvVal", func(t *testing.T) {
		result := lvVal("id", 42)
		if result["id"] != 42 {
			t.Error("expected id=42")
		}
	})

	t.Run("lvDebounce", func(t *testing.T) {
		result := lvDebounce(300)
		expected := `data-lv-debounce="300"`
		if string(result) != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("lvThrottle", func(t *testing.T) {
		result := lvThrottle(500)
		expected := `data-lv-throttle="500"`
		if string(result) != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("lvLoading", func(t *testing.T) {
		result := lvLoading("opacity-50")
		expected := `data-lv-loading-class="opacity-50"`
		if string(result) != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("lvTarget", func(t *testing.T) {
		result := lvTarget("modal")
		expected := `data-lv-target="modal"`
		if string(result) != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("funcs work in templates", func(t *testing.T) {
		tmpl, err := template.New("test").Funcs(funcs).Parse(`
			<button {{lvClick "save" (lvVal "id" 1)}}>Save</button>
		`)
		if err != nil {
			t.Fatalf("template parse error: %v", err)
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, nil); err != nil {
			t.Fatalf("template execute error: %v", err)
		}

		html := buf.String()
		if !strings.Contains(html, `data-lv-click="save"`) {
			t.Error("missing click in output")
		}
		if !strings.Contains(html, `data-lv-value-id="1"`) {
			t.Error("missing value in output")
		}
	})
}

func TestInjectRuntime(t *testing.T) {
	lv := New(Options{})

	t.Run("injects before body close", func(t *testing.T) {
		html := `<html><body><div>content</div></body></html>`
		result := lv.injectRuntime(html, "sess123", "/counter")

		if !strings.Contains(result, `<script src="/_live/runtime.js"></script>`) {
			t.Error("missing runtime script")
		}
		if !strings.Contains(result, `MizuLive.connect`) {
			t.Error("missing connect call")
		}
		if !strings.Contains(result, `"sess123"`) {
			t.Error("missing session id")
		}
		if !strings.Contains(result, `"/counter"`) {
			t.Error("missing path")
		}
		if !strings.HasSuffix(result, `</body></html>`) {
			t.Error("should end with </body></html>")
		}
	})

	t.Run("appends if no body close", func(t *testing.T) {
		html := `<div>content</div>`
		result := lv.injectRuntime(html, "sess", "/test")

		if !strings.Contains(result, `<script src="/_live/runtime.js"></script>`) {
			t.Error("missing runtime script")
		}
	})
}

func TestGenerateSessionID(t *testing.T) {
	ids := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		id := generateSessionID()

		if len(id) != 32 { // 16 bytes = 32 hex chars
			t.Errorf("expected 32 chars, got %d", len(id))
		}

		if ids[id] {
			t.Error("duplicate session ID generated")
		}
		ids[id] = true
	}
}

// Example page for testing.
type TestCounterPage struct{}

type TestCounterState struct {
	Count int
	Log   []string
}

func (p *TestCounterPage) Mount(ctx *Ctx, s *Session[TestCounterState]) error {
	s.State = TestCounterState{Count: 0}
	s.MarkAll()
	return nil
}

func (p *TestCounterPage) Render(ctx *Ctx, s *Session[TestCounterState]) (View, error) {
	return View{
		Page: "counter/index",
		Regions: map[string]string{
			"count": "counter/count",
			"log":   "counter/log",
		},
	}, nil
}

func (p *TestCounterPage) Handle(ctx *Ctx, s *Session[TestCounterState], e Event) error {
	switch e.Name {
	case "inc":
		by := e.GetInt("by")
		if by == 0 {
			by = 1
		}
		s.State.Count += by
		s.State.Log = append(s.State.Log, "inc")
		s.Mark("count", "log")
	case "dec":
		s.State.Count--
		s.State.Log = append(s.State.Log, "dec")
		s.Mark("count", "log")
	case "reset":
		s.State.Count = 0
		s.State.Log = nil
		s.MarkAll()
	}
	return nil
}

func (p *TestCounterPage) Info(ctx *Ctx, s *Session[TestCounterState], msg any) error {
	return nil
}

func TestCounterPageLogic(t *testing.T) {
	page := &TestCounterPage{}
	ctx := NewTestCtx()
	session := NewTestSession[TestCounterState]()

	// Mount
	if err := page.Mount(ctx, session); err != nil {
		t.Fatalf("mount error: %v", err)
	}

	if session.State.Count != 0 {
		t.Error("expected initial count 0")
	}

	// Increment
	if err := page.Handle(ctx, session, Event{Name: "inc"}); err != nil {
		t.Fatalf("handle error: %v", err)
	}

	if session.State.Count != 1 {
		t.Errorf("expected count 1, got %d", session.State.Count)
	}
	if len(session.State.Log) != 1 || session.State.Log[0] != "inc" {
		t.Error("expected log entry")
	}

	// Increment by 5
	if err := page.Handle(ctx, session, Event{
		Name:   "inc",
		Values: map[string]string{"by": "5"},
	}); err != nil {
		t.Fatalf("handle error: %v", err)
	}

	if session.State.Count != 6 {
		t.Errorf("expected count 6, got %d", session.State.Count)
	}

	// Decrement
	if err := page.Handle(ctx, session, Event{Name: "dec"}); err != nil {
		t.Fatalf("handle error: %v", err)
	}

	if session.State.Count != 5 {
		t.Errorf("expected count 5, got %d", session.State.Count)
	}

	// Reset
	if err := page.Handle(ctx, session, Event{Name: "reset"}); err != nil {
		t.Fatalf("handle error: %v", err)
	}

	if session.State.Count != 0 {
		t.Error("expected count 0 after reset")
	}
	if len(session.State.Log) != 0 {
		t.Error("expected empty log after reset")
	}
}
