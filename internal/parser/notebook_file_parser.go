package parser

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"web-rss-parser/internal/domain"
)

// NotebookFileParser handles both RSS and HTML parsing for notebooks
type NotebookFileParser struct {
	category domain.Category
}

// NewNotebookFileParser creates a new notebook file parser
func NewNotebookFileParser() *NotebookFileParser {
	category := domain.GetCategoryByName("notebooks")
	if category == nil {
		// Fallback if category not found
		category = &domain.Category{
			Name: "notebooks",
			Props: []domain.PropertyMeta{
				{ID: 4, Name: "Screen", Type: "string"},
				{ID: 5, Name: "HDD", Type: "number"},
				{ID: 6, Name: "RAM", Type: "number"},
				{ID: 7, Name: "Brand", Type: "string"},
				{ID: 8, Name: "Model", Type: "string"},
			},
		}
	}
	return &NotebookFileParser{category: *category}
}

// GetCategory returns the category object
func (p *NotebookFileParser) GetCategory() domain.Category {
	return p.category
}

// ParseFile parses a file (RSS or HTML) and returns parsed items
func (p *NotebookFileParser) ParseFile(ctx context.Context, filePath string) ([]domain.ParsedItem, error) {
	if strings.HasSuffix(strings.ToLower(filePath), ".xml") {
		return p.parseRSSFile(filePath)
	} else if strings.HasSuffix(strings.ToLower(filePath), ".html") {
		return p.parseHTMLFile(filePath)
	}
	return nil, fmt.Errorf("unsupported file format: %s", filePath)
}

// parseRSSFile parses RSS XML file
func (p *NotebookFileParser) parseRSSFile(filePath string) ([]domain.ParsedItem, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.ParseRSSData(context.Background(), data)
}

// ParseRSSData parses RSS XML data from bytes (for remote fetching)
func (p *NotebookFileParser) ParseRSSData(ctx context.Context, data []byte) ([]domain.ParsedItem, error) {
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
func (p *NotebookFileParser) parseHTMLFile(filePath string) ([]domain.ParsedItem, error) {
	return nil, fmt.Errorf("HTML parsing not yet implemented for notebooks")
}

// convertRSSItemToParsedItem converts an RSS item to a ParsedItem
func (p *NotebookFileParser) convertRSSItemToParsedItem(rssItem domain.RSSItem) (domain.ParsedItem, error) {
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
	screen := extractField(rssItem.Description, "Screen")
	hddStr := extractField(rssItem.Description, "HDD")
	ramStr := extractField(rssItem.Description, "RAM")


	// Skip items without price
	if price <= 0 {
		return domain.ParsedItem{}, fmt.Errorf("item has no valid price")
	}

	// Create properties
	var properties []domain.Property
	if screen != "" {
		meta := domain.PropertyMeta{Name: "Screen", Type: "string"}
		properties = append(properties, createStringProperty(meta, screen))
	}
	if hddStr != "" {
		if hdd, err := strconv.ParseFloat(hddStr, 64); err == nil {
			meta := domain.PropertyMeta{Name: "HDD", Type: "number"}
			properties = append(properties, createNumberProperty(meta, hdd))
		}
	}
	if ramStr != "" {
		if ram, err := strconv.ParseFloat(ramStr, 64); err == nil {
			meta := domain.PropertyMeta{Name: "RAM", Type: "number"}
			properties = append(properties, createNumberProperty(meta, ram))
		}
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
