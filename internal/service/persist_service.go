package service

import (
	"context"
	"log"

	"web-rss-parser/internal/database"
	"web-rss-parser/internal/domain"
)

// PersistService persists parsed items to database
type PersistService struct {
	repo *database.Repository
}

// NewPersistService creates a new persist service
func NewPersistService(repo *database.Repository) *PersistService {
	return &PersistService{repo: repo}
}

// SaveItems saves items to database (skips duplicates based on link)
// Returns statistics: total items, new items added, duplicates skipped
func (s *PersistService) SaveItems(ctx context.Context, items []domain.ParsedItem) (total, added, skipped int, err error) {
	total = len(items)
	log.Printf("Persisting %d items to database...", total)

	for i, item := range items {
		isNew, err := s.repo.SaveItem(&item)
		if err != nil {
			log.Printf("Error saving item %s (%s): %v", item.ID, item.Link, err)
			continue // Skip this item and continue with others
		}

		if isNew {
			added++
			log.Printf("[%d/%d] Added new item: %s - %s (Category: %s, Price: %d€)",
				i+1, total, item.ID, item.Title, item.Category.Name, item.Price)
		} else {
			skipped++
			log.Printf("[%d/%d] Skipped duplicate: %s - %s",
				i+1, total, item.ID, item.Title)
		}
	}

	log.Printf("Persistence complete: %d total, %d added, %d skipped", total, added, skipped)
	return total, added, skipped, nil
}

// GetItemByID retrieves an item by its ID
func (s *PersistService) GetItemByID(ctx context.Context, id string) (*domain.ParsedItem, error) {
	return s.repo.GetItemByID(id)
}

// GetStatistics returns database statistics
func (s *PersistService) GetStatistics(ctx context.Context) (map[string]int, error) {
	return s.repo.CountItemsByCategory()
}
