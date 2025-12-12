// Package validator provides request validation middleware for Mizu.
package validator

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
)

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Field+": "+err.Message)
	}
	return strings.Join(msgs, "; ")
}

// Rule represents a validation rule.
type Rule struct {
	Field    string
	Rules    []string
	Message  string
	Optional bool
}

// Options configures the validator middleware.
type Options struct {
	// Rules defines validation rules.
	Rules []Rule

	// ErrorHandler handles validation errors.
	ErrorHandler func(c *mizu.Ctx, errors ValidationErrors) error
}

// New creates validator middleware with rules.
func New(rules ...Rule) mizu.Middleware {
	return WithOptions(Options{Rules: rules})
}

// WithOptions creates validator middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.ErrorHandler == nil {
		opts.ErrorHandler = defaultErrorHandler
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			errors := validate(c, opts.Rules)
			if len(errors) > 0 {
				return opts.ErrorHandler(c, errors)
			}
			return next(c)
		}
	}
}

func validate(c *mizu.Ctx, rules []Rule) ValidationErrors {
	var errors ValidationErrors

	for _, rule := range rules {
		value := getValue(c, rule.Field)

		if value == "" && rule.Optional {
			continue
		}

		for _, r := range rule.Rules {
			if err := applyRule(rule.Field, value, r, rule.Message); err != nil {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

func getValue(c *mizu.Ctx, field string) string {
	// Try query parameter
	if val := c.Query(field); val != "" {
		return val
	}

	// Try form value
	if c.Request().Method == http.MethodPost || c.Request().Method == http.MethodPut {
		if val := c.Request().FormValue(field); val != "" {
			return val
		}
	}

	// Try header
	if val := c.Request().Header.Get(field); val != "" {
		return val
	}

	// Try path parameter
	if val := c.Param(field); val != "" {
		return val
	}

	return ""
}

func applyRule(field, value, rule, customMessage string) *ValidationError {
	parts := strings.SplitN(rule, ":", 2)
	ruleName := parts[0]
	var ruleParam string
	if len(parts) > 1 {
		ruleParam = parts[1]
	}

	var message string
	var valid bool

	switch ruleName {
	case "required":
		valid = value != ""
		message = "is required"

	case "min":
		minLen, _ := strconv.Atoi(ruleParam)
		valid = len(value) >= minLen
		message = "must be at least " + ruleParam + " characters"

	case "max":
		maxLen, _ := strconv.Atoi(ruleParam)
		valid = len(value) <= maxLen
		message = "must be at most " + ruleParam + " characters"

	case "email":
		valid = strings.Contains(value, "@") && strings.Contains(value, ".")
		message = "must be a valid email"

	case "numeric":
		_, err := strconv.ParseFloat(value, 64)
		valid = err == nil
		message = "must be numeric"

	case "integer":
		_, err := strconv.ParseInt(value, 10, 64)
		valid = err == nil
		message = "must be an integer"

	case "alpha":
		valid = true
		for _, r := range value {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
				valid = false
				break
			}
		}
		message = "must contain only letters"

	case "alphanum":
		valid = true
		for _, r := range value {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				valid = false
				break
			}
		}
		message = "must contain only letters and numbers"

	case "in":
		options := strings.Split(ruleParam, ",")
		valid = false
		for _, opt := range options {
			if value == opt {
				valid = true
				break
			}
		}
		message = "must be one of: " + ruleParam

	case "url":
		valid = strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
		message = "must be a valid URL"

	case "uuid":
		valid = len(value) == 36 &&
			value[8] == '-' && value[13] == '-' &&
			value[18] == '-' && value[23] == '-'
		message = "must be a valid UUID"

	default:
		return nil
	}

	if !valid {
		if customMessage != "" {
			message = customMessage
		}
		return &ValidationError{Field: field, Message: message}
	}

	return nil
}

func defaultErrorHandler(c *mizu.Ctx, errors ValidationErrors) error {
	return c.JSON(http.StatusBadRequest, map[string]any{
		"error":  "validation failed",
		"errors": errors,
	})
}

// JSON validates JSON body against rules.
func JSON(rules map[string][]string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Read body
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return c.Text(http.StatusBadRequest, "invalid body")
			}

			// Parse JSON
			var data map[string]any
			if err := json.Unmarshal(body, &data); err != nil {
				return c.Text(http.StatusBadRequest, "invalid JSON")
			}

			// Validate
			var errors ValidationErrors
			for field, fieldRules := range rules {
				value, _ := data[field].(string)
				if value == "" {
					if v, ok := data[field]; ok {
						value = stringify(v)
					}
				}

				for _, rule := range fieldRules {
					if err := applyRule(field, value, rule, ""); err != nil {
						errors = append(errors, *err)
					}
				}
			}

			if len(errors) > 0 {
				return c.JSON(http.StatusBadRequest, map[string]any{
					"error":  "validation failed",
					"errors": errors,
				})
			}

			return next(c)
		}
	}
}

func stringify(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return ""
	}
}

// Field creates a validation rule for a field.
func Field(name string, rules ...string) Rule {
	return Rule{Field: name, Rules: rules}
}

// OptionalField creates an optional validation rule.
func OptionalField(name string, rules ...string) Rule {
	return Rule{Field: name, Rules: rules, Optional: true}
}
