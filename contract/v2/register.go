package contract

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// invoker is the concrete runtime implementation produced by Register.
// It is intentionally unexported. Users depend only on the Invoker interface.
type invoker struct {
	service *Service
	impl    any

	// "resource.method" -> bound method value (receiver already bound)
	methods map[string]reflect.Value

	// "resource.method" -> input type (nil means no input)
	inputs map[string]reflect.Type
}

// Descriptor returns the contract service descriptor.
func (r *invoker) Descriptor() *Service { return r.service }

// Call implements Invoker.Call.
func (r *invoker) Call(ctx context.Context, resource, method string, in any) (any, error) {
	key := resource + "." + method
	fn, ok := r.methods[key]
	if !ok || !fn.IsValid() {
		return nil, fmt.Errorf("contract: method %s not found", key)
	}

	wantIn, hasWant := r.inputs[key]
	if !hasWant {
		return nil, fmt.Errorf("contract: method %s not found", key)
	}

	args := []reflect.Value{reflect.ValueOf(ctx)}

	if wantIn != nil {
		if in == nil {
			return nil, fmt.Errorf("contract: method %s requires input", key)
		}
		v := reflect.ValueOf(in)
		if !v.IsValid() {
			return nil, fmt.Errorf("contract: method %s requires input", key)
		}
		if !v.Type().AssignableTo(wantIn) {
			return nil, fmt.Errorf("contract: method %s: input type %s not assignable to %s", key, v.Type(), wantIn)
		}
		args = append(args, v)
	} else {
		if in != nil {
			return nil, fmt.Errorf("contract: method %s does not accept input", key)
		}
	}

	results := fn.Call(args)

	switch len(results) {
	case 1:
		// (error)
		if !results[0].IsNil() {
			return nil, results[0].Interface().(error)
		}
		return nil, nil

	case 2:
		// (output, error)
		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}

		out := results[0]
		if isNilable(out.Kind()) && out.IsNil() {
			return nil, err
		}
		return out.Interface(), err

	default:
		return nil, fmt.Errorf("contract: method %s returned %d values (expected 1 or 2)", key, len(results))
	}
}

// NewInput implements Invoker.NewInput.
func (r *invoker) NewInput(resource, method string) (any, error) {
	key := resource + "." + method
	t, ok := r.inputs[key]
	if !ok {
		return nil, fmt.Errorf("contract: method %s not found", key)
	}
	if t == nil {
		return nil, nil
	}

	// Always return a pointer when input is a pointer type.
	if t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem()).Interface(), nil
	}

	// For non-pointer input types, return a value instance.
	// (This is allowed, but Register validates inputs are pointer-to-struct by default.)
	return reflect.New(t).Elem().Interface(), nil
}

// Stream implements Invoker.Stream.
func (r *invoker) Stream(ctx context.Context, resource, method string, in any) (Stream, error) {
	return nil, ErrUnsupported
}

