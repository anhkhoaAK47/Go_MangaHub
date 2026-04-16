package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go_mangahub/manga_hub/pkg/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var db *sql.DB

// SetDB sets the package-level database handle for controllers.
func SetDB(d *sql.DB) {
	db = d
}


func GetAllManga(c *gin.Context) {
	// Query
	query := `SELECT id, title, author, artist, genres, status, year, total_chapters, total_volumes, serialization, publisher, description, my_anime_list, manga_dx FROM manga`

	// Execute and return results
	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	// Create mangalist
	var mangaList []models.Manga
	for rows.Next() {
		var m models.Manga
		var genresString string // string to hold JSON text from sqlite

		err := rows.Scan(&m.ID, &m.Title, &m.Author, &m.Artist, &genresString, &m.Status, &m.Year, &m.TotalChapters, &m.TotalVolumes, &m.Serialization, &m.Publisher, &m.Description, &m.MyAnimeList, &m.MangaDx)
		if err != nil {
			continue
		}

		// Convert JSON into go string slice []string
		json.Unmarshal([]byte(genresString), &m.Genres)

		mangaList = append(mangaList, m)
	}

	c.JSON(http.StatusOK, mangaList)
}

func GetMangaInfo(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		userID = nil
	}

	var m models.Manga
	var genresString string

	query := `SELECT id, title, author, artist, genres, status, year, total_chapters, total_volumes, serialization, publisher, description, my_anime_list, manga_dx FROM manga WHERE id = ?`
	err := db.QueryRow(query, id).Scan(
		&m.ID, &m.Title, &m.Author, &m.Artist, &genresString, &m.Status, &m.Year, &m.TotalChapters, &m.TotalVolumes, &m.Serialization, &m.Publisher, &m.Description, &m.MyAnimeList, &m.MangaDx,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Manga not found: %s", id),
		})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Convert JSON into go string slice []string
	json.Unmarshal([]byte(genresString), &m.Genres)

	response := models.MangaInfoResponse{
		Manga: m,
	}

	// Try to get user progress if userID is present
	if userID != nil {
		var p models.UserProgress
		var startedReadingStr, updatedAtStr string
		progressQuery := `SELECT user_id, manga_id, current_chapter, status, rating, started_reading, updated_at FROM user_progress WHERE user_id = ? AND manga_id = ?`
		err = db.QueryRow(progressQuery, userID, id).Scan(
			&p.UserID, &p.MangaID, &p.CurrentChapter, &p.Status, &p.Rating, &startedReadingStr, &updatedAtStr,
		)
		if err == nil {
			p.StartedReading, _ = time.Parse(time.RFC3339, startedReadingStr)
			p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
			response.Progress = &p
		}
	}

	c.JSON(http.StatusOK, response)
}