package amazon

import (
	"path/filepath"
	"testing"
)

func TestStateEnqueueReportsDuplicateInsertions(t *testing.T) {
	t.Parallel()

	state, err := OpenState(filepath.Join(t.TempDir(), "state.duckdb"))
	if err != nil {
		t.Fatalf("OpenState: %v", err)
	}
	defer state.Close()

	inserted, err := state.Enqueue("https://www.amazon.com/dp/B0GL7WD892", EntityProduct, 10)
	if err != nil {
		t.Fatalf("first Enqueue: %v", err)
	}
	if !inserted {
		t.Fatalf("first enqueue should insert")
	}

	inserted, err = state.Enqueue("https://www.amazon.com/dp/B0GL7WD892", EntityProduct, 10)
	if err != nil {
		t.Fatalf("second Enqueue: %v", err)
	}
	if inserted {
		t.Fatalf("duplicate enqueue should not insert")
	}
}
