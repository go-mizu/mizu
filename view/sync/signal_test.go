package sync

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestSignal_GetSet(t *testing.T) {
	s := NewSignal(42)

	if got := s.Get(); got != 42 {
		t.Errorf("Get() = %d, want 42", got)
	}

	s.Set(100)

	if got := s.Get(); got != 100 {
		t.Errorf("Get() = %d, want 100", got)
	}
}

func TestSignal_Update(t *testing.T) {
	s := NewSignal(10)

	s.Update(func(v int) int { return v * 2 })

	if got := s.Get(); got != 20 {
		t.Errorf("Get() = %d, want 20", got)
	}
}

func TestSignal_Version(t *testing.T) {
	s := NewSignal(0)

	v1 := s.Version()
	s.Set(1)
	v2 := s.Version()

	if v2 <= v1 {
		t.Errorf("Version should increase after Set")
	}
}

func TestComputed_Basic(t *testing.T) {
	count := NewSignal(5)
	doubled := NewComputed(func() int {
		return count.Get() * 2
	})

	if got := doubled.Get(); got != 10 {
		t.Errorf("Computed.Get() = %d, want 10", got)
	}

	count.Set(7)

	if got := doubled.Get(); got != 14 {
		t.Errorf("Computed.Get() = %d, want 14", got)
	}
}

func TestComputed_Caching(t *testing.T) {
	callCount := 0
	count := NewSignal(5)

	doubled := NewComputed(func() int {
		callCount++
		return count.Get() * 2
	})

	// First call computes
	_ = doubled.Get()
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Second call should be cached
	_ = doubled.Get()
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (cached)", callCount)
	}

	// After signal change, should recompute
	count.Set(10)
	_ = doubled.Get()
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (recomputed)", callCount)
	}
}

func TestComputed_Chained(t *testing.T) {
	a := NewSignal(2)
	b := NewComputed(func() int { return a.Get() * 2 })
	c := NewComputed(func() int { return b.Get() + 1 })

	if got := c.Get(); got != 5 {
		t.Errorf("c.Get() = %d, want 5", got)
	}

	a.Set(3)

	if got := c.Get(); got != 7 {
		t.Errorf("c.Get() = %d, want 7", got)
	}
}

func TestEffect_Basic(t *testing.T) {
	count := NewSignal(0)
	var effectCount atomic.Int32

	effect := NewEffect(func() {
		_ = count.Get()
		effectCount.Add(1)
	})
	defer effect.Stop()

	// Effect runs immediately (synchronously)
	if got := effectCount.Load(); got != 1 {
		t.Errorf("effectCount = %d, want 1", got)
	}

	// Effect runs on change (asynchronously)
	count.Set(1)

	// Wait for async effect with polling
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if effectCount.Load() >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := effectCount.Load(); got < 2 {
		t.Errorf("effectCount = %d, want >= 2", got)
	}
}

func TestEffect_Stop(t *testing.T) {
	count := NewSignal(0)
	var effectCount atomic.Int32

	effect := NewEffect(func() {
		_ = count.Get()
		effectCount.Add(1)
	})

	time.Sleep(10 * time.Millisecond)
	before := effectCount.Load()

	effect.Stop()
	count.Set(1)
	time.Sleep(50 * time.Millisecond)

	after := effectCount.Load()
	if after > before {
		t.Errorf("Effect should not run after Stop")
	}
}

func TestSignal_ConcurrentAccess(t *testing.T) {
	s := NewSignal(0)
	done := make(chan bool)

	// Writer
	go func() {
		for i := 0; i < 100; i++ {
			s.Set(i)
		}
		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 100; i++ {
			_ = s.Get()
		}
		done <- true
	}()

	<-done
	<-done
}