// Register creates a contract service from a Go interface and implementation.
// T must be an interface type. impl must implement T.
func Register[T any](impl T, opts ...Option) Invoker {
	var t T
	ifaceType := reflect.TypeOf(&t).Elem()
	if ifaceType.Kind() != reflect.Interface {
		panic("contract: Register requires an interface type parameter")
	}

	implVal := reflect.ValueOf(impl)
	implType := implVal.Type()

	// Verify impl implements the interface.
	if !implType.Implements(ifaceType) {
		panic(fmt.Sprintf("contract: %s does not implement %s", implType, ifaceType))
	}

	// Apply options.
	o := &registerOptions{
		name:      ifaceType.Name(),
		resources: make(map[string][]string),
		http:      make(map[string]HTTPBinding),
		streaming: make(map[string]StreamMode),
	}
	for _, opt := range opts {
		opt(o)
	}

	// Build runtime.
	rs := &invoker{
		impl:    impl,
		methods: make(map[string]reflect.Value),
		inputs:  make(map[string]reflect.Type),
	}

	// Build descriptor.
	svc := &Service{
		Name:        o.name,
		Description: o.description,
	}
	if o.defaults != nil {
		svc.Defaults = o.defaults
	}

	reg := newTypeRegistry()
	resourceMethods := make(map[string][]*Method)

	for i := 0; i < ifaceType.NumMethod(); i++ {
		im := ifaceType.Method(i)

		// Validate the method signature early to avoid confusing runtime behavior.
		inType, outType, sigErr := validateAndExtractSignature(ifaceType, im)
		if sigErr != nil {
			panic(sigErr.Error())
		}

		methodName := toLowerCamel(im.Name)

		// Determine resource for this method.
		resourceName := o.defaultResource
		if resourceName == "" {
			resourceName = toLowerSnake(svc.Name)
		}
		for res, methods := range o.resources {
			for _, mn := range methods {
				if mn == im.Name {
					resourceName = res
					break
				}
			}
		}

		// Build method descriptor.
		m := &Method{
			Name:        methodName,
			Description: "",
		}

		// Register types.
		if inType != nil {
			m.Input = reg.register(inType)
		}
		if outType != nil {
			m.Output = reg.register(outType)
		}

		// Default HTTP binding if not provided.
		if m.HTTP == nil {
			m.HTTP = inferHTTPBinding(im.Name, resourceName, inType)
		}

		// Explicit HTTP override.
		if binding, ok := o.http[im.Name]; ok {
			m.HTTP = &MethodHTTP{
				Method: binding.Method,
				Path:   binding.Path,
			}
		}

		// Streaming.
		if mode, ok := o.streaming[im.Name]; ok {
			m.Stream = &struct {
				Mode      string  `json:"mode,omitempty" yaml:"mode,omitempty"`
				Item      TypeRef `json:"item" yaml:"item"`
				Done      TypeRef `json:"done,omitempty" yaml:"done,omitempty"`
				Error     TypeRef `json:"error,omitempty" yaml:"error,omitempty"`
				InputItem TypeRef `json:"input_item,omitempty" yaml:"input_item,omitempty"`
			}{
				Mode: string(mode),
				Item: m.Output,
			}
		}

		resourceMethods[resourceName] = append(resourceMethods[resourceName], m)

		// Bind method for invocation using a direct reflect.Value (avoid Index pitfalls).
		bound := implVal.MethodByName(im.Name)
		if !bound.IsValid() {
			// Should not happen if Implements check passed, but keep deterministic.
			panic(fmt.Sprintf("contract: method %s not found on implementation %s", im.Name, implType))
		}

		key := resourceName + "." + methodName
		rs.methods[key] = bound
		rs.inputs[key] = inType
	}

	// Build resources.
	for name, methods := range resourceMethods {
		res := &Resource{
			Name:    name,
			Methods: methods,
		}
		svc.Resources = append(svc.Resources, res)
	}

	// Build types.
	svc.Types = reg.types()
	rs.service = svc
	return rs
}

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

// validateAndExtractSignature enforces supported shapes and returns:
// - inType: the single non-context input type (or nil)
// - outType: the single non-error output type (or nil)
// Supported:
//
//	M(ctx) error
//	M(ctx) (*Out, error)
//	M(ctx, *In) error
//	M(ctx, *In) (*Out, error)
func validateAndExtractSignature(iface reflect.Type, m reflect.Method) (inType reflect.Type, outType reflect.Type, err error) {
	mt := m.Type

	// Inputs: first param must be context.Context.
	if mt.NumIn() < 1 || !mt.In(0).Implements(contextType) {
		return nil, nil, fmt.Errorf("contract: %s.%s: first parameter must be context.Context", iface.Name(), m.Name)
	}
	if mt.NumIn() > 2 {
		return nil, nil, fmt.Errorf("contract: %s.%s: too many parameters (expected ctx or ctx+*In)", iface.Name(), m.Name)
	}
	if mt.NumIn() == 2 {
		inType = mt.In(1)
		// Keep v2 strict: require pointer input for stable optional semantics.
		if inType.Kind() != reflect.Ptr || inType.Elem().Kind() != reflect.Struct {
			return nil, nil, fmt.Errorf("contract: %s.%s: input must be pointer to struct (got %s)", iface.Name(), m.Name, inType)
		}
	}

	// Outputs: either (error) or (Out, error).
	switch mt.NumOut() {
	case 1:
		if !mt.Out(0).Implements(errorType) {
			return nil, nil, fmt.Errorf("contract: %s.%s: single return value must be error", iface.Name(), m.Name)
		}
		return inType, nil, nil

	case 2:
		if mt.Out(0).Implements(errorType) || !mt.Out(1).Implements(errorType) {
			return nil, nil, fmt.Errorf("contract: %s.%s: expected (Out, error)", iface.Name(), m.Name)
		}
		outType = mt.Out(0)
		return inType, outType, nil

	default:
		return nil, nil, fmt.Errorf("contract: %s.%s: invalid return arity %d (expected 1 or 2)", iface.Name(), m.Name, mt.NumOut())
	}
}

func isNilable(k reflect.Kind) bool {
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Interface, reflect.Slice:
		return true
	default:
		return false
	}
}

//
// Type registry (internal)
//

type typeRegistry struct {
	seen     map[string]*Type // key -> *Type
	order    []string         // keys in insertion order
	nameUsed map[string]int   // public schema name -> count
	keyToRef map[string]TypeRef
}

func newTypeRegistry() *typeRegistry {
	return &typeRegistry{
		seen:     make(map[string]*Type),
		nameUsed: make(map[string]int),
		keyToRef: make(map[string]TypeRef),
	}
}

