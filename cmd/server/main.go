package main

import (
	"context"
	"log"
	"os"

	"web-rss-parser/internal/config"
	"web-rss-parser/internal/database"
	"web-rss-parser/internal/service"
)

func main() {
	log.Println("=== Web RSS Parser Starting ===")

	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Configuration loaded: Database=%s, Files Folder=%s, Port=%d",
		cfg.Database.Path, cfg.Files.FolderPath, cfg.Server.Port)

	// Initialize database
	db, err := database.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create repository and services
	repo := database.NewRepository(db)
	fetcherService := service.NewFetcherService()
	persistService := service.NewPersistService(repo)

	// Process files on startup
	ctx := context.Background()
	if err := processFiles(ctx, fetcherService, persistService, cfg.Files.FolderPath); err != nil {
		log.Printf("Error processing files: %v", err)
	}

	// Print final statistics
	printStatistics(ctx, persistService)

	log.Println("=== Processing Complete ===")
	log.Println("TODO: Implement scheduled fetching with time.NewTicker")
	log.Println("TODO: Implement WebService to serve UI and API")
}

// processFiles fetches and persists items from all files
func processFiles(ctx context.Context, fetcherService *service.FetcherService, persistService *service.PersistService, folderPath string) error {
	log.Println("\n--- Starting File Processing ---")

	// Fetch items from folder
	items, err := fetcherService.FetchFromFolder(ctx, folderPath)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		log.Println("No items fetched from files")
		return nil
	}

	// Persist items to database
	total, added, skipped, err := persistService.SaveItems(ctx, items)
	if err != nil {
		return err
	}

	log.Printf("\n--- Processing Summary ---")
	log.Printf("Total items processed: %d", total)
	log.Printf("New items added: %d", added)
	log.Printf("Duplicates skipped: %d", skipped)

	return nil
}

// printStatistics prints database statistics
func printStatistics(ctx context.Context, persistService *service.PersistService) {
	log.Println("\n--- Database Statistics ---")

	stats, err := persistService.GetStatistics(ctx)
	if err != nil {
		log.Printf("Error getting statistics: %v", err)
		return
	}

	if len(stats) == 0 {
		log.Println("No items in database")
		return
	}

	totalItems := 0
	for category, count := range stats {
		log.Printf("Category '%s': %d items", category, count)
		totalItems += count
	}
	log.Printf("Total items in database: %d", totalItems)
}

// init sets up logging
func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
