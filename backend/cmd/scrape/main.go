package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/fantasysuperleague/backend/internal/database"
	"github.com/fantasysuperleague/backend/scraper"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	action := flag.String("action", "all", "Action: all | clubs | players | stats | import")
	gameweek := flag.Int("gw", 0, "Gameweek number (for stats action)")
	file := flag.String("file", "position_review.csv", "CSV file (for import action)")
	flag.Parse()

	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	s := scraper.NewScraper(db)

	switch *action {
	case "all":
		if err := s.ScrapeAll(); err != nil {
			log.Fatalf("Scrape failed: %v", err)
		}
	case "clubs":
		if err := s.ScrapeClubs(); err != nil {
			log.Fatalf("Scrape clubs failed: %v", err)
		}
	case "players":
		if err := s.ScrapePlayers(); err != nil {
			log.Fatalf("Scrape players failed: %v", err)
		}
	case "stats":
		if *gameweek == 0 {
			// Try from env or args
			gwStr := os.Getenv("GAMEWEEK")
			if gwStr != "" {
				*gameweek, _ = strconv.Atoi(gwStr)
			}
		}
		if *gameweek == 0 {
			fmt.Println("Usage: scrape -action=stats -gw=<gameweek_number>")
			os.Exit(1)
		}
		if err := s.ScrapeMatchStats(*gameweek); err != nil {
			log.Fatalf("Scrape stats failed: %v", err)
		}
	case "import":
		if err := s.ImportPositions(*file); err != nil {
			log.Fatalf("Import failed: %v", err)
		}
	default:
		fmt.Printf("Unknown action: %s\n", *action)
		fmt.Println("Available: all | clubs | players | stats | import")
		os.Exit(1)
	}
}
