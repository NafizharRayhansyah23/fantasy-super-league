package scraper

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

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

	// Struktur aktual ileague.id: .item-team > .info-team h5 (nama),
	// a[href*=clubs/single] (slug = segmen terakhir, UPPERCASE),
	// .logo-team img (logo), .group-team (stadion)
	c.OnHTML(".item-team", func(e *colly.HTMLElement) {
		name := strings.TrimSpace(e.ChildText(".info-team h5"))
		href := e.ChildAttr("a[href*='clubs/single']", "href")
		logoURL := e.ChildAttr(".logo-team img", "src")
		stadium := strings.TrimSpace(e.ChildText(".group-team"))

		slug := ""
		if href != "" {
			href = strings.Split(href, "?")[0]
			parts := strings.Split(strings.TrimSuffix(href, "/"), "/")
			if len(parts) > 0 {
				slug = parts[len(parts)-1]
			}
		}

		if name == "" || slug == "" {
			return
		}

		tier := 2 // default mid tier
		lower := strings.ToLower(slug)
		for _, bc := range []string{"persija", "persib", "arema", "persebaya", "bali", "psm"} {
			if strings.Contains(lower, bc) {
				tier = 1
				break
			}
		}
		for _, sc := range []string{"garudayaksa", "isenmulang", "java_united", "persijap", "psim"} {
			if strings.Contains(lower, sc) {
				tier = 3
				break
			}
		}

		clubs = append(clubs, models.Club{
			ID:        uuid.New().String(),
			Name:      name,
			ShortName: shortNameFrom(name),
			Slug:      slug,
			LogoURL:   ensureAbsURL(logoURL),
			Stadium:   stadium,
			Tier:      tier,
		})
	})

	url := fmt.Sprintf("%s/clubs?param=%s", baseURL, leagueSlug)
	if err := c.Visit(url); err != nil {
		log.Printf("Warning: could not visit clubs page: %v", err)
		// Insert default clubs for offline development
		s.insertDefaultClubs()
		return nil
	}

	if len(clubs) == 0 {
		log.Println("Warning: clubs page parsed 0 clubs (markup may have changed), seeding defaults")
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

	rows, err := s.db.Query("SELECT id, name, slug, tier FROM clubs ORDER BY name")
	if err != nil {
		return err
	}
	defer rows.Close()

	type clubInfo struct {
		ID   string
		Name string
		Slug string
		Tier int
	}
	clubs := []clubInfo{}
	for rows.Next() {
		var ci clubInfo
		rows.Scan(&ci.ID, &ci.Name, &ci.Slug, &ci.Tier)
		clubs = append(clubs, ci)
	}

	review := []reviewRow{}
	for _, club := range clubs {
		log.Printf("  Scraping players for club: %s", club.Slug)
		s.scrapeClubPlayers(club.ID, club.Name, club.Slug, club.Tier, &review)
		time.Sleep(500 * time.Millisecond) // polite delay
	}

	if len(review) > 0 {
		if err := writeReviewCSV("position_review.csv", review); err != nil {
			log.Printf("Warning: gagal tulis position_review.csv: %v", err)
		} else {
			log.Printf("📝 File review posisi: position_review.csv (%d baris)", len(review))
		}
	}
	return nil
}

// reviewRow mencatat asal posisi tiap pemain untuk dikoreksi manual.
// Kolom source: ileague | known | number | fallback.
type reviewRow struct {
	Slug, Name, Club string
	Jersey           int
	Position, Source string
}

func writeReviewCSV(path string, rows []reviewRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"slug", "name", "club", "jersey", "position", "source"}); err != nil {
		return err
	}
	for _, r := range rows {
		if err := w.Write([]string{r.Slug, r.Name, r.Club, strconv.Itoa(r.Jersey), r.Position, r.Source}); err != nil {
			return err
		}
	}
	return nil
}

