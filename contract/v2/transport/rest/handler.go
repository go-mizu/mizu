package rest

import (
	"net/http"

	"github.com/go-mizu/mizu"
	contract "github.com/go-mizu/mizu/contract/v2"
)

// errorResponse is the standard error response format.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// makeHandler creates a mizu.Handler for a contract method.
func makeHandler(inv contract.Invoker, rt mizuRoute, opts *options) mizu.Handler {
	return func(c *mizu.Ctx) error {
		ctx := c.Context()

		var in any
		if rt.hasInput {
			var err error
			in, err = inv.NewInput(rt.resource, rt.method)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, errorResponse{
					Error:   "internal_error",
					Message: "failed to create input: " + err.Error(),
				})
			}
			if in == nil {
				return c.JSON(http.StatusInternalServerError, errorResponse{
					Error:   "internal_error",
					Message: "invoker returned nil input",
				})
			}

			// Fill from path parameters first (highest priority)
			if err := fillFromPath(in, c, rt.pathParams); err != nil {
				return c.JSON(http.StatusBadRequest, errorResponse{
					Error:   "bad_path",
					Message: err.Error(),
				})
			}

			// Fill from query or body based on method
			if c.Request().Method == http.MethodGet {
				// GET: use query parameters
				if err := fillFromQuery(in, c.QueryValues()); err != nil {
					return c.JSON(http.StatusBadRequest, errorResponse{
						Error:   "bad_query",
						Message: err.Error(),
					})
				}
			} else if c.Request().ContentLength > 0 {
				// Non-GET with body: parse JSON
				if err := c.BindJSON(in, opts.maxBodySize); err != nil {
					return c.JSON(http.StatusBadRequest, errorResponse{
						Error:   "bad_json",
						Message: err.Error(),
					})
				}
			} else if !opts.strictRouting {
				// Non-GET without body: try query params as fallback
				if err := fillFromQuery(in, c.QueryValues()); err != nil {
					return c.JSON(http.StatusBadRequest, errorResponse{
						Error:   "bad_query",
						Message: err.Error(),
					})
				}
			}
		}

		// Invoke the contract method
		out, err := inv.Call(ctx, rt.resource, rt.method, in)
		if err != nil {
			status, code, msg := opts.errorMapper(err)
			return c.JSON(status, errorResponse{
				Error:   code,
				Message: msg,
			})
		}

		// Return response
		if !rt.hasOutput {
			return c.NoContent()
		}
		return c.JSON(http.StatusOK, out)
	}
}
