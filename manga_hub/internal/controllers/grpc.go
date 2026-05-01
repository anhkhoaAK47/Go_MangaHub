package controllers

import (
	"context"
	"net/http"
	"time"

	internalgrpc "go_mangahub/manga_hub/internal/grpc"
	"go_mangahub/manga_hub/internal/grpcpb"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	grpcClient grpcpb.MangaServiceClient
	grpcConn   *grpc.ClientConn
	grpcService *internalgrpc.GrpcService // legacy in-memory service (kept for extra endpoints)
)

// InitializeGrpcClient initializes the gRPC client used by HTTP handlers.
func InitializeGrpcClient(addr string) error {
	if grpcConn != nil {
		return nil
	}
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	grpcConn = conn
	grpcClient = grpcpb.NewMangaServiceClient(conn)
	return nil
}

// InitializeGrpcService initializes the legacy in-memory gRPC service (extra endpoints).
// Kept to avoid deleting existing behavior while we migrate to real grpc-go.
func InitializeGrpcService(service *internalgrpc.GrpcService) {
	grpcService = service
}

// GetMangaByID retrieves a manga by ID via gRPC
func GetMangaByID(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	mangaID := c.Param("id")
	if mangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Manga ID is required"})
		return
	}

	if grpcClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC service not available"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	manga, err := grpcClient.GetManga(ctx, &grpcpb.GetMangaRequest{MangaId: mangaID})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            manga.Id,
		"title":         manga.Title,
		"author":        manga.Author,
		"artist":        manga.Artist,
		"status":        manga.Status,
		"year":          manga.Year,
		"totalChapters": manga.TotalChapters,
		"totalVolumes":  manga.TotalVolumes,
		"publisher":     manga.Publisher,
		"description":   manga.Description,
		"genres":        manga.Genres,
	})
}

// SearchManga searches for manga via gRPC
func SearchManga(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	if grpcClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC service not available"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	resp, err := grpcClient.SearchManga(ctx, &grpcpb.SearchRequest{Query: query})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var results []gin.H
	for _, manga := range resp.Mangas {
		results = append(results, gin.H{
			"id":            manga.Id,
			"title":         manga.Title,
			"author":        manga.Author,
			"status":        manga.Status,
			"totalChapters": manga.TotalChapters,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(results),
		"mangas": results,
	})
}

// UpdateProgressGrpc updates user progress via gRPC
func UpdateProgressGrpc(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request struct {
		MangaID string `json:"manga_id" binding:"required"`
		Chapter int    `json:"chapter" binding:"required,min=1"`
		Status  string `json:"status"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if grpcClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC service not available"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	progress, err := grpcClient.UpdateProgress(ctx, &grpcpb.ProgressRequest{
		UserId:  userID,
		MangaId: request.MangaID,
		Chapter: int32(request.Chapter),
		Status:  request.Status,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"user_id":        progress.UserId,
		"manga_id":       progress.MangaId,
		"chapter":        progress.Chapter,
		"reading_status": progress.ReadingStatus,
		"rating":         progress.Rating,
		"updated_at":     progress.UpdatedAt,
	})
}

// GetProgress retrieves user progress (legacy in-memory implementation)
func GetProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	mangaID := c.Query("manga_id")
	if mangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Manga ID is required"})
		return
	}

	if grpcService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC legacy service not available"})
		return
	}

	progress, err := grpcService.GetProgress(userID, mangaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":         progress.UserID,
		"manga_id":        progress.MangaID,
		"chapter":         progress.CurrentChapter,
		"reading_status":  progress.Status,
		"rating":          progress.Rating,
		"started_reading": progress.StartedReading,
		"updated_at":      progress.UpdatedAt,
	})
}

// ListAllManga lists all manga (legacy in-memory implementation)
func ListAllManga(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if grpcService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC legacy service not available"})
		return
	}

	mangas := grpcService.ListAllManga()

	var results []gin.H
	for _, manga := range mangas {
		results = append(results, gin.H{
			"id":            manga.ID,
			"title":         manga.Title,
			"author":        manga.Author,
			"status":        manga.Status,
			"totalChapters": manga.TotalChapters,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(results),
		"mangas": results,
	})
}

// GetServiceStats retrieves gRPC service statistics (legacy in-memory implementation)
func GetServiceStats(c *gin.Context) {
	if grpcService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC legacy service not available"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service":        "grpc",
		"manga_count":    grpcService.GetMangaCount(),
		"progress_count": grpcService.GetProgressCount(),
		"status":         "operational",
	})
}

// BatchGetManga retrieves multiple manga (legacy in-memory implementation)
func BatchGetManga(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request struct {
		MangaIDs []string `json:"manga_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if grpcService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC legacy service not available"})
		return
	}

	mangas, err := grpcService.BatchGetManga(request.MangaIDs)
	if err != nil {
		c.JSON(http.StatusPartialContent, gin.H{
			"partial": true,
			"error":   err.Error(),
			"count":   len(mangas),
			"mangas":  mangas,
		})
		return
	}

	var results []gin.H
	for _, manga := range mangas {
		results = append(results, gin.H{
			"id":            manga.ID,
			"title":         manga.Title,
			"author":        manga.Author,
			"status":        manga.Status,
			"totalChapters": manga.TotalChapters,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(results),
		"mangas": results,
	})
}

// GetMangaStats retrieves query statistics for a manga (legacy in-memory implementation)
func GetMangaStats(c *gin.Context) {
	mangaID := c.Param("id")
	if mangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Manga ID is required"})
		return
	}

	if grpcService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC legacy service not available"})
		return
	}

	stats := grpcService.GetQueryStats(mangaID)
	if stats == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No stats found for manga"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"manga_id":      stats.MangaID,
		"times_queried": stats.TimesQueried,
		"last_queried":  stats.LastQueried,
	})
}
