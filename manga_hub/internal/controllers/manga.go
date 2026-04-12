package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go_mangahub/manga_hub/pkg/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

var db *sql.DB

// SetDB sets the package-level database handle for controllers.
func SetDB(d *sql.DB) {
	db = d
}


func GetAllManga(c *gin.Context) {	
	// Query
	query := `SELECT * FROM manga`

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

		err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresString, &m.Status, &m.TotalChapters, &m.Description)
		if err != nil {
			continue
		}

		// Convert JSON into go string slice []string
		json.Unmarshal([]byte(genresString), &m.Genres)

		mangaList = append(mangaList, m)
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

	var m models.Manga
	var genresString string
	

	err := db.QueryRow("SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE id = ?", id).
	Scan(&m.ID, &m.Title, &m.Author, &genresString, &m.Status, &m.TotalChapters, &m.Description)

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

	c.JSON(http.StatusOK, m)
}

