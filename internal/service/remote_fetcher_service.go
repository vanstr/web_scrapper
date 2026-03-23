package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"web-rss-parser/internal/domain"
	"web-rss-parser/internal/parser"
)

// RSSParser interface for parsers that can parse RSS data
type RSSParser interface {
	ParseRSSData(ctx context.Context, data []byte) ([]domain.ParsedItem, error)
	GetCategory() domain.Category
}

// FeedConfig represents a remote feed configuration
type FeedConfig struct {
	URL      string `yaml:"url"`
	Category string `yaml:"category"`
	Type     string `yaml:"type"` // "rss" or "html"
}

// RemoteFetcherService fetches and parses feeds from remote URLs
type RemoteFetcherService struct {
	httpClient *http.Client
	parsers    map[string]RSSParser
}

// NewRemoteFetcherService creates a new remote fetcher service
func NewRemoteFetcherService() *RemoteFetcherService {
	return &RemoteFetcherService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		parsers: map[string]RSSParser{
			"monitors":       parser.NewMonitorFileParser(),
			"notebooks":      parser.NewNotebookFileParser(),
			"baby_bicycles":  parser.NewBabyBicycleFileParser(),
		},
	}
}

// FetchFromURL fetches and parses a single RSS feed from URL
func (s *RemoteFetcherService) FetchFromURL(ctx context.Context, url string, category string) ([]domain.ParsedItem, error) {
	log.Printf("Fetching RSS from URL: %s (category: %s)", url, category)

	// Get parser for this category
	p, ok := s.parsers[category]
	if !ok {
		return nil, fmt.Errorf("no parser found for category: %s", category)
	}

	// Fetch RSS data
	data, err := s.fetchURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}

	// Parse RSS data
	items, err := p.ParseRSSData(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS data: %w", err)
	}

	log.Printf("Parsed %d items from %s", len(items), url)
	return items, nil
}

// FetchFromFeeds fetches and parses multiple feeds
func (s *RemoteFetcherService) FetchFromFeeds(ctx context.Context, feeds []FeedConfig) ([]domain.ParsedItem, error) {
	var allItems []domain.ParsedItem

	for _, feed := range feeds {
		if feed.Type != "rss" {
			log.Printf("Skipping non-RSS feed: %s (type: %s)", feed.URL, feed.Type)
			continue
		}

		items, err := s.FetchFromURL(ctx, feed.URL, feed.Category)
		if err != nil {
			log.Printf("Error fetching feed %s: %v", feed.URL, err)
			continue
		}
		allItems = append(allItems, items...)
	}

	log.Printf("Total items fetched from remote feeds: %d", len(allItems))
	return allItems, nil
}

// fetchURL fetches data from a URL
func (s *RemoteFetcherService) fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WebScraper/1.0)")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// DetectCategoryFromURL tries to detect category from URL patterns
func (s *RemoteFetcherService) DetectCategoryFromURL(url string) string {
	lowerURL := strings.ToLower(url)

	patterns := map[string]string{
		"monitors":             "monitors",
		"noutbooks":            "notebooks",
		"notebooks":            "notebooks",
		"laptop":               "notebooks",
		"childrens-for-tots":   "baby_bicycles",
		"bycycles":             "baby_bicycles",
	}

	for pattern, category := range patterns {
		if strings.Contains(lowerURL, pattern) {
			return category
		}
	}

	return ""
}

// Add ParseRSSData to existing parsers by updating the interface check
// We need to verify which parsers implement RSSParser

// GenericRSSParser wraps any FileParser to provide RSS parsing from bytes
type GenericRSSParser struct {
	category domain.Category
}

