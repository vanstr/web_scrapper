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

	"golang.org/x/net/html"
)

// CarFileParser handles both RSS and HTML parsing for cars
type CarFileParser struct {
	category domain.Category
}

// NewCarFileParser creates a new car file parser
func NewCarFileParser() *CarFileParser {
	category := domain.GetCategoryByName("cars")
	if category == nil {
		// Fallback if category not found
		category = &domain.Category{
			Name: "cars",
			Props: []domain.PropertyMeta{
				{ID: 13, Name: "Brand", Type: "string"},
				{ID: 14, Name: "Model", Type: "string"},
				{ID: 15, Name: "Year", Type: "number"},
				{ID: 16, Name: "Engine", Type: "string"},
				{ID: 17, Name: "Mileage", Type: "number"},
			},
		}
	}
	return &CarFileParser{category: *category}
}

// GetCategory returns the category object
func (p *CarFileParser) GetCategory() domain.Category {
	return p.category
}

// ParseFile parses a file (RSS or HTML) and returns parsed items
func (p *CarFileParser) ParseFile(ctx context.Context, filePath string) ([]domain.ParsedItem, error) {
	if strings.HasSuffix(strings.ToLower(filePath), ".xml") {
		return p.parseRSSFile(filePath)
	} else if strings.HasSuffix(strings.ToLower(filePath), ".html") {
		return p.parseHTMLFile(filePath)
	}
	return nil, fmt.Errorf("unsupported file format: %s", filePath)
}

// parseRSSFile parses RSS XML file
func (p *CarFileParser) parseRSSFile(filePath string) ([]domain.ParsedItem, error) {
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

// parseHTMLFile parses HTML file with table structure
func (p *CarFileParser) parseHTMLFile(filePath string) ([]domain.ParsedItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	doc, err := html.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var items []domain.ParsedItem
	p.extractCarListings(doc, &items)

	return items, nil
}

// convertRSSItemToParsedItem converts an RSS item to a ParsedItem
func (p *CarFileParser) convertRSSItemToParsedItem(rssItem domain.RSSItem) (domain.ParsedItem, error) {
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
	yearStr := extractField(rssItem.Description, "Year")
	engine := extractField(rssItem.Description, "Engine")
	mileageStr := extractField(rssItem.Description, "Mileage")

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
	if engine != "" {
		meta := domain.PropertyMeta{Name: "Engine", Type: "string"}
		properties = append(properties, createStringProperty(meta, engine))
	}
	if mileageStr != "" {
		if mileage, err := strconv.ParseFloat(mileageStr, 64); err == nil {
			meta := domain.PropertyMeta{Name: "Mileage", Type: "number"}
			properties = append(properties, createNumberProperty(meta, mileage))
		}
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

// extractCarListings recursively searches for car listing rows in HTML
func (p *CarFileParser) extractCarListings(n *html.Node, items *[]domain.ParsedItem) {
	if n.Type == html.ElementNode && n.Data == "tr" {
		id := getAttr(n, "id")
		if strings.HasPrefix(id, "tr_") && id != "tr_head_line" && id != "head_line" {
			item := p.parseCarRow(n)
			if item != nil {
				*items = append(*items, *item)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		p.extractCarListings(c, items)
	}
}

// parseCarRow parses a single car listing row from HTML table
func (p *CarFileParser) parseCarRow(n *html.Node) *domain.ParsedItem {
	var (
		link     string
		title    string
		imageURL string
		brand    string
		model    string
		year     int
		engine   string
		mileage  int
		price    int
	)

	cellIndex := 0
	for td := n.FirstChild; td != nil; td = td.NextSibling {
		if td.Type != html.ElementNode || td.Data != "td" {
			continue
		}

		switch cellIndex {
		case 1:
			imageURL = extractImageFromCell(td)
		case 2:
			link, title = extractLinkAndTitle(td)
		case 3:
			brand, model = extractBrandModel(td)
		case 4:
			yearStr := extractText(td)
			year, _ = strconv.Atoi(strings.TrimSpace(yearStr))
		case 5:
			engine = strings.TrimSpace(extractText(td))
		case 6:
			mileageStr := extractText(td)
			mileageStr = strings.ReplaceAll(mileageStr, ",", "")
			mileageStr = strings.TrimSpace(mileageStr)
			if mileageStr != "-" && mileageStr != "" {
				mileage, _ = strconv.Atoi(mileageStr)
			}
		case 7:
			priceStr := extractText(td)
			price = parsePrice(priceStr)
		}
		cellIndex++
	}

	if link == "" {
		return nil
	}

	id := extractIDFromLink(link)
	if id == "" {
		return nil
	}

	// Skip items without price
	if price <= 0 {
		return nil
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
	if year > 0 {
		yearFloat := float64(year)
		meta := domain.PropertyMeta{Name: "Year", Type: "number"}
		properties = append(properties, createNumberProperty(meta, yearFloat))
	}
	if engine != "" {
		meta := domain.PropertyMeta{Name: "Engine", Type: "string"}
		properties = append(properties, createStringProperty(meta, engine))
	}
	if mileage > 0 {
		mileageFloat := float64(mileage)
		meta := domain.PropertyMeta{Name: "Mileage", Type: "number"}
		properties = append(properties, createNumberProperty(meta, mileageFloat))
	}

	item := domain.ParsedItem{
		ID:          id,
		Link:        link,
		Title:       title,
		PubDate:     time.Now(),
		Description: fmt.Sprintf("Brand: %s, Model: %s, Year: %d, Engine: %s, Mileage: %d km", brand, model, year, engine, mileage),
		Brand:       brand,
		Model:       model,
		Price:       price,
		ImageURL:    imageURL,
		Category:    p.category,
		Properties:  properties,
		ProcessedAt: time.Now(),
	}

	return &item
}

// HTML helper functions

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func extractImageFromCell(td *html.Node) string {
	for c := td.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "a" {
			for img := c.FirstChild; img != nil; img = img.NextSibling {
				if img.Type == html.ElementNode && img.Data == "img" {
					return getAttr(img, "src")
				}
			}
		}
	}
	return ""
}

func extractLinkAndTitle(td *html.Node) (string, string) {
	for c := td.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" {
			for a := c.FirstChild; a != nil; a = a.NextSibling {
				if a.Type == html.ElementNode && a.Data == "a" {
					link := getAttr(a, "href")
					title := extractText(a)
					if !strings.HasPrefix(link, "http") {
						link = "https://www.ss.lv" + link
					}
					return link, title
				}
			}
		}
	}
	return "", ""
}

func extractBrandModel(td *html.Node) (string, string) {
	text := extractText(td)
	lines := strings.Split(text, "\n")

	var brand, model string
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i == 0 {
			brand = line
		} else if i == 1 {
			model = line
			break
		}
	}
	return brand, model
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += extractText(c)
	}
	return text
}

func parsePrice(priceStr string) int {
	priceStr = strings.TrimSpace(priceStr)
	priceStr = strings.ReplaceAll(priceStr, ",", "")
	priceStr = strings.ReplaceAll(priceStr, "€", "")
	priceStr = strings.TrimSpace(priceStr)

	price, _ := strconv.Atoi(priceStr)
	return price
}
