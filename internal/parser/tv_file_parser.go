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

// TVFileParser handles both RSS and HTML parsing for TVs
type TVFileParser struct {
	category domain.Category
}

// NewTVFileParser creates a new TV file parser
func NewTVFileParser() *TVFileParser {
	category := domain.GetCategoryByName("tvs")
	if category == nil {
		// Fallback if category not found
		category = &domain.Category{
			Name: "tvs",
			Props: []domain.PropertyMeta{
				{ID: 9, Name: "Diagonal", Type: "string"},
				{ID: 10, Name: "Condition", Type: "string"},
				{ID: 11, Name: "Brand", Type: "string"},
				{ID: 12, Name: "Model", Type: "string"},
			},
		}
	}
	return &TVFileParser{category: *category}
}

// GetCategory returns the category object
func (p *TVFileParser) GetCategory() domain.Category {
	return p.category
}

// ParseFile parses a file (RSS or HTML) and returns parsed items
func (p *TVFileParser) ParseFile(ctx context.Context, filePath string) ([]domain.ParsedItem, error) {
	if strings.HasSuffix(strings.ToLower(filePath), ".xml") {
		return p.parseRSSFile(filePath)
	} else if strings.HasSuffix(strings.ToLower(filePath), ".html") {
		return p.parseHTMLFile(filePath)
	}
	return nil, fmt.Errorf("unsupported file format: %s", filePath)
}

// parseRSSFile parses RSS XML file
func (p *TVFileParser) parseRSSFile(filePath string) ([]domain.ParsedItem, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var rssFeed domain.RSSFeed
	if err := xml.Unmarshal(data, &rssFeed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	var items []domain.ParsedItem
	for _, rssItem := range rssFeed.Channel.Items {
		item, err := p.convertRSSItemToParsedItem(rssItem)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// parseHTMLFile parses HTML file
func (p *TVFileParser) parseHTMLFile(filePath string) ([]domain.ParsedItem, error) {
	return nil, fmt.Errorf("HTML parsing not yet implemented for TVs")
}

// convertRSSItemToParsedItem converts an RSS item to a ParsedItem
func (p *TVFileParser) convertRSSItemToParsedItem(rssItem domain.RSSItem) (domain.ParsedItem, error) {
	id := extractIDFromLink(rssItem.Link)
	if id == "" {
		return domain.ParsedItem{}, fmt.Errorf("failed to extract ID from link: %s", rssItem.Link)
	}

	pubDate, err := parsePubDate(rssItem.PubDate)
	if err != nil {
		pubDate = time.Now()
	}

	// Extract fields from description
	brand := extractField(rssItem.Description, "Brand")
	model := extractField(rssItem.Description, "Model")
	price := extractPrice(rssItem.Description)
	imageURL := extractImageURL(rssItem.Description)
	diagonal := extractField(rssItem.Description, "Diagonal")
	condition := extractField(rssItem.Description, "Cond\\.")

	// Skip items without price
	if price <= 0 {
		return domain.ParsedItem{}, fmt.Errorf("item has no valid price")
	}

	// Create properties
	var properties []domain.Property
	if diagonal != "" {
		meta := domain.PropertyMeta{Name: "Diagonal", Type: "string"}
		properties = append(properties, createStringProperty(meta, diagonal))
	}
	if condition != "" {
		meta := domain.PropertyMeta{Name: "Condition", Type: "string"}
		properties = append(properties, createStringProperty(meta, condition))
	}
	if brand != "" {
		meta := domain.PropertyMeta{Name: "Brand", Type: "string"}
		properties = append(properties, createStringProperty(meta, brand))
	}
	if model != "" {
		meta := domain.PropertyMeta{Name: "Model", Type: "string"}
		properties = append(properties, createStringProperty(meta, model))
	}

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
