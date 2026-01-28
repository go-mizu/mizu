package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// WidgetStore handles widget settings and cheat sheets.
type WidgetStore struct {
	db *sql.DB
}

// GetWidgetSettings retrieves widget settings for a user.
func (s *WidgetStore) GetWidgetSettings(ctx context.Context, userID string) ([]*store.WidgetSetting, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, widget_type, enabled, position, created_at
		FROM widget_settings
		WHERE user_id = ?
		ORDER BY position ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*store.WidgetSetting
	for rows.Next() {
		var setting store.WidgetSetting
		var enabled int

		if err := rows.Scan(&setting.ID, &setting.UserID, &setting.WidgetType,
			&enabled, &setting.Position, &setting.CreatedAt); err != nil {
			return nil, err
		}

		setting.Enabled = enabled == 1
		settings = append(settings, &setting)
	}

	// If no settings exist, return defaults
	if len(settings) == 0 {
		settings = s.defaultWidgetSettings(userID)
	}

	return settings, nil
}

// defaultWidgetSettings returns default widget settings.
func (s *WidgetStore) defaultWidgetSettings(userID string) []*store.WidgetSetting {
	allTypes := types.AllWidgetTypes()
	settings := make([]*store.WidgetSetting, len(allTypes))
	for i, wt := range allTypes {
		settings[i] = &store.WidgetSetting{
			UserID:     userID,
			WidgetType: wt,
			Enabled:    true,
			Position:   i,
			CreatedAt:  time.Now(),
		}
	}
	return settings
}

// SetWidgetSetting updates a widget setting.
func (s *WidgetStore) SetWidgetSetting(ctx context.Context, setting *store.WidgetSetting) error {
	setting.CreatedAt = time.Now()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO widget_settings (user_id, widget_type, enabled, position, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, widget_type) DO UPDATE SET
			enabled = excluded.enabled,
			position = excluded.position
	`, setting.UserID, setting.WidgetType, boolToInt(setting.Enabled), setting.Position, setting.CreatedAt)

	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	setting.ID = id
	return nil
}

// GetCheatSheet retrieves a cheat sheet by language.
func (s *WidgetStore) GetCheatSheet(ctx context.Context, language string) (*store.CheatSheet, error) {
	var sheet store.CheatSheet
	var contentJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT language, title, content
		FROM cheat_sheets WHERE language = ?
	`, language).Scan(&sheet.Language, &sheet.Title, &contentJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(contentJSON), &sheet.Sections); err != nil {
		return nil, err
	}

	return &sheet, nil
}

