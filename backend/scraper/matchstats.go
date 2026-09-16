package scraper

// Pencatatan statistik gameweek dari 3 sumber ileague.id:
//  1. Match Summary PDF (assets/.../matchsum/...) -> skor, susunan XI +
//     cadangan, kode posisi, menit main (dari anotasi substitusi).
//  2. Halaman result HTML -> gol & kartu (ikon) per pemain.
//  3. AJAX get_statistikpemain -> assist, saves, kebobolan per pemain.
//
// Alur per match: result page -> link PDF + events + ID pemain AJAX ->
// download PDF -> gabung semua -> upsert player_stats.

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/fantasysuperleague/backend/internal/models"
	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
)

// --- Struktur perantara ---

type pdfPlayer struct {
	Number  int
	PosCode string
	Name    string
	Starter bool
	Captain bool
	offAt   int // menit keluar definitif (starter), 0 = main penuh
	onAt    int // menit masuk definitif (cadangan), 0 = starter/tidak main
	bareMin int // menit telanjang pertama (butuh pasangan substitusi)
}

type pdfMatch struct {
	HomeSlug  string
	AwaySlug  string
	HomeGoals int
	AwayGoals int
	Home      []pdfPlayer
	Away      []pdfPlayer
}

type matchEvent struct {
	Goals  int
	Yellow int
	Red    int
	Minute int // menit kejadian terakhir (untuk kartu merah)
}

type resultPage struct {
	pdfURL  string
	events  map[string]*matchEvent // nama UPPER -> agregat
	ajaxIDs map[string]string      // nama UPPER -> playerID AJAX
	matchID string
}

// --- Entry point (menggantikan versi lama) ---

func (s *Scraper) ScrapeMatchStats(gameweekNum int) error {
	log.Printf("📊 Scraping stats for gameweek %d...", gameweekNum)

	var gwID string
	err := s.db.QueryRow("SELECT id FROM gameweeks WHERE number = $1", gameweekNum).Scan(&gwID)
	if err == sql.ErrNoRows {
		gwID = uuid.New().String()
		s.db.Exec(
			`INSERT INTO gameweeks (id, number, name, is_finished) VALUES ($1, $2, $3, TRUE)`,
			gwID, gameweekNum, fmt.Sprintf("Gameweek %d", gameweekNum),
		)
	} else if err != nil {
		return err
	}

	// Halaman fixtures/index/<N> = daftar laga matchweek N.
	c := newCollector()
	var matchURLs []string
	seen := map[string]bool{}
	c.OnHTML("a[href*='result/detail']", func(e *colly.HTMLElement) {
		u := ensureAbsURL(e.Attr("href"))
		if !seen[u] {
			seen[u] = true
			matchURLs = append(matchURLs, u)
		}
	})
	url := fmt.Sprintf("%s/fixtures/index/%s/%d", baseURL, leagueSlug, gameweekNum)
	if err := c.Visit(url); err != nil {
		return fmt.Errorf("fixtures gw %d: %w", gameweekNum, err)
	}
	log.Printf("  %d laga ditemukan", len(matchURLs))

	for _, mu := range matchURLs {
		s.scrapeMatchResult(gwID, mu)
		time.Sleep(500 * time.Millisecond)
	}

	s.calculateGameweekPoints(gwID)
	log.Printf("✅ Gameweek %d stats scraped!", gameweekNum)
	return nil
}

