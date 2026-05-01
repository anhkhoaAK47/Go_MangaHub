package controllers

import (
	"net/http"

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
	genre := c.GetHeader("X-Genre")

	if notifyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service not available"})
		return
	}

	if mangaID == "" && genre == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either manga-id or genre is required"})
		return
	}

	err := notifyManager.Subscribe(userID, mangaID, genre)
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
	if genre != "" {
		response["genre"] = genre
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
	genre := c.GetHeader("X-Genre")

	if notifyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service not available"})
		return
	}

	if mangaID == "" && genre == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either manga-id or genre is required"})
		return
	}

	err := notifyManager.Unsubscribe(userID, mangaID, genre)
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
	if genre != "" {
		response["genre"] = genre
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
			SubscribedGenres:     []string{},
			SubscribedMangas:     []string{},
			NotificationsEnabled: true,
			EmailNotifications:   true,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"preferences": gin.H{
			"user_id":               userID,
			"subscribed_genres":     prefs.SubscribedGenres,
			"subscribed_mangas":     prefs.SubscribedMangas,
			"notifications_enabled": prefs.NotificationsEnabled,
			"email_notifications":   prefs.EmailNotifications,
			"updated_at":            prefs.UpdatedAt,
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
		"status":  "success",
		"message": "Test notification sent",
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