// SaveCheatSheet saves a cheat sheet.
func (s *WidgetStore) SaveCheatSheet(ctx context.Context, sheet *store.CheatSheet) error {
	contentJSON, err := json.Marshal(sheet.Sections)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cheat_sheets (language, title, content, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(language) DO UPDATE SET
			title = excluded.title,
			content = excluded.content,
			updated_at = datetime('now')
	`, sheet.Language, sheet.Title, string(contentJSON))

	return err
}

// ListCheatSheets returns all cheat sheets.
func (s *WidgetStore) ListCheatSheets(ctx context.Context) ([]*store.CheatSheet, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT language, title, content
		FROM cheat_sheets
		ORDER BY language ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sheets []*store.CheatSheet
	for rows.Next() {
		var sheet store.CheatSheet
		var contentJSON string

		if err := rows.Scan(&sheet.Language, &sheet.Title, &contentJSON); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(contentJSON), &sheet.Sections); err != nil {
			return nil, err
		}

		sheets = append(sheets, &sheet)
	}

	return sheets, nil
}

// SeedCheatSheets inserts default cheat sheets.
func (s *WidgetStore) SeedCheatSheets(ctx context.Context) error {
	sheets := getDefaultCheatSheets()
	for _, sheet := range sheets {
		if err := s.SaveCheatSheet(ctx, sheet); err != nil {
			return err
		}
	}
	return nil
}

// GetRelatedSearches retrieves cached related searches.
func (s *WidgetStore) GetRelatedSearches(ctx context.Context, queryHash string) ([]string, error) {
	var relatedJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT related FROM related_searches
		WHERE query_hash = ? AND (expires_at IS NULL OR expires_at > datetime('now'))
	`, queryHash).Scan(&relatedJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var related []string
	if err := json.Unmarshal([]byte(relatedJSON), &related); err != nil {
		return nil, err
	}

	return related, nil
}

// SaveRelatedSearches caches related searches.
func (s *WidgetStore) SaveRelatedSearches(ctx context.Context, queryHash, query string, related []string) error {
	relatedJSON, err := json.Marshal(related)
	if err != nil {
		return err
	}

	// Cache for 1 hour
	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO related_searches (query_hash, query, related, created_at, expires_at)
		VALUES (?, ?, ?, datetime('now'), ?)
		ON CONFLICT(query_hash) DO UPDATE SET
			related = excluded.related,
			expires_at = excluded.expires_at
	`, queryHash, query, string(relatedJSON), expiresAt)

	return err
}

// getDefaultCheatSheets returns default programming cheat sheets.
func getDefaultCheatSheets() []*store.CheatSheet {
	return []*store.CheatSheet{
		{
			Language: "go",
			Title:    "Go Cheat Sheet",
			Sections: []store.CheatSection{
				{
					Name: "Variables",
					Items: []store.CheatItem{
						{Code: "var x int", Description: "Declare variable with type"},
						{Code: "x := 10", Description: "Short declaration"},
						{Code: "const Pi = 3.14", Description: "Constant declaration"},
					},
				},
				{
					Name: "Control Flow",
					Items: []store.CheatItem{
						{Code: "if x > 0 { }", Description: "If statement"},
						{Code: "for i := 0; i < 10; i++ { }", Description: "For loop"},
						{Code: "for _, v := range slice { }", Description: "Range loop"},
						{Code: "switch x { case 1: ... }", Description: "Switch statement"},
					},
				},
				{
					Name: "Functions",
					Items: []store.CheatItem{
						{Code: "func add(a, b int) int { return a + b }", Description: "Function with return"},
						{Code: "func (r *Rect) Area() int { }", Description: "Method with receiver"},
						{Code: "defer f.Close()", Description: "Defer execution"},
					},
				},
				{
					Name: "Slices & Maps",
					Items: []store.CheatItem{
						{Code: "s := make([]int, 0)", Description: "Create empty slice"},
						{Code: "s = append(s, 1)", Description: "Append to slice"},
						{Code: "m := make(map[string]int)", Description: "Create map"},
						{Code: "m[\"key\"] = 1", Description: "Set map value"},
					},
				},
				{
					Name: "Concurrency",
					Items: []store.CheatItem{
						{Code: "go func() { }()", Description: "Start goroutine"},
						{Code: "ch := make(chan int)", Description: "Create channel"},
						{Code: "ch <- 1", Description: "Send to channel"},
						{Code: "x := <-ch", Description: "Receive from channel"},
					},
				},
			},
		},
		{
			Language: "python",
			Title:    "Python Cheat Sheet",
			Sections: []store.CheatSection{
				{
					Name: "Variables & Types",
					Items: []store.CheatItem{
						{Code: "x = 10", Description: "Variable assignment"},
						{Code: "x: int = 10", Description: "Type hint"},
						{Code: "s = \"hello\"", Description: "String"},
						{Code: "l = [1, 2, 3]", Description: "List"},
						{Code: "d = {\"a\": 1}", Description: "Dictionary"},
					},
				},
				{
					Name: "Control Flow",
					Items: []store.CheatItem{
						{Code: "if x > 0:", Description: "If statement"},
						{Code: "for i in range(10):", Description: "For loop"},
						{Code: "for k, v in d.items():", Description: "Dict iteration"},
						{Code: "[x*2 for x in range(5)]", Description: "List comprehension"},
					},
				},
				{
					Name: "Functions",
					Items: []store.CheatItem{
						{Code: "def add(a, b): return a + b", Description: "Function definition"},
						{Code: "lambda x: x * 2", Description: "Lambda function"},
						{Code: "def f(*args, **kwargs):", Description: "Variable arguments"},
					},
				},
				{
					Name: "Classes",
					Items: []store.CheatItem{
						{Code: "class Dog:", Description: "Class definition"},
						{Code: "def __init__(self):", Description: "Constructor"},
						{Code: "@property", Description: "Property decorator"},
						{Code: "@staticmethod", Description: "Static method"},
					},
				},
			},
		},
		{
			Language: "javascript",
			Title:    "JavaScript Cheat Sheet",
			Sections: []store.CheatSection{
				{
					Name: "Variables",
					Items: []store.CheatItem{
						{Code: "const x = 10", Description: "Constant (immutable binding)"},
						{Code: "let y = 20", Description: "Block-scoped variable"},
						{Code: "var z = 30", Description: "Function-scoped variable"},
					},
				},
				{
					Name: "Arrays",
					Items: []store.CheatItem{
						{Code: "arr.map(x => x * 2)", Description: "Transform elements"},
						{Code: "arr.filter(x => x > 0)", Description: "Filter elements"},
						{Code: "arr.reduce((a, b) => a + b)", Description: "Reduce to single value"},
						{Code: "[...arr1, ...arr2]", Description: "Spread operator"},
					},
				},
				{
					Name: "Objects",
					Items: []store.CheatItem{
						{Code: "const { a, b } = obj", Description: "Destructuring"},
						{Code: "{ ...obj, c: 3 }", Description: "Spread with new property"},
						{Code: "Object.keys(obj)", Description: "Get keys"},
						{Code: "Object.entries(obj)", Description: "Get key-value pairs"},
					},
				},
				{
					Name: "Async",
					Items: []store.CheatItem{
						{Code: "async function f() { }", Description: "Async function"},
						{Code: "await promise", Description: "Await promise"},
						{Code: "Promise.all([p1, p2])", Description: "Wait for all promises"},
						{Code: "fetch(url).then(r => r.json())", Description: "Fetch with promise"},
					},
				},
			},
		},
		{
			Language: "typescript",
			Title:    "TypeScript Cheat Sheet",
			Sections: []store.CheatSection{
				{
					Name: "Types",
					Items: []store.CheatItem{
						{Code: "let x: number = 10", Description: "Number type"},
						{Code: "let s: string = \"hi\"", Description: "String type"},
						{Code: "let arr: number[] = [1, 2]", Description: "Array type"},
						{Code: "let tuple: [string, number]", Description: "Tuple type"},
					},
				},
				{
					Name: "Interfaces",
					Items: []store.CheatItem{
						{Code: "interface User { name: string }", Description: "Interface definition"},
						{Code: "interface Props extends Base { }", Description: "Extend interface"},
						{Code: "type ID = string | number", Description: "Union type"},
						{Code: "type Partial<T>", Description: "Utility type"},
					},
				},
				{
					Name: "Generics",
					Items: []store.CheatItem{
						{Code: "function id<T>(x: T): T", Description: "Generic function"},
						{Code: "class Box<T> { value: T }", Description: "Generic class"},
						{Code: "<T extends Base>", Description: "Generic constraint"},
					},
				},
			},
		},
		{
			Language: "rust",
			Title:    "Rust Cheat Sheet",
			Sections: []store.CheatSection{
				{
					Name: "Variables",
					Items: []store.CheatItem{
						{Code: "let x = 5;", Description: "Immutable variable"},
						{Code: "let mut x = 5;", Description: "Mutable variable"},
						{Code: "const MAX: u32 = 100;", Description: "Constant"},
					},
				},
				{
					Name: "Ownership",
					Items: []store.CheatItem{
						{Code: "let s2 = s1.clone();", Description: "Deep copy"},
						{Code: "fn f(s: &String)", Description: "Borrow reference"},
						{Code: "fn f(s: &mut String)", Description: "Mutable borrow"},
					},
				},
				{
					Name: "Pattern Matching",
					Items: []store.CheatItem{
						{Code: "match x { 1 => ... }", Description: "Match expression"},
						{Code: "if let Some(v) = opt", Description: "If let"},
						{Code: "while let Some(v) = iter.next()", Description: "While let"},
					},
				},
				{
					Name: "Error Handling",
					Items: []store.CheatItem{
						{Code: "Result<T, E>", Description: "Result type"},
						{Code: "Option<T>", Description: "Option type"},
						{Code: "value?", Description: "Propagate error"},
						{Code: ".unwrap()", Description: "Panic on None/Err"},
					},
				},
			},
		},
	}
}
