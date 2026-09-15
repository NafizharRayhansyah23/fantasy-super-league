package teams

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/fantasysuperleague/backend/internal/models"
	"github.com/fantasysuperleague/backend/internal/scoring"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// GetMyTeam godoc
// GET /api/team
func (h *Handler) GetMyTeam(c *gin.Context) {
	userID, _ := c.Get("userID")

	var team models.FantasyTeam
	err := h.db.QueryRow(`
		SELECT id, user_id, team_name, total_budget, remaining_budget,
		       wildcard_used, free_transfers, total_points, created_at
		FROM fantasy_teams WHERE user_id = $1
	`, userID).Scan(
		&team.ID, &team.UserID, &team.TeamName, &team.TotalBudget,
		&team.RemainingBudget, &team.WildcardUsed, &team.FreeTransfers,
		&team.TotalPoints, &team.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tim belum dibuat", "has_team": false})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get players in team
	rows, err := h.db.Query(`
		SELECT 
			ftp.id, ftp.team_id, ftp.player_id, ftp.is_captain, ftp.is_vice_captain,
			ftp.is_starting, ftp.position_slot,
			p.id, p.name, p.slug, p.position, p.price, p.photo_url, p.total_points, p.jersey_number,
			c.id, c.name, c.short_name, c.logo_url
		FROM fantasy_team_players ftp
		JOIN players p ON ftp.player_id = p.id
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE ftp.team_id = $1
		ORDER BY ftp.position_slot ASC
	`, team.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var slot models.FantasyTeamPlayer
		var player models.Player
		var club models.Club
		rows.Scan(
			&slot.ID, &slot.TeamID, &slot.PlayerID, &slot.IsCaptain, &slot.IsViceCaptain,
			&slot.IsStarting, &slot.PositionSlot,
			&player.ID, &player.Name, &player.Slug, &player.Position, &player.Price,
			&player.PhotoURL, &player.TotalPoints, &player.JerseyNumber,
			&club.ID, &club.Name, &club.ShortName, &club.LogoURL,
		)
		player.Club = &club
		slot.Player = &player
		team.Players = append(team.Players, slot)
	}

	c.JSON(http.StatusOK, gin.H{"data": team, "has_team": true})
}

// SaveTeam godoc
// POST /api/team
func (h *Handler) SaveTeam(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req models.SaveTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate team composition (15 players: 2 GK, 5 DEF, 5 MID, 3 FWD)
	if err := validateTeamComposition(h.db, req.Players); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate total price
	totalPrice, err := calculateTotalPrice(h.db, req.Players)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate price"})
		return
	}
	if totalPrice > 100.0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Budget melebihi batas (Rp 100 juta)",
			"total_price": totalPrice,
		})
		return
	}

	// Check if team already exists
	var existingTeamID string
	err = h.db.QueryRow("SELECT id FROM fantasy_teams WHERE user_id = $1", userID).Scan(&existingTeamID)

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	if existingTeamID != "" {
		// Update existing team
		_, err = tx.Exec(
			`UPDATE fantasy_teams SET team_name = $1, remaining_budget = $2, updated_at = NOW() WHERE id = $3`,
			req.TeamName, 100.0-totalPrice, existingTeamID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update team"})
			return
		}
		// Remove old players
		tx.Exec("DELETE FROM fantasy_team_players WHERE team_id = $1", existingTeamID)
		// Insert new players
		if err := insertTeamPlayers(tx, existingTeamID, req.Players); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Create new team
		teamID := uuid.New().String()
		_, err = tx.Exec(
			`INSERT INTO fantasy_teams (id, user_id, team_name, remaining_budget) VALUES ($1, $2, $3, $4)`,
			teamID, userID, req.TeamName, 100.0-totalPrice,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create team"})
			return
		}
		if err := insertTeamPlayers(tx, teamID, req.Players); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save team"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Tim berhasil disimpan!",
		"remaining_budget": 100.0 - totalPrice,
		"total_price":     totalPrice,
	})
}

