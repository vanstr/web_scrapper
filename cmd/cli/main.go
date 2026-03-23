package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"web-rss-parser/internal/config"
	"web-rss-parser/internal/database"
	"web-rss-parser/internal/domain"
	"web-rss-parser/internal/service"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "scrape":
		handleScrape(os.Args[2:])
	case "list":
		handleList(os.Args[2:])
	case "stats":
		handleStats(os.Args[2:])
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Web Scraper CLI

Usage:
  webscraper <command> [options]

Commands:
  scrape    Fetch and parse items from RSS feeds or local files
  list      List items from database
  stats     Show database statistics
  help      Show this help message

Scrape Options:
  --url <url>         Fetch from a specific RSS URL
  --category <cat>    Category for the URL (monitors, notebooks, baby_bicycles)
  --folder <path>     Process local files from folder
  --config <path>     Config file path (default: config.yaml)

List Options:
  --category <cat>    Filter by category
  --limit <n>         Limit number of results (default: 20)

Examples:
  webscraper scrape --url "https://www.ss.lv/ru/electronics/computers/monitors/rss/" --category monitors
  webscraper scrape --url "https://www.ss.lv/ru/transport/bycycles/childrens-for-tots/rss/" --category baby_bicycles
  webscraper scrape --folder ./toprocess
  webscraper list --category monitors --limit 10
  webscraper stats`)
}

func handleScrape(args []string) {
	fs := flag.NewFlagSet("scrape", flag.ExitOnError)
	url := fs.String("url", "", "RSS feed URL to scrape")
	category := fs.String("category", "", "Category (monitors, notebooks, baby_bicycles)")
	folder := fs.String("folder", "", "Folder with local files to process")
	configPath := fs.String("config", "config.yaml", "Config file path")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	// Load config and initialize database
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	persistService := service.NewPersistService(repo)
	ctx := context.Background()

	var items []domain.ParsedItem

	if *url != "" {
		// Scrape from URL
		if *category == "" {
			// Try to auto-detect category
			remoteFetcher := service.NewRemoteFetcherService()
			*category = remoteFetcher.DetectCategoryFromURL(*url)
			if *category == "" {
				log.Fatalf("Cannot auto-detect category for URL. Please specify --category")
			}
			log.Printf("Auto-detected category: %s", *category)
		}

		remoteFetcher := service.NewRemoteFetcherService()
		items, err = remoteFetcher.FetchFromURL(ctx, *url, *category)
		if err != nil {
			log.Fatalf("Failed to scrape URL: %v", err)
		}
	} else if *folder != "" {
		// Process local folder
		fetcherService := service.NewFetcherService()
		items, err = fetcherService.FetchFromFolder(ctx, *folder)
		if err != nil {
			log.Fatalf("Failed to process folder: %v", err)
		}
	} else {
		// Use default folder from config
		fetcherService := service.NewFetcherService()
		items, err = fetcherService.FetchFromFolder(ctx, cfg.Files.FolderPath)
		if err != nil {
			log.Fatalf("Failed to process folder: %v", err)
		}
	}

	if len(items) == 0 {
		log.Println("No items found")
		return
	}

	// Save to database
	total, added, skipped, err := persistService.SaveItems(ctx, items)
	if err != nil {
		log.Fatalf("Failed to save items: %v", err)
	}

	fmt.Printf("\n=== Scrape Summary ===\n")
	fmt.Printf("Total processed: %d\n", total)
	fmt.Printf("New items added: %d\n", added)
	fmt.Printf("Duplicates skipped: %d\n", skipped)
}

func handleList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	category := fs.String("category", "", "Filter by category")
	limit := fs.Int("limit", 20, "Limit number of results")
	configPath := fs.String("config", "config.yaml", "Config file path")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	ctx := context.Background()

	items, err := repo.ListItems(ctx, *category, *limit)
	if err != nil {
		log.Fatalf("Failed to list items: %v", err)
	}

	if len(items) == 0 {
		fmt.Println("No items found")
		return
	}

	fmt.Printf("\n=== Items (%d) ===\n", len(items))
	for _, item := range items {
		fmt.Printf("\n[%s] %s\n", item.Category.Name, truncate(item.Title, 60))
		fmt.Printf("  Price: %d €\n", item.Price)
		if item.Brand != "" {
			fmt.Printf("  Brand: %s\n", item.Brand)
		}
		if item.Model != "" {
			fmt.Printf("  Model: %s\n", item.Model)
		}
		fmt.Printf("  Link: %s\n", item.Link)
		fmt.Printf("  Date: %s\n", item.PubDate.Format("2006-01-02 15:04"))
	}
}

func handleStats(args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Config file path")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	ctx := context.Background()

	stats, err := repo.GetStatistics(ctx)
	if err != nil {
		log.Fatalf("Failed to get statistics: %v", err)
	}

	fmt.Printf("\n=== Database Statistics ===\n")
	if len(stats) == 0 {
		fmt.Println("No items in database")
		return
	}

	total := 0
	for category, count := range stats {
		fmt.Printf("  %s: %d items\n", strings.Title(category), count)
		total += count
	}
	fmt.Printf("  Total: %d items\n", total)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
