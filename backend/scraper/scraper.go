package scraper

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/fantasysuperleague/backend/internal/models"
	"github.com/fantasysuperleague/backend/internal/scoring"
	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"
)

const (
	baseURL    = "https://ileague.id"
	leagueSlug = "BRI_SUPER_LEAGUE_2026-27"
)

// Scraper holds the database connection for inserting data
type Scraper struct {
	db *sql.DB
}

func NewScraper(db *sql.DB) *Scraper {
	return &Scraper{db: db}
}

// ScrapeAll runs the full scraping pipeline: clubs → players → stats
func (s *Scraper) ScrapeAll() error {
	log.Println("🔄 Starting full scrape from ileague.id...")

	if err := s.ScrapeClubs(); err != nil {
		return fmt.Errorf("failed to scrape clubs: %w", err)
	}

	if err := s.ScrapePlayers(); err != nil {
		return fmt.Errorf("failed to scrape players: %w", err)
	}

	log.Println("✅ Full scrape completed!")
	return nil
}

// ScrapeClubs fetches all clubs from Liga 1
func (s *Scraper) ScrapeClubs() error {
	log.Println("🏟️ Scraping clubs...")

	c := newCollector()
	clubs := []models.Club{}

	c.OnHTML(".club-card, .team-card, [data-club]", func(e *colly.HTMLElement) {
		name := e.ChildText(".club-name, .team-name, h3, h4")
		slug := e.Attr("data-slug")
		logoURL := e.ChildAttr("img", "src")

		if slug == "" {
			href := e.ChildAttr("a", "href")
			parts := strings.Split(href, "/")
			if len(parts) > 0 {
				slug = parts[len(parts)-1]
			}
		}

		if name != "" && slug != "" {
			tier := 2 // default mid tier
			bigClubs := []string{"persija", "persib", "bali-united", "arema", "pss-sleman", "borneo"}
			for _, bc := range bigClubs {
				if strings.Contains(strings.ToLower(slug), bc) {
					tier = 1
					break
				}
			}

			clubs = append(clubs, models.Club{
				ID:      uuid.New().String(),
				Name:    name,
				Slug:    slug,
				LogoURL: ensureAbsURL(logoURL),
				Tier:    tier,
			})
		}
	})

	url := fmt.Sprintf("%s/clubs/index/%s", baseURL, leagueSlug)
	if err := c.Visit(url); err != nil {
		log.Printf("Warning: could not visit clubs page: %v", err)
		// Insert default clubs for offline development
		s.insertDefaultClubs()
		return nil
	}

	for _, club := range clubs {
		s.upsertClub(club)
	}

	log.Printf("✅ Scraped %d clubs", len(clubs))
	return nil
}

// ScrapePlayers fetches all players from all clubs
func (s *Scraper) ScrapePlayers() error {
	log.Println("👤 Scraping players from all clubs...")

	rows, err := s.db.Query("SELECT id, slug, tier FROM clubs")
	if err != nil {
		return err
	}
	defer rows.Close()

	type clubInfo struct {
		ID   string
		Slug string
		Tier int
	}
	clubs := []clubInfo{}
	for rows.Next() {
		var ci clubInfo
		rows.Scan(&ci.ID, &ci.Slug, &ci.Tier)
		clubs = append(clubs, ci)
	}

	for _, club := range clubs {
		log.Printf("  Scraping players for club: %s", club.Slug)
		s.scrapeClubPlayers(club.ID, club.Slug, club.Tier)
		time.Sleep(500 * time.Millisecond) // polite delay
	}

	return nil
}

func (s *Scraper) scrapeClubPlayers(clubID, clubSlug string, clubTier int) {
	c := newCollector()

	c.OnHTML(".player-card, .squad-player, [data-player]", func(e *colly.HTMLElement) {
		name := e.ChildText(".player-name, .name, h3, h4")
		position := normalizePosition(e.ChildText(".position, .pos"))
		photoURL := e.ChildAttr("img.player-photo, img.photo", "src")
		slug := e.Attr("data-slug")
		jerseyStr := e.ChildText(".jersey, .number")
		token := e.Attr("data-token")

		if slug == "" {
			href := e.ChildAttr("a", "href")
			parts := strings.Split(href, "/")
			if len(parts) > 0 {
				slug = parts[len(parts)-1]
			}
		}

		if name == "" || position == "" {
			return
		}

		jersey, _ := strconv.Atoi(strings.TrimSpace(jerseyStr))
		isNational := e.ChildText(".national, .timnas") != ""
		price := scoring.GetPriceByPositionAndTier(position, clubTier, isNational)

		player := models.Player{
			ID:             uuid.New().String(),
			Name:           strings.TrimSpace(name),
			Slug:           slug,
			ClubID:         clubID,
			Position:       position,
			IsNationalTeam: isNational,
			Price:          price,
			PhotoURL:       ensureAbsURL(photoURL),
			IleagueToken:   token,
			JerseyNumber:   jersey,
			IsActive:       true,
		}

		s.upsertPlayer(player)
	})

	url := fmt.Sprintf("%s/clubs/single/%s/%s", baseURL, leagueSlug, clubSlug)
	if err := c.Visit(url); err != nil {
		log.Printf("  Warning: could not scrape players for %s: %v", clubSlug, err)
	}
}

