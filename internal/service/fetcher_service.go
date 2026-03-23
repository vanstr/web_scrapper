package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"web-rss-parser/internal/domain"
	"web-rss-parser/internal/parser"
)

// FetcherService fetches and parses files from a single folder
type FetcherService struct {
	parsers map[string]parser.FileParser
}

// NewFetcherService creates a new fetcher service with all parsers
func NewFetcherService() *FetcherService {
	return &FetcherService{
		parsers: map[string]parser.FileParser{
			"monitors":      parser.NewMonitorFileParser(),
			"notebooks":     parser.NewNotebookFileParser(),
			"tvs":           parser.NewTVFileParser(),
			"cars":          parser.NewCarFileParser(),
			"baby_bicycles": parser.NewBabyBicycleFileParser(),
		},
	}
}

// FetchFromFolder reads all files from folder and parses them based on filename
func (s *FetcherService) FetchFromFolder(ctx context.Context, folderPath string) ([]domain.ParsedItem, error) {
	log.Printf("Fetching files from folder: %s", folderPath)

	// Read all files from folder
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read folder %s: %w", folderPath, err)
	}

	var allItems []domain.ParsedItem

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Detect category from filename
		category := s.detectCategoryFromFilename(file.Name())
		if category == "" {
			log.Printf("Warning: cannot detect category from filename %s, skipping", file.Name())
			continue
		}

		// Get parser for this category
		fileParser, ok := s.parsers[category]
		if !ok {
			log.Printf("Warning: no parser found for category %s, skipping file %s", category, file.Name())
			continue
		}

		// Parse file (parser decides RSS vs HTML)
		filePath := fmt.Sprintf("%s/%s", folderPath, file.Name())
		items, err := fileParser.ParseFile(ctx, filePath)
		if err != nil {
			log.Printf("Error parsing file %s: %v", file.Name(), err)
			continue // Skip this file and continue with others
		}

		log.Printf("Parsed %d items from %s (category: %s)", len(items), file.Name(), category)
		allItems = append(allItems, items...)
	}

	log.Printf("Total items fetched: %d", len(allItems))
	return allItems, nil
}

// detectCategoryFromFilename detects category from filename
// Examples: rss-monitor.xml -> monitors, ss-cars.html -> cars
func (s *FetcherService) detectCategoryFromFilename(filename string) string {
	patterns := map[string]string{
		"monitor":       "monitors",
		"notebook":      "notebooks",
		"noutbook":      "notebooks", // Alternative spelling
		"tv":            "tvs",
		"car":           "cars",
		"bicycle":       "baby_bicycles",
		"childrens":     "baby_bicycles",
		"tots":          "baby_bicycles",
	}

	lowerFilename := strings.ToLower(filename)
	for pattern, category := range patterns {
		if strings.Contains(lowerFilename, pattern) {
			return category
		}
	}

	return ""
}
