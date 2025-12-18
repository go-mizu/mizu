package contract

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// RegisteredService combines the Invoker interface with service metadata.
type RegisteredService struct {
	service *Service
	impl    any
	methods map[string]reflect.Method // "resource.method" -> reflect.Method
	inputs  map[string]reflect.Type   // "resource.method" -> input type
}

// Descriptor returns the contract service descriptor.
func (r *RegisteredService) Descriptor() *Service {
	return r.service
}

// Call implements Invoker.Call.
func (r *RegisteredService) Call(ctx context.Context, resource, method string, in any) (any, error) {
	key := resource + "." + method
	m, ok := r.methods[key]
	if !ok {
		return nil, fmt.Errorf("contract: method %s not found", key)
	}

	implVal := reflect.ValueOf(r.impl)
	args := []reflect.Value{reflect.ValueOf(ctx)}

	if in != nil {
		args = append(args, reflect.ValueOf(in))
	}

	results := implVal.Method(m.Index).Call(args)

	// Handle return values: (output, error) or (error)
	if len(results) == 1 {
		// Only error returned
		if !results[0].IsNil() {
			return nil, results[0].Interface().(error)
		}
		return nil, nil
	}

	// (output, error)
	var err error
	if !results[1].IsNil() {
		err = results[1].Interface().(error)
	}

	if results[0].IsNil() {
		return nil, err
	}
	return results[0].Interface(), err
}

// NewInput implements Invoker.NewInput.
func (r *RegisteredService) NewInput(resource, method string) (any, error) {
	key := resource + "." + method
	t, ok := r.inputs[key]
	if !ok {
		return nil, fmt.Errorf("contract: method %s not found", key)
	}
	if t == nil {
		return nil, nil // No input for this method
	}

	// Create new instance (handling pointer types)
	if t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem()).Interface(), nil
	}
	return reflect.New(t).Elem().Interface(), nil
}

// Stream implements Invoker.Stream.
func (r *RegisteredService) Stream(ctx context.Context, resource, method string, in any) (Stream, error) {
	// TODO: Implement streaming support
	return nil, ErrUnsupported
}

// Register creates a contract service from a Go interface and implementation.
// T must be an interface type. impl must implement T.
func Register[T any](impl T, opts ...Option) *RegisteredService {
	var t T
	ifaceType := reflect.TypeOf(&t).Elem()
	if ifaceType.Kind() != reflect.Interface {
		panic("contract: Register requires an interface type parameter")
	}

	implVal := reflect.ValueOf(impl)
	implType := implVal.Type()

	// Verify impl implements the interface
	if !implType.Implements(ifaceType) {
		panic(fmt.Sprintf("contract: %s does not implement %s", implType, ifaceType))
	}

	// Apply options
	o := &registerOptions{
		name:      ifaceType.Name(),
		resources: make(map[string][]string),
		http:      make(map[string]HTTPBinding),
		streaming: make(map[string]StreamMode),
	}
	for _, opt := range opts {
		opt(o)
	}

	// Build the service descriptor
	rs := &RegisteredService{
		impl:    impl,
		methods: make(map[string]reflect.Method),
		inputs:  make(map[string]reflect.Type),
	}

	svc := &Service{
		Name:        o.name,
		Description: o.description,
	}
	if o.defaults != nil {
		svc.Defaults = o.defaults
	}

	// Extract types from methods
	typeRegistry := newTypeRegistry()

	// Collect methods by resource
	resourceMethods := make(map[string][]*Method)

	for i := 0; i < ifaceType.NumMethod(); i++ {
		m := ifaceType.Method(i)
		methodName := toLowerCamel(m.Name)

		// Determine resource for this method
		resourceName := o.defaultResource
		if resourceName == "" {
			resourceName = toLowerSnake(svc.Name)
		}
		for res, methods := range o.resources {
			for _, mn := range methods {
				if mn == m.Name {
					resourceName = res
					break
				}
			}
		}

		// Extract method info
		method, inputType := extractMethod(m, methodName, typeRegistry, o)

		// Add HTTP binding if not provided
		if method.HTTP == nil {
			method.HTTP = inferHTTPBinding(m.Name, resourceName, inputType)
		}

		// Check for explicit HTTP override
		if binding, ok := o.http[m.Name]; ok {
			method.HTTP = &MethodHTTP{
				Method: binding.Method,
				Path:   binding.Path,
			}
		}

		// Check for streaming
		if mode, ok := o.streaming[m.Name]; ok {
			method.Stream = &struct {
				Mode      string  `json:"mode,omitempty" yaml:"mode,omitempty"`
				Item      TypeRef `json:"item" yaml:"item"`
				Done      TypeRef `json:"done,omitempty" yaml:"done,omitempty"`
				Error     TypeRef `json:"error,omitempty" yaml:"error,omitempty"`
				InputItem TypeRef `json:"input_item,omitempty" yaml:"input_item,omitempty"`
			}{
				Mode: string(mode),
				Item: method.Output,
			}
		}

		resourceMethods[resourceName] = append(resourceMethods[resourceName], method)

		// Store method mapping for invocation
		// Find the method on the implementation type
		implMethod, ok := implType.MethodByName(m.Name)
		if ok {
			rs.methods[resourceName+"."+methodName] = implMethod
			rs.inputs[resourceName+"."+methodName] = inputType
		}
	}

	// Build resources
	for name, methods := range resourceMethods {
		res := &Resource{
			Name:    name,
			Methods: methods,
		}
		svc.Resources = append(svc.Resources, res)
	}

	// Build types
	svc.Types = typeRegistry.types()

	rs.service = svc
	return rs
}

