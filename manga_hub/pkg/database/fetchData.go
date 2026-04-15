package database

import (
	"encoding/json"
	"go_mangahub/manga_hub/pkg/models"
	"io"
	"net/http"
)

func FetchMangaDex() ([]models.Manga, error) {
	url := "https://api.mangadex.org/manga?limit=100&includes[]=author&includes[]=cover_art"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response body
	body, _ := io.ReadAll(resp.Body)

	var mangaDexResponse models.MangaDexResponse
	if err := json.Unmarshal(body, &mangaDexResponse); err != nil {
		return nil, err
	}


	var result []models.Manga
	for _, items := range mangaDexResponse.Data {
		// Map manga dex data to our manga struct
		m := models.Manga {
			ID: items.ID,
			Title: items.Attributes.Title["en"],
			Description: items.Attributes.Description["en"],
			Status: items.Attributes.Status,
		}

		// Extract author from relationships
		for _, rel := range items.Relationships {
			if rel.Type == "author" {
				m.Author = rel.Attributes.Name
			}
		}
		result = append(result, m)
	} 

	return result, nil
} 