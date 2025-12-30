package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/cms/feature/menus"
)

// Menu tests

func TestMenusStore_CreateMenu(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	menu := &menus.Menu{
		ID:        "menu-001",
		Name:      "Main Navigation",
		Slug:      "main-navigation",
		Location:  "header",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}

	err := store.CreateMenu(ctx, menu)
	assertNoError(t, err)

	got, err := store.GetMenu(ctx, menu.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, menu.ID)
	assertEqual(t, "Name", got.Name, menu.Name)
	assertEqual(t, "Slug", got.Slug, menu.Slug)
	assertEqual(t, "Location", got.Location, menu.Location)
}

func TestMenusStore_CreateMenu_WithLocation(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	menu := &menus.Menu{
		ID:        "menu-loc",
		Name:      "Footer Menu",
		Slug:      "footer-menu",
		Location:  "footer",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}

	err := store.CreateMenu(ctx, menu)
	assertNoError(t, err)

	got, _ := store.GetMenu(ctx, menu.ID)
	assertEqual(t, "Location", got.Location, "footer")
}

func TestMenusStore_CreateMenu_DuplicateSlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	menu1 := &menus.Menu{
		ID:        "menu-dup-1",
		Name:      "Menu 1",
		Slug:      "duplicate-slug",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu1))

	menu2 := &menus.Menu{
		ID:        "menu-dup-2",
		Name:      "Menu 2",
		Slug:      "duplicate-slug",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	err := store.CreateMenu(ctx, menu2)
	assertError(t, err)
}

func TestMenusStore_GetMenu(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	menu := &menus.Menu{
		ID:        "menu-get",
		Name:      "Get Menu",
		Slug:      "get-menu",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	got, err := store.GetMenu(ctx, menu.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, menu.ID)
}

func TestMenusStore_GetMenu_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	got, err := store.GetMenu(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent menu")
	}
}

func TestMenusStore_GetMenuBySlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	menu := &menus.Menu{
		ID:        "menu-slug",
		Name:      "Slug Menu",
		Slug:      "my-unique-menu-slug",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	got, err := store.GetMenuBySlug(ctx, "my-unique-menu-slug")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, menu.ID)
}

func TestMenusStore_GetMenuByLocation(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	menu := &menus.Menu{
		ID:        "menu-location",
		Name:      "Location Menu",
		Slug:      "location-menu",
		Location:  "sidebar",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	got, err := store.GetMenuByLocation(ctx, "sidebar")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, menu.ID)
}

func TestMenusStore_ListMenus(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		menu := &menus.Menu{
			ID:        "menu-list-" + string(rune('a'+i)),
			Name:      "Menu " + string(rune('A'+i)),
			Slug:      "menu-" + string(rune('a'+i)),
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.CreateMenu(ctx, menu))
	}

	list, err := store.ListMenus(ctx)
	assertNoError(t, err)
	assertLen(t, list, 3)
}

