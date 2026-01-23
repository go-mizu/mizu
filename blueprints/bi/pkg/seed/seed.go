package seed

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/go-mizu/blueprints/bi/pkg/password"
	"github.com/go-mizu/blueprints/bi/store"
)

// Seeder seeds the database with sample data.
type Seeder struct {
	store   store.Store
	dataDir string
}

// New creates a new Seeder.
func New(s store.Store) *Seeder {
	return &Seeder{store: s}
}

// SetDataDir sets the data directory for the seeder.
func (s *Seeder) SetDataDir(dir string) {
	s.dataDir = dir
}

// Run seeds all sample data.
func (s *Seeder) Run(ctx context.Context) error {
	slog.Info("Starting database seed with Northwind data")

	// Create admin user
	if err := s.seedUsers(ctx); err != nil {
		return fmt.Errorf("seed users: %w", err)
	}

	// Create sample data source with Northwind database
	dsID, err := s.seedDataSource(ctx)
	if err != nil {
		return fmt.Errorf("seed data source: %w", err)
	}

	// Create collections
	collIDs, err := s.seedCollections(ctx)
	if err != nil {
		return fmt.Errorf("seed collections: %w", err)
	}

	// Create comprehensive questions
	questionIDs, err := s.seedQuestions(ctx, dsID, collIDs)
	if err != nil {
		return fmt.Errorf("seed questions: %w", err)
	}

	// Create dashboards
	if err := s.seedDashboards(ctx, collIDs, questionIDs); err != nil {
		return fmt.Errorf("seed dashboards: %w", err)
	}

	slog.Info("Database seed complete",
		"questions", len(questionIDs),
		"collections", len(collIDs),
	)
	return nil
}

func (s *Seeder) seedUsers(ctx context.Context) error {
	// Check if admin user already exists
	existing, _ := s.store.Users().GetByEmail(ctx, "admin@example.com")
	if existing != nil {
		slog.Info("Admin user already exists, skipping")
		return nil
	}

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

	if err := s.store.Users().Create(ctx, user); err != nil {
		return err
	}

	slog.Info("Created admin user", "email", user.Email)
	return nil
}

func (s *Seeder) seedDataSource(ctx context.Context) (string, error) {
	// Check if data source already exists
	existing, _ := s.store.DataSources().List(ctx)
	for _, ds := range existing {
		if ds.Name == "Northwind" {
			slog.Info("Northwind data source already exists", "id", ds.ID)
			return ds.ID, nil
		}
	}

	// Determine data directory
	dataDir := s.dataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprint", "bi")
	}

	// Create Northwind database
	northwindDBPath := filepath.Join(dataDir, "northwind.db")
	if err := createNorthwindDatabase(northwindDBPath); err != nil {
		return "", fmt.Errorf("create northwind database: %w", err)
	}

	ds := &store.DataSource{
		Name:     "Northwind",
		Engine:   "sqlite",
		Database: northwindDBPath,
	}

	if err := s.store.DataSources().Create(ctx, ds); err != nil {
		return "", err
	}

	slog.Info("Created Northwind data source", "id", ds.ID, "path", northwindDBPath)

	// Sync tables
	if err := s.syncTables(ctx, ds.ID, northwindDBPath); err != nil {
		slog.Warn("Failed to sync tables", "error", err)
	}

	return ds.ID, nil
}

func (s *Seeder) syncTables(ctx context.Context, dsID, dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get table names
	rows, err := db.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}

		// Get row count
		var rowCount int64
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount)

		// Create table record
		table := &store.Table{
			DataSourceID: dsID,
			Name:         tableName,
			DisplayName:  formatTableName(tableName),
			RowCount:     rowCount,
		}
		if err := s.store.Tables().Create(ctx, table); err != nil {
			slog.Warn("Failed to create table", "table", tableName, "error", err)
			continue
		}

		// Get columns
		colRows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
		if err != nil {
			continue
		}

		pos := 0
		for colRows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dflt interface{}
			colRows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk)

			col := &store.Column{
				TableID:     table.ID,
				Name:        name,
				DisplayName: formatColumnName(name),
				Type:        mapType(colType),
				Position:    pos,
			}
			s.store.Tables().CreateColumn(ctx, col)
			pos++
		}
		colRows.Close()

		slog.Info("Synced table", "table", tableName, "rows", rowCount, "columns", pos)
	}

	return nil
}

func (s *Seeder) seedCollections(ctx context.Context) (map[string]string, error) {
	collections := []struct {
		name        string
		description string
		color       string
	}{
		{"Executive", "High-level business metrics and KPIs", "#509EE3"},
		{"Sales", "Sales performance and revenue analysis", "#84BB4C"},
		{"Products", "Product catalog and inventory analytics", "#F2A86F"},
		{"Customers", "Customer insights and segmentation", "#7172AD"},
		{"Operations", "Operational metrics and logistics", "#ED6E6E"},
		{"Finance", "Financial analysis and profitability metrics", "#88BF4D"},
		{"Geographic", "Geographic and regional analysis", "#A989C5"},
		{"Trends", "Time-series analysis and forecasting", "#EF8C8C"},
	}

	ids := make(map[string]string)
	for _, c := range collections {
		// Check if exists
		existing, _ := s.store.Collections().List(ctx)
		found := false
		for _, e := range existing {
			if e.Name == c.name {
				ids[c.name] = e.ID
				found = true
				break
			}
		}
		if found {
			continue
		}

		coll := &store.Collection{
			Name:        c.name,
			Description: c.description,
			Color:       c.color,
		}
		if err := s.store.Collections().Create(ctx, coll); err != nil {
			return nil, err
		}
		ids[c.name] = coll.ID
		slog.Info("Created collection", "name", c.name)
	}

	return ids, nil
}