// MakeTransfer godoc
// POST /api/team/transfer
func (h *Handler) MakeTransfer(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req models.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get team
	var team models.FantasyTeam
	err := h.db.QueryRow(
		`SELECT id, remaining_budget, wildcard_used, free_transfers FROM fantasy_teams WHERE user_id = $1`,
		userID,
	).Scan(&team.ID, &team.RemainingBudget, &team.WildcardUsed, &team.FreeTransfers)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tim tidak ditemukan"})
		return
	}

	// Validate player out is in team
	var slotID, positionSlot string
	var isStarting bool
	err = h.db.QueryRow(
		`SELECT id, position_slot, is_starting FROM fantasy_team_players WHERE team_id = $1 AND player_id = $2`,
		team.ID, req.PlayerOutID,
	).Scan(&slotID, &positionSlot, &isStarting)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pemain tidak ada di tim kamu"})
		return
	}

	// Get prices for both players
	var priceOut, priceIn float64
	h.db.QueryRow("SELECT price FROM players WHERE id = $1", req.PlayerOutID).Scan(&priceOut)
	h.db.QueryRow("SELECT price FROM players WHERE id = $1", req.PlayerInID).Scan(&priceIn)

	// Check budget
	budgetAfterTransfer := team.RemainingBudget + priceOut - priceIn
	if budgetAfterTransfer < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Budget tidak cukup",
			"deficit": -budgetAfterTransfer,
		})
		return
	}

	// Calculate points deduction (unless wildcard or free transfer)
	pointsDeducted := 0
	isWildcard := req.UseWildcard && !team.WildcardUsed

	if !isWildcard && team.FreeTransfers <= 0 {
		pointsDeducted = 4
	}

	// Get active gameweek
	var gameweekID string
	h.db.QueryRow("SELECT id FROM gameweeks WHERE is_active = TRUE LIMIT 1").Scan(&gameweekID)

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Perform transfer: replace player_out with player_in in the same slot
	tx.Exec(
		`UPDATE fantasy_team_players SET player_id = $1 WHERE id = $2`,
		req.PlayerInID, slotID,
	)

	// Update budget
	tx.Exec(
		`UPDATE fantasy_teams SET remaining_budget = $1, free_transfers = GREATEST(free_transfers - 1, 0), updated_at = NOW() WHERE id = $2`,
		budgetAfterTransfer, team.ID,
	)

	// If wildcard, mark it used and no deduction
	if isWildcard {
		tx.Exec(`UPDATE fantasy_teams SET wildcard_used = TRUE WHERE id = $1`, team.ID)
	}

	// Record transfer history
	if gameweekID != "" {
		tx.Exec(`
			INSERT INTO transfer_history (id, team_id, gameweek_id, player_out_id, player_in_id, is_wildcard, points_deducted)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, uuid.New().String(), team.ID, gameweekID, req.PlayerOutID, req.PlayerInID, isWildcard, pointsDeducted)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to make transfer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Transfer berhasil!",
		"points_deducted":  pointsDeducted,
		"remaining_budget": budgetAfterTransfer,
		"is_wildcard":      isWildcard,
	})
}

// GetGameweekPoints godoc
// GET /api/team/points/:gameweek_number
func (h *Handler) GetGameweekPoints(c *gin.Context) {
	userID, _ := c.Get("userID")
	gwNum := c.Param("gameweek_number")

	var teamID string
	h.db.QueryRow("SELECT id FROM fantasy_teams WHERE user_id = $1", userID).Scan(&teamID)
	if teamID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tim tidak ditemukan"})
		return
	}

	rows, err := h.db.Query(`
		SELECT 
			p.id, p.name, p.position, p.photo_url,
			ftp.is_captain, ftp.is_vice_captain, ftp.is_starting,
			ps.goals, ps.assists, ps.minutes_played, ps.clean_sheet,
			ps.yellow_cards, ps.red_cards, ps.saves, ps.bonus, ps.fantasy_points
		FROM fantasy_team_players ftp
		JOIN players p ON ftp.player_id = p.id
		LEFT JOIN player_stats ps ON ps.player_id = p.id
		LEFT JOIN gameweeks gw ON ps.gameweek_id = gw.id AND gw.number = $1
		WHERE ftp.team_id = $2
		ORDER BY ftp.position_slot
	`, gwNum, teamID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type PlayerPoints struct {
		PlayerID       string `json:"player_id"`
		Name           string `json:"name"`
		Position       string `json:"position"`
		PhotoURL       string `json:"photo_url"`
		IsCaptain      bool   `json:"is_captain"`
		IsViceCaptain  bool   `json:"is_vice_captain"`
		IsStarting     bool   `json:"is_starting"`
		Goals          int    `json:"goals"`
		Assists        int    `json:"assists"`
		MinutesPlayed  int    `json:"minutes_played"`
		CleanSheet     bool   `json:"clean_sheet"`
		YellowCards    int    `json:"yellow_cards"`
		RedCards       int    `json:"red_cards"`
		Saves          int    `json:"saves"`
		Bonus          int    `json:"bonus"`
		FantasyPoints  int    `json:"fantasy_points"`
		TotalPoints    int    `json:"total_points"` // doubles if captain
	}

	results := []PlayerPoints{}
	totalPoints := 0

	for rows.Next() {
		var pp PlayerPoints
		rows.Scan(
			&pp.PlayerID, &pp.Name, &pp.Position, &pp.PhotoURL,
			&pp.IsCaptain, &pp.IsViceCaptain, &pp.IsStarting,
			&pp.Goals, &pp.Assists, &pp.MinutesPlayed, &pp.CleanSheet,
			&pp.YellowCards, &pp.RedCards, &pp.Saves, &pp.Bonus, &pp.FantasyPoints,
		)
		pp.TotalPoints = pp.FantasyPoints
		if pp.IsCaptain {
			pp.TotalPoints *= 2
		}
		if pp.IsStarting {
			totalPoints += pp.TotalPoints
		}
		results = append(results, pp)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":         results,
		"total_points": totalPoints,
	})
}

// GetLeaderboard godoc
// GET /api/leaderboard
func (h *Handler) GetLeaderboard(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT 
			u.id, u.username, u.team_name,
			COALESCE(ft.total_points, 0) as total_points
		FROM users u
		LEFT JOIN fantasy_teams ft ON u.id = ft.user_id
		ORDER BY total_points DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	leaderboard := []models.LeaderboardEntry{}
	rank := 1
	for rows.Next() {
		var entry models.LeaderboardEntry
		rows.Scan(&entry.UserID, &entry.Username, &entry.TeamName, &entry.TotalPoints)
		entry.Rank = rank
		leaderboard = append(leaderboard, entry)
		rank++
	}

	c.JSON(http.StatusOK, gin.H{"data": leaderboard})
}

// --- Helpers ---

func validateTeamComposition(db *sql.DB, slots []models.TeamPlayerSlot) error {
	if len(slots) != 15 {
		return fmt.Errorf("tim harus memiliki tepat 15 pemain (saat ini: %d)", len(slots))
	}

	positionCounts := map[string]int{}
	captainCount := 0
	vcCount := 0
	startingCount := 0

	for _, slot := range slots {
		var pos string
		db.QueryRow("SELECT position FROM players WHERE id = $1", slot.PlayerID).Scan(&pos)
		positionCounts[pos]++
		if slot.IsCaptain {
			captainCount++
		}
		if slot.IsViceCaptain {
			vcCount++
		}
		if slot.IsStarting {
			startingCount++
		}
	}

	if positionCounts["GK"] != 2 {
		return fmt.Errorf("harus ada 2 GK (saat ini: %d)", positionCounts["GK"])
	}
	if positionCounts["DEF"] != 5 {
		return fmt.Errorf("harus ada 5 DEF (saat ini: %d)", positionCounts["DEF"])
	}
	if positionCounts["MID"] != 5 {
		return fmt.Errorf("harus ada 5 MID (saat ini: %d)", positionCounts["MID"])
	}
	if positionCounts["FWD"] != 3 {
		return fmt.Errorf("harus ada 3 FWD (saat ini: %d)", positionCounts["FWD"])
	}
	if captainCount != 1 {
		return fmt.Errorf("harus ada tepat 1 kapten")
	}
	if vcCount != 1 {
		return fmt.Errorf("harus ada tepat 1 wakil kapten")
	}
	if startingCount != 11 {
		return fmt.Errorf("harus ada 11 pemain starting")
	}

	return nil
}

func calculateTotalPrice(db *sql.DB, slots []models.TeamPlayerSlot) (float64, error) {
	total := 0.0
	for _, slot := range slots {
		var price float64
		if err := db.QueryRow("SELECT price FROM players WHERE id = $1", slot.PlayerID).Scan(&price); err != nil {
			return 0, err
		}
		total += price
	}
	return total, nil
}

func insertTeamPlayers(tx *sql.Tx, teamID string, slots []models.TeamPlayerSlot) error {
	for _, slot := range slots {
		_, err := tx.Exec(`
			INSERT INTO fantasy_team_players (id, team_id, player_id, is_captain, is_vice_captain, is_starting, position_slot)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, uuid.New().String(), teamID, slot.PlayerID, slot.IsCaptain, slot.IsViceCaptain, slot.IsStarting, slot.PositionSlot)
		if err != nil {
			return fmt.Errorf("failed to insert player slot: %w", err)
		}
	}
	return nil
}

// needed for import
var _ = scoring.CalculatePoints