func TestMenusStore_UpdateMenu(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	menu := &menus.Menu{
		ID:        "menu-update",
		Name:      "Original Name",
		Slug:      "update-menu",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	err := store.UpdateMenu(ctx, menu.ID, &menus.UpdateMenuIn{
		Name:     ptr("Updated Name"),
		Location: ptr("new-location"),
	})
	assertNoError(t, err)

	got, _ := store.GetMenu(ctx, menu.ID)
	assertEqual(t, "Name", got.Name, "Updated Name")
	assertEqual(t, "Location", got.Location, "new-location")
}

func TestMenusStore_UpdateMenu_PartialFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	menu := &menus.Menu{
		ID:        "menu-partial",
		Name:      "Original",
		Slug:      "partial-menu",
		Location:  "header",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	// Only update name
	err := store.UpdateMenu(ctx, menu.ID, &menus.UpdateMenuIn{
		Name: ptr("New Name"),
	})
	assertNoError(t, err)

	got, _ := store.GetMenu(ctx, menu.ID)
	assertEqual(t, "Name", got.Name, "New Name")
	assertEqual(t, "Location", got.Location, "header") // Unchanged
}

func TestMenusStore_DeleteMenu(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	menu := &menus.Menu{
		ID:        "menu-delete",
		Name:      "Delete Menu",
		Slug:      "delete-menu",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	err := store.DeleteMenu(ctx, menu.ID)
	assertNoError(t, err)

	got, _ := store.GetMenu(ctx, menu.ID)
	if got != nil {
		t.Error("expected menu to be deleted")
	}
}

// Menu Item tests

func TestMenusStore_CreateItem(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu first
	menu := &menus.Menu{
		ID:        "menu-for-item",
		Name:      "Menu For Items",
		Slug:      "menu-for-items",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	item := &menus.MenuItem{
		ID:        "item-001",
		MenuID:    menu.ID,
		Title:     "Home",
		URL:       "/",
		Target:    "_self",
		LinkType:  "custom",
		CSSClass:  "nav-item",
		SortOrder: 1,
		CreatedAt: testTime,
	}

	err := store.CreateItem(ctx, item)
	assertNoError(t, err)

	got, err := store.GetItem(ctx, item.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, item.ID)
	assertEqual(t, "MenuID", got.MenuID, item.MenuID)
	assertEqual(t, "Title", got.Title, item.Title)
	assertEqual(t, "URL", got.URL, item.URL)
	assertEqual(t, "Target", got.Target, item.Target)
}

func TestMenusStore_CreateItem_WithParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu
	menu := &menus.Menu{
		ID:        "menu-parent-item",
		Name:      "Menu",
		Slug:      "menu-parent-item",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	// Create parent item
	parent := &menus.MenuItem{
		ID:        "item-parent",
		MenuID:    menu.ID,
		Title:     "Parent",
		Target:    "_self",
		SortOrder: 1,
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateItem(ctx, parent))

	// Create child item
	child := &menus.MenuItem{
		ID:        "item-child",
		MenuID:    menu.ID,
		ParentID:  parent.ID,
		Title:     "Child",
		Target:    "_self",
		SortOrder: 1,
		CreatedAt: testTime,
	}
	err := store.CreateItem(ctx, child)
	assertNoError(t, err)

	got, _ := store.GetItem(ctx, child.ID)
	assertEqual(t, "ParentID", got.ParentID, parent.ID)
}

func TestMenusStore_CreateItem_WithLink(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu
	menu := &menus.Menu{
		ID:        "menu-link-item",
		Name:      "Menu",
		Slug:      "menu-link-item",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	item := &menus.MenuItem{
		ID:        "item-link",
		MenuID:    menu.ID,
		Title:     "Blog Post",
		Target:    "_self",
		LinkType:  "post",
		LinkID:    "post-123",
		SortOrder: 1,
		CreatedAt: testTime,
	}

	err := store.CreateItem(ctx, item)
	assertNoError(t, err)

	got, _ := store.GetItem(ctx, item.ID)
	assertEqual(t, "LinkType", got.LinkType, "post")
	assertEqual(t, "LinkID", got.LinkID, "post-123")
}

func TestMenusStore_GetItem(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu
	menu := &menus.Menu{
		ID:        "menu-get-item",
		Name:      "Menu",
		Slug:      "menu-get-item",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	item := &menus.MenuItem{
		ID:        "item-get",
		MenuID:    menu.ID,
		Title:     "Get Item",
		Target:    "_self",
		SortOrder: 1,
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateItem(ctx, item))

	got, err := store.GetItem(ctx, item.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, item.ID)
}

func TestMenusStore_GetItem_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	got, err := store.GetItem(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent item")
	}
}

func TestMenusStore_GetItemsByMenu(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu
	menu := &menus.Menu{
		ID:        "menu-items-list",
		Name:      "Menu",
		Slug:      "menu-items-list",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	// Create items
	for i := 0; i < 3; i++ {
		item := &menus.MenuItem{
			ID:        "item-list-" + string(rune('a'+i)),
			MenuID:    menu.ID,
			Title:     "Item " + string(rune('A'+i)),
			Target:    "_self",
			SortOrder: i,
			CreatedAt: testTime,
		}
		assertNoError(t, store.CreateItem(ctx, item))
	}

	list, err := store.GetItemsByMenu(ctx, menu.ID)
	assertNoError(t, err)
	assertLen(t, list, 3)
}

func TestMenusStore_GetItemsByMenu_Ordering(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu
	menu := &menus.Menu{
		ID:        "menu-order-items",
		Name:      "Menu",
		Slug:      "menu-order-items",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	// Create items with different sort orders
	itemsData := []struct {
		title     string
		sortOrder int
	}{
		{"Third", 3},
		{"First", 1},
		{"Second", 2},
	}
	for i, id := range itemsData {
		item := &menus.MenuItem{
			ID:        "item-order-" + string(rune('a'+i)),
			MenuID:    menu.ID,
			Title:     id.title,
			Target:    "_self",
			SortOrder: id.sortOrder,
			CreatedAt: testTime,
		}
		assertNoError(t, store.CreateItem(ctx, item))
	}

	list, _ := store.GetItemsByMenu(ctx, menu.ID)
	assertEqual(t, "Title[0]", list[0].Title, "First")
	assertEqual(t, "Title[1]", list[1].Title, "Second")
	assertEqual(t, "Title[2]", list[2].Title, "Third")
}

func TestMenusStore_UpdateItem(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu
	menu := &menus.Menu{
		ID:        "menu-update-item",
		Name:      "Menu",
		Slug:      "menu-update-item",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	item := &menus.MenuItem{
		ID:        "item-update",
		MenuID:    menu.ID,
		Title:     "Original Title",
		Target:    "_self",
		SortOrder: 1,
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateItem(ctx, item))

	err := store.UpdateItem(ctx, item.ID, &menus.UpdateItemIn{
		Title:  ptr("Updated Title"),
		URL:    ptr("/new-url"),
		Target: ptr("_blank"),
	})
	assertNoError(t, err)

	got, _ := store.GetItem(ctx, item.ID)
	assertEqual(t, "Title", got.Title, "Updated Title")
	assertEqual(t, "URL", got.URL, "/new-url")
	assertEqual(t, "Target", got.Target, "_blank")
}

func TestMenusStore_UpdateItem_ChangeParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu
	menu := &menus.Menu{
		ID:        "menu-move-item",
		Name:      "Menu",
		Slug:      "menu-move-item",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	// Create two parents
	parent1 := &menus.MenuItem{
		ID:        "item-parent-1",
		MenuID:    menu.ID,
		Title:     "Parent 1",
		Target:    "_self",
		SortOrder: 1,
		CreatedAt: testTime,
	}
	parent2 := &menus.MenuItem{
		ID:        "item-parent-2",
		MenuID:    menu.ID,
		Title:     "Parent 2",
		Target:    "_self",
		SortOrder: 2,
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateItem(ctx, parent1))
	assertNoError(t, store.CreateItem(ctx, parent2))

	// Create child under parent1
	child := &menus.MenuItem{
		ID:        "item-movable",
		MenuID:    menu.ID,
		ParentID:  parent1.ID,
		Title:     "Movable Child",
		Target:    "_self",
		SortOrder: 1,
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateItem(ctx, child))

	// Move to parent2
	err := store.UpdateItem(ctx, child.ID, &menus.UpdateItemIn{
		ParentID: ptr(parent2.ID),
	})
	assertNoError(t, err)

	got, _ := store.GetItem(ctx, child.ID)
	assertEqual(t, "ParentID", got.ParentID, parent2.ID)
}

func TestMenusStore_DeleteItem(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu
	menu := &menus.Menu{
		ID:        "menu-delete-item",
		Name:      "Menu",
		Slug:      "menu-delete-item",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	item := &menus.MenuItem{
		ID:        "item-delete",
		MenuID:    menu.ID,
		Title:     "Delete Item",
		Target:    "_self",
		SortOrder: 1,
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateItem(ctx, item))

	err := store.DeleteItem(ctx, item.ID)
	assertNoError(t, err)

	got, _ := store.GetItem(ctx, item.ID)
	if got != nil {
		t.Error("expected item to be deleted")
	}
}

func TestMenusStore_DeleteItemsByMenu(t *testing.T) {
	db := setupTestDB(t)
	store := NewMenusStore(db)
	ctx := context.Background()

	// Create menu
	menu := &menus.Menu{
		ID:        "menu-delete-all",
		Name:      "Menu",
		Slug:      "menu-delete-all",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.CreateMenu(ctx, menu))

	// Create multiple items
	for i := 0; i < 3; i++ {
		item := &menus.MenuItem{
			ID:        "item-delall-" + string(rune('a'+i)),
			MenuID:    menu.ID,
			Title:     "Item " + string(rune('A'+i)),
			Target:    "_self",
			SortOrder: i,
			CreatedAt: testTime,
		}
		assertNoError(t, store.CreateItem(ctx, item))
	}

	// Delete all items for menu
	err := store.DeleteItemsByMenu(ctx, menu.ID)
	assertNoError(t, err)

	list, _ := store.GetItemsByMenu(ctx, menu.ID)
	assertLen(t, list, 0)
}
