// Package contract defines a transport-neutral service contract.
//
// A contract is derived from a plain Go service (struct + methods).
// The service contains zero framework dependencies and is easy to test.
//
// Supported canonical method signature:
//
//   func (s *S) Method(ctx context.Context, in *In) (*Out, error)
//
// Variants supported:
//   func (s *S) Method(ctx context.Context) (*Out, error)
//   func (s *S) Method(ctx context.Context, in *In) error
//   func (s *S) Method(ctx context.Context) error
//
// Reflection is performed once at registration time.
// Runtime calls use compiled invokers.
package contract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidService   = errors.New("contract: invalid service")
	ErrInvalidMethod    = errors.New("contract: invalid method")
	ErrInvalidSignature = errors.New("contract: invalid method signature")
	ErrUnsupportedType  = errors.New("contract: unsupported type")
	ErrNilInput         = errors.New("contract: nil input")
)

// Register inspects a service and returns its contract.
//
// The service must be a struct or pointer to struct.
// All exported methods are considered part of the contract.
func Register(name string, svc any) (*Service, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: empty name", ErrInvalidService)
	}
	if svc == nil {
		return nil, fmt.Errorf("%w: nil service", ErrInvalidService)
	}

	v := reflect.ValueOf(svc)
	t := v.Type()

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: expected struct, got %s", ErrInvalidService, t)
	}

	reg := newTypeRegistry()

	out := &Service{
		Name:         name,
		Types:        reg,
		methodByName: make(map[string]*Method),
	}

	rt := reflect.TypeOf(svc)

	for i := 0; i < rt.NumMethod(); i++ {
		rm := rt.Method(i)

		sig, inT, outT, err := parseSignature(rm.Type)
		if err != nil {
			return nil, fmt.Errorf("%w: %s.%s: %v", ErrInvalidSignature, name, rm.Name, err)
		}

		m := &Method{
			Service:  out,
			Name:     rm.Name,
			FullName: name + "." + rm.Name,
			sig:      sig,
			inType:   inT,
			outType:  outT,
			Errors:   &ErrorContract{},
		}

		if inT != nil {
			tr, err := reg.Add(inT)
			if err != nil {
				return nil, err
			}
			m.Input = tr
		}

		if outT != nil {
			tr, err := reg.Add(outT)
			if err != nil {
				return nil, err
			}
			m.Output = tr
		}

		m.Invoker = newInvoker(reflect.ValueOf(svc), rm, sig, inT, outT)

		out.Methods = append(out.Methods, m)
		out.methodByName[m.Name] = m
	}

	if len(out.Methods) == 0 {
		return nil, fmt.Errorf("%w: no exported methods", ErrInvalidService)
	}

	sort.Slice(out.Methods, func(i, j int) bool {
		return out.Methods[i].Name < out.Methods[j].Name
	})

	return out, nil
}

// Service represents a transport-neutral service contract.
type Service struct {
	Name    string
	Methods []*Method
	Types   *TypeRegistry

	methodByName map[string]*Method
}

func (s *Service) Method(name string) *Method {
	return s.methodByName[name]
}

// Method represents a callable service method.
type Method struct {
	Service  *Service
	Name     string
	FullName string

	Input  *TypeRef
	Output *TypeRef
	Errors *ErrorContract

	Invoker Invoker

	inType  reflect.Type
	outType reflect.Type
	sig     sigKind
}

func (m *Method) NewInput() any {
	if m.inType == nil {
		return nil
	}
	return reflect.New(m.inType.Elem()).Interface()
}

// Invoker calls a compiled method.
type Invoker interface {
	Call(ctx context.Context, in any) (any, error)
}

// ---- invocation ----

type sigKind int

const (
	sigCtxErr sigKind = iota
	sigCtxOutErr
	sigCtxInErr
	sigCtxInOutErr
)

var (
	typeContext = reflect.TypeOf((*context.Context)(nil)).Elem()
	typeError   = reflect.TypeOf((*error)(nil)).Elem()
	typeTime    = reflect.TypeOf(time.Time{})
)

