package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection and initializes schema
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Printf("Database initialized at: %s", dbPath)
	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// GetConn returns the underlying sql.DB connection
func (db *DB) GetConn() *sql.DB {
	return db.conn
}

// initSchema creates all necessary tables and indexes
func (db *DB) initSchema() error {
	schema := `
	-- categories table (predefined categories)
	CREATE TABLE IF NOT EXISTS categories (
		name TEXT PRIMARY KEY,
		display_name TEXT NOT NULL
	);

	-- property_meta table (defines available properties per category)
	CREATE TABLE IF NOT EXISTS property_meta (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category_name TEXT NOT NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL CHECK(type IN ('string', 'number', 'money', 'date')),
		FOREIGN KEY (category_name) REFERENCES categories(name) ON DELETE CASCADE,
		UNIQUE(category_name, name)
	);

	-- parsed_items table
	CREATE TABLE IF NOT EXISTS parsed_items (
		id TEXT PRIMARY KEY,
		link TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		pub_date DATETIME NOT NULL,
		description TEXT,
		brand TEXT,
		model TEXT,
		price INTEGER,
		image_url TEXT,
		category_name TEXT NOT NULL,
		processed_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (category_name) REFERENCES categories(name)
	);

	CREATE INDEX IF NOT EXISTS idx_category ON parsed_items(category_name);
	CREATE INDEX IF NOT EXISTS idx_brand ON parsed_items(brand);
	CREATE INDEX IF NOT EXISTS idx_price ON parsed_items(price);
	CREATE INDEX IF NOT EXISTS idx_pub_date ON parsed_items(pub_date);
	CREATE INDEX IF NOT EXISTS idx_created_at ON parsed_items(created_at);

	-- properties table with typed columns
	CREATE TABLE IF NOT EXISTS properties (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id TEXT NOT NULL,
		property_meta_id INTEGER NOT NULL,
		string_value TEXT,
		number_value REAL,
		money_value INTEGER,
		date_value DATETIME,
		FOREIGN KEY (item_id) REFERENCES parsed_items(id) ON DELETE CASCADE,
		FOREIGN KEY (property_meta_id) REFERENCES property_meta(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_item_id ON properties(item_id);
	CREATE INDEX IF NOT EXISTS idx_property_meta_id ON properties(property_meta_id);
	CREATE INDEX IF NOT EXISTS idx_number_value ON properties(number_value);
	CREATE INDEX IF NOT EXISTS idx_money_value ON properties(money_value);
	CREATE INDEX IF NOT EXISTS idx_date_value ON properties(date_value);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Initialize predefined categories
	if err := db.initCategories(); err != nil {
		return fmt.Errorf("failed to initialize categories: %w", err)
	}

	return nil
}

// initCategories initializes predefined categories and their properties
func (db *DB) initCategories() error {
	// Insert categories
	categories := []struct {
		name        string
		displayName string
	}{
		{"monitors", "Monitors"},
		{"notebooks", "Notebooks"},
		{"tvs", "TVs"},
		{"cars", "Cars"},
		{"baby_bicycles", "Baby Bicycles"},
	}

	for _, cat := range categories {
		_, err := db.conn.Exec(
			"INSERT OR IGNORE INTO categories (name, display_name) VALUES (?, ?)",
			cat.name, cat.displayName,
		)
		if err != nil {
			return fmt.Errorf("failed to insert category %s: %w", cat.name, err)
		}
	}

	// Insert property metadata
	properties := []struct {
		categoryName string
		name         string
		propType     string
	}{
		// Monitors
		{"monitors", "Size", "string"},
		{"monitors", "Brand", "string"},
		{"monitors", "Model", "string"},
		// Notebooks
		{"notebooks", "Screen", "string"},
		{"notebooks", "HDD", "number"},
		{"notebooks", "RAM", "number"},
		{"notebooks", "Brand", "string"},
		{"notebooks", "Model", "string"},
		// TVs
		{"tvs", "Diagonal", "string"},
		{"tvs", "Condition", "string"},
		{"tvs", "Brand", "string"},
		{"tvs", "Model", "string"},
		// Cars
		{"cars", "Brand", "string"},
		{"cars", "Model", "string"},
		{"cars", "Year", "number"},
		{"cars", "Engine", "string"},
		{"cars", "Mileage", "number"},
		// Baby Bicycles
		{"baby_bicycles", "Brand", "string"},
		{"baby_bicycles", "Model", "string"},
		{"baby_bicycles", "Year", "number"},
		{"baby_bicycles", "Condition", "string"},
	}

	for _, prop := range properties {
		_, err := db.conn.Exec(
			"INSERT OR IGNORE INTO property_meta (category_name, name, type) VALUES (?, ?, ?)",
			prop.categoryName, prop.name, prop.propType,
		)
		if err != nil {
			return fmt.Errorf("failed to insert property meta %s.%s: %w", prop.categoryName, prop.name, err)
		}
	}

	return nil
}
