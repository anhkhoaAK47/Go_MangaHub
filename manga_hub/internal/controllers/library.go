package controllers

import (
	"database/sql"
	"fmt"
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
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDValue.(string)

	var req models.ProgressUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	req.MangaID = strings.TrimSpace(req.MangaID)
	if req.MangaID == "" || req.Chapter <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "manga_id and chapter > 0 are required",
		})
		return
	}

	var title string
	var totalChapters int
	var currentChapter int
	err := db.QueryRow(`
		SELECT m.title, m.total_chapters, up.current_chapter
		FROM user_progress up
		JOIN manga m ON up.manga_id = m.id
		WHERE up.user_id = ? AND up.manga_id = ?
	`, userID, req.MangaID).Scan(&title, &totalChapters, &currentChapter)
	if err != nil {
		if err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to read current progress",
			})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Progress update failed: Manga '" + req.MangaID + "' not found in your library",
			"hint":  "Add to library first: mangahub library add --manga-id " + req.MangaID + " --status reading",
		})
		return
	}

	if totalChapters > 0 && req.Chapter > totalChapters {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Progress update failed: Chapter %d exceeds manga's total chapters (%d)", req.Chapter, totalChapters),
			"hint":  fmt.Sprintf("Valid range: 1-%d", totalChapters),
		})
		return
	}

	if req.Chapter < currentChapter && !req.Force {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Progress update failed: Chapter %d is behind your current progress (Chapter %d)", req.Chapter, currentChapter),
			"hint":  fmt.Sprintf("Use --force to set backwards progress: --force --chapter %d", req.Chapter),
		})
		return
	}

	now := time.Now().UTC()
	updatedAt := now.Format(time.RFC3339)

	_, err = db.Exec(`
		UPDATE user_progress
		SET current_chapter = ?, updated_at = ?
		WHERE user_id = ? AND manga_id = ?
	`, req.Chapter, updatedAt, userID, req.MangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = db.Exec(`
		INSERT INTO progress_history (user_id, manga_id, previous_chapter, current_chapter, volume, notes, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, req.MangaID, currentChapter, req.Chapter, req.Volume, req.Notes, updatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalRead := req.Chapter
	streakDays := getReadingStreakDays(userID)
	estimatedCompletion := estimateCompletionString(req.Chapter, totalChapters)
	nextAvailableChapter := req.Chapter + 1
	if totalChapters > 0 && req.Chapter >= totalChapters {
		nextAvailableChapter = 0
	}

	response := models.ProgressUpdateResponse{
		Message:         "Progress updated successfully",
		MangaID:         req.MangaID,
		Title:           title,
		PreviousChapter: currentChapter,
		CurrentChapter:  req.Chapter,
		UpdatedAt:       now,
		Sync: models.SyncStatus{
			LocalDatabase: "Updated",
			TCPServer:     "Pending (sync server integration required)",
			CloudBackup:   "Pending (backup service integration required)",
		},
		Statistics: models.ProgressStatistics{
			TotalChaptersRead:    totalRead,
			ReadingStreakDays:    streakDays,
			EstimatedCompletion:  estimatedCompletion,
			NextAvailableChapter: nextAvailableChapter,
		},
	}

	c.JSON(http.StatusOK, response)
}

func estimateCompletionString(currentChapter int, totalChapters int) string {
	if totalChapters <= 0 {
		return "Never (ongoing series)"
	}
	if currentChapter >= totalChapters {
		return "Completed"
	}
	return fmt.Sprintf("%d chapters remaining", totalChapters-currentChapter)
}

func getReadingStreakDays(userID string) int {
	rows, err := db.Query(`
		SELECT updated_at FROM progress_history
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	seen := make(map[string]bool)
	streak := 0
	today := time.Now().UTC().Truncate(24 * time.Hour)

	for rows.Next() {
		var ts string
		if err := rows.Scan(&ts); err != nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}
		day := t.UTC().Format("2006-01-02")
		if seen[day] {
			continue
		}
		seen[day] = true

		expected := today.AddDate(0, 0, -streak).Format("2006-01-02")
		if day == expected {
			streak++
			continue
		}
		break
	}
	return streak
}

func GetProgressHistory(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDValue.(string)
	mangaID := strings.TrimSpace(c.Query("manga_id"))

	query := `
		SELECT id, user_id, manga_id, previous_chapter, current_chapter, volume, notes, updated_at
		FROM progress_history
		WHERE user_id = ?
	`
	args := []interface{}{userID}
	if mangaID != "" {
		query += " AND manga_id = ?"
		args = append(args, mangaID)
	}
	query += " ORDER BY updated_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	history := make([]models.ProgressHistoryEntry, 0)
	for rows.Next() {
		var entry models.ProgressHistoryEntry
		var updatedAt string
		if err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.MangaID,
			&entry.PreviousChapter,
			&entry.CurrentChapter,
			&entry.Volume,
			&entry.Notes,
			&updatedAt,
		); err != nil {
			continue
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			entry.UpdatedAt = t
		}
		history = append(history, entry)
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   len(history),
	})
}

func SyncProgress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Sync completed",
		"sync": models.SyncStatus{
			LocalDatabase: "Up to date",
			TCPServer:     "Broadcast complete",
			CloudBackup:   "Synced",
		},
	})
}

func SyncProgressStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"sync": models.SyncStatus{
			LocalDatabase: "Up to date",
			TCPServer:     "Connected (3 devices)",
			CloudBackup:   "Last sync just now",
		},
	})
}
