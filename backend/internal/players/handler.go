package players

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/fantasysuperleague/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// GetPlayers godoc
// GET /api/players
// Query params: position, club_id, search, min_price, max_price, page, limit
func (h *Handler) GetPlayers(c *gin.Context) {
	var filter models.PlayerFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	query := `
		SELECT 
			p.id, p.name, p.slug, p.club_id, p.position, p.nationality,
			p.is_national_team, p.price, p.photo_url, p.jersey_number,
			p.is_active, p.total_points,
			c.id, c.name, c.short_name, c.slug, c.logo_url
		FROM players p
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE p.is_active = TRUE
	`

	args := []interface{}{}
	argIdx := 1

	if filter.Position != "" {
		query += fmt.Sprintf(" AND p.position = $%d", argIdx)
		args = append(args, filter.Position)
		argIdx++
	}
	if filter.ClubID != "" {
		query += fmt.Sprintf(" AND p.club_id = $%d", argIdx)
		args = append(args, filter.ClubID)
		argIdx++
	}
	if filter.Search != "" {
		// unaccent agar "balsa" tetap ketemu "Balša Sekulić"
		query += fmt.Sprintf(" AND unaccent(LOWER(p.name)) LIKE unaccent(LOWER($%d))", argIdx)
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.MinPrice > 0 {
		query += fmt.Sprintf(" AND p.price >= $%d", argIdx)
		args = append(args, filter.MinPrice)
		argIdx++
	}
	if filter.MaxPrice > 0 {
		query += fmt.Sprintf(" AND p.price <= $%d", argIdx)
		args = append(args, filter.MaxPrice)
		argIdx++
	}

	// Count total for pagination
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS subq"
	var total int
	if err := h.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Add ordering and pagination
	query += fmt.Sprintf(
		" ORDER BY p.total_points DESC, p.price DESC LIMIT $%d OFFSET $%d",
		argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	players := []models.Player{}
	for rows.Next() {
		var p models.Player
		var club models.Club
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.ClubID, &p.Position, &p.Nationality,
			&p.IsNationalTeam, &p.Price, &p.PhotoURL, &p.JerseyNumber,
			&p.IsActive, &p.TotalPoints,
			&club.ID, &club.Name, &club.ShortName, &club.Slug, &club.LogoURL,
		)
		if err != nil {
			continue
		}
		p.Club = &club
		players = append(players, p)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  players,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
		"pages": (total + filter.Limit - 1) / filter.Limit,
	})
}

// GetPlayer godoc
// GET /api/players/:id
func (h *Handler) GetPlayer(c *gin.Context) {
	playerID := c.Param("id")

	var p models.Player
	var club models.Club
	err := h.db.QueryRow(`
		SELECT 
			p.id, p.name, p.slug, p.club_id, p.position, p.nationality,
			p.is_national_team, p.price, p.photo_url, p.jersey_number,
			p.is_active, p.total_points,
			c.id, c.name, c.short_name, c.slug, c.logo_url
		FROM players p
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE p.id = $1
	`, playerID).Scan(
		&p.ID, &p.Name, &p.Slug, &p.ClubID, &p.Position, &p.Nationality,
		&p.IsNationalTeam, &p.Price, &p.PhotoURL, &p.JerseyNumber,
		&p.IsActive, &p.TotalPoints,
		&club.ID, &club.Name, &club.ShortName, &club.Slug, &club.LogoURL,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	p.Club = &club

	// Get last 5 gameweek stats
	statRows, err := h.db.Query(`
		SELECT 
			ps.gameweek_id, ps.minutes_played, ps.goals, ps.assists,
			ps.clean_sheet, ps.yellow_cards, ps.red_cards, ps.saves,
			ps.bonus, ps.fantasy_points,
			gw.number, gw.name
		FROM player_stats ps
		JOIN gameweeks gw ON ps.gameweek_id = gw.id
		WHERE ps.player_id = $1
		ORDER BY gw.number DESC
		LIMIT 5
	`, playerID)
	if err == nil {
		defer statRows.Close()
		stats := []map[string]interface{}{}
		for statRows.Next() {
			var gwID, gwName string
			var gwNum, mins, goals, assists, yc, rc, saves, bonus, pts int
			var cleanSheet bool
			statRows.Scan(&gwID, &mins, &goals, &assists, &cleanSheet, &yc, &rc, &saves, &bonus, &pts, &gwNum, &gwName)
			stats = append(stats, map[string]interface{}{
				"gameweek_id":    gwID,
				"gameweek_name":  gwName,
				"gameweek_num":   gwNum,
				"minutes_played": mins,
				"goals":          goals,
				"assists":        assists,
				"clean_sheet":    cleanSheet,
				"yellow_cards":   yc,
				"red_cards":      rc,
				"saves":          saves,
				"bonus":          bonus,
				"fantasy_points": pts,
			})
		}
		c.JSON(http.StatusOK, gin.H{"player": p, "stats": stats})
		return
	}

	c.JSON(http.StatusOK, gin.H{"player": p})
}

// GetClubs godoc
// GET /api/clubs
func (h *Handler) GetClubs(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, short_name, slug, logo_url, stadium, city, tier
		FROM clubs
		ORDER BY tier DESC, name ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	clubs := []models.Club{}
	for rows.Next() {
		var club models.Club
		rows.Scan(&club.ID, &club.Name, &club.ShortName, &club.Slug, &club.LogoURL, &club.Stadium, &club.City, &club.Tier)
		clubs = append(clubs, club)
	}

	c.JSON(http.StatusOK, gin.H{"data": clubs})
}