// ScrapeMatchStats scrapes match stats for a specific gameweek
func (s *Scraper) ScrapeMatchStats(gameweekNum int) error {
	log.Printf("📊 Scraping stats for gameweek %d...", gameweekNum)

	// Ensure gameweek exists
	var gwID string
	err := s.db.QueryRow("SELECT id FROM gameweeks WHERE number = $1", gameweekNum).Scan(&gwID)
	if err == sql.ErrNoRows {
		gwID = uuid.New().String()
		s.db.Exec(
			`INSERT INTO gameweeks (id, number, name, is_finished) VALUES ($1, $2, $3, TRUE)`,
			gwID, gameweekNum, fmt.Sprintf("Gameweek %d", gameweekNum),
		)
	}

	// Get match results for this gameweek
	c := newCollector()
	
	c.OnHTML(".match-result, .fixture-result", func(e *colly.HTMLElement) {
		matchURL := e.ChildAttr("a", "href")
		if matchURL != "" {
			// Visit each match detail page to get player stats
			s.scrapeMatchDetail(gwID, ensureAbsURL(matchURL))
			time.Sleep(300 * time.Millisecond)
		}
	})

	url := fmt.Sprintf("%s/fixtures/index/%s/%d/1", baseURL, leagueSlug, gameweekNum)
	c.Visit(url)

	// After scraping, calculate fantasy points
	s.calculateGameweekPoints(gwID)

	log.Printf("✅ Gameweek %d stats scraped!", gameweekNum)
	return nil
}

func (s *Scraper) scrapeMatchDetail(gwID, matchURL string) {
	c := newCollector()

	c.OnHTML(".player-stat-row, .lineup-player", func(e *colly.HTMLElement) {
		playerSlug := e.Attr("data-player")
		if playerSlug == "" {
			return
		}

		var playerID string
		s.db.QueryRow("SELECT id FROM players WHERE slug = $1", playerSlug).Scan(&playerID)
		if playerID == "" {
			return
		}

		stats := models.PlayerStats{
			ID:            uuid.New().String(),
			PlayerID:      playerID,
			GameweekID:    gwID,
			MinutesPlayed: parseInt(e.ChildText(".minutes")),
			Goals:         parseInt(e.ChildText(".goals")),
			Assists:       parseInt(e.ChildText(".assists")),
			YellowCards:   parseInt(e.ChildText(".yellow-cards")),
			RedCards:      parseInt(e.ChildText(".red-cards")),
			Saves:         parseInt(e.ChildText(".saves")),
		}

		s.upsertPlayerStats(stats)
	})

	c.Visit(matchURL)
}

func (s *Scraper) calculateGameweekPoints(gwID string) {
	rows, err := s.db.Query(`
		SELECT ps.id, ps.player_id, ps.minutes_played, ps.goals, ps.assists,
		       ps.clean_sheet, ps.goals_conceded, ps.yellow_cards, ps.red_cards,
		       ps.saves, ps.bonus, p.position
		FROM player_stats ps
		JOIN players p ON ps.player_id = p.id
		WHERE ps.gameweek_id = $1
	`, gwID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var statsID, playerID, position string
		var stats models.PlayerStats
		rows.Scan(
			&statsID, &playerID, &stats.MinutesPlayed, &stats.Goals, &stats.Assists,
			&stats.CleanSheet, &stats.GoalsConceded, &stats.YellowCards, &stats.RedCards,
			&stats.Saves, &stats.Bonus, &position,
		)

		pts := scoring.CalculatePoints(stats, position)
		s.db.Exec("UPDATE player_stats SET fantasy_points = $1 WHERE id = $2", pts, statsID)
		s.db.Exec("UPDATE players SET total_points = total_points + $1 WHERE id = $2", pts, playerID)
	}

	// Update fantasy teams total points
	s.db.Exec(`
		UPDATE fantasy_teams ft
		SET total_points = (
			SELECT COALESCE(SUM(
				CASE 
					WHEN ftp.is_captain THEN ps.fantasy_points * 2
					ELSE ps.fantasy_points
				END
			), 0)
			FROM fantasy_team_players ftp
			JOIN player_stats ps ON ps.player_id = ftp.player_id AND ps.gameweek_id = $1
			WHERE ftp.team_id = ft.id AND ftp.is_starting = TRUE
		) + ft.total_points
	`, gwID)
}

// --- DB helpers ---

func (s *Scraper) upsertClub(club models.Club) {
	s.db.Exec(`
		INSERT INTO clubs (id, name, slug, logo_url, tier)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name,
			logo_url = EXCLUDED.logo_url,
			tier = EXCLUDED.tier
	`, club.ID, club.Name, club.Slug, club.LogoURL, club.Tier)
}

