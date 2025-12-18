package contract

// registerOptions holds configuration for service registration.
type registerOptions struct {
	name            string
	description     string
	defaults        *Defaults
	resources       map[string][]string // resource name -> method names
	defaultResource string
	http            map[string]HTTPBinding
	streaming       map[string]StreamMode
}

// Option configures the registration process.
type Option func(*registerOptions)

// HTTPBinding specifies the HTTP method and path for an API method.
type HTTPBinding struct {
	Method string // GET, POST, PUT, DELETE, PATCH
	Path   string // /todos, /todos/{id}
}

// StreamMode specifies the streaming protocol.
type StreamMode string

const (
	StreamSSE  StreamMode = "sse"
	StreamWS   StreamMode = "ws"
	StreamGRPC StreamMode = "grpc"
	StreamAsync StreamMode = "async"
)

// WithName sets the service name (default: interface name).
func WithName(name string) Option {
	return func(o *registerOptions) {
		o.name = name
	}
}

// WithDescription sets the service description.
func WithDescription(desc string) Option {
	return func(o *registerOptions) {
		o.description = desc
	}
}

// WithResource groups methods into a named resource.
// Methods not explicitly grouped go into a default resource.
func WithResource(name string, methods ...string) Option {
	return func(o *registerOptions) {
		o.resources[name] = methods
	}
}

// WithDefaultResource sets the default resource name for methods
// not explicitly assigned to a resource.
func WithDefaultResource(name string) Option {
	return func(o *registerOptions) {
		o.defaultResource = name
	}
}

// WithDefaults sets service-level defaults.
func WithDefaults(defaults Defaults) Option {
	return func(o *registerOptions) {
		o.defaults = &defaults
	}
}

// WithHTTP provides explicit HTTP bindings for methods.
func WithHTTP(bindings map[string]HTTPBinding) Option {
	return func(o *registerOptions) {
		for k, v := range bindings {
			o.http[k] = v
		}
	}
}

// WithMethodHTTP provides an HTTP binding for a single method.
func WithMethodHTTP(method string, httpMethod string, path string) Option {
	return func(o *registerOptions) {
		o.http[method] = HTTPBinding{
			Method: httpMethod,
			Path:   path,
		}
	}
}

// WithStreaming marks methods as streaming with specified mode.
func WithStreaming(method string, mode StreamMode) Option {
	return func(o *registerOptions) {
		o.streaming[method] = mode
	}
}