// extractMethod extracts a Method descriptor from a reflect.Method.
func extractMethod(m reflect.Method, methodName string, reg *typeRegistry, opts *registerOptions) (*Method, reflect.Type) {
	method := &Method{
		Name:        methodName,
		Description: "", // Could be extracted from comments via go/doc
	}

	mType := m.Type

	// Input: first non-context param (if any)
	var inputType reflect.Type
	for i := 0; i < mType.NumIn(); i++ {
		in := mType.In(i)
		if in.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			continue
		}
		inputType = in
		method.Input = reg.register(in)
		break
	}

	// Output: first non-error return (if any)
	for i := 0; i < mType.NumOut(); i++ {
		out := mType.Out(i)
		if out.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			continue
		}
		method.Output = reg.register(out)
		break
	}

	return method, inputType
}

// typeRegistry tracks discovered types.
type typeRegistry struct {
	seen  map[string]*Type
	order []string
}

func newTypeRegistry() *typeRegistry {
	return &typeRegistry{
		seen: make(map[string]*Type),
	}
}

func (r *typeRegistry) register(t reflect.Type) TypeRef {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Primitive types
	switch t.Kind() {
	case reflect.String:
		return TypeRef("string")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return TypeRef("int")
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return TypeRef("int")
	case reflect.Float32, reflect.Float64:
		return TypeRef("float")
	case reflect.Bool:
		return TypeRef("bool")
	}

	// Special types
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return TypeRef("string") // ISO 8601
	}
	if t.PkgPath() == "encoding/json" && t.Name() == "RawMessage" {
		return TypeRef("any")
	}

	name := t.Name()
	if name == "" {
		// Anonymous type - generate name
		name = fmt.Sprintf("Type%d", len(r.seen))
	}

	// Already registered?
	if _, ok := r.seen[name]; ok {
		return TypeRef(name)
	}

	// Register based on kind
	switch t.Kind() {
	case reflect.Struct:
		r.registerStruct(name, t)
	case reflect.Slice:
		return r.registerSlice(name, t)
	case reflect.Map:
		r.registerMap(name, t)
	}

	return TypeRef(name)
}

func (r *typeRegistry) registerStruct(name string, t reflect.Type) {
	typ := &Type{
		Name: name,
		Kind: KindStruct,
	}
	r.seen[name] = typ
	r.order = append(r.order, name)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		field := Field{
			Name: getJSONName(f),
			Type: r.register(f.Type),
		}

		// Check for optional (omitempty)
		if tag := f.Tag.Get("json"); strings.Contains(tag, "omitempty") {
			field.Optional = true
		}

		// Check for nullable (pointer type or explicit tag)
		if f.Type.Kind() == reflect.Ptr {
			field.Nullable = true
		}
		if tag := f.Tag.Get("nullable"); tag == "true" {
			field.Nullable = true
		}

		// Check for required tag
		if tag := f.Tag.Get("required"); tag == "true" {
			field.Optional = false
		}

		// Check for enum
		if tag := f.Tag.Get("enum"); tag != "" {
			field.Enum = strings.Split(tag, ",")
		}

		// Check for description
		if tag := f.Tag.Get("desc"); tag != "" {
			field.Description = tag
		}

		typ.Fields = append(typ.Fields, field)
	}
}

func (r *typeRegistry) registerSlice(name string, t reflect.Type) TypeRef {
	elemRef := r.register(t.Elem())
	return TypeRef("[]" + string(elemRef))
}

func (r *typeRegistry) registerMap(name string, t reflect.Type) {
	typ := &Type{
		Name: name,
		Kind: KindMap,
		Elem: r.register(t.Elem()),
	}
	r.seen[name] = typ
	r.order = append(r.order, name)
}

func (r *typeRegistry) types() []*Type {
	result := make([]*Type, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.seen[name])
	}
	return result
}

// getJSONName returns the JSON field name from struct tag.
func getJSONName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" || tag == "-" {
		return toLowerSnake(f.Name)
	}
	parts := strings.Split(tag, ",")
	if parts[0] != "" {
		return parts[0]
	}
	return toLowerSnake(f.Name)
}

// toLowerCamel converts PascalCase to camelCase.
func toLowerCamel(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// toLowerSnake converts PascalCase to snake_case.
func toLowerSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
