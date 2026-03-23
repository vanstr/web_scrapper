package parser

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"web-rss-parser/internal/domain"
)

// FileParser interface for parsing files (RSS or HTML)
type FileParser interface {
	// ParseFile parses a file and returns parsed items
	// The parser decides whether to use RSS or HTML parsing based on file extension
	ParseFile(ctx context.Context, filePath string) ([]domain.ParsedItem, error)

	// GetCategory returns the category object
	GetCategory() domain.Category
}

// Helper functions for parsing

// extractImageURL extracts image URL from HTML img tag
func extractImageURL(html string) string {
	re := regexp.MustCompile(`src="([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractField extracts a field value from HTML like: FieldName: <b>Value</b>
// Supports both English and Russian field names
func extractField(html, fieldName string) string {
	// Map English field names to Russian equivalents for ss.lv
	russianNames := map[string]string{
		"Brand": "Марка",
		"Model": "Модель",
		"Size":  "Размер",
		"Price": "Цена",
		"Year":  "Год",
	}

	// Try English first, then Russian
	names := []string{fieldName}
	if ru, ok := russianNames[fieldName]; ok {
		names = append(names, ru)
	}

	for _, name := range names {
		pattern := name + `:\s*<b>(?:<b>)?([^<]+)`
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			value := strings.TrimSpace(matches[1])
			// Clean up nested tags and line breaks
			value = strings.ReplaceAll(value, "<br>", "")
			value = strings.ReplaceAll(value, "</b>", "")
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// extractPrice extracts price from HTML like: Price: <b>149 €</b> or <b>149</b> €
// Supports both English "Price" and Russian "Цена"
func extractPrice(html string) int {
	patterns := []string{
		`Price:\s*<b>(?:<b>)?(\d+)`,
		`Цена:\s*<b>(?:<b>)?(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			price, err := strconv.Atoi(matches[1])
			if err == nil && price > 0 {
				return price
			}
		}
	}
	return 0
}

// createStringProperty creates a string property
func createStringProperty(meta domain.PropertyMeta, value string) domain.Property {
	return domain.Property{
		PropertyMeta: meta,
		StringValue:  &value,
	}
}

// createNumberProperty creates a number property
func createNumberProperty(meta domain.PropertyMeta, value float64) domain.Property {
	return domain.Property{
		PropertyMeta: meta,
		NumberValue:  &value,
	}
}

// extractIDFromLink extracts ID from URL
// Example: https://www.ss.lv/msg/en/electronics/computers/monitors/okebe.html -> okebe
func extractIDFromLink(link string) string {
	re := regexp.MustCompile(`/([^/]+)\.html$`)
	matches := re.FindStringSubmatch(link)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parsePubDate parses RSS pub date string to time.Time
// Example: Wed, 25 Feb 2026 19:00:21 +0300
func parsePubDate(pubDateStr string) (time.Time, error) {
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
