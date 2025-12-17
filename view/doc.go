// Package view provides a standardized template system for Mizu.
//
// The view package builds on Go's html/template and adds:
//   - Clear conceptual model (Page, Layout, Component, Partial)
//   - Convention-based directory structure
//   - Layout templates with named slots
//   - Slot and stack mechanisms for composition
//   - Reusable components with structured input
//   - Development-time reload and error display
//   - Production-safe rendering with embed and caching
//
// # Directory Structure
//
// The package expects a conventional directory structure:
//
//	views/
//	├── layouts/      # Layout templates
//	├── pages/        # Page templates
//	├── components/   # Reusable components
//	└── partials/     # Template fragments
//
// # Basic Usage
//
//	// Create engine
//	engine := view.New(view.Config{
//	    Dir:         "views",
//	    Development: true,
//	})
//
//	// Add handler to Mizu app
//	app := mizu.New()
//	app.Use(engine.Handler())
//
//	// Render in handlers
//	func handler(c *mizu.Ctx) error {
//	    return view.Render(c, "home", view.Data{
//	        "Title": "Welcome",
//	        "User":  user,
//	    })
//	}
//
// # Layouts and Slots
//
// Layouts define the page structure with named slots:
//
//	<!-- layouts/default.html -->
//	<!DOCTYPE html>
//	<html>
//	<head>
//	    <title>{{slot "title" "Default Title"}}</title>
//	</head>
//	<body>
//	    {{slot "content"}}
//	</body>
//	</html>
//
// Pages fill slots using define blocks:
//
//	<!-- pages/home.html -->
//	{{define "title"}}Home{{end}}
//	{{define "content"}}
//	    <h1>Welcome</h1>
//	{{end}}
//
// # Components
//
// Components are reusable template units with isolated data:
//
//	<!-- components/button.html -->
//	<button class="btn btn-{{.Variant}}">{{.Label}}</button>
//
//	<!-- Usage -->
//	{{component "button" (dict "Variant" "primary" "Label" "Submit")}}
//
// # Production Mode
//
// For production, embed templates and disable development mode:
//
//	//go:embed views
//	var viewsFS embed.FS
//
//	engine := view.New(view.Config{
//	    FS: viewsFS,
//	})
package view
