package auth

import (
	"database/sql"
	"go_mangahub/manga_hub/pkg/models"
	"go_mangahub/manga_hub/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func ChangePassword(c *gin.Context, db *sql.DB) {
	// extract userID from context
	userID, _ := c.Get("user_id")
	
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Require current and new password",
		})
		return
	}

	// Fetch the current hashed password
	var hashedPassword string
	err := db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).
	Scan(&hashedPassword)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// compare hashedPassword to currentPassword
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.CurrentPassword))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Current password is incorrect",
		})
		return
	}

	// validate new password strength
	if !utils.IsPasswordStrong(req.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password must be at least 8 characters with mixed case and numbers",
		})
		return
	}

	// hash new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error hashing new password",
		})
		return
	} 


	// update database where password_hash
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(newHashedPassword), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": "Password changed successfully!",
	})
}