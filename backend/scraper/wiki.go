package scraper

// Posisi pemain diambil dari tabel "Current squad" Wikipedia (kolom Pos:
// GK/DF/MF/FW) yang tidak dipublikasi ileague.id. Diambil sekali per klub
// via MediaWiki API (ringan, ~1 request/klub) lalu dicocokkan nama ke kartu
// ileague dengan aturan yang sama ketatnya (lihat matchKnownPlayer).
// Klub tanpa halaman/tabel skuad -> dilewati (fallback ke sumber lain).

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// wikiTitles memetakan slug klub DB -> judul artikel Wikipedia (EN).
// Diverifikasi satu per satu (Agustus 2026); semua 18 klub punya tabel skuad.
var wikiTitles = map[string]string{
	"AREMA_FC":                       "Arema F.C.",
	"BALI_UNITED_FC":                 "Bali United F.C.",
	"BHAYANGKARA_PRESISI_LAMPUNG_FC": "Bhayangkara Presisi Lampung F.C.",
	"BORNEO_FC_SAMARINDA":            "Borneo F.C. Samarinda",
	"DEWA_UNITED_BANTEN_FC":          "Dewa United Banten F.C.",
	"GARUDAYAKSA_FC":                 "Garudayaksa F.C.",
	"ISENMULANG_KALTENG_FC":          "Isenmulang Kalteng F.C.",
	"JAVA_UNITED_FC":                 "Java United F.C.",
	"MADURA_UNITED_FC":               "Madura United F.C.",
	"PERSEBAYA_SURABAYA":             "Persebaya Surabaya",
	"PERSIB_BANDUNG":                 "Persib Bandung",
	"PERSIJA_JAKARTA":                "Persija Jakarta",
	"PERSIJAP_JEPARA":                "Persijap Jepara",
	"PERSIK_KEDIRI":                  "Persik Kediri",
	"PERSITA":                        "Persita Tangerang",
	"PSIM_YOGYAKARTA":                "PSIM Yogyakarta",
	"PSM_MAKASSAR":                   "PSM Makassar",
	"PSS_SLEMAN_":                    "PSS Sleman",
}

// wikiPlayer adalah satu baris skuad Wikipedia.
type wikiPlayer struct {
	Number   string
	Position string // GK | DEF | MID | FWD (sudah dipetakan)
	Name     string
}

var wikiHTTP = &http.Client{Timeout: 30 * time.Second}

const wikiUA = "FantasySuperLeague/1.0 (local dev research; contact admin)"

// Dua format template skuad: {{Fs player|...}} dan {{Football squad player|...}}
// (Bali United memakai yang kedua). Pola mengizinkan satu level [[...]]
// di dalam (nama ber-link seperti [[Queven (footballer)|Queven]]).
var fsPlayerRe = regexp.MustCompile(`\{\{(?:[Ff]s player|[Ff]ootball squad player)\|((?:[^}]|\}[^}])*)\}\}`)

