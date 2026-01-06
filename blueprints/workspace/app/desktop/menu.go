package main

import (
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// createMenu creates the native application menu
func (a *DesktopApp) createMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	// File menu
	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("New Page", keys.CmdOrCtrl("n"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:newPage")
	})
	fileMenu.AddText("New Database", keys.CmdOrCtrl("d"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:newDatabase")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Import...", keys.CmdOrCtrl("i"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:import")
	})
	fileMenu.AddText("Export Page...", keys.Combo("e", keys.CmdOrCtrlKey, keys.ShiftKey), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:export")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Close Window", keys.CmdOrCtrl("w"), func(cd *menu.CallbackData) {
		runtime.Quit(a.ctx)
	})

	// Edit menu
	editMenu := appMenu.AddSubmenu("Edit")
	editMenu.AddText("Undo", keys.CmdOrCtrl("z"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:undo")
	})
	editMenu.AddText("Redo", keys.Combo("z", keys.CmdOrCtrlKey, keys.ShiftKey), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:redo")
	})
	editMenu.AddSeparator()
	editMenu.AddText("Cut", keys.CmdOrCtrl("x"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:cut")
	})
	editMenu.AddText("Copy", keys.CmdOrCtrl("c"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:copy")
	})
	editMenu.AddText("Paste", keys.CmdOrCtrl("v"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:paste")
	})
	editMenu.AddText("Select All", keys.CmdOrCtrl("a"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:selectAll")
	})
	editMenu.AddSeparator()
	editMenu.AddText("Find...", keys.CmdOrCtrl("f"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:find")
	})
	editMenu.AddText("Find and Replace...", keys.Combo("h", keys.CmdOrCtrlKey, keys.ShiftKey), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:findReplace")
	})

	// View menu
	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Toggle Sidebar", keys.CmdOrCtrl("\\"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:toggleSidebar")
	})
	viewMenu.AddSeparator()
	viewMenu.AddText("Zoom In", keys.CmdOrCtrl("="), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:zoomIn")
	})
	viewMenu.AddText("Zoom Out", keys.CmdOrCtrl("-"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:zoomOut")
	})
	viewMenu.AddText("Reset Zoom", keys.CmdOrCtrl("0"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:zoomReset")
	})
	viewMenu.AddSeparator()
	viewMenu.AddText("Toggle Full Screen", keys.Key("f11"), func(cd *menu.CallbackData) {
		runtime.WindowFullscreen(a.ctx)
	})

	// Go menu
	goMenu := appMenu.AddSubmenu("Go")
	goMenu.AddText("Quick Find", keys.CmdOrCtrl("k"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:quickFind")
	})
	goMenu.AddSeparator()
	goMenu.AddText("Back", keys.CmdOrCtrl("["), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:back")
	})
	goMenu.AddText("Forward", keys.CmdOrCtrl("]"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:forward")
	})

	// Window menu
	windowMenu := appMenu.AddSubmenu("Window")
	windowMenu.AddText("Minimize", keys.CmdOrCtrl("m"), func(cd *menu.CallbackData) {
		runtime.WindowMinimise(a.ctx)
	})
	windowMenu.AddText("Zoom", nil, func(cd *menu.CallbackData) {
		runtime.WindowToggleMaximise(a.ctx)
	})

	// Help menu
	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("Documentation", nil, func(cd *menu.CallbackData) {
		runtime.BrowserOpenURL(a.ctx, "https://mizu.dev/docs/blueprints/workspace")
	})
	helpMenu.AddText("Keyboard Shortcuts", keys.CmdOrCtrl("/"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:shortcuts")
	})
	helpMenu.AddSeparator()
	helpMenu.AddText("Report Issue...", nil, func(cd *menu.CallbackData) {
		runtime.BrowserOpenURL(a.ctx, "https://github.com/go-mizu/mizu/issues")
	})

	return appMenu
}