func (s *Scraper) upsertPlayer(player models.Player) {
	s.db.Exec(`
		INSERT INTO players (id, name, slug, club_id, position, is_national_team, price, photo_url, ileague_token, jersey_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name,
			club_id = EXCLUDED.club_id,
			position = EXCLUDED.position,
			price = EXCLUDED.price,
			photo_url = EXCLUDED.photo_url,
			updated_at = NOW()
	`, player.ID, player.Name, player.Slug, player.ClubID, player.Position,
		player.IsNationalTeam, player.Price, player.PhotoURL, player.IleagueToken, player.JerseyNumber)
}

func (s *Scraper) upsertPlayerStats(stats models.PlayerStats) {
	s.db.Exec(`
		INSERT INTO player_stats (id, player_id, gameweek_id, minutes_played, goals, assists, yellow_cards, red_cards, saves)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (player_id, gameweek_id) DO UPDATE SET
			minutes_played = EXCLUDED.minutes_played,
			goals = EXCLUDED.goals,
			assists = EXCLUDED.assists,
			yellow_cards = EXCLUDED.yellow_cards,
			red_cards = EXCLUDED.red_cards,
			saves = EXCLUDED.saves
	`, stats.ID, stats.PlayerID, stats.GameweekID, stats.MinutesPlayed, stats.Goals,
		stats.Assists, stats.YellowCards, stats.RedCards, stats.Saves)
}

// insertDefaultClubs seeds known Liga 1 clubs for offline dev
func (s *Scraper) insertDefaultClubs() {
	defaultClubs := []struct {
		name      string
		shortName string
		slug      string
		stadium   string
		city      string
		tier      int
	}{
		{"Persija Jakarta", "PJK", "persija-jakarta", "Gelora Bung Karno", "Jakarta", 1},
		{"Persib Bandung", "PIB", "persib-bandung", "Gelora Bandung Lautan Api", "Bandung", 1},
		{"Bali United", "BAL", "bali-united", "Kapten I Wayan Dipta", "Gianyar", 1},
		{"Arema FC", "ARM", "arema-fc", "Kanjuruhan", "Malang", 1},
		{"PSS Sleman", "PSS", "pss-sleman", "Maguwoharjo", "Sleman", 2},
		{"Borneo FC", "BFC", "borneo-fc", "Segiri", "Samarinda", 2},
		{"PSIS Semarang", "PSI", "psis-semarang", "Jatidiri", "Semarang", 2},
		{"Persebaya Surabaya", "PBS", "persebaya-surabaya", "Gelora Bung Tomo", "Surabaya", 2},
		{"Madura United", "MDU", "madura-united", "Gelora Ratu Pamelingan", "Pamekasan", 2},
		{"Dewa United", "DWU", "dewa-united", "Indomilk Arena", "Tangerang", 2},
		{"Persis Solo", "PRS", "persis-solo", "Manahan", "Solo", 2},
		{"Persik Kediri", "PRK", "persik-kediri", "Brawijaya", "Kediri", 3},
		{"PSBS Biak", "PSB", "psbs-biak", "Mandala Jayapura", "Biak", 3},
		{"Malut United", "MLU", "malut-united", "Gelora Kie Raha", "Ternate", 3},
		{"Semen Padang", "SMP", "semen-padang", "Haji Agus Salim", "Padang", 3},
		{"Barito Putera", "BAP", "barito-putera", "17 Mei", "Banjarmasin", 3},
	}

	for _, club := range defaultClubs {
		s.db.Exec(`
			INSERT INTO clubs (id, name, short_name, slug, stadium, city, tier)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (slug) DO NOTHING
		`, uuid.New().String(), club.name, club.shortName, club.slug, club.stadium, club.city, club.tier)
	}
	log.Printf("✅ Inserted %d default clubs", len(defaultClubs))
}

// --- Utils ---

func newCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		colly.AllowURLRevisit(),
	)
	c.SetRequestTimeout(30 * time.Second)
	return c
}

func normalizePosition(raw string) string {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	switch {
	case upper == "GK" || upper == "GOALKEEPER" || upper == "KIPER" || upper == "PENJAGA GAWANG":
		return "GK"
	case upper == "DEF" || upper == "DEFENDER" || upper == "CB" || upper == "LB" || upper == "RB" || upper == "BEK":
		return "DEF"
	case upper == "MID" || upper == "MIDFIELDER" || upper == "CM" || upper == "AM" || upper == "DM" || upper == "GELANDANG":
		return "MID"
	case upper == "FWD" || upper == "FORWARD" || upper == "ST" || upper == "CF" || upper == "LW" || upper == "RW" || upper == "STRIKER" || upper == "PENYERANG":
		return "FWD"
	default:
		return "MID" // default fallback
	}
}

func ensureAbsURL(url string) string {
	if url == "" {
		return ""
	}
	if strings.HasPrefix(url, "http") {
		return url
	}
	return baseURL + url
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	n, _ := strconv.Atoi(s)
	return n
}