func (r *typeRegistry) register(t reflect.Type) TypeRef {
	// Handle pointer types.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Primitives.
	switch t.Kind() {
	case reflect.String:
		return TypeRef("string")
	case reflect.Bool:
		return TypeRef("bool")
	case reflect.Int:
		return TypeRef("int")
	case reflect.Int8:
		return TypeRef("int8")
	case reflect.Int16:
		return TypeRef("int16")
	case reflect.Int32:
		return TypeRef("int32")
	case reflect.Int64:
		return TypeRef("int64")
	case reflect.Uint:
		return TypeRef("uint")
	case reflect.Uint8:
		return TypeRef("uint8")
	case reflect.Uint16:
		return TypeRef("uint16")
	case reflect.Uint32:
		return TypeRef("uint32")
	case reflect.Uint64:
		return TypeRef("uint64")
	case reflect.Float32:
		return TypeRef("float32")
	case reflect.Float64:
		return TypeRef("float64")
	}

	// Special types.
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return TypeRef("time.Time")
	}
	if t.PkgPath() == "encoding/json" && t.Name() == "RawMessage" {
		return TypeRef("any")
	}

	// Slices: keep as a TypeRef grammar for now (minimal change).
	if t.Kind() == reflect.Slice {
		elemRef := r.register(t.Elem())
		return TypeRef("[]" + string(elemRef))
	}

	// Compute a stable registry key.
	key := typeKey(t)

	// Already registered?
	if ref, ok := r.keyToRef[key]; ok {
		return ref
	}

	// Choose a schema name, ensuring no collisions across packages.
	name := t.Name()
	if name == "" {
		name = fmt.Sprintf("Type%d", len(r.seen)+1)
	}
	name = r.uniqueName(name)

	// Register based on kind.
	switch t.Kind() {
	case reflect.Struct:
		r.registerStruct(key, name, t)
	case reflect.Map:
		r.registerMap(key, name, t)
	default:
		// For unnamed / unsupported complex kinds, treat as external/any.
		// Keep minimal behavior: do not panic, but make it explicit.
		return TypeRef("any")
	}

	ref := TypeRef(name)
	r.keyToRef[key] = ref
	return ref
}

func typeKey(t reflect.Type) string {
	// Named types: fully qualified.
	if t.Name() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	// Unnamed types: include kind and element info where relevant.
	switch t.Kind() {
	case reflect.Map:
		return "map[string]" + typeKey(t.Elem())
	case reflect.Struct:
		// Anonymous struct types are hard to canonicalize; treat as unique by address.
		// reflect.Type.String() includes field layout and is stable enough for this use.
		return "struct:" + t.String()
	default:
		return t.Kind().String() + ":" + t.String()
	}
}

func (r *typeRegistry) uniqueName(base string) string {
	n := r.nameUsed[base]
	if n == 0 {
		r.nameUsed[base] = 1
		return base
	}
	n++
	r.nameUsed[base] = n
	return fmt.Sprintf("%s_%d", base, n)
}

func (r *typeRegistry) registerStruct(key, name string, t reflect.Type) {
	typ := &Type{
		Name: name,
		Kind: KindStruct,
	}
	r.seen[key] = typ
	r.order = append(r.order, key)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			// Minimal fix: skip anonymous fields to avoid surprising flattening.
			// (Future enhancement can expand embedded fields explicitly.)
			continue
		}

		field := Field{
			Name: getJSONName(f),
			Type: r.register(f.Type),
		}

		// Optional (omitempty).
		if tag := f.Tag.Get("json"); strings.Contains(tag, "omitempty") {
			field.Optional = true
		}

		// Nullable (pointer type or explicit tag).
		if f.Type.Kind() == reflect.Ptr {
			field.Nullable = true
		}
		if tag := f.Tag.Get("nullable"); tag == "true" {
			field.Nullable = true
		}

		// Required override.
		if tag := f.Tag.Get("required"); tag == "true" {
			field.Optional = false
		}

		// Enum.
		if tag := f.Tag.Get("enum"); tag != "" {
			field.Enum = strings.Split(tag, ",")
		}

		// Description.
		if tag := f.Tag.Get("desc"); tag != "" {
			field.Description = tag
		}

		typ.Fields = append(typ.Fields, field)
	}
}

func (r *typeRegistry) registerMap(key, name string, t reflect.Type) {
	// Only support string keys as per contract design.
	if t.Key().Kind() != reflect.String {
		// Keep minimal behavior: represent as any.
		r.seen[key] = &Type{Name: name, Kind: KindMap, Elem: TypeRef("any")}
		r.order = append(r.order, key)
		return
	}

	typ := &Type{
		Name: name,
		Kind: KindMap,
		Elem: r.register(t.Elem()),
	}
	r.seen[key] = typ
	r.order = append(r.order, key)
}

func (r *typeRegistry) types() []*Type {
	result := make([]*Type, 0, len(r.order))
	for _, key := range r.order {
		result = append(result, r.seen[key])
	}
	return result
}

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

func toLowerCamel(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func toLowerSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