func (s *Seeder) seedQuestions(ctx context.Context, dsID string, collIDs map[string]string) (map[string]string, error) {
	questions := []struct {
		name       string
		desc       string
		collection string
		queryType  string
		query      map[string]interface{}
		viz        map[string]interface{}
	}{
		// =====================================================================
		// EXECUTIVE QUESTIONS - KPIs and Overview
		// =====================================================================
		{
			name:       "Total Revenue",
			desc:       "Sum of all order revenue",
			collection: "Executive",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					printf("$%,.0f", SUM(od.unit_price * od.quantity * (1 - od.discount))) as total_revenue
				FROM order_details od`,
			},
			viz: map[string]interface{}{
				"type": "number",
				"settings": map[string]interface{}{
					"prefix": "",
				},
			},
		},
		{
			name:       "Total Orders",
			desc:       "Count of all orders",
			collection: "Executive",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT COUNT(*) as total_orders FROM orders`,
			},
			viz: map[string]interface{}{
				"type": "number",
			},
		},
		{
			name:       "Average Order Value",
			desc:       "Average revenue per order",
			collection: "Executive",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					printf("$%.2f", AVG(order_total)) as avg_order_value
				FROM (
					SELECT o.id, SUM(od.unit_price * od.quantity * (1 - od.discount)) as order_total
					FROM orders o
					JOIN order_details od ON o.id = od.order_id
					GROUP BY o.id
				)`,
			},
			viz: map[string]interface{}{
				"type": "number",
			},
		},
		{
			name:       "Active Customers",
			desc:       "Customers with orders in the last 90 days",
			collection: "Executive",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT COUNT(DISTINCT customer_id) as active_customers
				FROM orders
				WHERE order_date >= date('now', '-90 days')`,
			},
			viz: map[string]interface{}{
				"type": "number",
			},
		},
		{
			name:       "Revenue by Month",
			desc:       "Monthly revenue trend over time",
			collection: "Executive",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-%m', o.order_date) as month,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM orders o
				JOIN order_details od ON o.id = od.order_id
				GROUP BY month
				ORDER BY month`,
			},
			viz: map[string]interface{}{
				"type": "line",
				"settings": map[string]interface{}{
					"x_axis":   "month",
					"y_axis":   "revenue",
					"showArea": true,
				},
			},
		},
		{
			name:       "Orders by Month",
			desc:       "Monthly order count trend",
			collection: "Executive",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-%m', order_date) as month,
					COUNT(*) as orders
				FROM orders
				GROUP BY month
				ORDER BY month`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "month",
					"y_axis": "orders",
				},
			},
		},
		{
			name:       "Revenue Growth Rate",
			desc:       "Month-over-month revenue growth percentage",
			collection: "Executive",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `WITH monthly_revenue AS (
					SELECT
						strftime('%Y-%m', o.order_date) as month,
						SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
					FROM orders o
					JOIN order_details od ON o.id = od.order_id
					GROUP BY month
				)
				SELECT
					month,
					revenue,
					LAG(revenue) OVER (ORDER BY month) as prev_revenue,
					CASE
						WHEN LAG(revenue) OVER (ORDER BY month) IS NULL THEN NULL
						ELSE printf("%.1f%%", (revenue - LAG(revenue) OVER (ORDER BY month)) / LAG(revenue) OVER (ORDER BY month) * 100)
					END as growth_rate
				FROM monthly_revenue
				ORDER BY month`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		// =====================================================================
		// SALES QUESTIONS - Performance Analysis
		// =====================================================================
		{
			name:       "Sales by Category",
			desc:       "Revenue breakdown by product category",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.name as category,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM order_details od
				JOIN products p ON od.product_id = p.id
				JOIN categories c ON p.category_id = c.id
				GROUP BY c.id
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "category",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Top 10 Products by Revenue",
			desc:       "Best selling products by total revenue",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					p.name as product,
					SUM(od.quantity) as units_sold,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM order_details od
				JOIN products p ON od.product_id = p.id
				GROUP BY p.id
				ORDER BY revenue DESC
				LIMIT 10`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "product",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Sales by Region",
			desc:       "Revenue distribution by customer region",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.region,
					COUNT(DISTINCT o.id) as orders,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM orders o
				JOIN customers c ON o.customer_id = c.id
				JOIN order_details od ON o.id = od.order_id
				WHERE c.region IS NOT NULL
				GROUP BY c.region
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "pie",
				"settings": map[string]interface{}{
					"dimension": "region",
					"metric":    "revenue",
				},
			},
		},
		{
			name:       "Sales Rep Performance",
			desc:       "Revenue by sales representative",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					e.first_name || ' ' || e.last_name as employee,
					COUNT(DISTINCT o.id) as orders,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM orders o
				JOIN employees e ON o.employee_id = e.id
				JOIN order_details od ON o.id = od.order_id
				GROUP BY e.id
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "employee",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Category Revenue Share",
			desc:       "Revenue percentage by product category",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.name as category,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue,
					printf("%.1f%%", 100.0 * SUM(od.unit_price * od.quantity * (1 - od.discount)) /
						(SELECT SUM(unit_price * quantity * (1 - discount)) FROM order_details)) as share
				FROM order_details od
				JOIN products p ON od.product_id = p.id
				JOIN categories c ON p.category_id = c.id
				GROUP BY c.id
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "pie",
				"settings": map[string]interface{}{
					"dimension": "category",
					"metric":    "revenue",
				},
			},
		},
		{
			name:       "Discount Analysis",
			desc:       "Revenue impact of discounts",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					CASE
						WHEN discount = 0 THEN 'No Discount'
						WHEN discount <= 0.05 THEN '5% Discount'
						WHEN discount <= 0.10 THEN '10% Discount'
						ELSE '15%+ Discount'
					END as discount_tier,
					COUNT(*) as line_items,
					SUM(unit_price * quantity) as gross_revenue,
					SUM(unit_price * quantity * discount) as discount_amount,
					SUM(unit_price * quantity * (1 - discount)) as net_revenue
				FROM order_details
				GROUP BY discount_tier
				ORDER BY net_revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "discount_tier",
					"y_axis": "net_revenue",
				},
			},
		},
		// =====================================================================
		// PRODUCTS QUESTIONS - Inventory and Catalog
		// =====================================================================
		{
			name:       "Product Inventory Status",
			desc:       "Current stock levels by product",
			collection: "Products",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					p.name as product,
					c.name as category,
					p.units_in_stock as stock,
					p.unit_price as price,
					CASE
						WHEN p.units_in_stock = 0 THEN 'Out of Stock'
						WHEN p.units_in_stock < 10 THEN 'Low Stock'
						ELSE 'In Stock'
					END as status
				FROM products p
				JOIN categories c ON p.category_id = c.id
				WHERE p.discontinued = 0
				ORDER BY p.units_in_stock ASC`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		{
			name:       "Products by Category",
			desc:       "Product count and average price by category",
			collection: "Products",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.name as category,
					COUNT(*) as product_count,
					printf("$%.2f", AVG(p.unit_price)) as avg_price,
					SUM(p.units_in_stock) as total_stock
				FROM products p
				JOIN categories c ON p.category_id = c.id
				WHERE p.discontinued = 0
				GROUP BY c.id
				ORDER BY product_count DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "category",
					"y_axis": "product_count",
				},
			},
		},
		{
			name:       "Low Stock Alert",
			desc:       "Products with stock below 10 units",
			collection: "Products",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					p.name as product,
					c.name as category,
					s.company_name as supplier,
					p.units_in_stock as stock,
					p.reorder_level
				FROM products p
				JOIN categories c ON p.category_id = c.id
				JOIN suppliers s ON p.supplier_id = s.id
				WHERE p.units_in_stock < 10 AND p.discontinued = 0
				ORDER BY p.units_in_stock ASC`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		{
			name:       "Inventory Value by Category",
			desc:       "Total inventory value breakdown",
			collection: "Products",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.name as category,
					SUM(p.units_in_stock) as total_units,
					SUM(p.units_in_stock * p.unit_price) as inventory_value
				FROM products p
				JOIN categories c ON p.category_id = c.id
				WHERE p.discontinued = 0
				GROUP BY c.id
				ORDER BY inventory_value DESC`,
			},
			viz: map[string]interface{}{
				"type": "pie",
				"settings": map[string]interface{}{
					"dimension": "category",
					"metric":    "inventory_value",
				},
			},
		},
		{
			name:       "Supplier Product Count",
			desc:       "Number of products by supplier",
			collection: "Products",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					s.company_name as supplier,
					s.country,
					COUNT(*) as products,
					SUM(p.units_in_stock) as total_stock
				FROM suppliers s
				LEFT JOIN products p ON s.id = p.supplier_id
				GROUP BY s.id
				ORDER BY products DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "supplier",
					"y_axis": "products",
				},
			},
		},
		// =====================================================================
		// CUSTOMERS QUESTIONS - Insights and Segmentation
		// =====================================================================
		{
			name:       "Top 10 Customers",
			desc:       "Highest revenue customers",
			collection: "Customers",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.company_name as customer,
					c.country,
					COUNT(DISTINCT o.id) as orders,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM customers c
				JOIN orders o ON c.id = o.customer_id
				JOIN order_details od ON o.id = od.order_id
				GROUP BY c.id
				ORDER BY revenue DESC
				LIMIT 10`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		{
			name:       "Customers by Country",
			desc:       "Customer distribution by country",
			collection: "Customers",
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
			name:       "Customer Order Frequency",
			desc:       "Distribution of orders per customer",
			collection: "Customers",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					CASE
						WHEN order_count = 1 THEN '1 order'
						WHEN order_count BETWEEN 2 AND 5 THEN '2-5 orders'
						WHEN order_count BETWEEN 6 AND 10 THEN '6-10 orders'
						ELSE '10+ orders'
					END as frequency,
					COUNT(*) as customers
				FROM (
					SELECT c.id, COUNT(o.id) as order_count
					FROM customers c
					LEFT JOIN orders o ON c.id = o.customer_id
					GROUP BY c.id
				)
				GROUP BY frequency
				ORDER BY MIN(order_count)`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "frequency",
					"y_axis": "customers",
				},
			},
		},
		{
			name:       "Customer Lifetime Value",
			desc:       "Average order value and frequency by customer",
			collection: "Customers",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.company_name as customer,
					c.country,
					COUNT(DISTINCT o.id) as total_orders,
					MIN(o.order_date) as first_order,
					MAX(o.order_date) as last_order,
					printf("$%.2f", AVG(order_total)) as avg_order_value,
					printf("$%.2f", SUM(order_total)) as lifetime_value
				FROM customers c
				JOIN orders o ON c.id = o.customer_id
				JOIN (
					SELECT order_id, SUM(unit_price * quantity * (1 - discount)) as order_total
					FROM order_details
					GROUP BY order_id
				) ot ON o.id = ot.order_id
				GROUP BY c.id
				ORDER BY SUM(order_total) DESC
				LIMIT 20`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		{
			name:       "Revenue by Country",
			desc:       "Total revenue by customer country",
			collection: "Customers",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.country,
					COUNT(DISTINCT c.id) as customers,
					COUNT(DISTINCT o.id) as orders,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM customers c
				JOIN orders o ON c.id = o.customer_id
				JOIN order_details od ON o.id = od.order_id
				GROUP BY c.country
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "country",
					"y_axis": "revenue",
				},
			},
		},
		// =====================================================================
		// OPERATIONS QUESTIONS - Fulfillment and Logistics
		// =====================================================================
		{
			name:       "Recent Orders",
			desc:       "Latest 100 orders with details",
			collection: "Operations",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					o.id as order_id,
					c.company_name as customer,
					e.first_name || ' ' || e.last_name as employee,
					o.order_date,
					o.shipped_date,
					s.company_name as shipper,
					printf("$%.2f", o.freight) as freight,
					printf("$%.2f", SUM(od.unit_price * od.quantity * (1 - od.discount))) as total
				FROM orders o
				JOIN customers c ON o.customer_id = c.id
				LEFT JOIN employees e ON o.employee_id = e.id
				LEFT JOIN shippers s ON o.shipper_id = s.id
				JOIN order_details od ON o.id = od.order_id
				GROUP BY o.id
				ORDER BY o.order_date DESC
				LIMIT 100`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		{
			name:       "Shipping Performance",
			desc:       "On-time delivery rate by shipper",
			collection: "Operations",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					s.company_name as shipper,
					COUNT(*) as total_shipments,
					SUM(CASE WHEN o.shipped_date <= o.required_date THEN 1 ELSE 0 END) as on_time,
					printf("%.1f%%", 100.0 * SUM(CASE WHEN o.shipped_date <= o.required_date THEN 1 ELSE 0 END) / COUNT(*)) as on_time_rate
				FROM orders o
				JOIN shippers s ON o.shipper_id = s.id
				WHERE o.shipped_date IS NOT NULL
				GROUP BY s.id
				ORDER BY on_time_rate DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "shipper",
					"y_axis": "total_shipments",
				},
			},
		},
		{
			name:       "Orders by Day of Week",
			desc:       "Order volume by day of week",
			collection: "Operations",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					CASE strftime('%w', order_date)
						WHEN '0' THEN 'Sunday'
						WHEN '1' THEN 'Monday'
						WHEN '2' THEN 'Tuesday'
						WHEN '3' THEN 'Wednesday'
						WHEN '4' THEN 'Thursday'
						WHEN '5' THEN 'Friday'
						WHEN '6' THEN 'Saturday'
					END as day_of_week,
					COUNT(*) as orders
				FROM orders
				GROUP BY strftime('%w', order_date)
				ORDER BY strftime('%w', order_date)`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "day_of_week",
					"y_axis": "orders",
				},
			},
		},
		{
			name:       "Pending Orders",
			desc:       "Orders not yet shipped",
			collection: "Operations",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					o.id as order_id,
					c.company_name as customer,
					o.order_date,
					o.required_date,
					CAST(julianday(o.required_date) - julianday('now') AS INTEGER) as days_until_due,
					printf("$%.2f", SUM(od.unit_price * od.quantity * (1 - od.discount))) as total
				FROM orders o
				JOIN customers c ON o.customer_id = c.id
				JOIN order_details od ON o.id = od.order_id
				WHERE o.shipped_date IS NULL
				GROUP BY o.id
				ORDER BY o.required_date ASC`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		{
			name:       "Average Shipping Time",
			desc:       "Days between order and shipment by shipper",
			collection: "Operations",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					s.company_name as shipper,
					COUNT(*) as shipments,
					printf("%.1f days", AVG(julianday(o.shipped_date) - julianday(o.order_date))) as avg_ship_time,
					MIN(julianday(o.shipped_date) - julianday(o.order_date)) as min_days,
					MAX(julianday(o.shipped_date) - julianday(o.order_date)) as max_days
				FROM orders o
				JOIN shippers s ON o.shipper_id = s.id
				WHERE o.shipped_date IS NOT NULL
				GROUP BY s.id
				ORDER BY avg_ship_time`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		{
			name:       "Freight Cost Analysis",
			desc:       "Average freight by destination country",
			collection: "Operations",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					ship_country as country,
					COUNT(*) as orders,
					printf("$%.2f", AVG(freight)) as avg_freight,
					printf("$%.2f", SUM(freight)) as total_freight
				FROM orders
				WHERE ship_country IS NOT NULL
				GROUP BY ship_country
				ORDER BY SUM(freight) DESC
				LIMIT 15`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "country",
					"y_axis": "orders",
				},
			},
		},
		// =====================================================================
		// FINANCE QUESTIONS - Profitability and Cost Analysis
		// =====================================================================
		{
			name:       "Gross Profit Margin",
			desc:       "Overall profit margin percentage",
			collection: "Finance",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					printf("%.1f%%", 100.0 * (SUM(od.unit_price * od.quantity * (1 - od.discount)) - SUM(od.unit_price * od.quantity * 0.6)) / SUM(od.unit_price * od.quantity * (1 - od.discount))) as gross_margin
				FROM order_details od`,
			},
			viz: map[string]interface{}{
				"type": "number",
			},
		},
		{
			name:       "Profit by Category",
			desc:       "Estimated profit margin by product category",
			collection: "Finance",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.name as category,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue,
					SUM(od.unit_price * od.quantity * (1 - od.discount) * 0.4) as estimated_profit,
					printf("%.1f%%", 40.0) as margin
				FROM order_details od
				JOIN products p ON od.product_id = p.id
				JOIN categories c ON p.category_id = c.id
				GROUP BY c.id
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "category",
					"y_axis": "estimated_profit",
				},
			},
		},
		{
			name:       "Monthly Profit Trend",
			desc:       "Estimated profit over time",
			collection: "Finance",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-%m', o.order_date) as month,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue,
					SUM(od.unit_price * od.quantity * (1 - od.discount) * 0.4) as profit
				FROM orders o
				JOIN order_details od ON o.id = od.order_id
				GROUP BY month
				ORDER BY month`,
			},
			viz: map[string]interface{}{
				"type": "area",
				"settings": map[string]interface{}{
					"x_axis":   "month",
					"y_axis":   "profit",
					"showArea": true,
				},
			},
		},
		{
			name:       "Revenue vs Freight Cost",
			desc:       "Compare revenue to shipping costs",
			collection: "Finance",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-%m', o.order_date) as month,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue,
					SUM(o.freight) as freight_cost,
					printf("%.2f%%", 100.0 * SUM(o.freight) / SUM(od.unit_price * od.quantity * (1 - od.discount))) as freight_pct
				FROM orders o
				JOIN order_details od ON o.id = od.order_id
				GROUP BY month
				ORDER BY month`,
			},
			viz: map[string]interface{}{
				"type": "combo",
				"settings": map[string]interface{}{
					"x_axis": "month",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Discount Impact Analysis",
			desc:       "Revenue lost to discounts",
			collection: "Finance",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-%m', o.order_date) as month,
					SUM(od.unit_price * od.quantity) as gross_revenue,
					SUM(od.unit_price * od.quantity * od.discount) as discount_amount,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as net_revenue
				FROM orders o
				JOIN order_details od ON o.id = od.order_id
				GROUP BY month
				ORDER BY month`,
			},
			viz: map[string]interface{}{
				"type": "line",
				"settings": map[string]interface{}{
					"x_axis": "month",
					"y_axis": "discount_amount",
				},
			},
		},
		// =====================================================================
		// GEOGRAPHIC QUESTIONS - Regional Analysis
		// =====================================================================
		{
			name:       "Revenue by Country Map",
			desc:       "Geographic distribution of revenue",
			collection: "Geographic",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.country,
					COUNT(DISTINCT o.id) as orders,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM orders o
				JOIN customers c ON o.customer_id = c.id
				JOIN order_details od ON o.id = od.order_id
				GROUP BY c.country
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "country",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Regional Growth Comparison",
			desc:       "Revenue growth by region",
			collection: "Geographic",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.region,
					COUNT(DISTINCT c.id) as customers,
					COUNT(DISTINCT o.id) as orders,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue,
					printf("$%.2f", SUM(od.unit_price * od.quantity * (1 - od.discount)) / COUNT(DISTINCT o.id)) as avg_order_value
				FROM customers c
				JOIN orders o ON c.id = o.customer_id
				JOIN order_details od ON o.id = od.order_id
				WHERE c.region IS NOT NULL
				GROUP BY c.region
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "region",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "City Performance",
			desc:       "Top performing cities by revenue",
			collection: "Geographic",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.city,
					c.country,
					COUNT(DISTINCT o.id) as orders,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM customers c
				JOIN orders o ON c.id = o.customer_id
				JOIN order_details od ON o.id = od.order_id
				GROUP BY c.city, c.country
				ORDER BY revenue DESC
				LIMIT 15`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		{
			name:       "Supplier Geography",
			desc:       "Supplier distribution by country",
			collection: "Geographic",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					s.country,
					COUNT(*) as suppliers,
					COUNT(DISTINCT p.id) as products
				FROM suppliers s
				LEFT JOIN products p ON s.id = p.supplier_id
				GROUP BY s.country
				ORDER BY suppliers DESC`,
			},
			viz: map[string]interface{}{
				"type": "pie",
				"settings": map[string]interface{}{
					"dimension": "country",
					"metric":    "suppliers",
				},
			},
		},
		{
			name:       "Shipping Destination Analysis",
			desc:       "Where orders are shipped to",
			collection: "Geographic",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					ship_country as country,
					ship_city as city,
					COUNT(*) as shipments,
					SUM(freight) as total_freight
				FROM orders
				WHERE ship_country IS NOT NULL
				GROUP BY ship_country, ship_city
				ORDER BY shipments DESC
				LIMIT 20`,
			},
			viz: map[string]interface{}{
				"type": "table",
			},
		},
		// =====================================================================
		// TRENDS QUESTIONS - Time Series Analysis
		// =====================================================================
		{
			name:       "Daily Order Volume",
			desc:       "Orders per day over time",
			collection: "Trends",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					order_date as date,
					COUNT(*) as orders
				FROM orders
				GROUP BY order_date
				ORDER BY order_date
				LIMIT 90`,
			},
			viz: map[string]interface{}{
				"type": "line",
				"settings": map[string]interface{}{
					"x_axis": "date",
					"y_axis": "orders",
				},
			},
		},
		{
			name:       "Weekly Revenue Trend",
			desc:       "Revenue aggregated by week",
			collection: "Trends",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-W%W', o.order_date) as week,
					COUNT(DISTINCT o.id) as orders,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM orders o
				JOIN order_details od ON o.id = od.order_id
				GROUP BY week
				ORDER BY week`,
			},
			viz: map[string]interface{}{
				"type": "area",
				"settings": map[string]interface{}{
					"x_axis": "week",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Quarterly Performance",
			desc:       "Revenue by quarter",
			collection: "Trends",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y', o.order_date) || '-Q' || ((CAST(strftime('%m', o.order_date) AS INTEGER) - 1) / 3 + 1) as quarter,
					COUNT(DISTINCT o.id) as orders,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM orders o
				JOIN order_details od ON o.id = od.order_id
				GROUP BY quarter
				ORDER BY quarter`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "quarter",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Product Sales Velocity",
			desc:       "Units sold per product over time",
			collection: "Trends",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					p.name as product,
					SUM(od.quantity) as total_units,
					COUNT(DISTINCT o.id) as order_count,
					printf("%.1f", 1.0 * SUM(od.quantity) / COUNT(DISTINCT strftime('%Y-%m', o.order_date))) as avg_monthly_units
				FROM products p
				JOIN order_details od ON p.id = od.product_id
				JOIN orders o ON od.order_id = o.id
				GROUP BY p.id
				ORDER BY total_units DESC
				LIMIT 15`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "product",
					"y_axis": "total_units",
				},
			},
		},
		{
			name:       "Seasonal Patterns",
			desc:       "Revenue by month of year (seasonality)",
			collection: "Trends",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					CASE strftime('%m', o.order_date)
						WHEN '01' THEN 'January'
						WHEN '02' THEN 'February'
						WHEN '03' THEN 'March'
						WHEN '04' THEN 'April'
						WHEN '05' THEN 'May'
						WHEN '06' THEN 'June'
						WHEN '07' THEN 'July'
						WHEN '08' THEN 'August'
						WHEN '09' THEN 'September'
						WHEN '10' THEN 'October'
						WHEN '11' THEN 'November'
						WHEN '12' THEN 'December'
					END as month_name,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue,
					COUNT(DISTINCT o.id) as orders
				FROM orders o
				JOIN order_details od ON o.id = od.order_id
				GROUP BY strftime('%m', o.order_date)
				ORDER BY strftime('%m', o.order_date)`,
			},
			viz: map[string]interface{}{
				"type": "bar",
				"settings": map[string]interface{}{
					"x_axis": "month_name",
					"y_axis": "revenue",
				},
			},
		},
		{
			name:       "Customer Acquisition Trend",
			desc:       "New customers over time",
			collection: "Trends",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-%m', first_order) as month,
					COUNT(*) as new_customers
				FROM (
					SELECT customer_id, MIN(order_date) as first_order
					FROM orders
					GROUP BY customer_id
				)
				GROUP BY month
				ORDER BY month`,
			},
			viz: map[string]interface{}{
				"type": "area",
				"settings": map[string]interface{}{
					"x_axis": "month",
					"y_axis": "new_customers",
				},
			},
		},
		{
			name:       "Category Trends Over Time",
			desc:       "Revenue by category per month",
			collection: "Trends",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-%m', o.order_date) as month,
					c.name as category,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM orders o
				JOIN order_details od ON o.id = od.order_id
				JOIN products p ON od.product_id = p.id
				JOIN categories c ON p.category_id = c.id
				GROUP BY month, c.id
				ORDER BY month, revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "line",
				"settings": map[string]interface{}{
					"x_axis": "month",
					"y_axis": "revenue",
				},
			},
		},
		// =====================================================================
		// SHOWCASE QUESTIONS - Additional visualization types
		// =====================================================================
		{
			name:       "Monthly Revenue Changes (Waterfall)",
			desc:       "Revenue changes month over month shown as waterfall",
			collection: "Executive",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `WITH monthly AS (
					SELECT
						strftime('%Y-%m', o.order_date) as month,
						SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
					FROM orders o
					JOIN order_details od ON o.id = od.order_id
					GROUP BY month
					ORDER BY month
					LIMIT 6
				),
				changes AS (
					SELECT
						month as period,
						revenue - LAG(revenue, 1, 0) OVER (ORDER BY month) as change
					FROM monthly
				)
				SELECT period, change FROM changes
				UNION ALL
				SELECT 'Total', SUM(change) FROM changes`,
			},
			viz: map[string]interface{}{
				"type": "waterfall",
			},
		},
		{
			name:       "Products by Price and Quantity (Bubble)",
			desc:       "Product analysis with price, quantity, and revenue as bubble size",
			collection: "Products",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					AVG(p.unit_price) as avg_price,
					SUM(od.quantity) as total_quantity,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as total_revenue
				FROM products p
				JOIN order_details od ON p.id = od.product_id
				GROUP BY p.id
				HAVING total_quantity > 100
				ORDER BY total_revenue DESC
				LIMIT 20`,
			},
			viz: map[string]interface{}{
				"type": "bubble",
			},
		},
		{
			name:       "Revenue Goal Progress",
			desc:       "Progress toward annual revenue goal",
			collection: "Executive",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as current_revenue
				FROM order_details od`,
			},
			viz: map[string]interface{}{
				"type": "progress",
				"settings": map[string]interface{}{
					"goal": 2000000,
				},
			},
		},
		{
			name:       "Order Completion Rate (Gauge)",
			desc:       "Percentage of orders shipped on time",
			collection: "Operations",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					CAST(100.0 * SUM(CASE WHEN shipped_date <= required_date THEN 1 ELSE 0 END) / COUNT(*) AS INTEGER) as on_time_rate
				FROM orders
				WHERE shipped_date IS NOT NULL`,
			},
			viz: map[string]interface{}{
				"type": "gauge",
				"settings": map[string]interface{}{
					"min": 0,
					"max": 100,
				},
			},
		},
		{
			name:       "Sales Pipeline (Funnel)",
			desc:       "Order stages from placed to delivered",
			collection: "Operations",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					'Total Orders' as stage, COUNT(*) as count FROM orders
				UNION ALL
				SELECT 'Shipped', COUNT(*) FROM orders WHERE shipped_date IS NOT NULL
				UNION ALL
				SELECT 'On Time', COUNT(*) FROM orders WHERE shipped_date IS NOT NULL AND shipped_date <= required_date`,
			},
			viz: map[string]interface{}{
				"type": "funnel",
			},
		},
		{
			name:       "Revenue by Country (Map)",
			desc:       "Geographic distribution of revenue by country",
			collection: "Geographic",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.country,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM orders o
				JOIN customers c ON o.customer_id = c.id
				JOIN order_details od ON o.id = od.order_id
				GROUP BY c.country
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "map-region",
			},
		},
		{
			name:       "Category Distribution (Donut)",
			desc:       "Revenue share by product category as donut chart",
			collection: "Sales",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					c.name as category,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM order_details od
				JOIN products p ON od.product_id = p.id
				JOIN categories c ON p.category_id = c.id
				GROUP BY c.id
				ORDER BY revenue DESC`,
			},
			viz: map[string]interface{}{
				"type": "donut",
			},
		},
		{
			name:       "Monthly Revenue Trend (Area)",
			desc:       "Revenue over time as filled area chart",
			collection: "Trends",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					strftime('%Y-%m', o.order_date) as month,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM orders o
				JOIN order_details od ON o.id = od.order_id
				GROUP BY month
				ORDER BY month`,
			},
			viz: map[string]interface{}{
				"type": "area",
				"settings": map[string]interface{}{
					"stacked": false,
				},
			},
		},
		{
			name:       "Top Products (Horizontal Bar)",
			desc:       "Best selling products as horizontal bars",
			collection: "Products",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					p.name as product,
					SUM(od.unit_price * od.quantity * (1 - od.discount)) as revenue
				FROM order_details od
				JOIN products p ON od.product_id = p.id
				GROUP BY p.id
				ORDER BY revenue DESC
				LIMIT 10`,
			},
			viz: map[string]interface{}{
				"type": "row",
			},
		},
		{
			name:       "Price vs Quantity (Scatter)",
			desc:       "Product price vs quantity sold scatter plot",
			collection: "Products",
			queryType:  "native",
			query: map[string]interface{}{
				"sql": `SELECT
					p.unit_price as price,
					SUM(od.quantity) as quantity_sold
				FROM products p
				JOIN order_details od ON p.id = od.product_id
				GROUP BY p.id`,
			},
			viz: map[string]interface{}{
				"type": "scatter",
			},
		},
	}

	ids := make(map[string]string)
	for _, q := range questions {
		question := &store.Question{
			Name:          q.name,
			Description:   q.desc,
			CollectionID:  collIDs[q.collection],
			DataSourceID:  dsID,
			QueryType:     q.queryType,
			Query:         q.query,
			Visualization: q.viz,
		}
		if err := s.store.Questions().Create(ctx, question); err != nil {
			slog.Warn("Failed to create question", "name", q.name, "error", err)
			continue
		}
		ids[q.name] = question.ID
		slog.Info("Created question", "name", q.name)
	}

	return ids, nil
}

