package seed

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/blueprints/bi/pkg/password"
	"github.com/go-mizu/blueprints/bi/store"
)

// Seeder seeds the database with sample data.
type Seeder struct {
	store store.Store
}

// New creates a new Seeder.
func New(s store.Store) *Seeder {
	return &Seeder{store: s}
}

// Run seeds all sample data.
func (s *Seeder) Run(ctx context.Context) error {
	slog.Info("Starting database seed")

	// Create admin user
	if err := s.seedUsers(ctx); err != nil {
		return fmt.Errorf("seed users: %w", err)
	}

	// Create sample data source
	dsID, err := s.seedDataSource(ctx)
	if err != nil {
		return fmt.Errorf("seed data source: %w", err)
	}

	// Create collections
	collIDs, err := s.seedCollections(ctx)
	if err != nil {
		return fmt.Errorf("seed collections: %w", err)
	}

	// Create questions
	questionIDs, err := s.seedQuestions(ctx, dsID, collIDs)
	if err != nil {
		return fmt.Errorf("seed questions: %w", err)
	}

	// Create dashboards
	if err := s.seedDashboards(ctx, collIDs, questionIDs); err != nil {
		return fmt.Errorf("seed dashboards: %w", err)
	}

	slog.Info("Database seed complete")
	return nil
}

func (s *Seeder) seedUsers(ctx context.Context) error {
	// Hash the password "admin" using Argon2
	passwordHash, err := password.Hash("admin", nil)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := &store.User{
		Email:        "admin@example.com",
		Name:         "Admin User",
		PasswordHash: passwordHash,
		Role:         "admin",
	}

	return s.store.Users().Create(ctx, user)
}

func (s *Seeder) seedDataSource(ctx context.Context) (string, error) {
	// Get data directory from store
	sqliteStore, ok := s.store.(*sqliteStoreWrapper)
	var dataDir string
	if ok {
		dataDir = sqliteStore.dataDir
	} else {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprint", "bi")
	}

	// Create sample database
	sampleDBPath := filepath.Join(dataDir, "sample_data.db")
	if err := createSampleDatabase(sampleDBPath); err != nil {
		return "", err
	}

	ds := &store.DataSource{
		Name:     "Sample Data",
		Engine:   "sqlite",
		Database: sampleDBPath,
	}

	if err := s.store.DataSources().Create(ctx, ds); err != nil {
		return "", err
	}

	// Sync tables
	db, err := sql.Open("sqlite3", sampleDBPath)
	if err != nil {
		return ds.ID, nil
	}
	defer db.Close()

	// Get table info
	tables := []string{"orders", "products", "customers", "analytics"}
	for _, tableName := range tables {
		table := &store.Table{
			DataSourceID: ds.ID,
			Name:         tableName,
			DisplayName:  tableName,
		}
		s.store.Tables().Create(ctx, table)

		// Get columns
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
		if err != nil {
			continue
		}

		pos := 0
		for rows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dflt interface{}
			rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk)

			col := &store.Column{
				TableID:     table.ID,
				Name:        name,
				DisplayName: name,
				Type:        mapType(colType),
				Position:    pos,
			}
			s.store.Tables().CreateColumn(ctx, col)
			pos++
		}
		rows.Close()
	}

	return ds.ID, nil
}

func (s *Seeder) seedCollections(ctx context.Context) (map[string]string, error) {
	collections := []struct {
		name  string
		color string
	}{
		{"Sales", "#509EE3"},
		{"Marketing", "#88BF4D"},
		{"Operations", "#A989C5"},
	}

	ids := make(map[string]string)
	for _, c := range collections {
		coll := &store.Collection{
			Name:  c.name,
			Color: c.color,
		}
		if err := s.store.Collections().Create(ctx, coll); err != nil {
			return nil, err
		}
		ids[c.name] = coll.ID
	}

	return ids, nil
}

