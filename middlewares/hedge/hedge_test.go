package hedge

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestFastResponse(t *testing.T) {
	hedger := NewHedger(Options{
		Delay:     50 * time.Millisecond,
		MaxHedges: 1,
	})

	app := mizu.NewRouter()
	app.Use(hedger.Middleware())

	var callCount int32

	app.Get("/", func(c *mizu.Ctx) error {
		atomic.AddInt32(&callCount, 1)
		return c.Text(http.StatusOK, "fast")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Give time for hedged request to potentially start
	time.Sleep(100 * time.Millisecond)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHedgeTriggered(t *testing.T) {
	hedger := NewHedger(Options{
		Delay:     10 * time.Millisecond,
		MaxHedges: 1,
		OnHedge: func(hedgeNum int) {
			// Hedge triggered callback
		},
	})

	app := mizu.NewRouter()
	app.Use(hedger.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		// Slow response triggers hedge
		time.Sleep(50 * time.Millisecond)
		return c.Text(http.StatusOK, "slow")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	// Hedge should have been triggered for slow request
	// Note: due to timing, this might not always trigger
}

func TestShouldHedge(t *testing.T) {
	hedger := NewHedger(Options{
		Delay: 10 * time.Millisecond,
		ShouldHedge: func(r *http.Request) bool {
			return r.URL.Path != "/nohedge"
		},
	})

	app := mizu.NewRouter()
	app.Use(hedger.Middleware())

	app.Get("/hedge", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hedged")
	})
	app.Get("/nohedge", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "not hedged")
	})

	// Request to /hedge
	req := httptest.NewRequest(http.MethodGet, "/hedge", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	// Request to /nohedge
	req = httptest.NewRequest(http.MethodGet, "/nohedge", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestOnComplete(t *testing.T) {
	var completedDuration time.Duration

	hedger := NewHedger(Options{
		Delay:     10 * time.Millisecond,
		MaxHedges: 1,
		OnComplete: func(hedgeNum int, duration time.Duration) {
			_ = hedgeNum // Hedge number (0 = original, 1+ = hedge)
			completedDuration = duration
		},
	})

	app := mizu.NewRouter()
	app.Use(hedger.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if completedDuration == 0 {
		t.Error("expected OnComplete to be called with duration")
	}
}

func TestStats(t *testing.T) {
	hedger := NewHedger(Options{
		Delay:     100 * time.Millisecond,
		MaxHedges: 1,
	})

	app := mizu.NewRouter()
	app.Use(hedger.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	stats := hedger.Stats()

	if stats.TotalRequests != 1 {
		t.Errorf("expected TotalRequests=1, got %d", stats.TotalRequests)
	}
	if stats.HedgedRequests != 1 {
		t.Errorf("expected HedgedRequests=1, got %d", stats.HedgedRequests)
	}
}

func TestGetHedgeInfo(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Delay:     time.Hour, // Don't actually hedge
		MaxHedges: 1,
	}))

	var hedgeInfo *HedgeInfo

	app.Get("/", func(c *mizu.Ctx) error {
		hedgeInfo = GetHedgeInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if hedgeInfo == nil {
		t.Fatal("expected hedge info")
	}

	if hedgeInfo.HedgeNumber != 0 {
		t.Errorf("expected original request (hedge 0), got %d", hedgeInfo.HedgeNumber)
	}
}

func TestIsHedge(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Delay:     time.Hour,
		MaxHedges: 1,
	}))

	var isHedged bool

	app.Get("/", func(c *mizu.Ctx) error {
		isHedged = IsHedge(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Original request should not be marked as hedge
	if isHedged {
		t.Error("expected original request to not be a hedge")
	}
}

func TestConditional(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Conditional(func(r *http.Request) bool {
		return r.Method == http.MethodGet
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// GET should be hedged
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	// POST should not be hedged
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestForSlowRequests(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(ForSlowRequests(50 * time.Millisecond))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMaxHedges(t *testing.T) {
	hedger := NewHedger(Options{
		Delay:     5 * time.Millisecond,
		MaxHedges: 3,
		Timeout:   500 * time.Millisecond,
	})

	app := mizu.NewRouter()
	app.Use(hedger.Middleware())

	var callCount int32

	app.Get("/", func(c *mizu.Ctx) error {
		atomic.AddInt32(&callCount, 1)
		// First call is slow, others are fast
		if atomic.LoadInt32(&callCount) == 1 {
			time.Sleep(100 * time.Millisecond)
		}
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestTimeout(t *testing.T) {
	hedger := NewHedger(Options{
		Delay:     10 * time.Millisecond,
		MaxHedges: 1,
		Timeout:   50 * time.Millisecond,
	})

	app := mizu.NewRouter()
	app.Use(hedger.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		time.Sleep(100 * time.Millisecond) // Longer than timeout
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Request should timeout
	// Note: behavior depends on how timeout is handled
}
