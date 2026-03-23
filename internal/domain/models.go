package domain

import "time"

// Category represents a product category (monitors, notebooks, tvs, etc.)
type Category struct {
	Name  string         `json:"name"`  // e.g., "monitors", "notebooks", "tvs"
	Props []PropertyMeta `json:"props"` // supported properties for this category
}

// PropertyMeta defines metadata for a property
type PropertyMeta struct {
	ID   int    `json:"id"`   // unique identifier
	Name string `json:"name"` // e.g., "Brand", "Size", "Screen", "HDD", "RAM", "Diagonal"
	Type string `json:"type"` // "string", "number", "money", "date"
}

// Property represents an actual property value with typed storage
type Property struct {
	ID           int          `json:"id"`
	PropertyMeta PropertyMeta `json:"property_meta"`
	StringValue  *string      `json:"string_value,omitempty"` // For string type
	NumberValue  *float64     `json:"number_value,omitempty"` // For number type
	MoneyValue   *int         `json:"money_value,omitempty"`  // For money type (cents/minor units)
	DateValue    *time.Time   `json:"date_value,omitempty"`   // For date type
}

// ParsedItem represents a parsed RSS item stored in the database
type ParsedItem struct {
	ID          string     `json:"id"`           // Unique ID (extracted from URL, e.g., "okebe")
	Link        string     `json:"link"`         // Full URL to the advertisement
	Title       string     `json:"title"`        // Item title
	PubDate     time.Time  `json:"pub_date"`     // Publication date
	Description string     `json:"description"`  // Full HTML description
	Brand       string     `json:"brand"`        // Extracted brand (e.g., "Dell", "Samsung", "HP")
	Model       string     `json:"model"`        // Extracted model (e.g., "P2425H")
	Price       int        `json:"price"`        // Extracted price in euros (whole number)
	ImageURL    string     `json:"image_url"`    // Extracted image URL
	Category    Category   `json:"category"`     // Category object with metadata
	Properties  []Property `json:"properties"`   // Additional extracted properties
	ProcessedAt time.Time  `json:"processed_at"` // When the item was processed
	CreatedAt   time.Time  `json:"created_at"`   // When record was created in DB
}

// PredefinedCategories contains all available categories with their property definitions
var PredefinedCategories = []Category{
	{
		Name: "monitors",
		Props: []PropertyMeta{
			{ID: 1, Name: "Size", Type: "string"},
			{ID: 2, Name: "Brand", Type: "string"},
			{ID: 3, Name: "Model", Type: "string"},
		},
	},
	{
		Name: "notebooks",
		Props: []PropertyMeta{
			{ID: 4, Name: "Screen", Type: "string"},
			{ID: 5, Name: "HDD", Type: "number"},
			{ID: 6, Name: "RAM", Type: "number"},
			{ID: 7, Name: "Brand", Type: "string"},
			{ID: 8, Name: "Model", Type: "string"},
		},
	},
	{
		Name: "tvs",
		Props: []PropertyMeta{
			{ID: 9, Name: "Diagonal", Type: "string"},
			{ID: 10, Name: "Condition", Type: "string"},
			{ID: 11, Name: "Brand", Type: "string"},
			{ID: 12, Name: "Model", Type: "string"},
		},
	},
	{
		Name: "cars",
		Props: []PropertyMeta{
			{ID: 13, Name: "Brand", Type: "string"},
			{ID: 14, Name: "Model", Type: "string"},
			{ID: 15, Name: "Year", Type: "number"},
			{ID: 16, Name: "Engine", Type: "string"},
			{ID: 17, Name: "Mileage", Type: "number"},
		},
	},
	{
		Name: "baby_bicycles",
		Props: []PropertyMeta{
			{ID: 18, Name: "Brand", Type: "string"},
			{ID: 19, Name: "Model", Type: "string"},
			{ID: 20, Name: "Year", Type: "number"},
			{ID: 21, Name: "Condition", Type: "string"},
		},
	},
}

// GetCategoryByName returns a category by name or nil if not found
func GetCategoryByName(name string) *Category {
	for _, cat := range PredefinedCategories {
		if cat.Name == name {
			return &cat
		}
	}
	return nil
}
