package rest

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/menus"
)

// Menus handles menu endpoints.
type Menus struct {
	menus menus.API
}

// NewMenus creates a new menus handler.
func NewMenus(menus menus.API) *Menus {
	return &Menus{menus: menus}
}

// List lists all menus.
func (h *Menus) List(c *mizu.Ctx) error {
	list, err := h.menus.ListMenus(c.Context())
	if err != nil {
		return InternalError(c, "failed to list menus")
	}

	return OK(c, list)
}

// Create creates a new menu.
func (h *Menus) Create(c *mizu.Ctx) error {
	var in menus.CreateMenuIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	menu, err := h.menus.CreateMenu(c.Context(), &in)
	if err != nil {
		if err == menus.ErrMissingName {
			return BadRequest(c, err.Error())
		}
		return InternalError(c, "failed to create menu")
	}

	return Created(c, menu)
}

// Get retrieves a menu by ID with items.
func (h *Menus) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	menu, err := h.menus.GetMenu(c.Context(), id)
	if err != nil {
		if err == menus.ErrMenuNotFound {
			return NotFound(c, "menu not found")
		}
		return InternalError(c, "failed to get menu")
	}

	return OK(c, menu)
}

// GetByLocation retrieves a menu by location.
func (h *Menus) GetByLocation(c *mizu.Ctx) error {
	location := c.Param("location")
	menu, err := h.menus.GetMenuByLocation(c.Context(), location)
	if err != nil {
		if err == menus.ErrMenuNotFound {
			return NotFound(c, "menu not found")
		}
		return InternalError(c, "failed to get menu")
	}

	return OK(c, menu)
}

// Update updates a menu.
func (h *Menus) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in menus.UpdateMenuIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	menu, err := h.menus.UpdateMenu(c.Context(), id, &in)
	if err != nil {
		if err == menus.ErrMenuNotFound {
			return NotFound(c, "menu not found")
		}
		return InternalError(c, "failed to update menu")
	}

	return OK(c, menu)
}

// Delete deletes a menu.
func (h *Menus) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.menus.DeleteMenu(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete menu")
	}

	return OK(c, map[string]string{"message": "menu deleted"})
}

// CreateItem creates a new menu item.
func (h *Menus) CreateItem(c *mizu.Ctx) error {
	menuID := c.Param("id")

	var in menus.CreateItemIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	item, err := h.menus.CreateItem(c.Context(), menuID, &in)
	if err != nil {
		if err == menus.ErrMissingTitle {
			return BadRequest(c, err.Error())
		}
		return InternalError(c, "failed to create menu item")
	}

	return Created(c, item)
}

// UpdateItem updates a menu item.
func (h *Menus) UpdateItem(c *mizu.Ctx) error {
	itemID := c.Param("itemID")

	var in menus.UpdateItemIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	item, err := h.menus.UpdateItem(c.Context(), itemID, &in)
	if err != nil {
		if err == menus.ErrItemNotFound {
			return NotFound(c, "menu item not found")
		}
		return InternalError(c, "failed to update menu item")
	}

	return OK(c, item)
}

// DeleteItem deletes a menu item.
func (h *Menus) DeleteItem(c *mizu.Ctx) error {
	itemID := c.Param("itemID")

	if err := h.menus.DeleteItem(c.Context(), itemID); err != nil {
		return InternalError(c, "failed to delete menu item")
	}

	return OK(c, map[string]string{"message": "menu item deleted"})
}

// ReorderItems reorders menu items.
func (h *Menus) ReorderItems(c *mizu.Ctx) error {
	menuID := c.Param("id")

	var itemIDs []string
	if err := c.BindJSON(&itemIDs, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.menus.ReorderItems(c.Context(), menuID, itemIDs); err != nil {
		return InternalError(c, "failed to reorder items")
	}

	return OK(c, map[string]string{"message": "items reordered"})
}
