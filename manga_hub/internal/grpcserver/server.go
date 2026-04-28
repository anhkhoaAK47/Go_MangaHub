package grpcserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go_mangahub/manga_hub/internal/grpcpb"
	"go_mangahub/manga_hub/internal/tcp"
)

type TCPBroadcaster interface {
	BroadcastUpdate(update tcp.ProgressUpdate)
}

type Server struct {
	grpcpb.UnimplementedMangaServiceServer
	db  *sql.DB
	tcp TCPBroadcaster
}

func New(db *sql.DB, tcp TCPBroadcaster) *Server {
	return &Server{db: db, tcp: tcp}
}

func (s *Server) GetManga(ctx context.Context, req *grpcpb.GetMangaRequest) (*grpcpb.MangaResponse, error) {
	id := strings.TrimSpace(req.GetMangaId())
	if id == "" {
		return nil, fmt.Errorf("manga_id is required")
	}

	var (
		mangaID       string
		title         string
		author        string
		artist        string
		genresJSON    string
		status        string
		year          int
		totalChapters int
		totalVolumes  int
		serialization string
		publisher     string
		description   string
		myAnimeList   string
		mangaDx       string
	)

	_ = serialization
	_ = myAnimeList
	_ = mangaDx

	query := `SELECT id, title, author, artist, genres, status, year, total_chapters, total_volumes, serialization, publisher, description, my_anime_list, manga_dx
			  FROM manga WHERE id = ?`
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&mangaID, &title, &author, &artist, &genresJSON, &status, &year,
		&totalChapters, &totalVolumes, &serialization, &publisher, &description, &myAnimeList, &mangaDx,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("manga not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	var genres []string
	_ = json.Unmarshal([]byte(genresJSON), &genres)

	return &grpcpb.MangaResponse{
		Id:           mangaID,
		Title:        title,
		Author:       author,
		Artist:       artist,
		Genres:       genres,
		Status:       status,
		Year:         int32(year),
		TotalChapters: int32(totalChapters),
		TotalVolumes:  int32(totalVolumes),
		Publisher:     publisher,
		Description:   description,
	}, nil
}

func (s *Server) SearchManga(ctx context.Context, req *grpcpb.SearchRequest) (*grpcpb.SearchResponse, error) {
	q := strings.TrimSpace(req.GetQuery())
	if q == "" {
		return nil, fmt.Errorf("query is required")
	}

	like := "%" + strings.ToLower(q) + "%"
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, author, artist, genres, status, year, total_chapters, total_volumes, publisher, description
		 FROM manga
		 WHERE lower(title) LIKE ? OR lower(author) LIKE ?
		 LIMIT 50`, like, like,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*grpcpb.MangaResponse
	for rows.Next() {
		var (
			mangaID       string
			title         string
			author        string
			artist        string
			genresJSON    string
			status        string
			year          int
			totalChapters int
			totalVolumes  int
			publisher     string
			description   string
		)
		if err := rows.Scan(&mangaID, &title, &author, &artist, &genresJSON, &status, &year, &totalChapters, &totalVolumes, &publisher, &description); err != nil {
			continue
		}
		var genres []string
		_ = json.Unmarshal([]byte(genresJSON), &genres)
		out = append(out, &grpcpb.MangaResponse{
			Id:            mangaID,
			Title:         title,
			Author:        author,
			Artist:        artist,
			Genres:        genres,
			Status:        status,
			Year:          int32(year),
			TotalChapters: int32(totalChapters),
			TotalVolumes:  int32(totalVolumes),
			Publisher:     publisher,
			Description:   description,
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no manga found matching query: %s", q)
	}

	return &grpcpb.SearchResponse{Mangas: out}, nil
}

func (s *Server) UpdateProgress(ctx context.Context, req *grpcpb.ProgressRequest) (*grpcpb.ProgressResponse, error) {
	userID := strings.TrimSpace(req.GetUserId())
	mangaID := strings.TrimSpace(req.GetMangaId())
	chapter := int(req.GetChapter())
	status := strings.TrimSpace(req.GetStatus())

	if userID == "" || mangaID == "" || chapter < 1 {
		return nil, fmt.Errorf("user_id, manga_id, and chapter (>=1) are required")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Ensure record exists / update.
	// Note: schema in this repo includes rating/started_reading columns, so we preserve those if present.
	_, err := s.db.ExecContext(ctx, `
	INSERT INTO user_progress (user_id, manga_id, current_chapter, status, updated_at)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(user_id, manga_id) DO UPDATE SET
		current_chapter = excluded.current_chapter,
		status = CASE WHEN excluded.status != '' THEN excluded.status ELSE user_progress.status END,
		updated_at = excluded.updated_at
	`, userID, mangaID, chapter, status, now)
	if err != nil {
		return nil, err
	}

	// Broadcast via TCP sync server (UC-016 step 4)
	if s.tcp != nil {
		s.tcp.BroadcastUpdate(tcp.ProgressUpdate{
			UserID:  userID,
			MangaID: mangaID,
			Chapter: chapter,
		})
	}

	return &grpcpb.ProgressResponse{
		UserId:        userID,
		MangaId:       mangaID,
		Chapter:       int32(chapter),
		ReadingStatus: status,
		Rating:        0,
		UpdatedAt:     now,
	}, nil
}

