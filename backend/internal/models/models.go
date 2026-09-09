package models

import (
	"time"
)

// User represents a registered user
type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	TeamName     string    `json:"team_name,omitempty" db:"team_name"`
	TotalPoints  int       `json:"total_points" db:"total_points"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Club represents a football club in Liga 1
type Club struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	ShortName string    `json:"short_name" db:"short_name"`
	Slug      string    `json:"slug" db:"slug"`
	LogoURL   string    `json:"logo_url" db:"logo_url"`
	Stadium   string    `json:"stadium" db:"stadium"`
	City      string    `json:"city" db:"city"`
	Tier      int       `json:"tier" db:"tier"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Player represents a football player
type Player struct {
	ID             string    `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Slug           string    `json:"slug" db:"slug"`
	ClubID         string    `json:"club_id" db:"club_id"`
	Club           *Club     `json:"club,omitempty"`
	Position       string    `json:"position" db:"position"` // GK, DEF, MID, FWD
	Nationality    string    `json:"nationality" db:"nationality"`
	IsNationalTeam bool      `json:"is_national_team" db:"is_national_team"`
	Price          float64   `json:"price" db:"price"` // in millions (Rp juta)
	PhotoURL       string    `json:"photo_url" db:"photo_url"`
	IleagueToken   string    `json:"ileague_token,omitempty" db:"ileague_token"`
	JerseyNumber   int       `json:"jersey_number" db:"jersey_number"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	TotalPoints    int       `json:"total_points" db:"total_points"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Gameweek represents a round of matches
type Gameweek struct {
	ID         string    `json:"id" db:"id"`
	Number     int       `json:"number" db:"number"`
	Name       string    `json:"name" db:"name"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	IsFinished bool      `json:"is_finished" db:"is_finished"`
	Deadline   time.Time `json:"deadline" db:"deadline"`
	StartDate  string    `json:"start_date" db:"start_date"`
	EndDate    string    `json:"end_date" db:"end_date"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// PlayerStats represents stats for a player in a gameweek
type PlayerStats struct {
	ID            string    `json:"id" db:"id"`
	PlayerID      string    `json:"player_id" db:"player_id"`
	Player        *Player   `json:"player,omitempty"`
	GameweekID    string    `json:"gameweek_id" db:"gameweek_id"`
	MinutesPlayed int       `json:"minutes_played" db:"minutes_played"`
	Goals         int       `json:"goals" db:"goals"`
	Assists       int       `json:"assists" db:"assists"`
	CleanSheet    bool      `json:"clean_sheet" db:"clean_sheet"`
	GoalsConceded int       `json:"goals_conceded" db:"goals_conceded"`
	YellowCards   int       `json:"yellow_cards" db:"yellow_cards"`
	RedCards      int       `json:"red_cards" db:"red_cards"`
	Saves         int       `json:"saves" db:"saves"`
	Bonus         int       `json:"bonus" db:"bonus"`
	FantasyPoints int       `json:"fantasy_points" db:"fantasy_points"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// FantasyTeam represents a user's fantasy team
type FantasyTeam struct {
	ID              string    `json:"id" db:"id"`
	UserID          string    `json:"user_id" db:"user_id"`
	TeamName        string    `json:"team_name" db:"team_name"`
	TotalBudget     float64   `json:"total_budget" db:"total_budget"`
	RemainingBudget float64   `json:"remaining_budget" db:"remaining_budget"`
	WildcardUsed    bool      `json:"wildcard_used" db:"wildcard_used"`
	FreeTransfers   int       `json:"free_transfers" db:"free_transfers"`
	TotalPoints     int       `json:"total_points" db:"total_points"`
	Players         []FantasyTeamPlayer `json:"players,omitempty"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// FantasyTeamPlayer represents a player slot in a fantasy team
type FantasyTeamPlayer struct {
	ID           string  `json:"id" db:"id"`
	TeamID       string  `json:"team_id" db:"team_id"`
	PlayerID     string  `json:"player_id" db:"player_id"`
	Player       *Player `json:"player,omitempty"`
	IsCaptain    bool    `json:"is_captain" db:"is_captain"`
	IsViceCaptain bool   `json:"is_vice_captain" db:"is_vice_captain"`
	IsStarting   bool    `json:"is_starting" db:"is_starting"`
	PositionSlot int     `json:"position_slot" db:"position_slot"`
}

// TransferHistory records each transfer
type TransferHistory struct {
	ID             string    `json:"id" db:"id"`
	TeamID         string    `json:"team_id" db:"team_id"`
	GameweekID     string    `json:"gameweek_id" db:"gameweek_id"`
	PlayerOutID    string    `json:"player_out_id" db:"player_out_id"`
	PlayerOut      *Player   `json:"player_out,omitempty"`
	PlayerInID     string    `json:"player_in_id" db:"player_in_id"`
	PlayerIn       *Player   `json:"player_in,omitempty"`
	IsWildcard     bool      `json:"is_wildcard" db:"is_wildcard"`
	PointsDeducted int       `json:"points_deducted" db:"points_deducted"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// --- Request/Response DTOs ---

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	TeamName string `json:"team_name" binding:"required,min=3,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type SaveTeamRequest struct {
	TeamName string             `json:"team_name" binding:"required"`
	Players  []TeamPlayerSlot   `json:"players" binding:"required"`
}

type TeamPlayerSlot struct {
	PlayerID     string `json:"player_id" binding:"required"`
	IsCaptain    bool   `json:"is_captain"`
	IsViceCaptain bool  `json:"is_vice_captain"`
	IsStarting   bool   `json:"is_starting"`
	PositionSlot int    `json:"position_slot" binding:"required"`
}

type TransferRequest struct {
	PlayerOutID string `json:"player_out_id" binding:"required"`
	PlayerInID  string `json:"player_in_id" binding:"required"`
	UseWildcard bool   `json:"use_wildcard"`
}

type LeaderboardEntry struct {
	Rank        int    `json:"rank"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	TeamName    string `json:"team_name"`
	TotalPoints int    `json:"total_points"`
}

type PlayerFilter struct {
	Position string  `form:"position"`
	ClubID   string  `form:"club_id"`
	Search   string  `form:"search"`
	MinPrice float64 `form:"min_price"`
	MaxPrice float64 `form:"max_price"`
	Page     int     `form:"page,default=1"`
	Limit    int     `form:"limit,default=20"`
}
