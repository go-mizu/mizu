package seed

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

// Northwind data for realistic demo
var (
	// Categories with descriptions
	categories = []struct {
		name        string
		description string
	}{
		{"Beverages", "Soft drinks, coffees, teas, beers, and ales"},
		{"Condiments", "Sweet and savory sauces, relishes, spreads, and seasonings"},
		{"Confections", "Desserts, candies, and sweet breads"},
		{"Dairy Products", "Cheeses and other dairy items"},
		{"Grains/Cereals", "Breads, crackers, pasta, and cereal"},
		{"Meat/Poultry", "Prepared meats and poultry products"},
		{"Produce", "Dried fruit and bean curd"},
		{"Seafood", "Seaweed and fish products"},
	}

	// Suppliers with realistic data
	suppliers = []struct {
		companyName  string
		contactName  string
		contactTitle string
		city         string
		country      string
		phone        string
	}{
		{"Exotic Liquids", "Charlotte Cooper", "Purchasing Manager", "London", "UK", "(171) 555-2222"},
		{"New Orleans Cajun Delights", "Shelley Burke", "Order Administrator", "New Orleans", "USA", "(100) 555-4822"},
		{"Grandma Kelly's Homestead", "Regina Murphy", "Sales Representative", "Ann Arbor", "USA", "(313) 555-5735"},
		{"Tokyo Traders", "Yoshi Nagase", "Marketing Manager", "Tokyo", "Japan", "(03) 3555-5011"},
		{"Cooperativa de Quesos", "Antonio del Valle", "Export Administrator", "Oviedo", "Spain", "(98) 598 76 54"},
		{"Mayumi's", "Mayumi Ohno", "Marketing Representative", "Osaka", "Japan", "(06) 431-7877"},
		{"Pavlova, Ltd.", "Ian Devling", "Marketing Manager", "Melbourne", "Australia", "(03) 444-2343"},
		{"Specialty Biscuits, Ltd.", "Peter Wilson", "Sales Representative", "Manchester", "UK", "(161) 555-4448"},
		{"PB Knäckebröd AB", "Lars Peterson", "Sales Agent", "Göteborg", "Sweden", "031-987 65 43"},
		{"Refrescos Americanas LTDA", "Carlos Diaz", "Marketing Manager", "São Paulo", "Brazil", "(11) 555 4640"},
	}

	// Products with pricing
	products = []struct {
		name        string
		categoryIdx int
		supplierIdx int
		unitPrice   float64
		unitsStock  int
		discontinued bool
	}{
		{"Chai", 0, 0, 18.00, 39, false},
		{"Chang", 0, 0, 19.00, 17, false},
		{"Aniseed Syrup", 1, 0, 10.00, 13, false},
		{"Chef Anton's Cajun Seasoning", 1, 1, 22.00, 53, false},
		{"Chef Anton's Gumbo Mix", 1, 1, 21.35, 0, true},
		{"Grandma's Boysenberry Spread", 1, 2, 25.00, 120, false},
		{"Uncle Bob's Organic Dried Pears", 6, 2, 30.00, 15, false},
		{"Northwoods Cranberry Sauce", 1, 2, 40.00, 6, false},
		{"Mishi Kobe Niku", 5, 3, 97.00, 29, true},
		{"Ikura", 7, 3, 31.00, 31, false},
		{"Queso Cabrales", 3, 4, 21.00, 22, false},
		{"Queso Manchego La Pastora", 3, 4, 38.00, 86, false},
		{"Konbu", 7, 5, 6.00, 24, false},
		{"Tofu", 6, 5, 23.25, 35, false},
		{"Genen Shouyu", 1, 5, 15.50, 39, false},
		{"Pavlova", 2, 6, 17.45, 29, false},
		{"Alice Mutton", 5, 6, 39.00, 0, true},
		{"Carnarvon Tigers", 7, 6, 62.50, 42, false},
		{"Teatime Chocolate Biscuits", 2, 7, 9.20, 25, false},
		{"Sir Rodney's Marmalade", 2, 7, 81.00, 40, false},
		{"Sir Rodney's Scones", 2, 7, 10.00, 3, false},
		{"Gustaf's Knäckebröd", 4, 8, 21.00, 104, false},
		{"Tunnbröd", 4, 8, 9.00, 61, false},
		{"Guaraná Fantástica", 0, 9, 4.50, 20, true},
		{"NuNuCa Nuß-Nougat-Creme", 2, 9, 14.00, 76, false},
	}

	// Customers with realistic data
	customers = []struct {
		companyName  string
		contactName  string
		contactTitle string
		city         string
		country      string
		region       string
	}{
		{"Alfreds Futterkiste", "Maria Anders", "Sales Representative", "Berlin", "Germany", "Western Europe"},
		{"Ana Trujillo Emparedados", "Ana Trujillo", "Owner", "México D.F.", "Mexico", "Central America"},
		{"Antonio Moreno Taquería", "Antonio Moreno", "Owner", "México D.F.", "Mexico", "Central America"},
		{"Around the Horn", "Thomas Hardy", "Sales Representative", "London", "UK", "British Isles"},
		{"Berglunds snabbköp", "Christina Berglund", "Order Administrator", "Luleå", "Sweden", "Scandinavia"},
		{"Blauer See Delikatessen", "Hanna Moos", "Sales Representative", "Mannheim", "Germany", "Western Europe"},
		{"Blondesddsl père et fils", "Frédérique Citeaux", "Marketing Manager", "Strasbourg", "France", "Western Europe"},
		{"Bólido Comidas preparadas", "Martín Sommer", "Owner", "Madrid", "Spain", "Southern Europe"},
		{"Bon app'", "Laurence Lebihan", "Owner", "Marseille", "France", "Western Europe"},
		{"Bottom-Dollar Markets", "Elizabeth Lincoln", "Accounting Manager", "Tsawassen", "Canada", "North America"},
		{"B's Beverages", "Victoria Ashworth", "Sales Representative", "London", "UK", "British Isles"},
		{"Cactus Comidas para llevar", "Patricio Simpson", "Sales Agent", "Buenos Aires", "Argentina", "South America"},
		{"Centro comercial Moctezuma", "Francisco Chang", "Marketing Manager", "México D.F.", "Mexico", "Central America"},
		{"Chop-suey Chinese", "Yang Wang", "Owner", "Bern", "Switzerland", "Western Europe"},
		{"Comércio Mineiro", "Pedro Afonso", "Sales Associate", "São Paulo", "Brazil", "South America"},
		{"Consolidated Holdings", "Elizabeth Brown", "Sales Representative", "London", "UK", "British Isles"},
		{"Drachenblut Delikatessen", "Sven Ottlieb", "Order Administrator", "Aachen", "Germany", "Western Europe"},
		{"Du monde entier", "Janine Labrune", "Owner", "Nantes", "France", "Western Europe"},
		{"Eastern Connection", "Ann Devon", "Sales Agent", "London", "UK", "British Isles"},
		{"Ernst Handel", "Roland Mendel", "Sales Manager", "Graz", "Austria", "Western Europe"},
		{"FISSA Fabrica Inter", "Diego Roel", "Accounting Manager", "Madrid", "Spain", "Southern Europe"},
		{"Folies gourmandes", "Martine Rancé", "Assistant Sales Agent", "Lille", "France", "Western Europe"},
		{"Folk och fä HB", "Maria Larsson", "Owner", "Bräcke", "Sweden", "Scandinavia"},
		{"Frankenversand", "Peter Franken", "Marketing Manager", "München", "Germany", "Western Europe"},
		{"France restauration", "Carine Schmitt", "Marketing Manager", "Nantes", "France", "Western Europe"},
		{"Franchi S.p.A.", "Paolo Accorti", "Sales Representative", "Torino", "Italy", "Southern Europe"},
		{"Furia Bacalhau e Frutos do Mar", "Lino Rodriguez", "Sales Manager", "Lisbon", "Portugal", "Southern Europe"},
		{"Galería del gastrónomo", "Eduardo Saavedra", "Marketing Manager", "Barcelona", "Spain", "Southern Europe"},
		{"Godos Cocina Típica", "José Pedro Freyre", "Sales Manager", "Sevilla", "Spain", "Southern Europe"},
		{"Gourmet Lanchonetes", "André Fonseca", "Sales Associate", "Campinas", "Brazil", "South America"},
		{"Great Lakes Food Market", "Howard Snyder", "Marketing Manager", "Eugene", "USA", "North America"},
		{"GROSELLA-Restaurante", "Manuel Pereira", "Owner", "Caracas", "Venezuela", "South America"},
		{"Hanari Carnes", "Mario Pontes", "Accounting Manager", "Rio de Janeiro", "Brazil", "South America"},
		{"HILARION-Abastos", "Carlos Hernández", "Sales Representative", "San Cristóbal", "Venezuela", "South America"},
		{"Hungry Coyote Import Store", "Yoshi Latimer", "Sales Representative", "Elgin", "USA", "North America"},
		{"Hungry Owl All-Night Grocers", "Patricia McKenna", "Sales Associate", "Cork", "Ireland", "British Isles"},
		{"Island Trading", "Helen Bennett", "Marketing Manager", "Cowes", "UK", "British Isles"},
		{"Königlich Essen", "Philip Cramer", "Sales Associate", "Brandenburg", "Germany", "Western Europe"},
		{"La corne d'abondance", "Daniel Tonini", "Sales Representative", "Versailles", "France", "Western Europe"},
		{"La maison d'Asie", "Annette Roulet", "Sales Manager", "Toulouse", "France", "Western Europe"},
	}

	// Employees
	employees = []struct {
		firstName string
		lastName  string
		title     string
		hireDate  string
		city      string
		country   string
	}{
		{"Nancy", "Davolio", "Sales Representative", "2020-05-01", "Seattle", "USA"},
		{"Andrew", "Fuller", "Vice President, Sales", "2019-08-14", "Tacoma", "USA"},
		{"Janet", "Leverling", "Sales Representative", "2020-04-01", "Kirkland", "USA"},
		{"Margaret", "Peacock", "Sales Representative", "2021-05-03", "Redmond", "USA"},
		{"Steven", "Buchanan", "Sales Manager", "2021-10-17", "London", "UK"},
		{"Michael", "Suyama", "Sales Representative", "2022-10-17", "London", "UK"},
		{"Robert", "King", "Sales Representative", "2022-01-02", "London", "UK"},
		{"Laura", "Callahan", "Inside Sales Coordinator", "2022-03-05", "Seattle", "USA"},
		{"Anne", "Dodsworth", "Sales Representative", "2022-11-15", "London", "UK"},
	}

	// Shippers
	shippers = []struct {
		companyName string
		phone       string
	}{
		{"Speedy Express", "(503) 555-9831"},
		{"United Package", "(503) 555-3199"},
		{"Federal Shipping", "(503) 555-9931"},
	}

	// Regions
	regions = []struct {
		description string
	}{
		{"Eastern"},
		{"Western"},
		{"Northern"},
		{"Southern"},
	}
)

