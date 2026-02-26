package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"web-rss-parser/internal/domain"
)

// Repository provides database operations for parsed items
type Repository struct {
	db *DB
}

// NewRepository creates a new repository
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// SaveItem saves a parsed item to the database
// Returns true if item was newly inserted, false if it already existed
func (r *Repository) SaveItem(item *domain.ParsedItem) (bool, error) {
	// Check if item already exists
	var exists bool
	err := r.db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM parsed_items WHERE link = ?)", item.Link).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if item exists: %w", err)
	}

	if exists {
		return false, nil // Item already exists, skip
	}

	// Begin transaction
	tx, err := r.db.conn.Begin()
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert parsed item
	_, err = tx.Exec(`
		INSERT INTO parsed_items (
			id, link, title, pub_date, description, brand, model, price, 
			image_url, category_name, processed_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID, item.Link, item.Title, item.PubDate, item.Description,
		item.Brand, item.Model, item.Price, item.ImageURL, item.Category.Name,
		item.ProcessedAt, time.Now(),
	)
	if err != nil {
		return false, fmt.Errorf("failed to insert parsed item: %w", err)
	}

	// Insert properties
	for _, prop := range item.Properties {
		// Get property_meta_id
		var propMetaID int
		err := tx.QueryRow(
			"SELECT id FROM property_meta WHERE category_name = ? AND name = ?",
			item.Category.Name, prop.PropertyMeta.Name,
		).Scan(&propMetaID)
		if err != nil {
			log.Printf("Warning: property_meta not found for %s.%s, skipping", item.Category.Name, prop.PropertyMeta.Name)
			continue
		}

		// Insert property with appropriate typed value
		_, err = tx.Exec(`
			INSERT INTO properties (
				item_id, property_meta_id, string_value, number_value, money_value, date_value
			) VALUES (?, ?, ?, ?, ?, ?)
		`,
			item.ID, propMetaID, prop.StringValue, prop.NumberValue, prop.MoneyValue, prop.DateValue,
		)
		if err != nil {
			return false, fmt.Errorf("failed to insert property: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return true, nil
}

// GetItemByID retrieves an item by its ID
func (r *Repository) GetItemByID(id string) (*domain.ParsedItem, error) {
	var item domain.ParsedItem
	var categoryName string

	err := r.db.conn.QueryRow(`
		SELECT id, link, title, pub_date, description, brand, model, price, 
		       image_url, category_name, processed_at, created_at
		FROM parsed_items
		WHERE id = ?
	`, id).Scan(
		&item.ID, &item.Link, &item.Title, &item.PubDate, &item.Description,
		&item.Brand, &item.Model, &item.Price, &item.ImageURL, &categoryName,
		&item.ProcessedAt, &item.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	// Get category
	category := domain.GetCategoryByName(categoryName)
	if category != nil {
		item.Category = *category
	}

	// Load properties
	rows, err := r.db.conn.Query(`
		SELECT p.id, pm.name, pm.type, p.string_value, p.number_value, p.money_value, p.date_value
		FROM properties p
		JOIN property_meta pm ON p.property_meta_id = pm.id
		WHERE p.item_id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get properties: %w", err)
	}
	defer rows.Close()

	var properties []domain.Property
	for rows.Next() {
		var prop domain.Property
		var propName, propType string
		err := rows.Scan(
			&prop.ID, &propName, &propType,
			&prop.StringValue, &prop.NumberValue, &prop.MoneyValue, &prop.DateValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan property: %w", err)
		}
		prop.PropertyMeta = domain.PropertyMeta{Name: propName, Type: propType}
		properties = append(properties, prop)
	}
	item.Properties = properties

	return &item, nil
}

// CountItemsByCategory returns the count of items per category
func (r *Repository) CountItemsByCategory() (map[string]int, error) {
	rows, err := r.db.conn.Query(`
		SELECT category_name, COUNT(*) as count
		FROM parsed_items
		GROUP BY category_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to count items: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[category] = count
	}

	return counts, nil
}
