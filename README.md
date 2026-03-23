# Web Scraper

A Go-based RSS/HTML parser that fetches, parses, stores, and displays sale items from various websites.

## Current Status

✅ **Implemented:**
- **CLI Interface** - `webscraper` command with scrape, list, stats commands
- **Remote RSS Fetching** - Fetch directly from URLs
- **FetcherService** - Routes files by filename to appropriate category parser
- **FileParser implementations** - Handle both RSS and HTML formats per category
- **PersistService** - Saves parsed items to SQLite database with duplicate detection
- **Price validation** - Skips items without valid prices (price > 0)
- **Categories**: Monitors, Notebooks, TVs, Cars, Baby Bicycles
- **Auto-detection** - Automatically detects category from URL patterns
- **Database schema** with typed properties support
- **Configuration** via YAML file

⏳ **TODO:**
- ClassifyService - Assign ratings to items
- NotifyService - Send Telegram notifications
- WebService - Serve UI and API for browsing items
- Scheduled fetching with time.NewTicker
- HTML parsing for non-RSS sources (Wallapop, etc.)

## Quick Start

### Build

```bash
go build -o webscraper ./cmd/cli/
```

### Usage

```bash
# Scrape from RSS URL (auto-detects category)
./webscraper scrape --url "https://www.ss.lv/ru/electronics/computers/monitors/rss/"

# Scrape with explicit category
./webscraper scrape --url "https://www.ss.lv/ru/transport/bycycles/childrens-for-tots/rss/" --category baby_bicycles

# Process local files
./webscraper scrape --folder ./toprocess

# List items
./webscraper list --category monitors --limit 10

# Show statistics
./webscraper stats

# Help
./webscraper help
```

## Supported RSS Feeds

| Category | URL |
|----------|-----|
| Monitors | `https://www.ss.lv/ru/electronics/computers/monitors/rss/` |
| Notebooks | `https://www.ss.lv/ru/electronics/computers/noutbooks/rss/` |
| Baby Bicycles | `https://www.ss.lv/ru/transport/bycycles/childrens-for-tots/rss/` |

## Project Structure

```
web_scrapper/
├── cmd/
│   ├── cli/main.go                    # CLI application
│   └── server/main.go                 # Server (legacy)
├── internal/
│   ├── domain/                        # Domain models
│   │   ├── models.go                  # ParsedItem, Category, Property types
│   │   └── rss.go                     # RSS feed structures
│   ├── service/                       # Business logic
│   │   ├── fetcher_service.go         # Local file processing
│   │   ├── remote_fetcher_service.go  # Remote URL fetching
│   │   └── persist_service.go         # Database persistence
│   ├── parser/                        # File parsers (RSS + HTML)
│   │   ├── parser.go                  # FileParser interface + helpers
│   │   ├── monitor_file_parser.go     # Monitors (RSS/HTML)
│   │   ├── notebook_file_parser.go    # Notebooks (RSS/HTML)
│   │   ├── baby_bicycle_file_parser.go # Baby Bicycles (RSS)
│   │   ├── tv_file_parser.go          # TVs (RSS/HTML)
│   │   └── car_file_parser.go         # Cars (RSS/HTML)
│   ├── database/                      # Database layer
│   │   ├── db.go                      # SQLite connection + schema
│   │   └── repository.go              # CRUD operations
│   └── config/                        # Configuration
│       └── config.go
├── toprocess/                         # Folder for local files
├── config.yaml                        # Configuration file
└── data.db                            # SQLite database
```

## Configuration

Edit `config.yaml`:

```yaml
database:
  path: "./data.db"

server:
  port: 8080

files:
  folder_path: "./toprocess"
  fetch_interval: 60  # minutes
```

## Categories and Properties

### Monitors
- Brand, Model, Size

### Notebooks
- Brand, Model, Screen, HDD, RAM

### Baby Bicycles
- Brand, Model, Year, Condition

### TVs
- Brand, Model, Diagonal, Condition

### Cars
- Brand, Model, Year, Engine, Mileage