func (s *Seeder) seedQuestions(ctx context.Context, dsID string, collIDs map[string]string) ([]string, error) {
	questions := []struct {
		name       string
		collection string
		queryType  string
		query      map[string]interface{}
		viz        map[string]interface{}
	}{
		{
			name:       "Revenue by Month",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-%m', order_date) as month,
					SUM(total) as revenue
				FROM orders
				GROUP BY month
				ORDER BY month`,
			},
			viz: map[string]interface{}{
				"type": "line",
				"settings": map[string]interface{}{
					"x_axis": "month",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Top Products by Sales",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					p.name,
					SUM(o.quantity) as units_sold,
					SUM(o.quantity * p.price) as revenue
				FROM orders o
				JOIN products p ON o.product_id = p.id
				GROUP BY p.id
				ORDER BY revenue DESC
				LIMIT 10`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "name",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Customer Distribution",
			collection: "Marketing",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					country,
					COUNT(*) as customers
				FROM customers
				GROUP BY country
				ORDER BY customers DESC`,
			},
			viz: map[string]interface{}{
				"type": "pie",
				"settings": map[string]interface{}{
					"dimension": "country",
					"metric":    "customers",
				},
			},
		},
		{
			name:       "Recent Orders",
			collection: "Operations",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					o.id,
					c.name as customer,
					p.name as product,
					o.quantity,
					o.total,
					o.order_date
				FROM orders o
				JOIN customers c ON o.customer_id = c.id
				JOIN products p ON o.product_id = p.id
				ORDER BY o.order_date DESC
				LIMIT 100`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		{
			name:       "Total Revenue",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": "SELECT SUM(total) as total_revenue FROM orders",
			},
			viz: map[string]interface{}{
				"type": "number",
				"settings": map[string]interface{}{
					"prefix": "$",
				},
			},
		},
	}

	var ids []string
	for _, q := range questions {
		question := &store.Question{
			Name:          q.name,
			CollectionID:  collIDs[q.collection],
			DataSourceID:  dsID,
			QueryType:     q.queryType,
			Query:         q.query,
			Visualization: q.viz,
		}
		if err := s.store.Questions().Create(ctx, question); err != nil {
			return nil, err
		}
		ids = append(ids, question.ID)
	}

	return ids, nil
}

func (s *Seeder) seedDashboards(ctx context.Context, collIDs map[string]string, questionIDs []string) error {
	// Sales Overview Dashboard
	salesDash := &store.Dashboard{
		Name:         "Sales Overview",
		Description:  "Key sales metrics and trends",
		CollectionID: collIDs["Sales"],
	}
	if err := s.store.Dashboards().Create(ctx, salesDash); err != nil {
		return err
	}

	// Add cards to dashboard
	cards := []store.DashboardCard{
		{DashboardID: salesDash.ID, QuestionID: questionIDs[4], CardType: "question", Row: 0, Col: 0, Width: 4, Height: 2},  // Total Revenue
		{DashboardID: salesDash.ID, QuestionID: questionIDs[0], CardType: "question", Row: 0, Col: 4, Width: 8, Height: 4},  // Revenue by Month
		{DashboardID: salesDash.ID, QuestionID: questionIDs[1], CardType: "question", Row: 2, Col: 0, Width: 6, Height: 4},  // Top Products
		{DashboardID: salesDash.ID, QuestionID: questionIDs[3], CardType: "question", Row: 4, Col: 0, Width: 12, Height: 6}, // Recent Orders
	}
	for _, card := range cards {
		s.store.Dashboards().CreateCard(ctx, &card)
	}

	// Marketing Dashboard
	mktDash := &store.Dashboard{
		Name:         "Marketing Analytics",
		Description:  "Customer insights and marketing metrics",
		CollectionID: collIDs["Marketing"],
	}
	if err := s.store.Dashboards().Create(ctx, mktDash); err != nil {
		return err
	}

	// Add cards
	mktCards := []store.DashboardCard{
		{DashboardID: mktDash.ID, QuestionID: questionIDs[2], CardType: "question", Row: 0, Col: 0, Width: 6, Height: 4}, // Customer Distribution
	}
	for _, card := range mktCards {
		s.store.Dashboards().CreateCard(ctx, &card)
	}

	return nil
}