// ImportPositions menerapkan koreksi posisi manual dari CSV hasil edit
// position_review.csv. Format: slug,name,club,jersey,position,source.
// Hanya kolom position yang dibaca; harga dihitung ulang ikut tier klub.
// Idempoten (aman di-run ulang) — jadi setelah re-scrape, import lagi file ini.
func (s *Scraper) ImportPositions(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	recs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return err
	}
	if len(recs) < 2 {
		return fmt.Errorf("CSV kosong: %s", path)
	}

	updated, skipped := 0, 0
	for i, rec := range recs[1:] {
		if len(rec) < 5 {
			skipped++
			continue
		}
		slug := strings.TrimSpace(rec[0])
		pos := strings.ToUpper(strings.TrimSpace(rec[4]))
		if slug == "" {
			skipped++
			continue
		}
		if pos != "GK" && pos != "DEF" && pos != "MID" && pos != "FWD" {
			log.Printf("  baris %d (%s): posisi %q tidak valid, skip", i+2, slug, rec[4])
			skipped++
			continue
		}

		var id string
		var tier int
		var isNational bool
		err := s.db.QueryRow(`
			SELECT p.id, c.tier, p.is_national_team FROM players p
			JOIN clubs c ON c.id = p.club_id WHERE p.slug = $1
		`, slug).Scan(&id, &tier, &isNational)
		if err != nil {
			log.Printf("  baris %d (%s): pemain tidak ketemu, skip", i+2, slug)
			skipped++
			continue
		}
		price := scoring.GetPriceByPositionAndTier(pos, tier, isNational)
		if _, err := s.db.Exec(
			`UPDATE players SET position = $1, price = $2, updated_at = NOW() WHERE id = $3`,
			pos, price, id,
		); err != nil {
			log.Printf("  baris %d (%s): gagal update: %v", i+2, slug, err)
			skipped++
			continue
		}
		updated++
	}

	log.Printf("✅ Import selesai: %d update, %d skip", updated, skipped)
	return nil
}

func (s *Scraper) scrapeClubPlayers(clubID, clubName, clubSlug string, clubTier int, review *[]reviewRow) {
	c := newCollector()
	inserted := 0
	bySource := map[string]int{}

	// Struktur aktual ileague.id (tab #squad):
	// - a[href*=singleplayer] -> slug + ?token=...
	// - img[alt=foto-pemain] -> foto (ofisial pakai foto-ofisial & tanpa link -> ke-skip)
	// - td[colspan=2] -> nama, .number-player -> nomor punggung
	// - baris tabel "Negara" -> kewarganegaraan
	// Urutan penentuan posisi (SEMUA pemain dimasukkan):
	//  1. posisi asli situs (bila suatu saat ditampilkan),
	//  2. daftar pemain yang dikenal pasti (knownPositions),
	//  3. inferensi nomor punggung (konvensi umum, winger = MID),
	//  4. fallback MID + tercatat di position_review.csv untuk koreksi manual.
	for _, card := range fetchIleagueCards(c, clubSlug) {
		position, source := resolvePosition(card)
		price := scoring.GetPriceByPositionAndTier(position, clubTier, false)
		s.upsertPlayer(models.Player{
			ID:             uuid.New().String(),
			Name:           card.Name,
			Slug:           card.Slug,
			ClubID:         clubID,
			Position:       position,
			Nationality:    card.Nationality,
			IsNationalTeam: false,
			Price:          price,
			PhotoURL:       card.PhotoURL,
			IleagueToken:   card.Token,
			JerseyNumber:   card.Jersey,
			IsActive:       true,
		})
		inserted++
		bySource[source]++
		*review = append(*review, reviewRow{
			Slug: card.Slug, Name: card.Name, Club: clubName,
			Jersey: card.Jersey, Position: position, Source: source,
		})
	}
	log.Printf("  %s: %d masuk (ileague:%d known:%d nomor:%d fallback:%d)",
		clubSlug, inserted, bySource["ileague"], bySource["known"], bySource["number"], bySource["fallback"])
}

// resolvePosition menentukan posisi + sumbernya untuk satu kartu pemain.
func resolvePosition(card ileagueCard) (position, source string) {
	if pos, ok := normalizePositionStrict(card.RawPosition); ok {
		return pos, "ileague"
	}
	if pos, ok := matchKnownPlayer(card.Name, card.Slug); ok {
		return pos, "known"
	}
	if pos, ok := inferPositionByNumber(card.Jersey); ok {
		return pos, "number"
	}
	return "MID", "fallback"
}

