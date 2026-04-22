package mangahub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"go_mangahub/manga_hub/pkg/models"

	"github.com/spf13/cobra"
)

var ProgressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Track reading progress",
}

var (
	progressMangaID string
	progressChapter int
	progressVolume  int
	progressNotes   string
	progressForce   bool
)

var progressUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update reading progress",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		fmt.Println("Updating reading progress...")

		payload := models.ProgressUpdateRequest{
			MangaID: strings.TrimSpace(progressMangaID),
			Chapter: progressChapter,
			Volume:  progressVolume,
			Notes:   strings.TrimSpace(progressNotes),
			Force:   progressForce,
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest(http.MethodPut, "http://localhost:8080/users/progress", bytes.NewBuffer(body))
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
			fmt.Printf("✗ %s\n", extractErrorMessage(respBody))
			return
		}

		var result models.ProgressUpdateResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			fmt.Println("❌ Invalid server response")
			return
		}

		delta := result.CurrentChapter - result.PreviousChapter
		if delta >= 0 {
			fmt.Println("✓ Progress updated successfully!")
		} else {
			fmt.Println("✓ Progress updated successfully! (forced backwards update)")
		}
		fmt.Printf("Manga: %s\n", result.Title)
		fmt.Printf("Previous: Chapter %s\n", formatNumber(result.PreviousChapter))
		fmt.Printf("Current: Chapter %s (%+d)\n", formatNumber(result.CurrentChapter), delta)
		fmt.Printf("Updated: %s\n", result.UpdatedAt.UTC().Format("2006-01-02 15:04:05 MST"))
		fmt.Println("Sync Status:")
		fmt.Printf(" Local database: %s %s\n", syncMark(result.Sync.LocalDatabase), result.Sync.LocalDatabase)
		fmt.Printf(" TCP sync server: %s %s\n", syncMark(result.Sync.TCPServer), result.Sync.TCPServer)
		fmt.Printf(" Cloud backup: %s %s\n", syncMark(result.Sync.CloudBackup), result.Sync.CloudBackup)
		fmt.Println("Statistics:")
		fmt.Printf(" Total chapters read: %s\n", formatNumber(result.Statistics.TotalChaptersRead))
		fmt.Printf(" Reading streak: %d days\n", result.Statistics.ReadingStreakDays)
		fmt.Printf(" Estimated completion: %s\n", result.Statistics.EstimatedCompletion)
		fmt.Println("Next actions:")
		if result.Statistics.NextAvailableChapter > 0 {
			fmt.Printf(" Continue reading: Chapter %s available\n", formatNumber(result.Statistics.NextAvailableChapter))
		} else {
			fmt.Println(" Continue reading: No next chapter available")
		}
		fmt.Printf(" Rate this chapter: mangahub library update --manga-id %s --rating 9\n", result.MangaID)
	},
}

var progressHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "View progress history",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		url := "http://localhost:8080/users/progress/history"
		if strings.TrimSpace(progressMangaID) != "" {
			url += "?manga_id=" + strings.TrimSpace(progressMangaID)
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
			fmt.Printf("❌ Failed to get history: %s\n", extractErrorMessage(body))
			return
		}

		var payload struct {
			History []models.ProgressHistoryEntry `json:"history"`
			Total   int                           `json:"total"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			fmt.Println("❌ Invalid server response")
			return
		}

		if payload.Total == 0 {
			fmt.Println("No progress history yet.")
			return
		}

		fmt.Printf("Progress History (%d entries)\n", payload.Total)
		for _, h := range payload.History {
			base := fmt.Sprintf("- %s | %s: %s -> %s | %s",
				h.UpdatedAt.UTC().Format("2006-01-02 15:04:05"),
				h.MangaID,
				formatNumber(h.PreviousChapter),
				formatNumber(h.CurrentChapter),
				h.Notes,
			)
			if h.Notes == "" {
				base = fmt.Sprintf("- %s | %s: %s -> %s",
					h.UpdatedAt.UTC().Format("2006-01-02 15:04:05"),
					h.MangaID,
					formatNumber(h.PreviousChapter),
					formatNumber(h.CurrentChapter),
				)
			}
			fmt.Println(base)
		}
	},
}

var progressSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manually sync progress",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}
		req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/users/progress/sync", nil)
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
			fmt.Printf("❌ Sync failed: %s\n", extractErrorMessage(body))
			return
		}
		fmt.Println("✓ Progress synced successfully.")
	},
}

var progressSyncStatusCmd = &cobra.Command{
	Use:   "sync-status",
	Short: "Check sync status",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/users/progress/sync-status", nil)
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
			fmt.Printf("❌ Failed to fetch sync status: %s\n", extractErrorMessage(body))
			return
		}

		var payload struct {
			Sync models.SyncStatus `json:"sync"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			fmt.Println("❌ Invalid server response")
			return
		}

		fmt.Println("Sync Status:")
		fmt.Printf(" Local database: %s\n", payload.Sync.LocalDatabase)
		fmt.Printf(" TCP sync server: %s\n", payload.Sync.TCPServer)
		fmt.Printf(" Cloud backup: %s\n", payload.Sync.CloudBackup)
	},
}

func extractErrorMessage(body []byte) string {
	var errPayload map[string]interface{}
	if err := json.Unmarshal(body, &errPayload); err == nil {
		if msg, ok := errPayload["error"].(string); ok && msg != "" {
			if hint, ok := errPayload["hint"].(string); ok && hint != "" {
				return msg + "\n " + hint
			}
			return msg
		}
	}
	return string(body)
}

func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	out := ""
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(r)
	}
	return out
}

func syncMark(status string) string {
	lower := strings.ToLower(status)
	if strings.Contains(lower, "pending") || strings.Contains(lower, "failed") || strings.Contains(lower, "error") {
		return "⚠"
	}
	return "✓"
}

func init() {
	ProgressCmd.AddCommand(progressUpdateCmd)
	ProgressCmd.AddCommand(progressHistoryCmd)
	ProgressCmd.AddCommand(progressSyncCmd)
	ProgressCmd.AddCommand(progressSyncStatusCmd)

	progressUpdateCmd.Flags().StringVar(&progressMangaID, "manga-id", "", "manga id")
	progressUpdateCmd.Flags().IntVar(&progressChapter, "chapter", 0, "current chapter")
	progressUpdateCmd.Flags().IntVar(&progressVolume, "volume", 0, "current volume")
	progressUpdateCmd.Flags().StringVar(&progressNotes, "notes", "", "progress notes")
	progressUpdateCmd.Flags().BoolVar(&progressForce, "force", false, "allow backwards chapter update")
	progressUpdateCmd.MarkFlagRequired("manga-id")
	progressUpdateCmd.MarkFlagRequired("chapter")

	progressHistoryCmd.Flags().StringVar(&progressMangaID, "manga-id", "", "filter history by manga id")
}