// ParseRSSData parses RSS data using generic XML parsing
func (p *GenericRSSParser) ParseRSSData(ctx context.Context, data []byte) ([]domain.ParsedItem, error) {
	var rssFeed domain.RSSFeed
	if err := xml.Unmarshal(data, &rssFeed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	var items []domain.ParsedItem
	for _, rssItem := range rssFeed.Channel.Items {
		item, err := convertGenericRSSItem(rssItem, p.category)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

func (p *GenericRSSParser) GetCategory() domain.Category {
	return p.category
}

// convertGenericRSSItem converts an RSS item using ss.lv format
func convertGenericRSSItem(rssItem domain.RSSItem, category domain.Category) (domain.ParsedItem, error) {
	id := extractIDFromLink(rssItem.Link)
	if id == "" {
		return domain.ParsedItem{}, fmt.Errorf("failed to extract ID from link: %s", rssItem.Link)
	}

	pubDate, err := parsePubDateStr(rssItem.PubDate)
	if err != nil {
		pubDate = time.Now()
	}

	// Extract common fields (ss.lv Russian format)
	brand := extractSSField(rssItem.Description, "Марка")
	model := extractSSField(rssItem.Description, "Модель")
	price := extractSSPrice(rssItem.Description)
	imageURL := extractImageURLFromHTML(rssItem.Description)

	if price <= 0 {
		return domain.ParsedItem{}, fmt.Errorf("item has no valid price")
	}

	return domain.ParsedItem{
		ID:          id,
		Link:        rssItem.Link,
		Title:       strings.TrimSpace(rssItem.Title),
		PubDate:     pubDate,
		Description: rssItem.Description,
		Brand:       brand,
		Model:       model,
		Price:       price,
		ImageURL:    imageURL,
		Category:    category,
		ProcessedAt: time.Now(),
	}, nil
}

// Helper functions for generic parsing

func extractIDFromLink(link string) string {
	// Extract ID from ss.lv URL like /msg/ru/.../bjhmxk.html -> bjhmxk
	parts := strings.Split(link, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if strings.HasSuffix(last, ".html") {
			return strings.TrimSuffix(last, ".html")
		}
	}
	return ""
}

func parsePubDateStr(pubDateStr string) (time.Time, error) {
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, pubDateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse date: %s", pubDateStr)
}

func extractSSField(html, fieldName string) string {
	// Pattern for ss.lv: Марка: <b>Brand<br>Model</b>
	idx := strings.Index(html, fieldName+":")
	if idx == -1 {
		return ""
	}
	
	// Find <b> after field name
	rest := html[idx+len(fieldName)+1:]
	bStart := strings.Index(rest, "<b>")
	if bStart == -1 {
		return ""
	}
	
	rest = rest[bStart+3:]
	// Handle nested <b>
	if strings.HasPrefix(rest, "<b>") {
		rest = rest[3:]
	}
	
	// Find end - either <br>, </b>, or <
	endIdx := strings.IndexAny(rest, "<")
	if endIdx == -1 {
		return ""
	}
	
	return strings.TrimSpace(rest[:endIdx])
}

func extractSSPrice(html string) int {
	// Find Цена: <b>... or Price: <b>...
	for _, fieldName := range []string{"Цена", "Price"} {
		idx := strings.Index(html, fieldName+":")
		if idx == -1 {
			continue
		}
		
		rest := html[idx:]
		// Find digits after <b>
		bStart := strings.Index(rest, "<b>")
		if bStart == -1 {
			continue
		}
		
		rest = rest[bStart+3:]
		// Handle nested <b>
		if strings.HasPrefix(rest, "<b>") {
			rest = rest[3:]
		}
		
		// Extract digits
		var digits strings.Builder
		for _, c := range rest {
			if c >= '0' && c <= '9' {
				digits.WriteRune(c)
			} else if digits.Len() > 0 {
				break
			}
		}
		
		if digits.Len() > 0 {
			if price, err := fmt.Sscanf(digits.String(), "%d"); err == nil && price > 0 {
				var p int
				fmt.Sscanf(digits.String(), "%d", &p)
				return p
			}
		}
	}
	return 0
}

func extractImageURLFromHTML(html string) string {
	idx := strings.Index(html, `src="`)
	if idx == -1 {
		return ""
	}
	rest := html[idx+5:]
	end := strings.Index(rest, `"`)
	if end == -1 {
		return ""
	}
	return rest[:end]
}
