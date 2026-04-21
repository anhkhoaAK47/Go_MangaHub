package controllers

import (
	"net/http"
	"strings"
	"time"

	"go_mangahub/manga_hub/pkg/models"

	"github.com/gin-gonic/gin"
)

var validLibraryStatuses = map[string]bool{
	"reading":      true,
	"completed":    true,
	"plan-to-read": true,
	"on-hold":      true,
	"dropped":      true,
}

func isValidLibraryStatus(status string) bool {
	return validLibraryStatuses[status]
}

func AddToLibrary(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDValue.(string)

	var req models.AddLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.MangaID == "" || !isValidLibraryStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga_id and valid status are required"})
		return
	}
	if req.Rating < 0 || req.Rating > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 0 and 10"})
		return
	}
	
	var mangaExists int
	if err := db.QueryRow(`SELECT COUNT(1) FROM manga WHERE id = ?`, req.MangaID).Scan(&mangaExists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if mangaExists == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "manga not found"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	query := `
	INSERT INTO user_progress (user_id, manga_id, current_chapter, status, rating, started_reading, updated_at)
	VALUES (?, ?, 0, ?, ?, ?, ?)
	ON CONFLICT(user_id, manga_id) DO UPDATE SET
		status = excluded.status,
		rating = excluded.rating,
		updated_at = excluded.updated_at
	`
	_, err := db.Exec(query, userID, req.MangaID, req.Status, req.Rating, now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "library entry saved"})
}

func ListLibrary(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDValue.(string)

	statusFilter := strings.TrimSpace(strings.ToLower(c.Query("status")))
	sortBy := strings.TrimSpace(strings.ToLower(c.DefaultQuery("sort_by", "updated_at")))
	order := strings.TrimSpace(strings.ToLower(c.DefaultQuery("order", "desc")))

	if statusFilter != "" && !isValidLibraryStatus(statusFilter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status filter"})
		return
	}

	orderSQL := "DESC"
	if order == "asc" {
		orderSQL = "ASC"
	}

	sortColumn := "up.updated_at"
	switch sortBy {
	case "title":
		sortColumn = "m.title"
	case "last-updated", "updated_at":
		sortColumn = "up.updated_at"
	}

	baseQuery := `
	SELECT up.manga_id, m.title, up.current_chapter, m.total_chapters, up.status, up.rating, up.started_reading, up.updated_at
	FROM user_progress up
	JOIN manga m ON up.manga_id = m.id
	WHERE up.user_id = ?
	`
	args := []interface{}{userID}
	if statusFilter != "" {
		baseQuery += " AND up.status = ?"
		args = append(args, statusFilter)
	}
	baseQuery += " ORDER BY " + sortColumn + " " + orderSQL

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	entries := make([]models.LibraryEntry, 0)
	for rows.Next() {
		var entry models.LibraryEntry
		var startedStr string
		var updatedStr string
		if err := rows.Scan(
			&entry.MangaID,
			&entry.Title,
			&entry.CurrentChapter,
			&entry.TotalChapters,
			&entry.Status,
			&entry.Rating,
			&startedStr,
			&updatedStr,
		); err != nil {
			continue
		}

		if t, err := time.Parse(time.RFC3339, startedStr); err == nil {
			entry.StartedReading = t
		}
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			entry.UpdatedAt = t
		}

		entries = append(entries, entry)
	}

	c.JSON(http.StatusOK, models.LibraryListResponse{
		Entries: entries,
		Total:   len(entries),
	})
}

func RemoveFromLibrary(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDValue.(string)
	mangaID := c.Param("id")
	if mangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga id is required"})
		return
	}

	result, err := db.Exec(`DELETE FROM user_progress WHERE user_id = ? AND manga_id = ?`, userID, mangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "library entry not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "library entry removed"})
}

func UpdateLibraryEntry(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDValue.(string)
	mangaID := c.Param("id")

	var req models.UpdateLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status != "" && !isValidLibraryStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	if req.Rating != nil && (*req.Rating < 0 || *req.Rating > 10) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 0 and 10"})
		return
	}

	var existsCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM user_progress WHERE user_id = ? AND manga_id = ?`, userID, mangaID).Scan(&existsCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existsCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "library entry not found"})
		return
	}

	ratingToSet := -1
	if req.Rating != nil {
		ratingToSet = *req.Rating
	}

	_, err := db.Exec(`
		UPDATE user_progress
		SET status = CASE WHEN ? = '' THEN status ELSE ? END,
			rating = CASE WHEN ? = -1 THEN rating ELSE ? END,
			updated_at = ?
		WHERE user_id = ? AND manga_id = ?
	`, req.Status, req.Status, ratingToSet, ratingToSet, time.Now().UTC().Format(time.RFC3339), userID, mangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "library entry updated"})
}

func UpdateProgress(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "progress update endpoint is not implemented yet",
	})
}
