package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Book represents a book in the database
type Book struct {
	ID          int    `db:"id" json:"id"`
	Title       string `db:"title" json:"title" binding:"required"`
	Author      string `db:"author" json:"author" binding:"required"`
	Description string `db:"description" json:"description"`
	ISBN        string `db:"isbn" json:"isbn"`
	CoverURL    string `db:"cover_url" json:"cover_url"`
}

// UserBook represents a user's book relationship
type UserBook struct {
	ID       int    `db:"id" json:"id"`
	UserID   int    `db:"user_id" json:"user_id" binding:"required"`
	BookID   int    `db:"book_id" json:"book_id" binding:"required"`
	Status   string `db:"status" json:"status" binding:"required"`
	Rating   *int   `db:"rating" json:"rating"`
	Review   string `db:"review" json:"review"`
	Progress int    `db:"progress" json:"progress"`
}

// AddBookRequest for adding a new book
type AddBookRequest struct {
	Book
}

// UpdateRatingRequest for updating a book rating
type UpdateRatingRequest struct {
	Rating int    `json:"rating" binding:"required,min=1,max=5"`
	Review string `json:"review"`
}

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

// AddBookHandler handles adding a new book to the books table
func AddBookHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AddBookRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query := `INSERT INTO books (title, author, description, isbn, cover_url) 
				  VALUES ($1, $2, $3, $4, $5) RETURNING id`

		var bookID int
		err := db.QueryRow(query, req.Title, req.Author, req.Description, req.ISBN, req.CoverURL).Scan(&bookID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add book"})
			return
		}

		req.ID = bookID
		c.JSON(http.StatusCreated, gin.H{
			"message": "Book added successfully",
			"book":    req,
		})
	}
}

// AddUserBookHandler adds a book to user's collection
func AddUserBookHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UserBook
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query := `INSERT INTO user_books (user_id, book_id, status, rating, review, progress) 
				  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

		var userBookID int
		err := db.QueryRow(query, req.UserID, req.BookID, req.Status, req.Rating, req.Review, req.Progress).Scan(&userBookID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add book to user collection"})
			return
		}

		req.ID = userBookID
		c.JSON(http.StatusCreated, gin.H{
			"message":   "Book added to collection successfully",
			"user_book": req,
		})
	}
}

// UpdateRatingHandler updates the rating and review for a user's book
func UpdateRatingHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userBookID := c.Param("id")

		var req UpdateRatingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query := `UPDATE user_books 
				  SET rating = $1, review = $2, updated_at = NOW() 
				  WHERE id = $3`

		result, err := db.Exec(query, req.Rating, req.Review, userBookID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rating"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User book not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Rating updated successfully",
			"rating":  req.Rating,
			"review":  req.Review,
		})
	}
}

// RemoveBookHandler removes a book from user's collection
func RemoveBookHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userBookID := c.Param("id")

		query := `DELETE FROM user_books WHERE id = $1`

		result, err := db.Exec(query, userBookID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove book"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User book not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Book removed from collection successfully",
		})
	}
}

// GetUserBooksHandler retrieves all books for a user
func GetUserBooksHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.Query("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query parameter is required"})
			return
		}

		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
			return
		}

		query := `SELECT ub.id, ub.user_id, ub.book_id, ub.status, ub.rating, ub.review, ub.progress,
				  b.title, b.author, b.description, b.isbn, b.cover_url
				  FROM user_books ub
				  JOIN books b ON ub.book_id = b.id
				  WHERE ub.user_id = $1`

		type UserBookWithDetails struct {
			UserBook
			Title       string `db:"title" json:"title"`
			Author      string `db:"author" json:"author"`
			Description string `db:"description" json:"description"`
			ISBN        string `db:"isbn" json:"isbn"`
			CoverURL    string `db:"cover_url" json:"cover_url"`
		}

		var books []UserBookWithDetails
		err = db.Select(&books, query, userID)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve books"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"books":   books,
			"count":   len(books),
		})
	}
}

// SetupRouter initializes and configures the Gin router with all endpoints
func SetupRouter(db *sqlx.DB) *gin.Engine {
	r := gin.Default()

	// Health check endpoints
	r.GET("/ping", PingHandler(db))
	r.GET("/db-info", DBInfoHandler(db))

	// Book management endpoints
	r.POST("/books", AddBookHandler(db))
	r.POST("/user-books", AddUserBookHandler(db))
	r.GET("/user-books", GetUserBooksHandler(db))
	r.PUT("/user-books/:id/rating", UpdateRatingHandler(db))
	r.DELETE("/user-books/:id", RemoveBookHandler(db))

	return r
}
