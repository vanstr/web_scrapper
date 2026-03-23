package database

import (
	"context"
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
	var existingID string
	err := r.db.conn.QueryRow("SELECT id FROM parsed_items WHERE link = ?", item.Link).Scan(&existingID)
	if err == sql.ErrNoRows {
		exists = false
	} else if err != nil {
		return false, fmt.Errorf("failed to check if item exists: %w", err)
	} else {
		exists = true
	}

	// Begin transaction
	tx, err := r.db.conn.Begin()
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	itemID := item.ID
	if exists {
		itemID = existingID

		// Update parsed item fields and refresh properties
		_, err = tx.Exec(`
			UPDATE parsed_items
			SET title = ?, pub_date = ?, description = ?, brand = ?, model = ?, price = ?,
			    image_url = ?, category_name = ?, processed_at = ?
			WHERE link = ?
		`,
			item.Title, item.PubDate, item.Description, item.Brand, item.Model, item.Price,
			item.ImageURL, item.Category.Name, item.ProcessedAt, item.Link,
		)
		if err != nil {
			return false, fmt.Errorf("failed to update parsed item: %w", err)
		}

		_, err = tx.Exec(`DELETE FROM properties WHERE item_id = ?`, itemID)
		if err != nil {
			return false, fmt.Errorf("failed to delete properties: %w", err)
		}
	} else {
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
			itemID, propMetaID, prop.StringValue, prop.NumberValue, prop.MoneyValue, prop.DateValue,
		)
		if err != nil {
			return false, fmt.Errorf("failed to insert property: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return !exists, nil
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

// ListItems returns items from the database with optional category filter
func (r *Repository) ListItems(ctx context.Context, category string, limit int) ([]domain.ParsedItem, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = `
			SELECT id, link, title, pub_date, description, brand, model, price, 
			       image_url, category_name, processed_at, created_at
			FROM parsed_items
			WHERE category_name = ?
			ORDER BY pub_date DESC
			LIMIT ?
		`
		args = []interface{}{category, limit}
	} else {
		query = `
			SELECT id, link, title, pub_date, description, brand, model, price, 
			       image_url, category_name, processed_at, created_at
			FROM parsed_items
			ORDER BY pub_date DESC
			LIMIT ?
		`
		args = []interface{}{limit}
	}

	rows, err := r.db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	var items []domain.ParsedItem
	for rows.Next() {
		var item domain.ParsedItem
		var categoryName string
		var pubDateStr, processedAtStr, createdAtStr string

		err := rows.Scan(
			&item.ID, &item.Link, &item.Title, &pubDateStr, &item.Description,
			&item.Brand, &item.Model, &item.Price, &item.ImageURL, &categoryName,
			&processedAtStr, &createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		// Parse dates from SQLite string format
		item.PubDate = parseDateTime(pubDateStr)
		item.ProcessedAt = parseDateTime(processedAtStr)
		item.CreatedAt = parseDateTime(createdAtStr)

		cat := domain.GetCategoryByName(categoryName)
		if cat != nil {
			item.Category = *cat
		}

		items = append(items, item)
	}

	return items, nil
}

// parseDateTime parses SQLite datetime string to time.Time
func parseDateTime(s string) time.Time {
	layouts := []string{
		"2006-01-02 15:04:05 -0700 -0700", // Go time.Time with duplicate zone
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02T15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// GetStatistics returns item counts by category
func (r *Repository) GetStatistics(ctx context.Context) (map[string]int, error) {
	return r.CountItemsByCategory()
}