func (s *Scraper) scrapeMatchResult(gwID, matchURL string) {
	rp, err := parseResultPage(matchURL)
	if err != nil {
		log.Printf("  Warning result %s: %v", matchURL, err)
		return
	}
	if rp.pdfURL == "" {
		log.Printf("  Skip (belum ada match summary): %s", matchURL)
		return
	}

	// Slug klub dari URL lebih andal daripada header PDF.
	homeSlug, awaySlug, _ := clubSlugsFromResultURL(matchURL)
	pm, err := downloadAndParseMatchPDF(rp.pdfURL,
		strings.ReplaceAll(homeSlug, "_", " "),
		strings.ReplaceAll(awaySlug, "_", " "))
	if err != nil {
		log.Printf("  Warning PDF %s: %v", rp.pdfURL, err)
		return
	}
	pm.HomeSlug, pm.AwaySlug = homeSlug, awaySlug

	// Peta nama-normal -> pemain DB (id). Nama PDF & slug DB sama-sama
	// lengkap sehingga pencocokan persis; fallback token bersama >=3.
	type dbPlayer struct {
		ID string
	}
	byName := map[string]string{} // key normal -> playerID
	rows, err := s.db.Query("SELECT id, name, slug FROM players")
	if err != nil {
		log.Printf("  Warning players query: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, slug string
		rows.Scan(&id, &name, &slug)
		for _, k := range []string{normKey(slug), normKey(name)} {
			if k != "" {
				if _, exists := byName[k]; !exists {
					byName[k] = id
				}
			}
		}
	}
	lookup := func(displayName string) string {
		if id, ok := byName[normKey(displayName)]; ok {
			return id
		}
		bestID, best := "", 0
		for k, id := range byName {
			if n := sharedFullTokens(normTokens(displayName), normTokens(k)); n > best {
				best, bestID = n, id
			}
		}
		if best >= 3 {
			return bestID
		}
		return ""
	}

	// Agregat per pemain DB.
	type agg struct {
		st      models.PlayerStats
		minutes int
		concede int
	}
	aggs := map[string]*agg{}
	ensure := func(displayName string) *agg {
		id := lookup(displayName)
		if id == "" {
			return nil
		}
		if a, exists := aggs[id]; exists {
			return a
		}
		a := &agg{st: models.PlayerStats{ID: uuid.New().String(), PlayerID: id, GameweekID: gwID}}
		aggs[id] = a
		return a
	}

	// 1. PDF: menit + kebobolan tim.
	applySquad := func(squad []pdfPlayer, conceded int) {
		for _, pl := range squad {
			mins := 0
			if pl.Starter {
				mins = 90
				if pl.offAt > 0 {
					mins = pl.offAt
				}
			} else if pl.onAt > 0 {
				mins = 90 - pl.onAt
			}
			if mins <= 0 {
				continue
			}
			a := ensure(pl.Name)
			if a == nil {
				log.Printf("    ? pemain tak dikenal: %s", pl.Name)
				continue
			}
			a.minutes = mins
			a.concede = conceded
		}
	}
	applySquad(pm.Home, pm.AwayGoals)
	applySquad(pm.Away, pm.HomeGoals)

	// 2. HTML events: gol & kartu.
	for name, ev := range rp.events {
		a := ensure(name)
		if a == nil {
			log.Printf("    ? event tak dikenal: %s", name)
			continue
		}
		a.st.Goals += ev.Goals
		a.st.YellowCards += ev.Yellow
		a.st.RedCards += ev.Red
		if ev.Red > 0 && ev.Minute > 0 && (a.minutes == 0 || ev.Minute < a.minutes) {
			a.minutes = ev.Minute
		}
		if a.minutes == 0 {
			a.minutes = 90 // cetak gol/kartu => pasti main
		}
	}

	// 3. AJAX: assist, saves, validasi silang gol/kartu.
	for name, pid := range rp.ajaxIDs {
		ax, err := fetchAjaxStats(rp.matchID, pid)
		if err != nil {
			continue
		}
		a := ensure(name)
		if a == nil {
			continue
		}
		a.st.Assists += ax["ASSIST"]
		a.st.Saves += ax["SAVES"]
		if g := ax["GOAL"] + ax["PENALTY GOAL"]; g > a.st.Goals {
			a.st.Goals = g
		}
		if y := ax["YELLOW CARD"]; y > a.st.YellowCards {
			a.st.YellowCards = y
		}
		if a.minutes == 0 {
			a.minutes = 90
		}
		time.Sleep(300 * time.Millisecond)
	}

	n := 0
	for _, a := range aggs {
		a.st.MinutesPlayed = a.minutes
		a.st.GoalsConceded = a.concede
		a.st.CleanSheet = a.concede == 0
		s.upsertPlayerStats(a.st)
		n++
	}
	log.Printf("  %s: %d pemain tercatat", matchURL, n)
}

func normKey(s string) string {
	return strings.Join(normTokens(s), " ")
}

// --- Halaman result HTML ---

var onclickRe = regexp.MustCompile(`statistikPemain\((\d+),([\d.]+)\)`)

func parseResultPage(matchURL string) (*resultPage, error) {
	rp := &resultPage{
		events:  map[string]*matchEvent{},
		ajaxIDs: map[string]string{},
	}
	c := newCollector()

	c.OnHTML("a[href*='matchsum']", func(e *colly.HTMLElement) {
		if rp.pdfURL == "" {
			rp.pdfURL = ensureAbsURL(e.Attr("href"))
		}
	})

	// Daftar event per tim: <div class="team"> ... <ul><li>NAMA [ikon] menit</li>
	c.OnHTML("div.team ul li", func(e *colly.HTMLElement) {
		name, minute := splitEventText(e.Text)
		if name == "" {
			return
		}
		key := strings.ToUpper(strings.Join(strings.Fields(name), " "))
		ev := rp.events[key]
		if ev == nil {
			ev = &matchEvent{}
			rp.events[key] = ev
		}
		e.ForEach("i", func(_ int, ico *colly.HTMLElement) {
			class := strings.ToLower(ico.Attr("class") + " " + ico.Attr("style"))
			switch {
			case strings.Contains(class, "futbol"):
				ev.Goals++
			case strings.Contains(class, "red"):
				ev.Red++
				if minute > 0 {
					ev.Minute = minute
				}
			case strings.Contains(class, "yellow"), strings.Contains(class, "square"):
				ev.Yellow++
			}
		})
	})

	// ID pemain AJAX: <a onclick="...statistikPemain(matchID,playerID...)">NAMA</a>
	c.OnHTML("a[onclick*='statistikPemain']", func(e *colly.HTMLElement) {
		m := onclickRe.FindStringSubmatch(e.Attr("onclick"))
		if len(m) != 3 {
			return
		}
		if rp.matchID == "" {
			rp.matchID = m[1]
		}
		name := strings.ToUpper(strings.Join(strings.Fields(e.Text), " "))
		if name != "" {
			if _, exists := rp.ajaxIDs[name]; !exists {
				rp.ajaxIDs[name] = strings.TrimSuffix(m[2], ".0")
			}
		}
	})

	if err := c.Visit(matchURL); err != nil {
		return nil, err
	}
	return rp, nil
}

var eventMinuteRe = regexp.MustCompile(`^(.*?)\s*(\d+\+?\d*'|HT)?\s*$`)

// eventMarkerRe membuang penanda seperti (P) penalti di akhir teks event.
var eventMarkerRe = regexp.MustCompile(`\s*\([A-Z]+\)\s*$`)

func splitEventText(t string) (string, int) {
	t = eventMarkerRe.ReplaceAllString(strings.TrimSpace(t), "")
	m := eventMinuteRe.FindStringSubmatch(t)
	if m == nil {
		return "", 0
	}
	return m[1], parseMinute(m[2])
}

func parseMinute(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.ToUpper(s) == "HT" {
		return 45
	}
	if i := strings.Index(s, "+"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSuffix(s, "'")
	n, _ := strconv.Atoi(s)
	return n
}

// --- AJAX statistik pemain ---

var ajaxCellRe = regexp.MustCompile(`<td[^>]*>([^<>]+)</td>\s*<td[^>]*>([^<>]+)</td>`)

func fetchAjaxStats(matchID, playerID string) (map[string]int, error) {
	out := map[string]int{}
	url := fmt.Sprintf("%s/result/get_statistikpemain/%s/%s", baseURL, matchID, playerID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(nil))
	if err != nil {
		return out, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return out, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return out, err
	}
	for _, m := range ajaxCellRe.FindAllStringSubmatch(string(body), -1) {
		label := strings.ToUpper(strings.TrimSpace(m[1]))
		val, _ := strconv.Atoi(strings.TrimSpace(m[2]))
		out[label] += val
	}
	return out, nil
}

// --- Match Summary PDF ---

var pdfScoreRe = regexp.MustCompile(`(?i)\bVS\b\s*(.+?)\s+(\d+)\s*-\s*(\d+)`)

func downloadAndParseMatchPDF(pdfURL, homeName, awayName string) (*pdfMatch, error) {
	tmp, err := os.CreateTemp("", "matchsum-*.pdf")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	req, err := http.NewRequest("GET", pdfURL, nil)
	if err != nil {
		tmp.Close()
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		tmp.Close()
		return nil, err
	}
	defer res.Body.Close()
	if _, err := io.Copy(tmp, res.Body); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()

	f, r, err := pdf.Open(tmpPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var sb strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		rows, err := p.GetTextByRow()
		if err != nil {
			continue
		}
		for _, row := range rows {
			for _, w := range row.Content {
				sb.WriteString(w.S)
				sb.WriteString(" ")
			}
			sb.WriteString("\n")
		}
	}
	return parseMatchText(sb.String(), homeName, awayName)
}

type pdfSection int

const (
	secNone pdfSection = iota
	secHomeXI
	secAwayXI
	secHomeSubs
	secAwaySubs
	secDone
)

// entri pemain: [ NN ] POS ... (nama + anotasi menyusul)
var pdfEntryRe = regexp.MustCompile(`\[\s*(\d+)\s*\]\s*([A-Z]{1,3})\s+`)

// minutePrefixRe mengenali awalan anotasi menit: HT, 76', 76'(1), 45+1'.
var minutePrefixRe = regexp.MustCompile(`^(HT|\d+\+?\d*')`)

// Kata kunci struktur dipisah jadi baris sendiri agar terdeteksi walau
// menempel teks lain dalam satu baris PDF.
var secKeyRe = regexp.MustCompile(`SUBSTITUTES|HEAD COACH|MATCH EVENTS|OFFICIAL`)

func parseMatchText(text, homeName, awayName string) (*pdfMatch, error) {
	pm := &pdfMatch{Home: []pdfPlayer{}, Away: []pdfPlayer{}}

	// Skor dulu dari teks asli: "HOME VS AWAY X - Y".
	for _, ln := range strings.Split(text, "\n") {
		if m := pdfScoreRe.FindStringSubmatch(ln); m != nil {
			pm.HomeGoals, _ = strconv.Atoi(m[2])
			pm.AwayGoals, _ = strconv.Atoi(m[3])
			break
		}
	}

	// Pecah jadi segmen berlabel: nama tim & kata kunci struktur.
	text = strings.ReplaceAll(text, homeName, "\n@@HOME@@\n")
	if awayName != "" && awayName != homeName {
		text = strings.ReplaceAll(text, awayName, "\n@@AWAY@@\n")
	}
	text = secKeyRe.ReplaceAllString(text, "\n$0\n")
	lines := strings.Split(text, "\n")

	sec := secNone
	subBlocks := 0
	teamFor := func() *[]pdfPlayer {
		switch sec {
		case secHomeXI, secHomeSubs:
			return &pm.Home
		case secAwayXI, secAwaySubs:
			return &pm.Away
		}
		return nil
	}
	var cur *pdfPlayer
	flush := func() {
		if cur != nil && cur.Name != "" {
			if dst := teamFor(); dst != nil {
				*dst = append(*dst, *cur)
			}
		}
		cur = nil
	}

	for _, rawLn := range lines {
		upper := strings.TrimSpace(rawLn)
		switch {
		case upper == "@@HOME@@":
			flush()
			sec = secHomeXI
			continue
		case upper == "@@AWAY@@":
			flush()
			sec = secAwayXI
			continue
		case strings.HasPrefix(upper, "SUBSTITUTES"):
			flush()
			// Blok SUBSTITUTES pertama = cadangan home, kedua = away.
			subBlocks++
			if subBlocks == 1 {
				sec = secHomeSubs
			} else {
				sec = secAwaySubs
			}
			continue
		case strings.HasPrefix(upper, "HEAD COACH"),
			strings.HasPrefix(upper, "MATCH EVENTS"),
			strings.HasPrefix(upper, "OFFICIAL"):
			flush()
			sec = secDone
			continue
		}
		if sec == secNone || sec == secDone {
			continue
		}

		pos := pdfEntryRe.FindStringSubmatchIndex(rawLn)
		for pos != nil {
			num, _ := strconv.Atoi(rawLn[pos[2]:pos[3]])
			code := rawLn[pos[4]:pos[5]]
			flush()
			cur = &pdfPlayer{
				Number:  num,
				PosCode: code,
				Starter: sec == secHomeXI || sec == secAwayXI,
			}
			rest := rawLn[pos[1]:]
			words := strings.Fields(rest)
			nameParts := []string{}
			i := 0
			for ; i < len(words); i++ {
				w := words[i]
				if w == "(C)" || w == "[" || minutePrefixRe.MatchString(w) || !isNameWord(w) {
					break
				}
				nameParts = append(nameParts, w)
			}
			cur.Name = strings.Join(nameParts, " ")
			for ; i < len(words); i++ {
				w := words[i]
				switch {
				case w == "(C)":
					cur.Captain = true
				case w == "[":
					i = len(words)
				case minutePrefixRe.MatchString(w):
					// Menit injury time (45+1') selalu menit gol/kartu,
					// bukan substitusi (pergantian dicatat HT/bulat).
					if !strings.Contains(w, "+") {
						commitMinute(cur, w)
					}
				}
				if w == "[" {
					break
				}
			}
			next := pdfEntryRe.FindStringSubmatchIndex(rest)
			if next == nil {
				break
			}
			for k := range next {
				if next[k] >= 0 {
					next[k] += pos[1]
				}
			}
			pos = next
		}
	}
	flush()

	// Pasangan menit telanjang: XI yang keluar di menit M selalu dibarengi
	// cadangan yang masuk di menit M yang sama (satu tim). Tanpa pasangan,
	// menit telanjang adalah menit gol/kartu -> abaikan.
	for _, squad := range []*[]pdfPlayer{&pm.Home, &pm.Away} {
		onMinutes := map[int]bool{}
		for _, p := range *squad {
			if !p.Starter && p.onAt > 0 {
				onMinutes[p.onAt] = true
			}
		}
		for i := range *squad {
			p := &(*squad)[i]
			if p.Starter && p.offAt == 0 && p.bareMin > 0 {
				if onMinutes[p.bareMin] {
					p.offAt = p.bareMin
				} else {
					log.Printf("    (info) menit %d %s diduga event, bukan substitusi", p.bareMin, p.Name)
				}
			}
		}
	}
	return pm, nil
}

// commitMinute mencatat menit definitif: berpenanda (N)/HT, atau menit
// telanjang (ditentukan pasangan substitusinya belakangan).
func commitMinute(p *pdfPlayer, tok string) {
	m := minutePrefixRe.FindString(tok)
	if m == "" {
		return
	}
	marked := strings.Contains(tok, "(") || strings.ToUpper(strings.TrimSuffix(tok, ",")) == "HT"
	v := parseMinute(m)
	if p.Starter {
		if marked {
			p.offAt = v
		} else if p.bareMin == 0 {
			p.bareMin = v
		}
	} else {
		if p.onAt == 0 {
			p.onAt = v
		}
	}
}

func isNameWord(w string) bool {
	if w == "" {
		return false
	}
	for _, r := range w {
		// Huruf kapital Unicode (Ç, Ñ, Š ...) + tanda nama (. ' ’ - ,).
		if unicode.IsUpper(r) || r == '.' || r == '\'' || r == '’' || r == '-' || r == ',' {
			continue
		}
		return false
	}
	return true
}

func clubSlugsFromResultURL(matchURL string) (home, away string, ok bool) {
	parts := strings.Split(strings.TrimSuffix(matchURL, "/"), "/")
	// .../result/detail/<league>/<date>/<HOME>/<AWAY>
	if len(parts) < 3 {
		return "", "", false
	}
	return parts[len(parts)-2], parts[len(parts)-1], true
}
