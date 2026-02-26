package parser

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"

	"web-rss-parser/internal/domain"
)

// MonitorFileParser handles both RSS and HTML parsing for monitors
type MonitorFileParser struct {
	category domain.Category
}

// NewMonitorFileParser creates a new monitor file parser
func NewMonitorFileParser() *MonitorFileParser {
	category := domain.GetCategoryByName("monitors")
	if category == nil {
		// Fallback if category not found
		category = &domain.Category{
			Name: "monitors",
			Props: []domain.PropertyMeta{
				{ID: 1, Name: "Size", Type: "string"},
				{ID: 2, Name: "Brand", Type: "string"},
				{ID: 3, Name: "Model", Type: "string"},
			},
		}
	}
	return &MonitorFileParser{category: *category}
}

// GetCategory returns the category object
func (p *MonitorFileParser) GetCategory() domain.Category {
	return p.category
}

// ParseFile parses a file (RSS or HTML) and returns parsed items
func (p *MonitorFileParser) ParseFile(ctx context.Context, filePath string) ([]domain.ParsedItem, error) {
	// Decide based on file extension
	if strings.HasSuffix(strings.ToLower(filePath), ".xml") {
		return p.parseRSSFile(filePath)
	} else if strings.HasSuffix(strings.ToLower(filePath), ".html") {
		return p.parseHTMLFile(filePath)
	}
	return nil, fmt.Errorf("unsupported file format: %s", filePath)
}

// parseRSSFile parses RSS XML file
func (p *MonitorFileParser) parseRSSFile(filePath string) ([]domain.ParsedItem, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse XML
	var rssFeed domain.RSSFeed
	if err := xml.Unmarshal(data, &rssFeed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	// Convert RSS items to ParsedItems
	var items []domain.ParsedItem
	for _, rssItem := range rssFeed.Channel.Items {
		item, err := p.convertRSSItemToParsedItem(rssItem)
		if err != nil {
			continue // Skip invalid items
		}
		items = append(items, item)
	}

	return items, nil
}

// parseHTMLFile parses HTML file
func (p *MonitorFileParser) parseHTMLFile(filePath string) ([]domain.ParsedItem, error) {
	// TODO: Implement HTML parsing for monitors if needed
	return nil, fmt.Errorf("HTML parsing not yet implemented for monitors")
}

// convertRSSItemToParsedItem converts an RSS item to a ParsedItem
func (p *MonitorFileParser) convertRSSItemToParsedItem(rssItem domain.RSSItem) (domain.ParsedItem, error) {
	// Extract ID from link
	id := extractIDFromLink(rssItem.Link)
	if id == "" {
		return domain.ParsedItem{}, fmt.Errorf("failed to extract ID from link: %s", rssItem.Link)
	}

	// Parse pub date
	pubDate, err := parsePubDate(rssItem.PubDate)
	if err != nil {
		pubDate = time.Now()
	}

	// Parse description to extract structured data
	brand := extractField(rssItem.Description, "Brand")
	model := extractField(rssItem.Description, "Model")
	price := extractPrice(rssItem.Description)
	imageURL := extractImageURL(rssItem.Description)
	size := extractField(rssItem.Description, "Size")

	// Skip items without price
	if price <= 0 {
		return domain.ParsedItem{}, fmt.Errorf("item has no valid price")
	}

	// Create properties
	var properties []domain.Property
	if size != "" {
		meta := domain.PropertyMeta{Name: "Size", Type: "string"}
		properties = append(properties, createStringProperty(meta, size))
	}
	if brand != "" {
		meta := domain.PropertyMeta{Name: "Brand", Type: "string"}
		properties = append(properties, createStringProperty(meta, brand))
	}
	if model != "" {
		meta := domain.PropertyMeta{Name: "Model", Type: "string"}
		properties = append(properties, createStringProperty(meta, model))
	}

	// Create ParsedItem
	item := domain.ParsedItem{
		ID:          id,
		Link:        rssItem.Link,
		Title:       strings.TrimSpace(rssItem.Title),
		PubDate:     pubDate,
		Description: rssItem.Description,
		Brand:       brand,
		Model:       model,
		Price:       price,
		ImageURL:    imageURL,
		Category:    p.category,
		Properties:  properties,
		ProcessedAt: time.Now(),
	}

	return item, nil
}