// createNorthwindDatabase creates a comprehensive Northwind database
func createNorthwindDatabase(dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	// Create schema
	if err := createNorthwindSchema(db); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Check if data already exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if count > 0 {
		return nil // Data already seeded
	}

	// Seed data
	if err := seedNorthwindData(db); err != nil {
		return fmt.Errorf("seed data: %w", err)
	}

	return nil
}

func createNorthwindSchema(db *sql.DB) error {
	schema := `
	-- Categories table
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Suppliers table
	CREATE TABLE IF NOT EXISTS suppliers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		company_name TEXT NOT NULL,
		contact_name TEXT,
		contact_title TEXT,
		city TEXT,
		country TEXT,
		phone TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Products table
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		category_id INTEGER REFERENCES categories(id),
		supplier_id INTEGER REFERENCES suppliers(id),
		unit_price REAL NOT NULL DEFAULT 0,
		units_in_stock INTEGER NOT NULL DEFAULT 0,
		units_on_order INTEGER NOT NULL DEFAULT 0,
		reorder_level INTEGER NOT NULL DEFAULT 0,
		discontinued INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Customers table
	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		company_name TEXT NOT NULL,
		contact_name TEXT,
		contact_title TEXT,
		city TEXT,
		country TEXT,
		region TEXT,
		phone TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Employees table
	CREATE TABLE IF NOT EXISTS employees (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		title TEXT,
		hire_date DATE,
		city TEXT,
		country TEXT,
		reports_to INTEGER REFERENCES employees(id),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Shippers table
	CREATE TABLE IF NOT EXISTS shippers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		company_name TEXT NOT NULL,
		phone TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Regions table
	CREATE TABLE IF NOT EXISTS regions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT NOT NULL
	);

	-- Orders table
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		customer_id INTEGER NOT NULL REFERENCES customers(id),
		employee_id INTEGER REFERENCES employees(id),
		order_date DATE NOT NULL,
		required_date DATE,
		shipped_date DATE,
		shipper_id INTEGER REFERENCES shippers(id),
		freight REAL DEFAULT 0,
		ship_city TEXT,
		ship_country TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Order Details table
	CREATE TABLE IF NOT EXISTS order_details (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id INTEGER NOT NULL REFERENCES orders(id),
		product_id INTEGER NOT NULL REFERENCES products(id),
		unit_price REAL NOT NULL,
		quantity INTEGER NOT NULL,
		discount REAL DEFAULT 0,
		UNIQUE(order_id, product_id)
	);

	-- Create indexes for better query performance
	CREATE INDEX IF NOT EXISTS idx_orders_customer ON orders(customer_id);
	CREATE INDEX IF NOT EXISTS idx_orders_employee ON orders(employee_id);
	CREATE INDEX IF NOT EXISTS idx_orders_date ON orders(order_date);
	CREATE INDEX IF NOT EXISTS idx_order_details_order ON order_details(order_id);
	CREATE INDEX IF NOT EXISTS idx_order_details_product ON order_details(product_id);
	CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_id);
	CREATE INDEX IF NOT EXISTS idx_products_supplier ON products(supplier_id);
	`

	_, err := db.Exec(schema)
	return err
}

