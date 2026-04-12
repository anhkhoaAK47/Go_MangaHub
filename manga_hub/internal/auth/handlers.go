package auth

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"go_mangahub/manga_hub/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email	 string `json:"email" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func HandleRegister(c *gin.Context, db *sql.DB) {
	var req AuthRequest
	// validate input
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	// Check email format
	if (!utils.IsValidEmail(req.Email)) {
		c.JSON(http.StatusBadRequest, gin.H{
		"error": "Invalid email format",
		"suggestion": "Please provide a valid email address",
	})
		return
	}

	// Check password strength
	if (!utils.IsPasswordStrong(req.Password)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password too weak",
			"suggestion": "Password must be at least 8 characters with mixed case and numbers",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Generate unique userID
	userID := uuid.New().String()

	// Insert into data
	query := `INSERT INTO users (id, username, password_hash, email) VALUES (?, ?, ?, ?)`
	_, err = db.Exec(query, userID, req.Username, string(hashedPassword), req.Email)

	if err != nil {
		// Check for duplicate username
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.username") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Registration failed: Username '" + req.Username + "' already exists",
				"suggestion": "Try: mangahub auth login --username " + req.Username,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
		
	c.JSON(http.StatusOK, gin.H{
		"message": "Account created successfully!",
		"username": req.Username,
		"user_id": userID,
		"email": req.Email,
		"created_at": time.Now().UTC(),
	})	

}

func HandleLogin(c *gin.Context, db *sql.DB, jwtSecret string) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch user from DB
	var userID, storedHash string
	query := `SELECT id, password_hash FROM users WHERE username = ?`
	err := db.QueryRow(query, req.Username).Scan(&userID, &storedHash)

	// Check if username doesn't exist
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Login failed: Account not found",
			"suggestion": "Try: mangahub auth register --username " + req.Username,
		})
		return
	} else if err != nil { 
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Login failed: Server connection error",
			"suggestion": "Check server status: mangahub server status",
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))

	// Check for valid credentials
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Login failed: Invalid credentials",
			"suggestion": "Check your username and password",
		})
		return
	}

	// Generate token
	token, err := utils.GenerateJWT(userID, jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// OK Status
	c.JSON(http.StatusOK, gin.H{
		"success": "Login successful!",
		"message": "Welcome back, " + req.Username + "!",
		"username": req.Username,
		"token": token,
		"expires_at": time.Now().Add(time.Hour * 24),
	})
}

