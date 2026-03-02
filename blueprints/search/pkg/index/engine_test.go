package index_test

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// fakeExternal is a minimal engine that implements AddrSetter for testing.
type fakeExternal struct {
	addr   string
	opened bool
}

func (f *fakeExternal) Name() string                                                   { return "fake-external" }
func (f *fakeExternal) Open(_ context.Context, _ string) error                         { f.opened = true; return nil }
func (f *fakeExternal) Close() error                                                   { return nil }
func (f *fakeExternal) Stats(_ context.Context) (index.EngineStats, error)             { return index.EngineStats{}, nil }
func (f *fakeExternal) Index(_ context.Context, _ []index.Document) error              { return nil }
func (f *fakeExternal) Search(_ context.Context, _ index.Query) (index.Results, error) { return index.Results{}, nil }
func (f *fakeExternal) SetAddr(a string)                                               { f.addr = a }

func TestAddrSetter(t *testing.T) {
	eng := &fakeExternal{}
	setter, ok := any(eng).(index.AddrSetter)
	if !ok {
		t.Fatal("fakeExternal does not implement AddrSetter")
	}
	setter.SetAddr("http://localhost:9999")
	if eng.addr != "http://localhost:9999" {
		t.Errorf("SetAddr: got %q, want %q", eng.addr, "http://localhost:9999")
	}
}

func TestBaseExternal_EffectiveAddr(t *testing.T) {
	b := &index.BaseExternal{}
	if got := b.EffectiveAddr("http://default:7700"); got != "http://default:7700" {
		t.Errorf("EffectiveAddr with empty Addr: got %q, want default", got)
	}
	b.SetAddr("http://custom:9000")
	if got := b.EffectiveAddr("http://default:7700"); got != "http://custom:9000" {
		t.Errorf("EffectiveAddr with set Addr: got %q, want custom", got)
	}
}

func TestRegistry_ListAndNew(t *testing.T) {
	name := "test-fake-external-registry-unique-xyz"
	index.Register(name, func() index.Engine { return &fakeExternal{} })

	names := index.List()
	found := false
	for _, n := range names {
		if n == name {
			found = true
		}
	}
	if !found {
		t.Errorf("List() does not include %q; got %v", name, names)
	}

	eng, err := index.NewEngine(name)
	if err != nil {
		t.Fatal(err)
	}
	if eng.Name() == "" {
		t.Error("NewEngine returned engine with empty Name()")
	}

	_, err = index.NewEngine("definitely-not-registered-xyz-abc")
	if err == nil {
		t.Error("expected error for unknown driver, got nil")
	}
}
