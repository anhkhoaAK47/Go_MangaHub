package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type StatsOverview struct {
	TotalManga		int  				`json:"total_manga"`
	TotalChapters	int	 				`json:"total_chapters"`
	StatusCounts	map[string]int		`json:"status_counts"`
	FavoriteGenre	string				`json:"favorite_genre"`
}

type StatsDetailed struct {
	TotalManga		int  				`json:"total_manga"`
	TotalChapters	int	 				`json:"total_chapters"`
	MostRead		string				`json:"status_counts"`
	MostIgnored		string				`json:"favorite_genre"`
	Mangas			[]MangaDetailedStat	`json:"mangas"`
}

type MangaDetailedStat struct {
	MangaID        string  `json:"manga_id"`
	Title          string  `json:"title"`
	Status         string  `json:"status"`
	CurrentChapter int     `json:"current_chapter"`
	TotalChapters  int     `json:"total_chapters"`
	ChaptersLeft   int     `json:"chapters_left"`
	Rating         int     `json:"rating"`
	UpdatesCount   int     `json:"updates_count"`   // how many times you updated progress
	AvgChaptersPerUpdate float64 `json:"avg_chapters_per_update"`
	StartedReading string  `json:"started_reading"`
	LastUpdated    string  `json:"last_updated"`
}

func GetStatsOverview(c *gin.Context, db *sql.DB) {
	userID, _ := c.Get("user_id")

	var stats StatsOverview
	stats.StatusCounts = make(map[string]int)

	// get total manga and total chapters read
	query := `SELECT COUNT(*), SUM(current_chapter) FROM user_progress WHERE user_id = ?`
	err := db.QueryRow(query, userID).Scan(&stats.TotalManga, &stats.TotalChapters)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database error getting total mangas and chapters",
		})
		return
	}

	// get total status (reading: 2, on-hold: 3, etc.)
	rows, _ := db.Query(`SELECT status, COUNT(*) FROM user_progress WHERE user_id = ? GROUP BY status`, userID)

	var status string
	var count int
	for rows.Next() {
		rows.Scan(&status, &count)
		stats.StatusCounts[status] = count // on-hold = 2
	}

	// get favorite genre
	query = `SELECT m.genres FROM user_progress up JOIN manga m ON up.manga_id = m.id WHERE up.user_id = ?`

	genreRows, _ := db.Query(query, userID)

	genreMap := make(map[string]int)

	for genreRows.Next() {
		var genreJSON string
		genreRows.Scan(&genreJSON)

		var genres []string
		json.Unmarshal([]byte(genreJSON), &genres) // unmarshal JSON into genres array

		for _, g := range genres {
			genreMap[g]++
		}
	}
	genreRows.Close()


	// find genre with highest count
	max := 0
	for genre, count := range genreMap {
		if count > max {
			max = count
			stats.FavoriteGenre = genre
		}
	}

	// return OK status with overview stats
	c.JSON(http.StatusOK, stats)
}

func GetStatsDetailed(c *gin.Context, db *sql.DB) {
	userID, _ := c.Get("user_id")

	query := `SELECT 
			up.manga_id,
			m.title,
			up.status,
			up.current_chapter,
			m.total_chapters,
			up.rating,
			up.started_reading,
			up.updated_at
		FROM user_progress up
		JOIN manga m ON up.manga_id = m.id
		WHERE up.user_id = ?
		ORDER BY up.current_chapter DESC
		`
	rows, err := db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database error",
		})
		return
	}
	defer rows.Close()

	var result StatsDetailed
	result.Mangas = []MangaDetailedStat{}

	mostReadChapters := 0
	mostIgnoredLeft := 0

	for rows.Next() {
		var m MangaDetailedStat
		err := rows.Scan(
			&m.MangaID,
			&m.Title,
			&m.Status,
			&m.CurrentChapter,
			&m.TotalChapters,
			&m.Rating,
			&m.StartedReading,
			&m.LastUpdated,
		)
		if err != nil {
			continue
		}

		// chapters left
		if m.TotalChapters > 0 {
			m.ChaptersLeft = m.TotalChapters - m.CurrentChapter
		}

		// count how many time this manga has been updated on progress
		query = `SELECT COUNT(*), COALESCE(AVG(current_chapter - previous_chapter), 0) FROM progress_history WHERE user_id = ? AND manga_id = ?`
		db.QueryRow(query, userID, m.MangaID).Scan(&m.UpdatesCount, &m.AvgChaptersPerUpdate)
		

		// track most read
		if m.CurrentChapter > mostReadChapters {
			mostReadChapters = m.CurrentChapter
			result.MostRead = m.Title
		}

		// track most ignored
		if (m.Status == "on-hold" || m.Status == "dropped" || m.Status == "plan-to-read") && m.ChaptersLeft > mostIgnoredLeft {
			mostIgnoredLeft = m.ChaptersLeft
			result.MostIgnored = m.Title
		}

		result.TotalChapters += m.CurrentChapter
		result.Mangas = append(result.Mangas, m)
	}

	result.TotalManga = len(result.Mangas)

	c.JSON(http.StatusOK, result)
} 