func seedNorthwindData(db *sql.DB) error {
	// Seed categories
	for _, c := range categories {
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, ?)",
			c.name, c.description)
		if err != nil {
			return fmt.Errorf("insert category: %w", err)
		}
	}

	// Seed suppliers
	for _, s := range suppliers {
		_, err := db.Exec("INSERT INTO suppliers (company_name, contact_name, contact_title, city, country, phone) VALUES (?, ?, ?, ?, ?, ?)",
			s.companyName, s.contactName, s.contactTitle, s.city, s.country, s.phone)
		if err != nil {
			return fmt.Errorf("insert supplier: %w", err)
		}
	}

	// Seed products
	for _, p := range products {
		discontinued := 0
		if p.discontinued {
			discontinued = 1
		}
		_, err := db.Exec("INSERT INTO products (name, category_id, supplier_id, unit_price, units_in_stock, discontinued) VALUES (?, ?, ?, ?, ?, ?)",
			p.name, p.categoryIdx+1, p.supplierIdx+1, p.unitPrice, p.unitsStock, discontinued)
		if err != nil {
			return fmt.Errorf("insert product: %w", err)
		}
	}

	// Seed customers
	for _, c := range customers {
		phone := fmt.Sprintf("(%d) %d-%04d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(10000))
		_, err := db.Exec("INSERT INTO customers (company_name, contact_name, contact_title, city, country, region, phone) VALUES (?, ?, ?, ?, ?, ?, ?)",
			c.companyName, c.contactName, c.contactTitle, c.city, c.country, c.region, phone)
		if err != nil {
			return fmt.Errorf("insert customer: %w", err)
		}
	}

	// Seed employees
	for i, e := range employees {
		reportsTo := interface{}(nil)
		if i > 0 {
			// Most employees report to Andrew Fuller (id=2)
			if i != 1 {
				reportsTo = 2
			}
		}
		_, err := db.Exec("INSERT INTO employees (first_name, last_name, title, hire_date, city, country, reports_to) VALUES (?, ?, ?, ?, ?, ?, ?)",
			e.firstName, e.lastName, e.title, e.hireDate, e.city, e.country, reportsTo)
		if err != nil {
			return fmt.Errorf("insert employee: %w", err)
		}
	}

	// Seed shippers
	for _, s := range shippers {
		_, err := db.Exec("INSERT INTO shippers (company_name, phone) VALUES (?, ?)",
			s.companyName, s.phone)
		if err != nil {
			return fmt.Errorf("insert shipper: %w", err)
		}
	}

	// Seed regions
	for _, r := range regions {
		_, err := db.Exec("INSERT INTO regions (description) VALUES (?)", r.description)
		if err != nil {
			return fmt.Errorf("insert region: %w", err)
		}
	}

	// Seed orders and order details (last 2 years of data)
	numCustomers := len(customers)
	numEmployees := len(employees)
	numProducts := len(products)
	numShippers := len(shippers)

	now := time.Now()
	startDate := now.AddDate(-2, 0, 0) // 2 years ago

	// Generate ~2500 orders over 2 years
	for i := 0; i < 2500; i++ {
		// Random date within the range
		daysOffset := rand.Intn(730) // 2 years in days
		orderDate := startDate.AddDate(0, 0, daysOffset)

		customerID := rand.Intn(numCustomers) + 1
		employeeID := rand.Intn(numEmployees) + 1
		shipperID := rand.Intn(numShippers) + 1

		// Required date is 1-2 weeks after order
		requiredDate := orderDate.AddDate(0, 0, rand.Intn(7)+7)

		// 90% of orders are shipped
		var shippedDate interface{}
		if rand.Float32() < 0.90 {
			// Shipped 1-5 days after order
			shippedDate = orderDate.AddDate(0, 0, rand.Intn(5)+1).Format("2006-01-02")
		}

		// Get customer's city and country for shipping
		var shipCity, shipCountry string
		db.QueryRow("SELECT city, country FROM customers WHERE id = ?", customerID).Scan(&shipCity, &shipCountry)

		freight := rand.Float64()*100 + 5 // $5-$105 freight

		res, err := db.Exec(`
			INSERT INTO orders (customer_id, employee_id, order_date, required_date, shipped_date, shipper_id, freight, ship_city, ship_country)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			customerID, employeeID, orderDate.Format("2006-01-02"), requiredDate.Format("2006-01-02"),
			shippedDate, shipperID, freight, shipCity, shipCountry)
		if err != nil {
			return fmt.Errorf("insert order: %w", err)
		}

		orderID, _ := res.LastInsertId()

		// Add 1-5 products per order
		numLineItems := rand.Intn(5) + 1
		usedProducts := make(map[int]bool)

		for j := 0; j < numLineItems; j++ {
			// Pick a random product that hasn't been used in this order
			productID := rand.Intn(numProducts) + 1
			for usedProducts[productID] {
				productID = rand.Intn(numProducts) + 1
			}
			usedProducts[productID] = true

			// Get product price
			var unitPrice float64
			db.QueryRow("SELECT unit_price FROM products WHERE id = ?", productID).Scan(&unitPrice)

			quantity := rand.Intn(50) + 1 // 1-50 units

			// 20% chance of discount
			discount := 0.0
			if rand.Float32() < 0.20 {
				discount = float64(rand.Intn(3)+1) * 0.05 // 5%, 10%, or 15%
			}

			_, err := db.Exec(`
				INSERT INTO order_details (order_id, product_id, unit_price, quantity, discount)
				VALUES (?, ?, ?, ?, ?)`,
				orderID, productID, unitPrice, quantity, discount)
			if err != nil {
				return fmt.Errorf("insert order detail: %w", err)
			}
		}
	}

	return nil
}
