package parser

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"web-rss-parser/internal/domain"
)

// BabyBicycleFileParser handles both RSS and HTML parsing for baby bicycles
type BabyBicycleFileParser struct {
	category domain.Category
}

// NewBabyBicycleFileParser creates a new baby bicycle file parser
func NewBabyBicycleFileParser() *BabyBicycleFileParser {
	category := domain.GetCategoryByName("baby_bicycles")
	if category == nil {
		// Fallback if category not found
		category = &domain.Category{
			Name: "baby_bicycles",
			Props: []domain.PropertyMeta{
				{ID: 18, Name: "Brand", Type: "string"},
				{ID: 19, Name: "Model", Type: "string"},
				{ID: 20, Name: "Year", Type: "number"},
				{ID: 21, Name: "Condition", Type: "string"},
			},
		}
	}
	return &BabyBicycleFileParser{category: *category}
}

// GetCategory returns the category object
func (p *BabyBicycleFileParser) GetCategory() domain.Category {
	return p.category
}

// ParseFile parses a file (RSS or HTML) and returns parsed items
func (p *BabyBicycleFileParser) ParseFile(ctx context.Context, filePath string) ([]domain.ParsedItem, error) {
	if strings.HasSuffix(strings.ToLower(filePath), ".xml") {
		return p.parseRSSFile(filePath)
	} else if strings.HasSuffix(strings.ToLower(filePath), ".html") {
		return p.parseHTMLFile(filePath)
	}
	return nil, fmt.Errorf("unsupported file format: %s", filePath)
}

// ParseRSSData parses RSS XML data from bytes (for remote fetching)
func (p *BabyBicycleFileParser) ParseRSSData(ctx context.Context, data []byte) ([]domain.ParsedItem, error) {
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

// parseRSSFile parses RSS XML file
func (p *BabyBicycleFileParser) parseRSSFile(filePath string) ([]domain.ParsedItem, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.ParseRSSData(context.Background(), data)
}

// parseHTMLFile parses HTML file
func (p *BabyBicycleFileParser) parseHTMLFile(filePath string) ([]domain.ParsedItem, error) {
	return nil, fmt.Errorf("HTML parsing not yet implemented for baby bicycles")
}

// convertRSSItemToParsedItem converts an RSS item to a ParsedItem
func (p *BabyBicycleFileParser) convertRSSItemToParsedItem(rssItem domain.RSSItem) (domain.ParsedItem, error) {
	id := extractIDFromLink(rssItem.Link)
	if id == "" {
		return domain.ParsedItem{}, fmt.Errorf("failed to extract ID from link: %s", rssItem.Link)
	}

	pubDate, err := parsePubDate(rssItem.PubDate)
	if err != nil {
		pubDate = time.Now()
	}

	// Extract fields from description (ss.lv Russian format)
	brand := p.extractSSField(rssItem.Description, "Марка")
	model := p.extractSSField(rssItem.Description, "Модель")
	yearStr := p.extractSSField(rssItem.Description, "Год")
	condition := p.extractSSField(rssItem.Description, "Сост.")
	price := p.extractSSPrice(rssItem.Description)
	imageURL := extractImageURL(rssItem.Description)

	// Skip items without price
	if price <= 0 {
		return domain.ParsedItem{}, fmt.Errorf("item has no valid price")
	}

	// Create properties
	var properties []domain.Property
	if brand != "" {
		meta := domain.PropertyMeta{Name: "Brand", Type: "string"}
		properties = append(properties, createStringProperty(meta, brand))
	}
	if model != "" {
		meta := domain.PropertyMeta{Name: "Model", Type: "string"}
		properties = append(properties, createStringProperty(meta, model))
	}
	if yearStr != "" {
		if year, err := strconv.ParseFloat(yearStr, 64); err == nil {
			meta := domain.PropertyMeta{Name: "Year", Type: "number"}
			properties = append(properties, createNumberProperty(meta, year))
		}
	}
	if condition != "" {
		meta := domain.PropertyMeta{Name: "Condition", Type: "string"}
		properties = append(properties, createStringProperty(meta, condition))
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

// extractSSField extracts a field value from ss.lv HTML format
// Pattern: FieldName: <b>Value</b> or FieldName: <b><b>Value</b></b>
func (p *BabyBicycleFileParser) extractSSField(html, fieldName string) string {
	// Pattern for ss.lv: Марка: <b>Brand<br>Model</b> or <b><b>Brand<br>Model</b></b>
	pattern := fieldName + `:\s*<b>(?:<b>)?([^<]+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		// Clean up nested tags and line breaks
		value = strings.ReplaceAll(value, "<br>", "")
		value = strings.ReplaceAll(value, "</b>", "")
		return strings.TrimSpace(value)
	}
	return ""
}

// extractSSPrice extracts price from ss.lv HTML format
// Pattern: Цена: <b>15 €</b> or Цена: <b><b>15</b> €</b>
func (p *BabyBicycleFileParser) extractSSPrice(html string) int {
	// Try pattern with nested bold
	re := regexp.MustCompile(`Цена:\s*<b>(?:<b>)?(\d+)`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		price, err := strconv.Atoi(matches[1])
		if err == nil {
			return price
		}
	}
	return 0
}
