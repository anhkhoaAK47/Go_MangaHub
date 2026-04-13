package auth

import (
	"database/sql"
	"fmt"
	"go_mangahub/manga_hub/pkg/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CheckStatus(c *gin.Context, db *sql.DB) {
	// Get user_id from auth middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized: No user session found",
		})
		return
	}

	// fetch user details from database
	var user models.User
	query := `SELECT username, created_at FROM users WHERE id = ?`
	err := db.QueryRow(query, userID).Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("User %s not found", user.Username),
		})
		return
	}


	// Return user info
	c.JSON(http.StatusOK, gin.H{
		"isAuthenticated": true,
		"user": user,
	})
}