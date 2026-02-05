package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// PingHandler handles the /ping endpoint
func PingHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if DB is still connected
		err := db.Ping()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message":   "pong",
				"db_status": "disconnected",
				"error":     err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "pong",
			"db_status": "connected",
		})
	}
}

// DBInfoHandler handles the /db-info endpoint
func DBInfoHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			version     string
			numTables   int
			currentDB   string
			currentUser string
		)

		db.QueryRow("SELECT version()").Scan(&version)
		db.QueryRow("SELECT current_database()").Scan(&currentDB)
		db.QueryRow("SELECT current_user").Scan(&currentUser)
		db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&numTables)

		c.JSON(http.StatusOK, gin.H{
			"version":      version,
			"database":     currentDB,
			"user":         currentUser,
			"tables_count": numTables,
		})
	}
}

// SetupRouter initializes and configures the Gin router with all endpoints
func SetupRouter(db *sqlx.DB) *gin.Engine {
	r := gin.Default()

	// Health check endpoints
	r.GET("/ping", PingHandler(db))
	r.GET("/db-info", DBInfoHandler(db))

	return r
}
