package mangahub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	model "go_mangahub/manga_hub/internal/controllers"

	"github.com/spf13/cobra"
)


var StatsCmd = &cobra.Command{
	Use: "stats",
	Short: "Check your manga reading progress, (total mangas, favorite manga, etc.)",
}

var overviewCmd = &cobra.Command{
	Use: "overview",
	Short: "Check your overall stats",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, _ := os.ReadFile(".token")

		client := &http.Client{}
		req, _ := http.NewRequest("GET", "http://localhost:8080/users/stats/overview", nil)
		req.Header.Add("Authorization", "Bearer " + string(tokenData))

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Server connection error.")
			return
		}
		defer resp.Body.Close()
		

		body, _ := io.ReadAll(resp.Body)
		var stats map[string]interface{}
		json.Unmarshal(body, &stats)

		fmt.Println("Stats Overview:")
		fmt.Println("-------------------------------")
		fmt.Println()

		fmt.Printf("Total Mangas in library: %.0f\n", stats["total_manga"])
		fmt.Printf("Total Chapters read: %.0f\n", stats["total_chapters"])
		fmt.Printf("Favorite genre: %v\n", stats["favorite_genre"])

		fmt.Println()
		fmt.Println("Manga Status:")
		counts := stats["status_counts"].(map[string]interface{})
		for status, count := range counts {
			fmt.Printf("  • %-12s: %.0f\n", status, count)
		}
	},
}

var detailedCmd = &cobra.Command{
	Use: "detailed",
	Short: "View your overall reading stats in more details",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, _ := os.ReadFile(".token")

		client := &http.Client{}
		req, _ := http.NewRequest("GET", "http://localhost:8080/users/stats/detailed", nil)
		req.Header.Add("Authorization", "Bearer " + string(tokenData))

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Server connection error.")
			return
		}
		defer resp.Body.Close()
		

		body, _ := io.ReadAll(resp.Body)
		var stats model.StatsDetailed

		if err := json.Unmarshal(body, &stats); err != nil {
			fmt.Println("❌ Failed to parse response")
			return
		}

		fmt.Println("Detailed Reading Stats")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Printf("Total Manga:    %d\n", int(stats.TotalManga))
		fmt.Printf("Total Chapters: %d\n", int(stats.TotalChapters))
		if stats.MostRead != "" {
			fmt.Printf("Most Read:      %s\n", stats.MostRead)
		}
		if stats.MostIgnored != "" {
			fmt.Printf("Most Ignored:   %s\n", stats.MostIgnored)
		}

		fmt.Println()
		fmt.Println("Per Manga Breakdown:")
		fmt.Println(strings.Repeat("─", 60))

		for _, m := range stats.Mangas {
			fmt.Printf("\n📖 %s\n", m.Title)
			fmt.Printf("   Status:        %s\n", m.Status)
			fmt.Printf("   Progress:      Ch.%d", int(m.CurrentChapter))

			if m.TotalChapters > 0 {
				fmt.Printf(" / %d (%d left)", int(m.TotalChapters), m.ChaptersLeft)
			} else {
				fmt.Printf(" / ongoing")
			}
			fmt.Println()

			if m.Rating > 0 {
				fmt.Printf("   Rating:        %d/10\n", int(m.Rating))
			}
			if m.UpdatesCount > 0 {
				fmt.Printf("   Updates:       %d times (avg %.1f ch/update)\n",
					m.UpdatesCount, m.AvgChaptersPerUpdate)
			}
			if m.StartedReading != "" && m.StartedReading != "0001-01-01T00:00:00Z" {
				fmt.Printf("   Started:       %s\n", formatDate(m.StartedReading))
			}
			if m.LastUpdated != "" {
				fmt.Printf("   Last Updated:  %s\n", formatDate(m.LastUpdated))
			}
		}

		fmt.Println()
		fmt.Println(strings.Repeat("─", 60))
	},
}

func formatDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}


func init() {
	// add overview command to stats command
	StatsCmd.AddCommand(overviewCmd)

	// add detailed command
	StatsCmd.AddCommand(detailedCmd)
}