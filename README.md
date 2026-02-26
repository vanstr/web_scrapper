# Web RSS Parser

A Go-based RSS/HTML parser that fetches, parses, stores, and displays parsed items from a unified folder structure.

## Current Status

✅ **Implemented:**
- **FetcherService** - Routes files by filename to appropriate category parser
- **FileParser** implementations - Handle both RSS and HTML formats per category
- **PersistService** - Saves parsed items to SQLite database with duplicate detection
- **Price validation** - Skips items without valid prices (price > 0)
- **Categories**: Monitors, Notebooks, TVs, Cars 
- **Database schema** with typed properties support
- **Configuration** via YAML file
- **Single folder approach** - All files in `toprocess/` folder

⏳ **TODO:**
- ClassifyService - Assign ratings to items
- NotifyService - Send Telegram notifications
- WebService - Serve UI and API for browsing items
- Scheduled fetching with time.NewTicker
- Remote RSS/HTML feed fetching from URLs
- HTML parsing for Monitors, Notebooks, TVs (currently only Cars)

## Project Structure

```
web-rss-parser/
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
├── internal/
│   ├── domain/                        # Domain models
│   │   ├── models.go                  # ParsedItem, Category, Property types
│   │   └── rss.go                     # RSS feed structures
│   ├── service/                       # Business logic
│   │   ├── fetcher_service.go         # Routes by filename to parsers
│   │   └── persist_service.go         # Database persistence
│   ├── parser/                        # File parsers (RSS + HTML)
│   │   ├── parser.go                  # FileParser interface + helpers
│   │   ├── monitor_file_parser.go     # Monitors (RSS/HTML)
│   │   ├── notebook_file_parser.go    # Notebooks (RSS/HTML)
│   │   ├── tv_file_parser.go          # TVs (RSS/HTML)
│   │   └── car_file_parser.go         # Cars (RSS/HTML with table parsing)
│   ├── database/                      # Database layer
│   │   ├── db.go                      # SQLite connection
│   │   └── repository.go              # CRUD operations
│   └── config/                        # Configuration
│       └── config.go
├── toprocess/                         # Unified folder for all files
├── config.yaml                        # Configuration file
├── go.mod
└── go.sum
```

## Quick Start

### 1. Install Dependencies

```bash
go mod tidy
```

### 2. Configure Application

Edit `config.yaml`:

### 3. Run Application

```bash
go run cmd/server/main.go
```

## Usage

### Processing Files

The application automatically processes all files from the `toprocess/` folder on startup:
- **Category Detection**: Based on filename patterns (monitor, notebook, tv, car)
- **Format Detection**: Based on file extension (.xml for RSS, .html for HTML)
- **Price Validation**: Items without prices (price <= 0) are skipped

## Architecture

### Simplified Design
```
FetcherService (filename routing)
    ↓
FileParser implementations (format detection)
    ↓
Category-specific extraction (RSS/HTML)
    ↓
PersistService (database storage)
```