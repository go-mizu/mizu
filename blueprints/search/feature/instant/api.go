// Package instant provides instant answer functionality.
package instant

import "github.com/go-mizu/mizu/blueprints/search/store"

// API defines the instant answer service contract.
type API interface {
	// Calculate evaluates a mathematical expression.
	Calculate(expr string) (*store.InstantAnswer, error)

	// ConvertUnit converts between units.
	ConvertUnit(value float64, from, to string) (*store.InstantAnswer, error)

	// ConvertCurrency converts between currencies.
	ConvertCurrency(amount float64, from, to string) (*store.InstantAnswer, error)

	// GetWeather returns weather for a location.
	GetWeather(location string) *store.InstantAnswer

	// Define returns dictionary definition for a word.
	Define(word string) (*store.InstantAnswer, error)

	// GetTime returns current time for a location.
	GetTime(location string) *store.InstantAnswer

	// Detect attempts to detect and compute an instant answer from a query.
	Detect(query string) *store.InstantAnswer
}