func parseSignature(t reflect.Type) (sig sigKind, in, out reflect.Type, err error) {
	if t.NumIn() < 2 || t.In(1) != typeContext {
		return 0, nil, nil, fmt.Errorf("first argument must be context.Context")
	}

	switch t.NumIn() {
	case 2:
		switch t.NumOut() {
		case 1:
			if t.Out(0) != typeError {
				return 0, nil, nil, fmt.Errorf("expected error return")
			}
			return sigCtxErr, nil, nil, nil
		case 2:
			if t.Out(1) != typeError {
				return 0, nil, nil, fmt.Errorf("last return must be error")
			}
			return sigCtxOutErr, nil, t.Out(0), nil
		}
	case 3:
		if t.In(2).Kind() != reflect.Pointer {
			return 0, nil, nil, fmt.Errorf("input must be pointer")
		}
		switch t.NumOut() {
		case 1:
			if t.Out(0) != typeError {
				return 0, nil, nil, fmt.Errorf("expected error return")
			}
			return sigCtxInErr, t.In(2), nil, nil
		case 2:
			if t.Out(1) != typeError {
				return 0, nil, nil, fmt.Errorf("last return must be error")
			}
			return sigCtxInOutErr, t.In(2), t.Out(0), nil
		}
	}

	return 0, nil, nil, fmt.Errorf("unsupported signature")
}

type reflectInvoker struct {
	recv   reflect.Value
	method reflect.Method
	sig    sigKind
	inType reflect.Type
}

func newInvoker(recv reflect.Value, m reflect.Method, sig sigKind, in, out reflect.Type) Invoker {
	return &reflectInvoker{recv: recv, method: m, sig: sig, inType: in}
}

func (x *reflectInvoker) Call(ctx context.Context, in any) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var args []reflect.Value

	switch x.sig {
	case sigCtxErr, sigCtxOutErr:
		args = []reflect.Value{x.recv, reflect.ValueOf(ctx)}
	case sigCtxInErr, sigCtxInOutErr:
		if in == nil {
			return nil, ErrNilInput
		}
		args = []reflect.Value{x.recv, reflect.ValueOf(ctx), reflect.ValueOf(in)}
	}

	out := x.method.Func.Call(args)

	switch x.sig {
	case sigCtxErr, sigCtxInErr:
		if !out[0].IsNil() {
			return nil, out[0].Interface().(error)
		}
		return nil, nil
	case sigCtxOutErr, sigCtxInOutErr:
		var err error
		if !out[1].IsNil() {
			err = out[1].Interface().(error)
		}
		if out[0].IsNil() {
			return nil, err
		}
		return out[0].Interface(), err
	}
	return nil, ErrInvalidMethod
}

// ---- types & schema ----

type TypeRegistry struct {
	mu      sync.RWMutex
	types   map[string]*TypeRef
	schemas map[string]Schema
}

type TypeRef struct {
	ID   string
	Name string
}

type Schema struct {
	ID   string         `json:"id"`
	JSON map[string]any `json:"json"`
}

func newTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types:   make(map[string]*TypeRef),
		schemas: make(map[string]Schema),
	}
}

func (r *TypeRegistry) Add(t reflect.Type) (*TypeRef, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Name() == "" {
		return nil, ErrUnsupportedType
	}

	id := typeID(t)

	r.mu.Lock()
	defer r.mu.Unlock()

	if tr, ok := r.types[id]; ok {
		return tr, nil
	}

	tr := &TypeRef{ID: id, Name: t.Name()}
	r.types[id] = tr

	s, err := schemaForType(t)
	if err != nil {
		return nil, err
	}
	r.schemas[id] = Schema{ID: id, JSON: s}

	return tr, nil
}

func (r *TypeRegistry) Schemas() []Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Schema, 0, len(r.schemas))
	for _, s := range r.schemas {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func typeID(t reflect.Type) string {
	if t.PkgPath() == "" {
		return t.Name()
	}
	parts := strings.Split(t.PkgPath(), "/")
	return parts[len(parts)-1] + "." + t.Name()
}

func schemaForType(t reflect.Type) (map[string]any, error) {
	if t == typeTime {
		return map[string]any{"type": "string", "format": "date-time"}, nil
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}, nil
	case reflect.Bool:
		return map[string]any{"type": "boolean"}, nil
	case reflect.Int, reflect.Int64:
		return map[string]any{"type": "integer"}, nil
	case reflect.Struct:
		props := map[string]any{}
		var req []string

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			name := strings.ToLower(f.Name[:1]) + f.Name[1:]
			props[name], _ = schemaForType(deref(f.Type))
			req = append(req, name)
		}

		return map[string]any{
			"type":       "object",
			"properties": props,
			"required":   req,
		}, nil
	}

	return nil, ErrUnsupportedType
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// ---- errors ----

type ErrorContract struct {
	Code string `json:"code,omitempty"`
}
