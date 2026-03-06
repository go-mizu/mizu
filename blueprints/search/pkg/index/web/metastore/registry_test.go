package metastore

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type testStore struct{}

func (testStore) Name() string               { return "test" }
func (testStore) Init(context.Context) error { return nil }
func (testStore) GetSummary(context.Context, string) (SummaryRecord, bool, error) {
	return SummaryRecord{}, false, nil
}
func (testStore) PutSummary(context.Context, SummaryRecord) error { return nil }
func (testStore) ListWARCs(context.Context, string) ([]WARCRecord, error) {
	return nil, nil
}
func (testStore) GetWARC(context.Context, string, string) (WARCRecord, bool, error) {
	return WARCRecord{}, false, nil
}
func (testStore) GetRefreshState(context.Context, string) (RefreshState, bool, error) {
	return RefreshState{}, false, nil
}
func (testStore) SetRefreshState(context.Context, RefreshState) error { return nil }
func (testStore) ListJobs(context.Context) ([]JobRecord, error)       { return nil, nil }
func (testStore) PutJob(context.Context, JobRecord) error             { return nil }
func (testStore) DeleteAllJobs(context.Context) error                 { return nil }
func (testStore) Close() error                                        { return nil }

type testDriver struct{}

func (testDriver) Open(string, Options) (Store, error) { return testStore{}, nil }

type errDriver struct{}

func (errDriver) Open(string, Options) (Store, error) { return nil, errors.New("boom") }

func TestOpen_UnknownDriver(t *testing.T) {
	_, err := Open("nonexistent", "", Options{})
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
	if !strings.Contains(err.Error(), "unknown driver") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterAndOpen(t *testing.T) {
	name := "metastore-test-register-open"
	Register(name, testDriver{})

	got, err := Open(name, "/tmp/test.db", Options{})
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if got.Name() != "test" {
		t.Fatalf("store name = %q, want %q", got.Name(), "test")
	}
}

func TestOpen_PropagatesDriverError(t *testing.T) {
	name := "metastore-test-open-error"
	Register(name, errDriver{})
	_, err := Open(name, "/tmp/test.db", Options{})
	if err == nil {
		t.Fatal("expected driver error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegister_DuplicatePanics(t *testing.T) {
	name := "metastore-test-duplicate-panic"
	Register(name, testDriver{})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate register")
		}
	}()
	Register(name, testDriver{})
}
