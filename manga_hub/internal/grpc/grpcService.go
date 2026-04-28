package grpc

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go_mangahub/manga_hub/pkg/models"
)

// GrpcService handles gRPC operations for manga and progress
type GrpcService struct {
	mangaCache    map[string]*models.Manga
	progressCache map[string]*models.UserProgress
	mu            sync.RWMutex
}

// NewGrpcService creates a new gRPC service instance
func NewGrpcService() *GrpcService {
	return &GrpcService{
		mangaCache:    make(map[string]*models.Manga),
		progressCache: make(map[string]*models.UserProgress),
	}
}

// GetMangaByID retrieves a manga by its ID via gRPC
func (gs *GrpcService) GetMangaByID(mangaID string) (*models.Manga, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	if manga, exists := gs.mangaCache[mangaID]; exists {
		return manga, nil
	}

	return nil, fmt.Errorf("manga with ID %s not found in cache", mangaID)
}

// SearchManga searches for manga by title or other criteria via gRPC
func (gs *GrpcService) SearchManga(query string) ([]*models.Manga, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	var results []*models.Manga

	for _, manga := range gs.mangaCache {
		if matches(manga.Title, query) || matches(manga.Author, query) {
			results = append(results, manga)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no manga found matching query: %s", query)
	}

	return results, nil
}

// UpdateProgress updates user reading progress via gRPC
func (gs *GrpcService) UpdateProgress(userID string, mangaID string, chapter int, status string) (*models.UserProgress, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	progressKey := fmt.Sprintf("%s-%s", userID, mangaID)

	progress, exists := gs.progressCache[progressKey]
	if !exists {
		progress = &models.UserProgress{
			UserID:         userID,
			MangaID:        mangaID,
			CurrentChapter: chapter,
			Status:         status,
			UpdatedAt:      time.Now(),
		}
	} else {
		progress.CurrentChapter = chapter
		if status != "" {
			progress.Status = status
		}
		progress.UpdatedAt = time.Now()
	}

	gs.progressCache[progressKey] = progress
	return progress, nil
}

// GetProgress retrieves user reading progress via gRPC
func (gs *GrpcService) GetProgress(userID string, mangaID string) (*models.UserProgress, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	progressKey := fmt.Sprintf("%s-%s", userID, mangaID)

	progress, exists := gs.progressCache[progressKey]
	if !exists {
		return nil, fmt.Errorf("no progress found for user %s and manga %s", userID, mangaID)
	}

	return progress, nil
}

// CacheManga adds or updates a manga in the cache
func (gs *GrpcService) CacheManga(manga *models.Manga) error {
	if manga == nil || manga.ID == "" {
		return fmt.Errorf("invalid manga: ID is required")
	}

	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.mangaCache[manga.ID] = manga
	return nil
}

// ClearCache clears the gRPC cache
func (gs *GrpcService) ClearCache() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.mangaCache = make(map[string]*models.Manga)
	gs.progressCache = make(map[string]*models.UserProgress)
}

// GetMangaCount returns the number of cached manga
func (gs *GrpcService) GetMangaCount() int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return len(gs.mangaCache)
}

// GetProgressCount returns the number of cached progress entries
func (gs *GrpcService) GetProgressCount() int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return len(gs.progressCache)
}

// matches checks if a value contains the query string (case-insensitive)
func matches(value string, query string) bool {
	if value == "" || query == "" {
		return false
	}

	return strings.Contains(strings.ToLower(value), strings.ToLower(query))
}

// BatchGetManga retrieves multiple manga at once via gRPC
func (gs *GrpcService) BatchGetManga(mangaIDs []string) ([]*models.Manga, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	var results []*models.Manga
	var notFound []string

	for _, id := range mangaIDs {
		if manga, exists := gs.mangaCache[id]; exists {
			results = append(results, manga)
		} else {
			notFound = append(notFound, id)
		}
	}

	if len(notFound) > 0 {
		return results, fmt.Errorf("manga not found: %v", notFound)
	}

	return results, nil
}

// ListAllManga returns all cached manga
func (gs *GrpcService) ListAllManga() []*models.Manga {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	var mangas []*models.Manga
	for _, manga := range gs.mangaCache {
		mangas = append(mangas, manga)
	}

	return mangas
}

// GetMangaStats returns statistics about a manga's requests
type MangaStats struct {
	MangaID      string
	TimesQueried int
	LastQueried  time.Time
}

var (
	queryStats = make(map[string]MangaStats)
	statsMu    sync.RWMutex
)

// RecordQuery records a query for statistics
func (gs *GrpcService) RecordQuery(mangaID string) {
	statsMu.Lock()
	defer statsMu.Unlock()

	stats := queryStats[mangaID]
	stats.MangaID = mangaID
	stats.TimesQueried++
	stats.LastQueried = time.Now()
	queryStats[mangaID] = stats
}

// GetQueryStats returns query statistics
func (gs *GrpcService) GetQueryStats(mangaID string) *MangaStats {
	statsMu.RLock()
	defer statsMu.RUnlock()

	if stats, exists := queryStats[mangaID]; exists {
		return &stats
	}

	return nil
}
