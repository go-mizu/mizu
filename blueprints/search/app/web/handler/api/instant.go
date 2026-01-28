package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/feature/instant"
)

// InstantHandler handles instant answer API requests
type InstantHandler struct {
	service *instant.Service
}

// NewInstantHandler creates a new instant handler
func NewInstantHandler() *InstantHandler {
	return &InstantHandler{service: instant.NewService()}
}

// Calculate handles calculator requests
func (h *InstantHandler) Calculate(c *mizu.Ctx) error {
	expr := c.Query("expr")
	if expr == "" {
		return c.JSON(400, map[string]string{"error": "expression required"})
	}

	answer, err := h.service.Calculate(expr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, answer)
}

// Convert handles unit conversion requests
func (h *InstantHandler) Convert(c *mizu.Ctx) error {
	value := c.Query("value")
	from := c.Query("from")
	to := c.Query("to")

	if value == "" || from == "" || to == "" {
		return c.JSON(400, map[string]string{"error": "value, from, and to parameters required"})
	}

	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid value"})
	}

	answer, err := h.service.ConvertUnit(val, from, to)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, answer)
}

// Currency handles currency conversion requests
func (h *InstantHandler) Currency(c *mizu.Ctx) error {
	amount := c.Query("amount")
	from := c.Query("from")
	to := c.Query("to")

	if amount == "" || from == "" || to == "" {
		return c.JSON(400, map[string]string{"error": "amount, from, and to parameters required"})
	}

	val, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid amount"})
	}

	answer, err := h.service.ConvertCurrency(val, from, to)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, answer)
}

// Weather handles weather requests
func (h *InstantHandler) Weather(c *mizu.Ctx) error {
	location := c.Query("location")
	return c.JSON(200, h.service.GetWeather(location))
}

// Define handles dictionary definition requests
func (h *InstantHandler) Define(c *mizu.Ctx) error {
	word := c.Query("word")
	if word == "" {
		return c.JSON(400, map[string]string{"error": "word parameter required"})
	}

	answer, err := h.service.Define(word)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "definition not found"})
	}

	return c.JSON(200, answer)
}

// Time handles world time requests
func (h *InstantHandler) Time(c *mizu.Ctx) error {
	location := c.Query("location")
	return c.JSON(200, h.service.GetTime(location))
}
