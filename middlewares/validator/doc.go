// Package validator provides declarative request validation middleware for Mizu.
//
// The validator middleware enables you to validate query parameters, form fields,
// headers, and path parameters using built-in rules or custom validation logic.
//
// # Basic Usage
//
// Create validation rules using the Field and OptionalField helper functions:
//
//	app := mizu.New()
//	app.Post("/users", createUser, validator.New(
//	    validator.Field("email", "required", "email"),
//	    validator.Field("name", "required", "min:2", "max:50"),
//	    validator.OptionalField("age", "numeric"),
//	))
//
// # Built-in Validation Rules
//
// The validator provides the following built-in rules:
//
//   - required: Field must not be empty
//   - min:n: Minimum string length (e.g., "min:3")
//   - max:n: Maximum string length (e.g., "max:100")
//   - email: Valid email format (contains @ and .)
//   - numeric: Numeric value (integer or float)
//   - integer: Integer value only
//   - alpha: Letters only (a-z, A-Z)
//   - alphanum: Letters and numbers only
//   - in:a,b,c: Value must be one of the listed options
//   - url: Valid URL (starts with http:// or https://)
//   - uuid: Valid UUID format (36 characters with hyphens)
//
// # Field Value Sources
//
// The validator checks for field values in the following order:
//
//  1. Query parameters (highest priority)
//  2. Form values (POST/PUT requests)
//  3. Headers
//  4. Path parameters (lowest priority)
//
// The first non-empty value found is used for validation.
//
// # JSON Body Validation
//
// For JSON request bodies, use the JSON function:
//
//	app.Post("/api/users", createUser, validator.JSON(map[string][]string{
//	    "email":    {"required", "email"},
//	    "name":     {"required", "min:2"},
//	    "age":      {"numeric"},
//	}))
//
// # Custom Error Messages
//
// Override default error messages using the Rule struct:
//
//	validator.New(
//	    validator.Rule{
//	        Field:   "email",
//	        Rules:   []string{"required", "email"},
//	        Message: "Please provide a valid email address",
//	    },
//	)
//
// # Custom Error Handler
//
// Customize error response format using WithOptions:
//
//	validator.WithOptions(validator.Options{
//	    Rules: []validator.Rule{
//	        validator.Field("email", "required", "email"),
//	    },
//	    ErrorHandler: func(c *mizu.Ctx, errors validator.ValidationErrors) error {
//	        return c.JSON(422, map[string]any{
//	            "message": "Validation failed",
//	            "errors":  errors,
//	        })
//	    },
//	})
//
// # Default Error Response
//
// When validation fails, the default error response is:
//
//	{
//	    "error": "validation failed",
//	    "errors": [
//	        {"field": "email", "message": "is required"},
//	        {"field": "name", "message": "must be at least 2 characters"}
//	    ]
//	}
//
// The response status code is 400 Bad Request by default.
//
// # Optional Fields
//
// Fields marked as optional are skipped if they are empty. If a value is provided,
// all validation rules are applied normally:
//
//	validator.New(
//	    validator.Field("username", "required"),
//	    validator.OptionalField("bio", "max:500"),      // only validated if present
//	    validator.OptionalField("website", "url"),      // only validated if present
//	)
//
// # Validation Flow
//
// The validation process follows these steps:
//
//  1. Extract field values from request (query, form, headers, path params)
//  2. For each validation rule:
//     - Skip optional fields that are empty
//     - Apply each rule to the field value
//     - Collect validation errors
//  3. If errors exist, call error handler (or use default)
//  4. If no errors, proceed to next middleware/handler
//
// # Performance Characteristics
//
// The validator is designed for efficiency:
//
//   - Memory: Minimal allocation for most validations; JSON validation reads the entire body into memory
//   - CPU: Linear time complexity with number of fields and rules; no regex for better performance
//   - Concurrency: Safe for concurrent use; no shared state between requests
//
// # Thread Safety
//
// All validator functions are safe for concurrent use. Each request has its own
// validation context with no shared state.
package validator
