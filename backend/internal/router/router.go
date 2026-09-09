package router

import (
	"database/sql"
	"net/http"

	"github.com/fantasysuperleague/backend/internal/auth"
	"github.com/fantasysuperleague/backend/internal/players"
	"github.com/fantasysuperleague/backend/internal/teams"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Setup initializes and returns the Gin router with all routes
func Setup(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// CORS config — allow Next.js frontend
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "Fantasy Super League API"})
	})

	// Initialize handlers
	authHandler := auth.NewHandler(db)
	playerHandler := players.NewHandler(db)
	teamHandler := teams.NewHandler(db)

	api := r.Group("/api")
	{
		// --- Auth routes (public) ---
		authRoutes := api.Group("/auth")
		{
			authRoutes.POST("/register", authHandler.Register)
			authRoutes.POST("/login", authHandler.Login)
		}

		// --- Public data routes ---
		api.GET("/players", playerHandler.GetPlayers)
		api.GET("/players/:id", playerHandler.GetPlayer)
		api.GET("/clubs", playerHandler.GetClubs)
		api.GET("/gameweeks", func(c *gin.Context) {
			// handled inline for simplicity
			rows, err := db.Query(`
				SELECT id, number, name, is_active, is_finished, deadline
				FROM gameweeks ORDER BY number DESC
			`)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				return
			}
			defer rows.Close()
			gws := []gin.H{}
			for rows.Next() {
				var id, name string
				var num int
				var isActive, isFinished bool
				var deadline interface{}
				rows.Scan(&id, &num, &name, &isActive, &isFinished, &deadline)
				gws = append(gws, gin.H{
					"id": id, "number": num, "name": name,
					"is_active": isActive, "is_finished": isFinished, "deadline": deadline,
				})
			}
			c.JSON(http.StatusOK, gin.H{"data": gws})
		})
		api.GET("/leaderboard", teamHandler.GetLeaderboard)

		// --- Protected routes (require JWT) ---
		protected := api.Group("/")
		protected.Use(auth.JWTMiddleware())
		{
			protected.GET("/auth/me", authHandler.GetMe)

			// Team management
			protected.GET("/team", teamHandler.GetMyTeam)
			protected.POST("/team", teamHandler.SaveTeam)
			protected.POST("/team/transfer", teamHandler.MakeTransfer)
			protected.GET("/team/points/:gameweek_number", teamHandler.GetGameweekPoints)
		}
	}

	return r
}