// fetchWikiSquad mengembalikan map nama-ternormalisasi -> posisi untuk sebuah klub.
// Key map = token-token nama digabung spasi agar pencocokan toleran terhadap
// singkatan ("R. RIDHO" cocok dengan "Rizky Ridho").
func fetchWikiSquad(clubSlug string) map[string]string {
	out := map[string]string{}
	title, ok := wikiTitles[clubSlug]
	if !ok || title == "" {
		return out
	}

	apiURL := "https://en.wikipedia.org/w/api.php?action=parse&page=" +
		url.QueryEscape(title) + "&prop=wikitext&format=json&redirects=1"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return out
	}
	req.Header.Set("User-Agent", wikiUA)

	// Retry dengan backoff karena Wikipedia me-rate-limit (429) bila agresif.
	var body []byte
	for attempt := 1; attempt <= 3; attempt++ {
		res, err := wikiHTTP.Do(req)
		if err != nil {
			log.Printf("  Warning: wiki %s percobaan %d: %v", clubSlug, attempt, err)
		} else {
			func() {
				defer res.Body.Close()
				if res.StatusCode == http.StatusOK {
					body, err = io.ReadAll(res.Body)
				} else {
					log.Printf("  Warning: wiki %s percobaan %d -> HTTP %d", clubSlug, attempt, res.StatusCode)
					err = fmt.Errorf("HTTP %d", res.StatusCode)
				}
			}()
			if err == nil && body != nil {
				break
			}
		}
		if attempt < 3 {
			time.Sleep(time.Duration(attempt*10) * time.Second)
		}
	}
	if body == nil {
		return out
	}
	// Bentuk parse.wikitext: {"*": "..."} — decode manual karena key "*".
	var raw struct {
		Parse map[string]json.RawMessage `json:"parse"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return out
	}
	wtRaw, ok := raw.Parse["wikitext"]
	if !ok {
		return out
	}
	var wtMap map[string]string
	if err := json.Unmarshal(wtRaw, &wtMap); err != nil {
		return out
	}
	wt := wtMap["*"]

	for _, m := range fsPlayerRe.FindAllStringSubmatch(wt, -1) {
		fields := map[string]string{}
		for _, kv := range strings.Split(m[1], "|") {
			if k, v, ok := strings.Cut(kv, "="); ok {
				fields[strings.TrimSpace(strings.ToLower(k))] = strings.TrimSpace(v)
			}
		}
		pos, ok := wikiPosition(fields["pos"])
		if !ok {
			continue
		}
		name := cleanWikiName(fields["name"])
		if name == "" {
			continue
		}
		out[strings.Join(normTokens(name), " ")] = pos
	}
	return out
}

// wikiPosition memetakan kode posisi Wikipedia ke kode internal.
func wikiPosition(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "GK", "GOALKEEPER":
		return "GK", true
	case "DF", "DEFENDER", "FB", "CB", "LB", "RB", "WB", "LWB", "RWB":
		return "DEF", true
	case "MF", "MIDFIELDER", "DM", "CM", "AM", "LM", "RM", "CDM", "CAM":
		return "MID", true
	case "FW", "FORWARD", "ST", "CF", "SS", "LW", "RW", "WG":
		return "FWD", true
	}
	return "", false
}

// cleanWikiName membersihkan markup wiki: [[Target|Tampil]] -> Tampil,
// [[Nama]] -> Nama, hapus <ref>, parenthetical, dan penanda kapten.
func cleanWikiName(raw string) string {
	s := raw
	for {
		start := strings.Index(s, "[[")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], "]]")
		if end < 0 {
			break
		}
		link := s[start+2 : start+end]
		display := link
		if i := strings.LastIndex(link, "|"); i >= 0 {
			display = link[i+1:]
		}
		s = s[:start] + display + s[start+end+2:]
	}
	// Sisa braket dari markup terpotong dibuang saja.
	s = strings.ReplaceAll(s, "[[", "")
	s = strings.ReplaceAll(s, "]]", "")
	// Hapus <ref>...</ref> dan penanda dalam kurung di akhir.
	for {
		start := strings.Index(s, "<ref")
		if start < 0 {
			break
		}
		if end := strings.Index(s[start:], "</ref>"); end >= 0 {
			s = s[:start] + s[start+end+len("</ref>"):]
		} else if end := strings.Index(s[start:], "/>"); end >= 0 {
			s = s[:start] + s[start+end+2:]
		} else {
			break
		}
	}
	if i := strings.Index(s, "("); i >= 0 {
		s = s[:i]
	}
	return strings.Join(strings.Fields(s), " ")
}

// matchWikiPlayer mencocokkan nama kartu ileague ke skuad wiki.
// Syarat: >=2 token penuh yang sama (aturan yang sama dengan knownPositions).
// Pengecualian: nama wiki satu kata (mononim seperti "Queven") cukup cocok
// persis satu token kartu/slug.
func matchWikiPlayer(cardName, cardSlug string, squad map[string]string) (string, bool) {
	if len(squad) == 0 {
		return "", false
	}
	combined := append(normTokens(cardName), normTokens(cardSlug)...)
	combinedSet := map[string]bool{}
	for _, t := range combined {
		combinedSet[t] = true
	}
	bestPos, bestShared := "", 0
	for name, pos := range squad {
		wt := normTokens(name)
		shared := sharedFullTokens(combined, wt)
		if len(wt) == 1 && len(wt[0]) >= 3 && combinedSet[wt[0]] {
			shared = 2 // mononim cocok persis
		}
		if shared > bestShared {
			bestPos, bestShared = pos, shared
		}
	}
	if bestShared >= 2 {
		return bestPos, true
	}
	return "", false
}