// knownPositions berisi pemain yang posisinya dipastikan dari pengetahuan
// umum (timnas, eks-timnas, asing top, veteran). Kunci = nama lengkap.
// Hanya yang high-confidence; sisanya lewat nomor punggung / koreksi CSV.
var knownPositions = map[string]string{
	// --- Kiper ---
	"nadeo arga winata":              "GK",
	"muchamad aqil savik":            "GK",
	"syahrul trisna fadillah":        "GK",
	"muhammad adi satryo":            "GK",
	"gianluca claudio pandeynuwu":    "GK",
	"awan setho raharjo":             "GK",
	"muhammad ridho":                 "GK",
	"sonny ricardo marciano stevens": "GK",
	"miswar saputra nurdin":          "GK",
	"cyrus ashkon margono":           "GK",
	"ernando ari sutaryadi":          "GK",
	"m reza arya pratama":            "GK",
	"teja paku alam":                 "GK",
	"andritany ardhiyasa":            "GK",
	"cahya supriadi":                 "GK",
	"kurniawan kartika ajie":         "GK",
	"hilman syah":                    "GK",
	"ega rizky pramana":              "GK",
	"fitrul dwi rustapa":             "GK",
	// --- Bek ---
	"rizky ridho ramadhani":       "DEF",
	"jordi amat":                  "DEF",
	"shayne pattynama":            "DEF",
	"pratama arhan":               "DEF",
	"ilham rio fahmi":             "DEF",
	"radovan pankov":              "DEF",
	"bagas adi nugroho":           "DEF",
	"rio fahmi":                   "DEF",
	"hansamu yama pranata":        "DEF",
	"alfeandra dewangga santosa":  "DEF",
	"johan ahmat farizi":          "DEF",
	"thomas anton rudolph lam":    "DEF",
	"rizky dwi febrianto":         "DEF",
	"tim henri victor geypens":    "DEF",
	"ricky fajrin saputra":        "DEF",
	"muhammad ferarri":            "DEF",
	"frengky deaner missa":        "DEF",
	"muhammad rifad marasabessy":  "DEF",
	"komang teguh trisnanda":      "DEF",
	"diego robbie michiels":       "DEF",
	"reva adi utama":              "DEF",
	"damion onandi lowe":          "DEF",
	"alta ballah":                 "DEF",
	"moh edo febriansah":          "DEF",
	"yance sayuri":                "DEF",
	"sandy henny walsh":           "DEF",
	"patricio martin matricardi":  "DEF",
	"danijel loncar":              "DEF",
	"denis kolinger":              "DEF",
	"muhammad fajar fathurrahman": "DEF",
	"dony tri pamungkas":          "DEF",
	"koko ari araya":              "DEF",
	"brandon marsel scheunemann":  "DEF",
	"ardi idrus":                  "DEF",
	"victor luiz prestes filho":   "DEF",
	"dusan lagator":               "DEF",
	"henhen herdiana":             "DEF",
	"fachruddin wahyudi aryanto":  "DEF",
	"jajang mulyana":              "DEF",
	"christophe nduwarugira":      "DEF",
	// --- Gelandang ---
	"witan sulaiman":                       "MID",
	"kwon chang hoon":                      "MID",
	"kwon changhoon":                       "MID",
	"dendi santoso":                        "MID",
	"septian david maulana":                "MID",
	"brandon james wilson":                 "MID",
	"tim charles pieter receveur":          "MID",
	"i kadek agung widnyana putra":         "MID",
	"irfan jaya":                           "MID",
	"moussa sidibe":                        "MID",
	"ryan kurnia":                          "MID",
	"jefferson brenes rojas":               "MID",
	"ivar jenner":                          "MID",
	"ricki kambuaya":                       "MID",
	"alexis nahuel messidoro":              "MID",
	"paulo oktavianus sitanggang":          "MID",
	"paulo domingos gali da costa freitas": "MID",
	"hugo gomes dos santos silva":          "MID",
	"frets listanto butuan":                "MID",
	"abrizal umanailo":                     "MID",
	"krisna bayu otto kartika":             "MID",
	"tyronne gustavo del pino ramos":       "MID",
	"ahmad agung setia budi":               "MID",
	"riyatno abiyoso":                      "MID",
	"malik risaldi":                        "MID",
	"rachmat irianto":                      "MID",
	"marc anthony klok":                    "MID",
	"thom jan marinus haye":                "MID",
	"gakuto notsuda":                       "MID",
	"adam alis setyano":                    "MID",
	"ragnar anthonius maria oratmangoen":   "MID",
	"saddil ramdani":                       "MID",
	"fabio da silva calonego":              "MID",
	"stjepan loncar":                       "MID",
	"ananda raehan alief":                  "MID",
	"kodai tanaka":                         "MID",
	"riko simanjuntak":                     "MID",
	"kim jeffrey kurniawan":                "MID",
	"yakob sayuri":                         "MID",
	// --- Penyerang ---
	"alexander jeremejeff":           "FWD",
	"ramadhan sananta":               "FWD",
	"ivan mamut":                     "FWD",
	"eksel timothy joseph runtukahu": "FWD",
	"mohammad rafli ariyanto":        "FWD",
	"david aparecido da silva":       "FWD",
	"jens raven":                     "FWD",
	"adrian dalmau vaquer":           "FWD",
	"nermin haljeta":                 "FWD",
	"muhammad dimas drajad":          "FWD",
	"rafael william struick":         "FWD",
}

