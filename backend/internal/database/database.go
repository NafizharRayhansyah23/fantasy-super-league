package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Connect establishes a connection to PostgreSQL
func Connect() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if dbname == "" {
		dbname = "fantasysuperleague"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

// Migrate runs all database migrations in order
func Migrate(db *sql.DB) error {
	migrations := []string{
		createUsersTable,
		createClubsTable,
		createPlayersTable,
		createGameweeksTable,
		createPlayerStatsTable,
		createFantasyTeamsTable,
		createFantasyTeamPlayersTable,
		createTransferHistoryTable,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w\nQuery: %s", err, migration)
		}
	}

	return nil
}

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username    VARCHAR(50) UNIQUE NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    team_name   VARCHAR(100),
    total_points INTEGER DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);`

const createClubsTable = `
CREATE TABLE IF NOT EXISTS clubs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    short_name  VARCHAR(10),
    slug        VARCHAR(100) UNIQUE NOT NULL,
    logo_url    VARCHAR(500),
    stadium     VARCHAR(100),
    city        VARCHAR(100),
    tier        INTEGER DEFAULT 2,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);`

const createPlayersTable = `
CREATE TABLE IF NOT EXISTS players (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(100) UNIQUE NOT NULL,
    club_id         UUID REFERENCES clubs(id) ON DELETE SET NULL,
    position        VARCHAR(5) NOT NULL CHECK (position IN ('GK', 'DEF', 'MID', 'FWD')),
    nationality     VARCHAR(100),
    is_national_team BOOLEAN DEFAULT FALSE,
    price           DECIMAL(10,1) NOT NULL,
    photo_url       VARCHAR(500),
    ileague_token   VARCHAR(200),
    jersey_number   INTEGER,
    is_active       BOOLEAN DEFAULT TRUE,
    total_points    INTEGER DEFAULT 0,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);`

const createGameweeksTable = `
CREATE TABLE IF NOT EXISTS gameweeks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number          INTEGER UNIQUE NOT NULL,
    name            VARCHAR(50),
    is_active       BOOLEAN DEFAULT FALSE,
    is_finished     BOOLEAN DEFAULT FALSE,
    deadline        TIMESTAMP WITH TIME ZONE,
    start_date      DATE,
    end_date        DATE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);`

const createPlayerStatsTable = `
CREATE TABLE IF NOT EXISTS player_stats (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id       UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    gameweek_id     UUID NOT NULL REFERENCES gameweeks(id) ON DELETE CASCADE,
    minutes_played  INTEGER DEFAULT 0,
    goals           INTEGER DEFAULT 0,
    assists         INTEGER DEFAULT 0,
    clean_sheet     BOOLEAN DEFAULT FALSE,
    goals_conceded  INTEGER DEFAULT 0,
    yellow_cards    INTEGER DEFAULT 0,
    red_cards       INTEGER DEFAULT 0,
    saves           INTEGER DEFAULT 0,
    bonus           INTEGER DEFAULT 0,
    fantasy_points  INTEGER DEFAULT 0,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(player_id, gameweek_id)
);`

const createFantasyTeamsTable = `
CREATE TABLE IF NOT EXISTS fantasy_teams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_name       VARCHAR(100) NOT NULL,
    total_budget    DECIMAL(10,1) DEFAULT 100.0,
    remaining_budget DECIMAL(10,1) DEFAULT 100.0,
    wildcard_used   BOOLEAN DEFAULT FALSE,
    free_transfers  INTEGER DEFAULT 1,
    total_points    INTEGER DEFAULT 0,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);`

const createFantasyTeamPlayersTable = `
CREATE TABLE IF NOT EXISTS fantasy_team_players (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES fantasy_teams(id) ON DELETE CASCADE,
    player_id       UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    is_captain      BOOLEAN DEFAULT FALSE,
    is_vice_captain BOOLEAN DEFAULT FALSE,
    is_starting     BOOLEAN DEFAULT TRUE,
    position_slot   INTEGER NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(team_id, player_id),
    UNIQUE(team_id, position_slot)
);`

const createTransferHistoryTable = `
CREATE TABLE IF NOT EXISTS transfer_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES fantasy_teams(id) ON DELETE CASCADE,
    gameweek_id     UUID NOT NULL REFERENCES gameweeks(id),
    player_out_id   UUID NOT NULL REFERENCES players(id),
    player_in_id    UUID NOT NULL REFERENCES players(id),
    is_wildcard     BOOLEAN DEFAULT FALSE,
    points_deducted INTEGER DEFAULT 0,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);`
