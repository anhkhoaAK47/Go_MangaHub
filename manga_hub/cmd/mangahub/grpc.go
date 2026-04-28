package mangahub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var GrpcCmd = &cobra.Command{
	Use:   "grpc",
	Short: "Perform operations via gRPC service",
}

var (
	grpcMangaID string
	grpcQuery   string
	grpcChapter int
	grpcStatus  string
)

// Manga command group for gRPC
var grpcMangaCmd = &cobra.Command{
	Use:   "manga",
	Short: "Query manga via gRPC",
}

// Get manga by ID via gRPC
var grpcMangaGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Query manga via gRPC",
	Run: func(cmd *cobra.Command, args []string) {
		if grpcMangaID == "" {
			fmt.Println("❌ Error: --id flag is required")
			fmt.Println("Usage: mangahub grpc manga get --id <manga-id>")
			return
		}

		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		client := &http.Client{}
		req, _ := http.NewRequest("GET", "http://localhost:8080/grpc/manga/"+strings.TrimSpace(grpcMangaID), nil)
		req.Header.Set("Authorization", "Bearer "+string(tokenData))

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Error connecting to gRPC service:", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		switch resp.StatusCode {
			case http.StatusOK:
			var manga map[string]interface{}
			json.Unmarshal(body, &manga)

			fmt.Println("\n📚 Manga Information (via gRPC)")
			fmt.Println(strings.Repeat("─", 50))

			if title, ok := manga["title"].(string); ok {
				fmt.Printf("  Title: %s\n", title)
			}
			if author, ok := manga["author"].(string); ok {
				fmt.Printf("  Author: %s\n", author)
			}
			if artist, ok := manga["artist"].(string); ok {
				fmt.Printf("  Artist: %s\n", artist)
			}
			if status, ok := manga["status"].(string); ok {
				fmt.Printf("  Status: %s\n", status)
			}
			if chapters, ok := manga["totalChapters"].(float64); ok {
				fmt.Printf("  Total Chapters: %.0f\n", chapters)
			}

			fmt.Println(strings.Repeat("─", 50))
			fmt.Println("✅ Data retrieved from gRPC service")
		case http.StatusNotFound:
			fmt.Printf("❌ Manga not found: '%s'\n", grpcMangaID)
		default:
			fmt.Printf("❌ Error: %s\n", string(body))
		}
	},
}

// Search manga via gRPC
var grpcMangaSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search manga via gRPC",
	Run: func(cmd *cobra.Command, args []string) {
		if grpcQuery == "" {
			fmt.Println("❌ Error: --query flag is required")
			fmt.Println("Usage: mangahub grpc manga search --query <search-term>")
			return
		}

		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		client := &http.Client{}
		req, _ := http.NewRequest("GET", "http://localhost:8080/grpc/manga/search?q="+strings.TrimSpace(grpcQuery), nil)
		req.Header.Set("Authorization", "Bearer "+string(tokenData))

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Error connecting to gRPC service:", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			var results map[string]interface{}
			json.Unmarshal(body, &results)

			fmt.Printf("\n🔍 Search Results for '%s' (via gRPC)\n", grpcQuery)
			fmt.Println(strings.Repeat("─", 50))

			if mangas, ok := results["mangas"].([]interface{}); ok {
				if len(mangas) == 0 {
					fmt.Println("  No manga found")
				} else {
					for i, m := range mangas {
						if manga, ok := m.(map[string]interface{}); ok {
							fmt.Printf("  %d. ", i+1)
							if title, ok := manga["title"].(string); ok {
								fmt.Printf("%s", title)
							}
							if status, ok := manga["status"].(string); ok {
								fmt.Printf(" [%s]", status)
							}
							fmt.Println()
						}
					}
				}
			}

			fmt.Println(strings.Repeat("─", 50))
			fmt.Println("✅ Search completed via gRPC service")
		} else {
			fmt.Printf("❌ Error: %s\n", string(body))
		}
	},
}

// Progress command group for gRPC
var grpcProgressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Manage progress via gRPC",
}

// Update progress via gRPC
var grpcProgressUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update progress via gRPC",
	Run: func(cmd *cobra.Command, args []string) {
		if grpcMangaID == "" || grpcChapter == 0 {
			fmt.Println("❌ Error: --manga-id and --chapter flags are required")
			fmt.Println("Usage: mangahub grpc progress update --manga-id <id> --chapter <number>")
			return
		}

		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		payload := map[string]interface{}{
			"manga_id": strings.TrimSpace(grpcMangaID),
			"chapter":  grpcChapter,
		}
		if grpcStatus != "" {
			payload["status"] = grpcStatus
		}

		bodyBytes, _ := json.Marshal(payload)

		client := &http.Client{}
		req, _ := http.NewRequest("POST", "http://localhost:8080/grpc/progress/update", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Authorization", "Bearer "+string(tokenData))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Error connecting to gRPC service:", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			json.Unmarshal(body, &result)

			fmt.Println("\n✅ Progress Updated (via gRPC)")
			fmt.Println(strings.Repeat("─", 50))
			fmt.Printf("  Manga ID: %s\n", grpcMangaID)
			fmt.Printf("  Chapter: %d\n", grpcChapter)
			if grpcStatus != "" {
				fmt.Printf("  Status: %s\n", grpcStatus)
			}
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println("✅ Update confirmed via gRPC service")
		} else {
			fmt.Printf("❌ Error: %s\n", string(body))
		}
	},
}

func init() {
	// Register manga subcommand to grpc
	GrpcCmd.AddCommand(grpcMangaCmd)

	// Register manga get command
	grpcMangaCmd.AddCommand(grpcMangaGetCmd)
	grpcMangaGetCmd.Flags().StringVar(&grpcMangaID, "id", "", "Manga ID")
	grpcMangaGetCmd.MarkFlagRequired("id")

	// Register manga search command
	grpcMangaCmd.AddCommand(grpcMangaSearchCmd)
	grpcMangaSearchCmd.Flags().StringVar(&grpcQuery, "query", "", "Search query")
	grpcMangaSearchCmd.MarkFlagRequired("query")

	// Register progress subcommand to grpc
	GrpcCmd.AddCommand(grpcProgressCmd)

	// Register progress update command
	grpcProgressCmd.AddCommand(grpcProgressUpdateCmd)
	grpcProgressUpdateCmd.Flags().StringVar(&grpcMangaID, "manga-id", "", "Manga ID")
	grpcProgressUpdateCmd.MarkFlagRequired("manga-id")
	grpcProgressUpdateCmd.Flags().IntVar(&grpcChapter, "chapter", 0, "Chapter number")
	grpcProgressUpdateCmd.MarkFlagRequired("chapter")
	grpcProgressUpdateCmd.Flags().StringVar(&grpcStatus, "status", "", "Reading status (optional)")
}
