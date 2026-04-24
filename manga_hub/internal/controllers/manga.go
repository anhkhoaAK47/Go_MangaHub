package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go_mangahub/manga_hub/pkg/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var db *sql.DB

// SetDB sets the package-level database handle for controllers.
func SetDB(d *sql.DB) {
	db = d
}


func GetAllManga(c *gin.Context) {

	// Read query params from URL
	search := strings.ToLower(c.Query("query"))
	genre := strings.ToLower(c.Query("genre"))
	status := strings.ToLower(c.Query("status"))
	
	// get pagination params
	limitStr := c.DefaultQuery("limit", "20")
	pageStr := c.DefaultQuery("page", "1")
	limit, _ := strconv.Atoi(limitStr)
	page, _ := strconv.Atoi(pageStr)


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
		
		// Filter by search query (title or author)
		if search != "" {
			titleMatch  := strings.Contains(strings.ToLower(m.Title), search)
			authorMatch := strings.Contains(strings.ToLower(m.Author), search)
			if !titleMatch && !authorMatch {
				continue
			}
		}

		// Filter by genre
		if genre != "" {
			matched := false
			for _, g := range m.Genres {
				if strings.Contains(strings.ToLower(g), genre) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Filter by status
		if status != "" {
			if !strings.Contains(strings.ToLower(m.Status), status) {
				continue
			}
		}
		mangaList = append(mangaList, m)
	}
	
	totalResults := len(mangaList)
	start := (page - 1) * limit
	end := start + limit

	// Return empty array instead of null
	if start > totalResults {
		mangaList = []models.Manga{}
	} else {
		if end > totalResults {
			end = totalResults
		}
		mangaList = mangaList[start:end]
	}

	c.JSON(http.StatusOK, mangaList)
}

func GetMangaByTitle(c *gin.Context) {
	title := c.Param("title")

	// Query
	var m models.Manga

	err := db.QueryRow("SELECT id, title, author, status, total_chapters FROM manga WHERE title LIKE ?", title).
	Scan(&m.ID, &m.Title, &m.Author, &m.Status, &m.TotalChapters)

	// if not found
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No manga found matching your search criteria",
		})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	
	c.JSON(http.StatusOK, m)
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