// normTokens menormalisasi nama menjadi token-token huruf kecil.
func normTokens(s string) []string {
	s = strings.ToLower(s)
	for _, r := range []string{".", "-", "_", "'", "’", "/"} {
		s = strings.ReplaceAll(s, r, " ")
	}
	return strings.Fields(s)
}

// matchKnownPlayer mencocokkan kartu (nama + slug ileague) dengan knownPositions.
// Butuh >=2 token SAMA PERSIS (min 3 huruf) — tanpa tebak inisial, tanpa
// token umum. Ini mencegah false positive seperti "M. DYAR" -> Marasabessy
// (berbagi "muhammad") atau "NATHAN" -> Tjoe-A-On (satu nama depan sama
// tapi orang berbeda). Hasil deterministik (tak tergantung urutan map).
func matchKnownPlayer(cardName, cardSlug string) (string, bool) {
	combined := append(normTokens(cardName), normTokens(cardSlug)...)
	if len(combined) == 0 {
		return "", false
	}
	bestPos, bestShared := "", 0
	for known, pos := range knownPositions {
		if shared := sharedFullTokens(combined, normTokens(known)); shared > bestShared {
			bestPos, bestShared = pos, shared
		}
	}
	if bestShared >= 2 {
		return bestPos, true
	}
	return "", false
}

// stopTokens adalah partikel nama yang terlalu umum untuk dijadikan bukti
// (gelar, patronimik Bali, partikel Portugis) — diabaikan saat mencocokkan.
var stopTokens = map[string]bool{
	"muhammad": true, "mohammad": true, "moh": true,
	"putra": true, "putri": true,
	"dos": true, "das": true, "del": true,
}

func sharedFullTokens(a, b []string) int {
	seen := map[string]bool{}
	setB := map[string]bool{}
	for _, y := range b {
		if len(y) >= 3 && !stopTokens[y] {
			setB[y] = true
		}
	}
	shared := 0
	for _, x := range a {
		if len(x) < 3 || stopTokens[x] || seen[x] {
			continue // abaikan token pendek/umum & ganda (nama & slug sering sama)
		}
		seen[x] = true
		if setB[x] {
			shared++
		}
	}
	return shared
}

// ileagueCard adalah satu kartu pemain di tab #squad halaman klub.
type ileagueCard struct {
	Name        string
	Slug        string
	Token       string
	PhotoURL    string
	Nationality string
	Jersey      int
	RawPosition string // kosong di markup saat ini; siap bila situs menampilkannya
}

// fetchIleagueCards mengambil semua kartu pemain (#squad .item-player) sebuah klub.
// Kartu ofisial (foto-ofisial / tanpa link singleplayer) otomatis terfilter.
func fetchIleagueCards(c *colly.Collector, clubSlug string) []ileagueCard {
	cards := []ileagueCard{}

	c.OnHTML("#squad .item-player", func(e *colly.HTMLElement) {
		href := e.ChildAttr("a[href*='singleplayer']", "href")
		photoURL := e.ChildAttr("img[alt='foto-pemain']", "src")
		name := strings.TrimSpace(e.ChildText("td[colspan='2']"))
		if href == "" || photoURL == "" || name == "" {
			return
		}
		slug, token := parsePlayerLink(href)
		if slug == "" {
			return
		}
		name = displayName(name, slug)
		jersey, _ := strconv.Atoi(strings.TrimSpace(e.ChildText(".number-player")))
		nationality := ""
		e.ForEach("table tr", func(_ int, row *colly.HTMLElement) {
			cells := row.ChildTexts("td")
			if len(cells) == 2 && strings.EqualFold(strings.TrimSpace(cells[0]), "negara") {
				nationality = strings.TrimSpace(cells[1])
			}
		})
		cards = append(cards, ileagueCard{
			Name:        name,
			Slug:        slug,
			Token:       token,
			PhotoURL:    ensureAbsURL(photoURL),
			Nationality: nationality,
			Jersey:      jersey,
			RawPosition: strings.TrimSpace(e.ChildText(".position, .pos, .posisi")),
		})
	})

	url := fmt.Sprintf("%s/clubs/single/%s/%s", baseURL, leagueSlug, clubSlug)
	if err := c.Visit(url); err != nil {
		log.Printf("  Warning: could not scrape players for %s: %v", clubSlug, err)
		return nil
	}
	return cards
}

