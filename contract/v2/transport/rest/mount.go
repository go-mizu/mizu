package rest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu"
	contract "github.com/go-mizu/mizu/contract/v2"
)

// Mount registers all contract routes on a mizu router.
// This is the primary API for integrating contracts with mizu.
func Mount(r *mizu.Router, inv contract.Invoker, opts ...Option) error {
	if r == nil {
		return errors.New("rest: nil router")
	}
	routes, err := Routes(inv, opts...)
	if err != nil {
		return err
	}
	for _, rt := range routes {
		r.Handle(rt.Method, rt.Path, rt.Handler)
	}
	return nil
}

// MountAt registers contract routes under a path prefix.
func MountAt(r *mizu.Router, prefix string, inv contract.Invoker, opts ...Option) error {
	if r == nil {
		return errors.New("rest: nil router")
	}
	return Mount(r.Prefix(prefix), inv, opts...)
}

// Handler returns a single mizu.Handler that routes all contract methods.
// Useful when you need manual control over routing or want to mount
// on an http.ServeMux using mizu.Adapt.
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error) {
	routes, err := buildMizuRoutes(inv, applyOptions(opts))
	if err != nil {
		return nil, err
	}

	// Build a simple route table for matching
	return func(c *mizu.Ctx) error {
		method := c.Request().Method
		path := c.Request().URL.Path

		for _, rt := range routes {
			if rt.httpMethod != method {
				continue
			}
			if matchPattern(rt.path, path) {
				return makeHandler(inv, rt, applyOptions(opts))(c)
			}
		}

		c.Status(404)
		return c.JSON(404, errorResponse{
			Error:   "not_found",
			Message: "no matching route",
		})
	}, nil
}

// Routes returns route definitions for manual registration.
// Each Route contains Method, Path, Resource, Name, and Handler.
func Routes(inv contract.Invoker, opts ...Option) ([]Route, error) {
	o := applyOptions(opts)
	internal, err := buildMizuRoutes(inv, o)
	if err != nil {
		return nil, err
	}

	routes := make([]Route, len(internal))
	for i, rt := range internal {
		routes[i] = Route{
			Method:   rt.httpMethod,
			Path:     rt.path,
			Resource: rt.resource,
			Name:     rt.method,
			Handler:  makeHandler(inv, rt, o),
		}
	}
	return routes, nil
}

// buildMizuRoutes creates internal route representations from a contract.
func buildMizuRoutes(inv contract.Invoker, opts *options) ([]mizuRoute, error) {
	if inv == nil {
		return nil, errors.New("rest: nil invoker")
	}
	svc := inv.Descriptor()
	if svc == nil {
		return nil, errors.New("rest: nil descriptor")
	}

	var routes []mizuRoute
	for _, res := range svc.Resources {
		if res == nil {
			continue
		}
		for _, m := range res.Methods {
			if m == nil || m.HTTP == nil {
				continue
			}

			httpMethod := strings.ToUpper(strings.TrimSpace(m.HTTP.Method))
			path := strings.TrimSpace(m.HTTP.Path)
			if httpMethod == "" || path == "" || !strings.HasPrefix(path, "/") {
				return nil, fmt.Errorf("rest: invalid http binding for %s.%s", res.Name, m.Name)
			}

			routes = append(routes, mizuRoute{
				httpMethod: httpMethod,
				path:       path,
				resource:   res.Name,
				method:     m.Name,
				pathParams: extractPathParams(path),
				hasInput:   m.Input != "",
				hasOutput:  m.Output != "",
			})
		}
	}

	if len(routes) == 0 {
		return nil, errors.New("rest: no http routes in descriptor")
	}
	return routes, nil
}

// matchPattern checks if a request path matches a route pattern.
// This is a simple implementation for the Handler() function.
// When using Mount(), mizu.Router handles matching natively.
func matchPattern(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, p := range patternParts {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			// Parameter - always matches
			continue
		}
		if p != pathParts[i] {
			return false
		}
	}
	return true
}

// OpenAPI generates an OpenAPI 3.0 specification from a contract descriptor.
// This is a convenience wrapper around OpenAPIDocument.
func OpenAPI(svc *contract.Service) ([]byte, error) {
	return OpenAPIDocument(svc)
}
