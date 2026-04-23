package mangahub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"go_mangahub/manga_hub/pkg/models"
	"go_mangahub/manga_hub/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	searchGenre string
	searchStatus string
	searchLimit int
)

var MangaCmd = &cobra.Command{
	Use:   "manga",
	Short: "Manage manga in the library",
}

var infoCmd = &cobra.Command{
	Use:   "info [manga-id]",
	Short: "Get detailed information about a manga",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mangaID := args[0]
		token, _ := os.ReadFile(".token")

		client := &http.Client{}
		req, _ := http.NewRequest("GET", "http://localhost:8080/manga/"+mangaID, nil)
		if len(token) > 0 {
			req.Header.Set("Authorization", "Bearer "+string(token))
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error connecting to server:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			fmt.Printf("✗ Manga not found: '%s'\n", mangaID)
			fmt.Println("Try searching instead:")
			fmt.Println("  mangahub manga search \"manga title\"")
			return
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Println("Error: Received status code", resp.StatusCode)
			return
		}

		body, _ := io.ReadAll(resp.Body)
		var info models.MangaInfoResponse
		if err := json.Unmarshal(body, &info); err != nil {
			fmt.Println("Error parsing response:", err)
			return
		}

		m := info.Manga
		p := info.Progress

		// Print Header
		titleLen := len(m.Title)
		boxWidth := titleLen + 4
		fmt.Printf("┌%s┐\n", strings.Repeat("─", boxWidth))
		fmt.Printf("│  %s  │\n", strings.ToUpper(m.Title))
		fmt.Printf("└%s┘\n", strings.Repeat("─", boxWidth))

		// Basic Information
		fmt.Println("Basic Information:")
		fmt.Printf("  ID: %s\n", m.ID)
		fmt.Printf("  Title: %s\n", m.Title)
		fmt.Printf("  Author: %s\n", m.Author)
		fmt.Printf("  Artist: %s\n", m.Artist)
		fmt.Printf("  Genres: %s\n", strings.Join(m.Genres, ", "))
		fmt.Printf("  Status: %s\n", m.Status)
		fmt.Printf("  Year: %d\n", m.Year)

		// Progress
		fmt.Println("Progress:")
		fmt.Printf("  Total Chapters: %d+\n", m.TotalChapters)
		fmt.Printf("  Total Volumes: %d+\n", m.TotalVolumes)
		fmt.Printf("  Serialization: %s\n", m.Serialization)
		fmt.Printf("  Publisher: %s\n", m.Publisher)

		if p != nil {
			fmt.Printf("  Your Status: %s\n", p.Status)
			fmt.Printf("  Current Chapter: %d\n", p.CurrentChapter)
			fmt.Printf("  Last Updated: %s\n", p.UpdatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Started Reading: %s\n", p.StartedReading.Format("2006-01-02"))
			fmt.Printf("  Personal Rating: %d/10\n", p.Rating)
		} else {
			fmt.Println("  Your Status: Not in Library")
		}

		// Description
		fmt.Println("Description:")
		desc := m.Description
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		// Simple word wrap for description
		words := strings.Fields(desc)
		line := "  "
		for _, word := range words {
			if len(line)+len(word) > 70 {
				fmt.Println(line)
				line = "  "
			}
			line += word + " "
		}
		fmt.Println(line)

		// External Links
		fmt.Println("External Links:")
		fmt.Printf("  MyAnimeList: %s\n", m.MyAnimeList)
		fmt.Printf("  MangaDx: %s\n", m.MangaDx)

		// Actions
		fmt.Println("Actions:")
		fmt.Printf("  Update Progress: mangahub progress update --manga-id %s --chapter %d\n", m.ID, m.TotalChapters)
		fmt.Printf("  Rate/Review: mangahub library update --manga-id %s --rating 10\n", m.ID)
		fmt.Printf("  Remove: mangahub library remove --manga-id %s\n", m.ID)
	},
}

var searchCmd = &cobra.Command{
	Use: "search [query]",
	Short: "Search for manga by title, genre, or status",
	Run: func(cmd *cobra.Command, args []string) {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		if query != "" {
			fmt.Printf("Searching for \"%s\"...\n", query)
		}

		params := url.Values{}
		if query != "" {
			params.Set("query", query)
		}
		if searchGenre != "" {
			params.Set("genre", searchGenre)
		}
		if searchStatus != "" {
			params.Set("status", searchStatus)
		}
		if searchLimit > 0 {
			params.Set("limit", strconv.Itoa(searchLimit))
		}


		fullUrl := "http://localhost:8080/manga/?" + params.Encode()

		resp, err := http.Get(fullUrl)
		if err != nil {
			fmt.Println("❌ Server connection error.")
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var mangaList []models.Manga
		if err := json.Unmarshal(body, &mangaList); err != nil {
			fmt.Println("❌ Failed to parse server response")
			return
		}

		// No result
		if len(mangaList) == 0 {
			if query != "" {
				fmt.Printf("No manga found for \"%s\"...", query)
			} else {
				fmt.Println("No manga found.")
			}

			fmt.Println("\nSuggestion:")
			fmt.Println("	-Browse by genre: mangahub manga list --genre action")
			return
		}

		// Print results
		fmt.Printf("Found %d result(s):\n\n", len(mangaList))

		utils.PrintMangaTable(mangaList)

		fmt.Println("\nUse 'mangahub manga info <id>' to view details")
		fmt.Println("Use 'mangahub library add --manga-id <id>' to add to your library")

	},
}

func init() {
	// add search command
	MangaCmd.AddCommand(searchCmd)

	// add info command
	MangaCmd.AddCommand(infoCmd)

}