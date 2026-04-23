package mangahub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"go_mangahub/manga_hub/pkg/models"

	"github.com/spf13/cobra"
)

var LibraryCmd = &cobra.Command{
	Use:   "library",
	Short: "Manage your manga library",
}

var (
	libraryMangaID string
	libraryStatus  string
	libraryRating  int
	sortBy         string
	order          string
)

var addLibraryCmd = &cobra.Command{
	Use:   "add",
	Short: "Add manga to library",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		payload := models.AddLibraryRequest{
			MangaID: libraryMangaID,
			Status:  strings.ToLower(strings.TrimSpace(libraryStatus)),
			Rating:  libraryRating,
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/users/library", bytes.NewBuffer(body))
		if err != nil {
			fmt.Println("❌ Failed to create request:", err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(tokenData)))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Failed to reach server. Is it running?")
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ Failed to add to library: %s\n", extractLibraryError(respBody))
			return
		}

		fmt.Printf("✅ Added '%s' to your library with status '%s'.\n", libraryMangaID, payload.Status)
	},
}

var listLibraryCmd = &cobra.Command{
	Use:   "list",
	Short: "View library entries",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		url := "http://localhost:8080/users/library"
		queryParts := make([]string, 0)
		if strings.TrimSpace(libraryStatus) != "" {
			queryParts = append(queryParts, "status="+strings.TrimSpace(strings.ToLower(libraryStatus)))
		}
		if strings.TrimSpace(sortBy) != "" {
			queryParts = append(queryParts, "sort_by="+strings.TrimSpace(strings.ToLower(sortBy)))
		}
		if strings.TrimSpace(order) != "" {
			queryParts = append(queryParts, "order="+strings.TrimSpace(strings.ToLower(order)))
		}
		if len(queryParts) > 0 {
			url += "?" + strings.Join(queryParts, "&")
		}

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			fmt.Println("❌ Failed to create request:", err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(tokenData)))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Failed to reach server. Is it running?")
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ Failed to list library: %s\n", extractLibraryError(body))
			return
		}

		var data models.LibraryListResponse
		if err := json.Unmarshal(body, &data); err != nil {
			fmt.Println("❌ Invalid server response")
			return
		}

		if data.Total == 0 {
			fmt.Println("Your library is empty.")
			fmt.Println("Get started by searching and adding manga:")
			fmt.Println("  mangahub manga search \"your favorite series\"")
			fmt.Println("  mangahub library add --manga-id <id> --status reading")
			return
		}

		fmt.Printf("Your Manga Library (%d entries)\n", data.Total)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTitle\tChapter\tRating\tStatus\tStarted\tUpdated")
		for _, e := range data.Entries {
			rating := "Unrated"
			if e.Rating > 0 {
				rating = fmt.Sprintf("%d/10", e.Rating)
			}
			chapters := fmt.Sprintf("%d/%d", e.CurrentChapter, e.TotalChapters)
			started := "-"
			if !e.StartedReading.IsZero() {
				started = e.StartedReading.Format("2006-01-02")
			}
			updated := "-"
			if !e.UpdatedAt.IsZero() {
				updated = humanDuration(time.Since(e.UpdatedAt))
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", e.MangaID, e.Title, chapters, rating, e.Status, started, updated)
		}
		w.Flush()
	},
}

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return "Just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	}
	if d < 24*time.Hour {
		hrs := int(d.Hours())
		if hrs == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hrs)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	if days < 7 {
		return fmt.Sprintf("%d days", days)
	}
	weeks := days / 7
	if weeks == 1 {
		return "1 week"
	}
	return fmt.Sprintf("%d weeks", weeks)
}

var removeLibraryCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove manga from library",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		url := "http://localhost:8080/users/library/" + strings.TrimSpace(libraryMangaID)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			fmt.Println("❌ Failed to create request:", err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(tokenData)))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Failed to reach server. Is it running?")
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ Failed to remove from library: %s\n", extractLibraryError(body))
			return
		}
		fmt.Printf("✅ Removed '%s' from your library.\n", libraryMangaID)
	},
}

var updateLibraryCmd = &cobra.Command{
	Use:   "update",
	Short: "Update library entry",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		payload := models.UpdateLibraryRequest{
			Status: strings.ToLower(strings.TrimSpace(libraryStatus)),
		}
		if cmd.Flags().Changed("rating") {
			r := libraryRating
			payload.Rating = &r
		}

		body, _ := json.Marshal(payload)
		url := "http://localhost:8080/users/library/" + strings.TrimSpace(libraryMangaID)
		req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
		if err != nil {
			fmt.Println("❌ Failed to create request:", err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(tokenData)))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Failed to reach server. Is it running?")
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ Failed to update library: %s\n", extractLibraryError(respBody))
			return
		}
		fmt.Printf("✅ Updated '%s' in your library.\n", libraryMangaID)
	},
}

func extractLibraryError(body []byte) string {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err == nil {
		if msg, ok := payload["error"].(string); ok && msg != "" {
			return msg
		}
	}
	return string(body)
}

func init() {
	LibraryCmd.AddCommand(addLibraryCmd)
	LibraryCmd.AddCommand(listLibraryCmd)
	LibraryCmd.AddCommand(removeLibraryCmd)
	LibraryCmd.AddCommand(updateLibraryCmd)

	addLibraryCmd.Flags().StringVar(&libraryMangaID, "manga-id", "", "manga id")
	addLibraryCmd.Flags().StringVar(&libraryStatus, "status", "", "status: reading, completed, plan-to-read, on-hold, dropped")
	addLibraryCmd.Flags().IntVar(&libraryRating, "rating", 0, "rating from 0 to 10")
	addLibraryCmd.MarkFlagRequired("manga-id")
	addLibraryCmd.MarkFlagRequired("status")

	listLibraryCmd.Flags().StringVar(&libraryStatus, "status", "", "filter by status")
	listLibraryCmd.Flags().StringVar(&sortBy, "sort-by", "last-updated", "sort by title or last-updated")
	listLibraryCmd.Flags().StringVar(&order, "order", "desc", "sort order asc or desc")

	removeLibraryCmd.Flags().StringVar(&libraryMangaID, "manga-id", "", "manga id")
	removeLibraryCmd.MarkFlagRequired("manga-id")

	updateLibraryCmd.Flags().StringVar(&libraryMangaID, "manga-id", "", "manga id")
	updateLibraryCmd.Flags().StringVar(&libraryStatus, "status", "", "new status")
	updateLibraryCmd.Flags().IntVar(&libraryRating, "rating", 0, "new rating from 0 to 10")
	updateLibraryCmd.MarkFlagRequired("manga-id")
}
