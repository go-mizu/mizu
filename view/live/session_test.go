package live

import (
	"testing"
)

func TestDirtySet(t *testing.T) {
	t.Run("new dirty set is empty", func(t *testing.T) {
		d := newDirtySet()
		if !d.IsEmpty() {
			t.Error("expected empty dirty set")
		}
	})

	t.Run("add marks region dirty", func(t *testing.T) {
		d := newDirtySet()
		d.Add("stats")

		if d.IsEmpty() {
			t.Error("expected non-empty dirty set")
		}
		if !d.Has("stats") {
			t.Error("expected stats to be dirty")
		}
		if d.Has("other") {
			t.Error("expected other to not be dirty")
		}
	})

	t.Run("add all marks everything dirty", func(t *testing.T) {
		d := newDirtySet()
		d.AddAll()

		if d.IsEmpty() {
			t.Error("expected non-empty dirty set")
		}
		if !d.IsAll() {
			t.Error("expected all flag to be set")
		}
		if !d.Has("anything") {
			t.Error("expected any region to be dirty when all is set")
		}
	})

	t.Run("list returns dirty regions", func(t *testing.T) {
		d := newDirtySet()
		d.Add("a")
		d.Add("b")
		d.Add("c")

		list := d.List()
		if len(list) != 3 {
			t.Errorf("expected 3 regions, got %d", len(list))
		}
	})

	t.Run("clear resets dirty set", func(t *testing.T) {
		d := newDirtySet()
		d.Add("stats")
		d.AddAll()
		d.Clear()

		if !d.IsEmpty() {
			t.Error("expected empty dirty set after clear")
		}
		if d.IsAll() {
			t.Error("expected all flag to be cleared")
		}
	})
}

func TestSession(t *testing.T) {
	t.Run("new session has empty state", func(t *testing.T) {
		s := NewTestSession[CounterState]()

		if s.ID == "" {
			t.Error("expected session ID")
		}
		if s.State.Count != 0 {
			t.Errorf("expected zero count, got %d", s.State.Count)
		}
	})

	t.Run("mark adds regions to dirty set", func(t *testing.T) {
		s := NewTestSession[CounterState]()
		s.Mark("stats", "log")

		if !s.IsDirty() {
			t.Error("expected session to be dirty")
		}
	})

	t.Run("mark all sets all flag", func(t *testing.T) {
		s := NewTestSession[CounterState]()
		s.MarkAll()

		if !s.IsDirty() {
			t.Error("expected session to be dirty")
		}
	})

	t.Run("replace state updates state and marks all", func(t *testing.T) {
		s := NewTestSession[CounterState]()
		s.ReplaceState(CounterState{Count: 42})

		if s.State.Count != 42 {
			t.Errorf("expected count 42, got %d", s.State.Count)
		}
		if !s.IsDirty() {
			t.Error("expected session to be dirty after replace")
		}
	})

	t.Run("push adds command", func(t *testing.T) {
		s := NewTestSession[CounterState]()
		s.Push(Redirect{To: "/home"})

		cmds := s.commands
		if len(cmds) != 1 {
			t.Errorf("expected 1 command, got %d", len(cmds))
		}
	})
}

func TestFlash(t *testing.T) {
	t.Run("empty flash", func(t *testing.T) {
		f := Flash{}
		if !f.IsEmpty() {
			t.Error("expected empty flash")
		}
	})

	t.Run("add messages", func(t *testing.T) {
		f := Flash{}
		f.AddSuccess("saved")
		f.AddError("failed")
		f.AddWarning("careful")
		f.AddInfo("note")

		if f.IsEmpty() {
			t.Error("expected non-empty flash")
		}
		if len(f.Success) != 1 || f.Success[0] != "saved" {
			t.Error("expected success message")
		}
		if len(f.Error) != 1 || f.Error[0] != "failed" {
			t.Error("expected error message")
		}
	})

	t.Run("clear removes all messages", func(t *testing.T) {
		f := Flash{}
		f.AddSuccess("saved")
		f.AddError("failed")
		f.Clear()

		if !f.IsEmpty() {
			t.Error("expected empty flash after clear")
		}
	})
}

// Test state type.
type CounterState struct {
	Count int
}
