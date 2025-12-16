// Package contract defines transport-neutral service contracts.
//
// A contract is derived from a plain Go struct with methods.
// The service contains zero framework dependencies.
//
// Supported method signatures:
//
//	func (s *S) Method(ctx context.Context, in *In) (*Out, error)
//	func (s *S) Method(ctx context.Context) (*Out, error)
//	func (s *S) Method(ctx context.Context, in *In) error
//	func (s *S) Method(ctx context.Context) error
package contract

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
)

var (
	ErrInvalidService   = errors.New("contract: invalid service")
	ErrInvalidMethod    = errors.New("contract: invalid method")
	ErrInvalidSignature = errors.New("contract: invalid method signature")
	ErrUnsupportedType  = errors.New("contract: unsupported type")
	ErrNilInput         = errors.New("contract: nil input")
)

// ServiceMeta is an optional interface services can implement
// to provide metadata about themselves.
type ServiceMeta interface {
	ContractServiceMeta() ServiceOptions
}

// ServiceOptions provides metadata about a service.
type ServiceOptions struct {
	Description string
	Version     string
	Tags        []string
}

// MethodMeta is an optional interface services can implement
// to provide metadata about their methods.
type MethodMeta interface {
	ContractMeta() map[string]MethodOptions
}

// MethodOptions provides metadata about a method.
type MethodOptions struct {
	Description string
	Summary     string
	Tags        []string
	Deprecated  bool
	HTTPMethod  string // REST verb override
	HTTPPath    string // REST path override
}

// Register creates a Service from a Go struct.
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

	types := newTypes()
	out := &Service{
		Name:         name,
		Types:        types,
		methodByName: make(map[string]*Method),
	}

	if sm, ok := svc.(ServiceMeta); ok {
		opts := sm.ContractServiceMeta()
		out.Description = opts.Description
		out.Version = opts.Version
		out.Tags = opts.Tags
	}

	var methodMeta map[string]MethodOptions
	if mm, ok := svc.(MethodMeta); ok {
		methodMeta = mm.ContractMeta()
	}

	rt := reflect.TypeOf(svc)
	for i := 0; i < rt.NumMethod(); i++ {
		rm := rt.Method(i)
		if isMetaMethod(rm.Name) {
			continue
		}

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
			recv:     v,
			method:   rm,
		}

		if methodMeta != nil {
			if opts, ok := methodMeta[rm.Name]; ok {
				m.Description = opts.Description
				m.Summary = opts.Summary
				m.Tags = opts.Tags
				m.Deprecated = opts.Deprecated
				m.HTTPMethod = opts.HTTPMethod
				m.HTTPPath = opts.HTTPPath
			}
		}

		if inT != nil {
			tr, err := types.Add(inT)
			if err != nil {
				return nil, err
			}
			m.Input = tr
		}

		if outT != nil {
			tr, err := types.Add(outT)
			if err != nil {
				return nil, err
			}
			m.Output = tr
		}

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

// Service represents a registered service contract.
type Service struct {
	Name        string
	Description string
	Version     string
	Tags        []string
	Methods     []*Method
	Types       *Types

	methodByName map[string]*Method
}

// Method returns a method by name, or nil if not found.
func (s *Service) Method(name string) *Method {
	return s.methodByName[name]
}

// MethodNames returns all method names in sorted order.
func (s *Service) MethodNames() []string {
	names := make([]string, len(s.Methods))
	for i, m := range s.Methods {
		names[i] = m.Name
	}
	return names
}

// Method represents a callable service method.
type Method struct {
	Service  *Service
	Name     string
	FullName string

	Description string
	Summary     string
	Tags        []string
	Deprecated  bool

	HTTPMethod string
	HTTPPath   string

	Input  *Type
	Output *Type

	inType  reflect.Type
	outType reflect.Type
	sig     sigKind
	recv    reflect.Value
	method  reflect.Method
}

// Call invokes the method with the given input.
func (m *Method) Call(ctx context.Context, in any) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var args []reflect.Value
	switch m.sig {
	case sigCtxErr, sigCtxOutErr:
		args = []reflect.Value{m.recv, reflect.ValueOf(ctx)}
	case sigCtxInErr, sigCtxInOutErr:
		if in == nil {
			return nil, ErrNilInput
		}
		args = []reflect.Value{m.recv, reflect.ValueOf(ctx), reflect.ValueOf(in)}
	}

	out := m.method.Func.Call(args)

	switch m.sig {
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

// NewInput creates a new instance of the input type.
func (m *Method) NewInput() any {
	if m.inType == nil {
		return nil
	}
	return reflect.New(m.inType.Elem()).Interface()
}

// HasInput returns true if the method accepts input.
func (m *Method) HasInput() bool { return m.inType != nil }

// HasOutput returns true if the method returns output.
func (m *Method) HasOutput() bool { return m.outType != nil }

// InputType returns the reflect.Type of the input, or nil.
func (m *Method) InputType() reflect.Type { return m.inType }

// OutputType returns the reflect.Type of the output, or nil.
func (m *Method) OutputType() reflect.Type { return m.outType }

// ---- internal ----

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
)

func isMetaMethod(name string) bool {
	switch name {
	case "ContractServiceMeta", "ContractMeta", "ContractEnum":
		return true
	}
	return false
}

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
