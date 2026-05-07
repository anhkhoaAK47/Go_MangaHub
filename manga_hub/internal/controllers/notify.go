package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go_mangahub/manga_hub/internal/udp"

	"github.com/gin-gonic/gin"
)

var notifyManager *udp.NotificationManager

// InitializeNotifyManager initializes the notification manager
func InitializeNotifyManager(manager *udp.NotificationManager) {
	notifyManager = manager
}

// Subscribe handles subscription to notifications
func Subscribe(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	mangaID := c.GetHeader("X-Manga-ID")

	if notifyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service not available"})
		return
	}

	if mangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga-id is required"})
		return
	}

	err := notifyManager.Subscribe(userID, mangaID, "")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"message": "Successfully subscribed to notifications",
		"status":  "subscribed",
	}

	if mangaID != "" {
		response["manga_id"] = mangaID
	}

	c.JSON(http.StatusOK, response)
}

// Unsubscribe handles unsubscription from notifications
func Unsubscribe(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	mangaID := c.GetHeader("X-Manga-ID")

	if notifyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service not available"})
		return
	}

	if mangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga-id is required"})
		return
	}

	err := notifyManager.Unsubscribe(userID, mangaID, "")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"message": "Successfully unsubscribed from notifications",
		"status":  "unsubscribed",
	}

	if mangaID != "" {
		response["manga_id"] = mangaID
	}

	
	c.JSON(http.StatusOK, response)
}

// GetPreferences retrieves user notification preferences
func GetPreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if notifyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service not available"})
		return
	}

	prefs, err := notifyManager.GetPreferences(userID)
	if err != nil {
		// Return default preferences if not found
		prefs = &udp.NotificationPreferences{
			UserID:               userID,
			SubscribedMangas:     []string{},
			NotificationsEnabled: true,
			EmailNotifications:   true,
			UpdatedAt:            time.Now(),
			LastUpdatedMangas:    []string{},
		}
	}

	// Format subscribed mangas as comma-separated string
	subscribed := ""
	if len(prefs.SubscribedMangas) > 0 {
		subscribed = strings.Join(prefs.SubscribedMangas, ", ")
	}

	// Format updated_at with list of last updated manga titles (if any)
	updatedAtStr := prefs.UpdatedAt.Format(time.RFC3339)
	if len(prefs.LastUpdatedMangas) > 0 {
		// quote each manga name and join with comma
		quoted := make([]string, 0, len(prefs.LastUpdatedMangas))
		for _, m := range prefs.LastUpdatedMangas {
			quoted = append(quoted, fmt.Sprintf("\"%s\"", m))
		}
		updatedAtStr = fmt.Sprintf("%s (updated: %s)", updatedAtStr, strings.Join(quoted, ", "))
	}

	c.JSON(http.StatusOK, gin.H{
		"preferences": gin.H{
			"user_id":               userID,
			"subscribed_mangas":     subscribed,
			"notifications_enabled": prefs.NotificationsEnabled,
			"email_notifications":   prefs.EmailNotifications,
			"updated_at":            updatedAtStr,
		},
	})
}

// TestNotification sends a test notification
func TestNotification(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if notifyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service not available"})
		return
	}

	testNotif, err := notifyManager.TestNotification(userID)
	if err != nil {
		// Create a default test notification anyway
		testNotif = &udp.Notification{
			ID:           "test-notification-001",
			UserID:       userID,
			MangaID:      "test-manga-123",
			MangaTitle:   "Test Manga Title",
			ChapterNum:   1,
			ChapterTitle: "Chapter 1: Getting Started",
			Genre:        "Adventure",
			Message:      "This is a test notification to verify your notification system is working properly",
		}
	}

	// Try to deliver over UDP (requires client registration).
	sendErr := notifyManager.SendNotification(testNotif)

	c.JSON(http.StatusOK, gin.H{
		"status":             "success",
		"message":            "Test notification sent",
		"delivered_over_udp": sendErr == nil,
		"udp_error": func() any {
			if sendErr == nil {
				return nil
			}
			return sendErr.Error()
		}(),
		"notification": gin.H{
			"id":            testNotif.ID,
			"manga_title":   testNotif.MangaTitle,
			"chapter_num":   testNotif.ChapterNum,
			"chapter_title": testNotif.ChapterTitle,
			"genre":         testNotif.Genre,
			"message":       testNotif.Message,
			"timestamp":     testNotif.Timestamp,
		},
	})
}

type notifyNewChapterRequest struct {
	MangaID      string `json:"manga_id"`
	ChapterNum   int    `json:"chapter_num"`
	ChapterTitle string `json:"chapter_title"`
	Genre        string `json:"genre,omitempty"`
}

// NotifyNewChapter triggers a server-side push event to matching subscribers.
// This is a control endpoint to simulate "a new chapter was released".
func NotifyNewChapter(c *gin.Context) {
	if notifyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service not available"})
		return
	}
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	var req notifyNewChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	req.MangaID = strings.TrimSpace(req.MangaID)
	req.ChapterTitle = strings.TrimSpace(req.ChapterTitle)
	req.Genre = strings.TrimSpace(req.Genre)
	if req.MangaID == "" || req.ChapterNum <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga_id and chapter_num (>0) are required"})
		return
	}

	var (
		title      string
		genresJSON string
	)
	err := db.QueryRow(`SELECT title, genres FROM manga WHERE id = ?`, req.MangaID).Scan(&title, &genresJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "manga not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If caller didn't specify genre, pick first genre from DB (if any).
	if req.Genre == "" && genresJSON != "" {
		var genres []string
		_ = json.Unmarshal([]byte(genresJSON), &genres)
		if len(genres) > 0 {
			req.Genre = genres[0]
		}
	}

	sent, pushErr := notifyManager.NotifyNewChapter(req.MangaID, title, req.ChapterNum, req.ChapterTitle, req.Genre)

	c.JSON(http.StatusOK, gin.H{
		"status":           "success",
		"manga_id":         req.MangaID,
		"manga_title":      title,
		"chapter_num":      req.ChapterNum,
		"chapter_title":    req.ChapterTitle,
		"genre":            req.Genre,
		"notified_clients": sent,
		"udp_error": func() any {
			if pushErr == nil {
				return nil
			}
			return pushErr.Error()
		}(),
	})
}