func (s *Seeder) seedDashboards(ctx context.Context, collIDs map[string]string, questionIDs map[string]string) error {
	dashboards := []struct {
		name        string
		description string
		collection  string
		cards       []struct {
			question string
			row, col int
			w, h     int
		}
	}{
		{
			name:        "Executive Overview",
			description: "High-level business metrics and KPIs for leadership",
			collection:  "Executive",
			cards: []struct {
				question string
				row, col int
				w, h     int
			}{
				{"Total Revenue", 0, 0, 3, 2},
				{"Total Orders", 0, 3, 3, 2},
				{"Average Order Value", 0, 6, 3, 2},
				{"Active Customers", 0, 9, 3, 2},
				{"Revenue by Month", 2, 0, 8, 4},
				{"Orders by Month", 2, 8, 4, 4},
				{"Sales by Category", 6, 0, 6, 4},
				{"Sales by Region", 6, 6, 6, 4},
			},
		},
		{
			name:        "Sales Performance",
			description: "Sales team and product performance analysis",
			collection:  "Sales",
			cards: []struct {
				question string
				row, col int
				w, h     int
			}{
				{"Top 10 Products by Revenue", 0, 0, 6, 4},
				{"Sales Rep Performance", 0, 6, 6, 4},
				{"Sales by Category", 4, 0, 4, 4},
				{"Category Revenue Share", 4, 4, 4, 4},
				{"Discount Analysis", 4, 8, 4, 4},
				{"Sales by Region", 8, 0, 6, 4},
			},
		},
		{
			name:        "Product Analytics",
			description: "Product catalog and inventory insights",
			collection:  "Products",
			cards: []struct {
				question string
				row, col int
				w, h     int
			}{
				{"Products by Category", 0, 0, 6, 4},
				{"Inventory Value by Category", 0, 6, 6, 4},
				{"Low Stock Alert", 4, 0, 6, 4},
				{"Supplier Product Count", 4, 6, 6, 4},
				{"Product Inventory Status", 8, 0, 12, 6},
			},
		},
		{
			name:        "Customer Insights",
			description: "Customer behavior and segmentation analysis",
			collection:  "Customers",
			cards: []struct {
				question string
				row, col int
				w, h     int
			}{
				{"Top 10 Customers", 0, 0, 8, 4},
				{"Customers by Country", 0, 8, 4, 4},
				{"Customer Order Frequency", 4, 0, 6, 4},
				{"Revenue by Country", 4, 6, 6, 4},
				{"Customer Lifetime Value", 8, 0, 12, 5},
			},
		},
		{
			name:        "Operations Dashboard",
			description: "Order fulfillment and logistics metrics",
			collection:  "Operations",
			cards: []struct {
				question string
				row, col int
				w, h     int
			}{
				{"Shipping Performance", 0, 0, 6, 4},
				{"Orders by Day of Week", 0, 6, 6, 4},
				{"Average Shipping Time", 4, 0, 6, 3},
				{"Freight Cost Analysis", 4, 6, 6, 4},
				{"Pending Orders", 7, 0, 6, 5},
				{"Recent Orders", 7, 6, 6, 5},
			},
		},
		{
			name:        "Financial Analysis",
			description: "Profitability and cost analysis metrics",
			collection:  "Finance",
			cards: []struct {
				question string
				row, col int
				w, h     int
			}{
				{"Gross Profit Margin", 0, 0, 4, 2},
				{"Total Revenue", 0, 4, 4, 2},
				{"Total Orders", 0, 8, 4, 2},
				{"Profit by Category", 2, 0, 6, 4},
				{"Monthly Profit Trend", 2, 6, 6, 4},
				{"Revenue vs Freight Cost", 6, 0, 6, 4},
				{"Discount Impact Analysis", 6, 6, 6, 4},
			},
		},
		{
			name:        "Geographic Insights",
			description: "Regional performance and distribution analysis",
			collection:  "Geographic",
			cards: []struct {
				question string
				row, col int
				w, h     int
			}{
				{"Revenue by Country Map", 0, 0, 6, 4},
				{"Regional Growth Comparison", 0, 6, 6, 4},
				{"City Performance", 4, 0, 6, 5},
				{"Supplier Geography", 4, 6, 6, 4},
				{"Shipping Destination Analysis", 9, 0, 12, 5},
			},
		},
		{
			name:        "Trends & Forecasting",
			description: "Time-series analysis and business trends",
			collection:  "Trends",
			cards: []struct {
				question string
				row, col int
				w, h     int
			}{
				{"Daily Order Volume", 0, 0, 8, 4},
				{"Quarterly Performance", 0, 8, 4, 4},
				{"Weekly Revenue Trend", 4, 0, 6, 4},
				{"Seasonal Patterns", 4, 6, 6, 4},
				{"Product Sales Velocity", 8, 0, 6, 4},
				{"Customer Acquisition Trend", 8, 6, 6, 4},
				{"Category Trends Over Time", 12, 0, 12, 4},
			},
		},
		{
			name:        "Chart Type Showcase",
			description: "All visualization types for verification and demonstration",
			collection:  "Executive",
			cards: []struct {
				question string
				row, col int
				w, h     int
			}{
				// Row 1: Scalar types (number, trend, gauge, progress)
				{"Total Revenue", 0, 0, 3, 2},
				{"Total Orders", 0, 3, 3, 2},
				{"Order Completion Rate (Gauge)", 0, 6, 3, 2},
				{"Revenue Goal Progress", 0, 9, 3, 2},
				// Row 2: Time series (line, area)
				{"Revenue by Month", 2, 0, 6, 4},
				{"Monthly Revenue Trend (Area)", 2, 6, 6, 4},
				// Row 3: Bar variants (bar, row, combo)
				{"Sales by Category", 6, 0, 4, 4},
				{"Top Products (Horizontal Bar)", 6, 4, 4, 4},
				{"Revenue vs Freight Cost", 6, 8, 4, 4},
				// Row 4: Parts of whole (pie, donut, funnel)
				{"Category Revenue Share", 10, 0, 4, 4},
				{"Category Distribution (Donut)", 10, 4, 4, 4},
				{"Sales Pipeline (Funnel)", 10, 8, 4, 4},
				// Row 5: Distribution (scatter, bubble, waterfall)
				{"Price vs Quantity (Scatter)", 14, 0, 4, 4},
				{"Products by Price and Quantity (Bubble)", 14, 4, 4, 4},
				{"Monthly Revenue Changes (Waterfall)", 14, 8, 4, 4},
				// Row 6: Geographic and Table
				{"Revenue by Country (Map)", 18, 0, 6, 4},
				{"Recent Orders", 18, 6, 6, 4},
			},
		},
	}

	for _, d := range dashboards {
		dashboard := &store.Dashboard{
			Name:         d.name,
			Description:  d.description,
			CollectionID: collIDs[d.collection],
		}
		if err := s.store.Dashboards().Create(ctx, dashboard); err != nil {
			slog.Warn("Failed to create dashboard", "name", d.name, "error", err)
			continue
		}

		// Add cards
		for _, c := range d.cards {
			qID, ok := questionIDs[c.question]
			if !ok {
				slog.Warn("Question not found for card", "question", c.question)
				continue
			}

			card := &store.DashboardCard{
				DashboardID: dashboard.ID,
				QuestionID:  qID,
				CardType:    "question",
				Row:         c.row,
				Col:         c.col,
				Width:       c.w,
				Height:      c.h,
			}
			if err := s.store.Dashboards().CreateCard(ctx, card); err != nil {
				slog.Warn("Failed to create card", "question", c.question, "error", err)
			}
		}

		slog.Info("Created dashboard", "name", d.name, "cards", len(d.cards))
	}

	return nil
}

// formatTableName converts snake_case to Title Case
func formatTableName(name string) string {
	words := make([]byte, 0, len(name))
	capitalize := true
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '_' {
			words = append(words, ' ')
			capitalize = true
		} else if capitalize {
			if c >= 'a' && c <= 'z' {
				c -= 32 // to uppercase
			}
			words = append(words, c)
			capitalize = false
		} else {
			words = append(words, c)
		}
	}
	return string(words)
}

// formatColumnName converts snake_case to Title Case
func formatColumnName(name string) string {
	return formatTableName(name)
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