// displayName menentukan nama tampil pemain. Situs sering menyingkat
// ("T.Paku Alam", "R. RIDHO") tapi slug URL selalu nama lengkap
// ("teja_paku_alam"). Bila kartu disingkat (ada titik) atau slug lebih
// lengkap, pakai versi slug; jika tidak, kapitalisasi nama kartu.
func displayName(cardName, slug string) string {
	if strings.Contains(cardName, ".") {
		if full := nameFromSlug(slug); full != "" {
			return full
		}
	}
	if len(normTokens(slug)) > len(normTokens(cardName)) {
		return nameFromSlug(slug)
	}
	return titleWords(cardName)
}

// nameFromSlug mengubah "teja_paku_alam" menjadi "Teja Paku Alam".
func nameFromSlug(slug string) string {
	s := slug
	for _, r := range []string{"_", ".", "-", ","} {
		s = strings.ReplaceAll(s, r, " ")
	}
	return titleWords(strings.Join(strings.Fields(s), " "))
}

func titleWords(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// inferPositionByNumber menebak posisi dari nomor punggung.
// Hanya nomor-nomor klasik yang hampir pasti (ok=true); sisanya ok=false
// agar tidak mengarang posisi pemain.
func inferPositionByNumber(jersey int) (string, bool) {
	switch jersey {
	case 1:
		return "GK", true // hampir selalu kiper utama
	case 2, 3, 4, 5:
		return "DEF", true // nomor bek klasik (RB/LB/CB)
	case 7, 8, 10, 11:
		return "MID", true // winger & playmaker
	case 9:
		return "FWD", true // striker klasik
	}
	return "", false
}

// parsePlayerLink memecah link detail pemain menjadi slug + token,
// mis. .../muchamad_aqil_savik?token=XYZ= -> ("muchamad_aqil_savik", "XYZ=")
func parsePlayerLink(href string) (slug, token string) {
	parts := strings.SplitN(href, "?", 2)
	segs := strings.Split(strings.TrimSuffix(parts[0], "/"), "/")
	if len(segs) > 0 {
		slug = segs[len(segs)-1]
	}
	if len(parts) == 2 {
		for _, kv := range strings.Split(parts[1], "&") {
			if k, v, ok := strings.Cut(kv, "="); ok && k == "token" {
				token = v
			}
		}
	}
	return slug, token
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
		INSERT INTO clubs (id, name, short_name, slug, logo_url, stadium, city, tier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name,
			short_name = EXCLUDED.short_name,
			logo_url = EXCLUDED.logo_url,
			stadium = EXCLUDED.stadium,
			city = EXCLUDED.city,
			tier = EXCLUDED.tier
	`, club.ID, club.Name, club.ShortName, club.Slug, club.LogoURL, club.Stadium, club.City, club.Tier)
}

func (s *Scraper) upsertPlayer(player models.Player) {
	s.db.Exec(`
		INSERT INTO players (id, name, slug, club_id, position, nationality, is_national_team, price, photo_url, ileague_token, jersey_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name,
			club_id = EXCLUDED.club_id,
			position = EXCLUDED.position,
			nationality = EXCLUDED.nationality,
			price = EXCLUDED.price,
			photo_url = EXCLUDED.photo_url,
			updated_at = NOW()
	`, player.ID, player.Name, player.Slug, player.ClubID, player.Position, player.Nationality,
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
		name    string
		slug    string
		stadium string
		city    string
		tier    int
	}{
		{"AREMA FC", "AREMA_FC", "Kanjuruhan", "Malang", 1},
		{"BALI UNITED FC", "BALI_UNITED_FC", "Kapten I Wayan Dipta", "Gianyar", 1},
		{"BHAYANGKARA PRESISI LAMPUNG FC", "BHAYANGKARA_PRESISI_LAMPUNG_FC", "PKOR Sumpah Pemuda", "Lampung", 2},
		{"BORNEO FC SAMARINDA", "BORNEO_FC_SAMARINDA", "Segiri", "Samarinda", 2},
		{"DEWA UNITED BANTEN FC", "DEWA_UNITED_BANTEN_FC", "Banten International Stadium", "Serang", 2},
		{"GARUDAYAKSA FC", "GARUDAYAKSA_FC", "Pakansari", "Bogor", 3},
		{"ISENMULANG KALTENG FC", "ISENMULANG_KALTENG_FC", "Tuah Pahoe", "Palangka Raya", 3},
		{"JAVA UNITED FC", "JAVA_UNITED_FC", "Jatidiri", "Semarang", 3},
		{"MADURA UNITED FC", "MADURA_UNITED_FC", "Gelora Madura Ratu Pamelingan", "Pamekasan", 2},
		{"PERSEBAYA SURABAYA", "PERSEBAYA_SURABAYA", "Gelora Bung Tomo", "Surabaya", 1},
		{"PERSIB BANDUNG", "PERSIB_BANDUNG", "Gelora Bandung Lautan Api", "Bandung", 1},
		{"PERSIJA JAKARTA", "PERSIJA_JAKARTA", "Gelora Bung Karno", "Jakarta", 1},
		{"PERSIJAP JEPARA", "PERSIJAP_JEPARA", "Gelora Bumi Kartini", "Jepara", 3},
		{"PERSIK KEDIRI", "PERSIK_KEDIRI", "Brawijaya", "Kediri", 2},
		{"PERSITA", "PERSITA", "Indomilk Arena", "Tangerang", 2},
		{"PSIM YOGYAKARTA", "PSIM_YOGYAKARTA", "Sultan Agung", "Yogyakarta", 3},
		{"PSM MAKASSAR", "PSM_MAKASSAR", "Gelora B.J. Habibie", "Makassar", 1},
		{"PSS SLEMAN", "PSS_SLEMAN_", "Maguwoharjo", "Sleman", 2},
	}

	for _, club := range defaultClubs {
		s.db.Exec(`
			INSERT INTO clubs (id, name, short_name, slug, stadium, city, tier)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (slug) DO NOTHING
		`, uuid.New().String(), club.name, shortNameFrom(club.name), club.slug, club.stadium, club.city, club.tier)
	}
	log.Printf("✅ Inserted %d default clubs", len(defaultClubs))
}

// --- Utils ---

func newCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		colly.AllowURLRevisit(),
	)
	// ileague.id lambat di balik Cloudflare, kasih timeout longgar
	c.SetRequestTimeout(90 * time.Second)
	return c
}

// shortNameFrom membuat singkatan dari inisial tiap kata, mis. "PERSIJA JAKARTA" -> "PJ"
func shortNameFrom(name string) string {
	words := strings.Fields(strings.ToUpper(name))
	var b strings.Builder
	for _, w := range words {
		if len(w) == 0 {
			continue
		}
		b.WriteByte(w[0])
		if b.Len() >= 3 {
			break
		}
	}
	return b.String()
}

func normalizePosition(raw string) string {
	pos, _ := normalizePositionStrict(raw)
	return pos
}

// normalizePositionStrict memetakan teks posisi ke GK/DEF/MID/FWD.
// ok=false bila teks kosong / tidak dikenal -> panggilannya bisa skip
// (jangan tebak posisi pemain).
func normalizePositionStrict(raw string) (pos string, ok bool) {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	switch {
	case upper == "GK" || upper == "GOALKEEPER" || upper == "KIPER" || upper == "PENJAGA GAWANG":
		return "GK", true
	case upper == "DEF" || upper == "DEFENDER" || upper == "CB" || upper == "LB" || upper == "RB" || upper == "BEK" || upper == "BELAKANG" || upper == "BERTAHAN":
		return "DEF", true
	case upper == "MID" || upper == "MIDFIELDER" || upper == "CM" || upper == "CDM" || upper == "CAM" || upper == "AM" || upper == "DM" || upper == "GELANDANG" || upper == "TENGAH":
		return "MID", true
	case upper == "FWD" || upper == "FORWARD" || upper == "ST" || upper == "CF" || upper == "LW" || upper == "RW" || upper == "STRIKER" || upper == "PENYERANG" || upper == "DEPAN":
		return "FWD", true
	default:
		return "MID", false
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