func createSampleDatabase(dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		price REAL NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		country TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		customer_id INTEGER NOT NULL REFERENCES customers(id),
		product_id INTEGER NOT NULL REFERENCES products(id),
		quantity INTEGER NOT NULL,
		total REAL NOT NULL,
		order_date DATE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS analytics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date DATE NOT NULL,
		page_views INTEGER NOT NULL,
		unique_visitors INTEGER NOT NULL,
		bounce_rate REAL NOT NULL,
		avg_session_duration REAL NOT NULL
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Check if data already exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if count > 0 {
		return nil // Data already seeded
	}

	// Seed products
	products := []struct {
		name     string
		category string
		price    float64
	}{
		{"Widget Pro", "Electronics", 99.99},
		{"Gadget Plus", "Electronics", 149.99},
		{"Super Tool", "Tools", 49.99},
		{"Mega Device", "Electronics", 299.99},
		{"Basic Item", "General", 19.99},
		{"Premium Package", "Services", 199.99},
		{"Standard Kit", "Tools", 79.99},
		{"Deluxe Set", "General", 129.99},
		{"Mini Gadget", "Electronics", 39.99},
		{"Pro Service", "Services", 499.99},
	}

	for _, p := range products {
		db.Exec("INSERT INTO products (name, category, price) VALUES (?, ?, ?)",
			p.name, p.category, p.price)
	}

	// Seed customers
	countries := []string{"USA", "UK", "Canada", "Germany", "France", "Australia", "Japan", "Brazil"}
	for i := 1; i <= 100; i++ {
		name := fmt.Sprintf("Customer %d", i)
		email := fmt.Sprintf("customer%d@example.com", i)
		country := countries[rand.Intn(len(countries))]
		db.Exec("INSERT INTO customers (name, email, country) VALUES (?, ?, ?)",
			name, email, country)
	}

	// Seed orders (for the past year)
	now := time.Now()
	for i := 0; i < 1000; i++ {
		customerID := rand.Intn(100) + 1
		productID := rand.Intn(10) + 1
		quantity := rand.Intn(5) + 1

		// Get product price
		var price float64
		db.QueryRow("SELECT price FROM products WHERE id = ?", productID).Scan(&price)
		total := price * float64(quantity)

		// Random date in the past year
		daysAgo := rand.Intn(365)
		orderDate := now.AddDate(0, 0, -daysAgo).Format("2006-01-02")

		db.Exec("INSERT INTO orders (customer_id, product_id, quantity, total, order_date) VALUES (?, ?, ?, ?, ?)",
			customerID, productID, quantity, total, orderDate)
	}

	// Seed analytics (for the past 30 days)
	for i := 0; i < 30; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		pageViews := rand.Intn(10000) + 5000
		uniqueVisitors := rand.Intn(5000) + 2000
		bounceRate := rand.Float64()*30 + 20
		avgSession := rand.Float64()*300 + 60

		db.Exec("INSERT INTO analytics (date, page_views, unique_visitors, bounce_rate, avg_session_duration) VALUES (?, ?, ?, ?, ?)",
			date, pageViews, uniqueVisitors, bounceRate, avgSession)
	}

	return nil
}

func mapType(sqlType string) string {
	switch sqlType {
	case "INTEGER", "INT", "REAL", "NUMERIC":
		return "number"
	case "TEXT", "VARCHAR", "CHAR":
		return "string"
	case "DATETIME", "DATE", "TIMESTAMP":
		return "datetime"
	case "BOOLEAN":
		return "boolean"
	default:
		return "string"
	}
}

// sqliteStoreWrapper is used to access the data directory from the store.
// This is a workaround since the store interface doesn't expose the data directory.
type sqliteStoreWrapper struct {
	store.Store
	dataDir string
}
