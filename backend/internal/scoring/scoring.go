package scoring

import "github.com/fantasysuperleague/backend/internal/models"

// CalculatePoints computes fantasy points for a player's gameweek stats
// Based on FPL-style scoring system adapted for Liga Indonesia
func CalculatePoints(stats models.PlayerStats, position string) int {
	points := 0

	// --- Appearance points ---
	if stats.MinutesPlayed >= 60 {
		points += 2
	} else if stats.MinutesPlayed > 0 {
		points += 1
	} else {
		// Didn't play, no points
		return 0
	}

	// --- Goal points ---
	switch position {
	case "GK", "DEF":
		points += stats.Goals * 6
	case "MID":
		points += stats.Goals * 5
	case "FWD":
		points += stats.Goals * 4
	}

	// --- Assist points ---
	points += stats.Assists * 3

	// --- Clean sheet points ---
	if stats.CleanSheet && stats.MinutesPlayed >= 60 {
		switch position {
		case "GK", "DEF":
			points += 4
		case "MID":
			points += 1
		}
	}

	// --- GK saves (every 3 saves = 1 point) ---
	if position == "GK" {
		points += stats.Saves / 3
	}

	// --- Goals conceded (GK and DEF, every 2 goals = -1 point) ---
	if position == "GK" || position == "DEF" {
		points -= stats.GoalsConceded / 2
	}

	// --- Card deductions ---
	points -= stats.YellowCards * 1
	points -= stats.RedCards * 3

	// --- Bonus points (admin-set, 1-3) ---
	points += stats.Bonus

	// Points can't go below -4 for a single match
	if points < -4 {
		points = -4
	}

	return points
}

// CalculateCaptainPoints doubles the captain's points
func CalculateCaptainPoints(basePoints int) int {
	return basePoints * 2
}

// GetPriceByPositionAndTier returns the suggested price for a player
// based on their position, club tier (1=big club, 2=mid, 3=small), 
// and whether they are in the national team
func GetPriceByPositionAndTier(position string, clubTier int, isNationalTeam bool) float64 {
	basePrice := map[string]map[int]float64{
		"GK":  {1: 6.5, 2: 5.5, 3: 4.5},
		"DEF": {1: 7.5, 2: 6.0, 3: 4.5},
		"MID": {1: 9.0, 2: 7.0, 3: 5.5},
		"FWD": {1: 10.5, 2: 8.0, 3: 6.0},
	}

	price, ok := basePrice[position][clubTier]
	if !ok {
		price = 5.0
	}

	// National team premium
	if isNationalTeam {
		price += 1.0
	}

	return price
}
