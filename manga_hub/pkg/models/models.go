package models

import (
	"time"
)

type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"password_hash" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Manga struct {
	ID            string   `json:"id" db:"id"`
	Title         string   `json:"title" db:"title"`
	Author        string   `json:"author" db:"author"`
	Artist        string   `json:"artist" db:"artist"`
	Genres        []string `json:"genres" db:"genres"`
	Status        string   `json:"status" db:"status"`
	Year          int      `json:"year" db:"year"`
	TotalChapters int      `json:"total_chapters" db:"total_chapters"`
	TotalVolumes  int      `json:"total_volumes" db:"total_volumes"`
	Serialization string   `json:"serialization" db:"serialization"`
	Publisher     string   `json:"publisher" db:"publisher"`
	Description   string   `json:"description" db:"description"`
	MyAnimeList   string   `json:"my_anime_list" db:"my_anime_list"`
	MangaDx       string   `json:"manga_dx" db:"manga_dx"`
}

type UserProgress struct {
	UserID         string    `json:"user_id" db:"user_id"`
	MangaID        string    `json:"manga_id" db:"manga_id"`
	CurrentChapter int       `json:"current_chapter" db:"current_chapter"`
	Status         string    `json:"status" db:"status"`
	Rating         int       `json:"rating" db:"rating"`
	StartedReading time.Time `json:"started_reading" db:"started_reading"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type MangaInfoResponse struct {
	Manga    Manga         `json:"manga"`
	Progress *UserProgress `json:"progress,omitempty"`
}

type LibraryEntry struct {
	MangaID        string    `json:"manga_id"`
	Title          string    `json:"title"`
	CurrentChapter int       `json:"current_chapter"`
	TotalChapters  int       `json:"total_chapters"`
	Status         string    `json:"status"`
	Rating         int       `json:"rating"`
	StartedReading time.Time `json:"started_reading"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type AddLibraryRequest struct {
	MangaID string `json:"manga_id"`
	Status  string `json:"status"`
	Rating  int    `json:"rating,omitempty"`
}

type UpdateLibraryRequest struct {
	Status string `json:"status,omitempty"`
	Rating *int   `json:"rating,omitempty"`
}

type LibraryListResponse struct {
	Entries []LibraryEntry `json:"entries"`
	Total   int            `json:"total"`
}

type ProgressUpdateRequest struct {
	MangaID string `json:"manga_id"`
	Chapter int    `json:"chapter"`
	Volume  int    `json:"volume,omitempty"`
	Notes   string `json:"notes,omitempty"`
	Force   bool   `json:"force,omitempty"`
}

type SyncStatus struct {
	LocalDatabase string `json:"local_database"`
	TCPServer     string `json:"tcp_server"`
	CloudBackup   string `json:"cloud_backup"`
}

type ProgressStatistics struct {
	TotalChaptersRead    int    `json:"total_chapters_read"`
	ReadingStreakDays    int    `json:"reading_streak_days"`
	EstimatedCompletion  string `json:"estimated_completion"`
	NextAvailableChapter int    `json:"next_available_chapter"`
}

type ProgressUpdateResponse struct {
	Message         string             `json:"message"`
	MangaID         string             `json:"manga_id"`
	Title           string             `json:"title"`
	PreviousChapter int                `json:"previous_chapter"`
	CurrentChapter  int                `json:"current_chapter"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Sync            SyncStatus         `json:"sync"`
	Statistics      ProgressStatistics `json:"statistics"`
}

type ProgressHistoryEntry struct {
	ID              int       `json:"id"`
	UserID          string    `json:"user_id"`
	MangaID         string    `json:"manga_id"`
	PreviousChapter int       `json:"previous_chapter"`
	CurrentChapter  int       `json:"current_chapter"`
	Volume          int       `json:"volume"`
	Notes           string    `json:"notes"`
	UpdatedAt       time.Time `json:"updated_at"`
}
