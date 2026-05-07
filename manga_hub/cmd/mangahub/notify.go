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

var NotifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Manage chapter release notifications",
}

var (
	notifyGenre   string
	notifyMangaID string
)

// Subscribe to chapter release notifications
var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to chapter release notifications",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		client := &http.Client{}
		req, _ := http.NewRequest("POST", "http://localhost:8080/notify/subscribe", nil)
		req.Header.Set("Authorization", "Bearer "+string(tokenData))

		if notifyMangaID != "" {
			req.Header.Set("X-Manga-ID", notifyMangaID)
		}
		if notifyGenre != "" {
			req.Header.Set("X-Genre", notifyGenre)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Error connecting to server:", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			json.Unmarshal(body, &result)
			fmt.Println("✅ Successfully subscribed to notifications")
			if notifyMangaID != "" {
				fmt.Printf("📚 Manga ID: %s\n", notifyMangaID)
			}
			if notifyGenre != "" {
				fmt.Printf("📖 Genre: %s\n", notifyGenre)
			}
		} else {
			fmt.Printf("❌ Error: %s\n", string(body))
		}
	},
}

// Unsubscribe from notifications
var unsubscribeCmd = &cobra.Command{
	Use:   "unsubscribe",
	Short: "Unsubscribe from chapter release notifications",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		client := &http.Client{}
		req, _ := http.NewRequest("POST", "http://localhost:8080/notify/unsubscribe", nil)
		req.Header.Set("Authorization", "Bearer "+string(tokenData))

		if notifyMangaID != "" {
			req.Header.Set("X-Manga-ID", notifyMangaID)
		}
		if notifyGenre != "" {
			req.Header.Set("X-Genre", notifyGenre)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Error connecting to server:", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			json.Unmarshal(body, &result)
			fmt.Println("✅ Successfully unsubscribed from notifications")
			if notifyMangaID != "" {
				fmt.Printf("📚 Manga ID: %s\n", notifyMangaID)
			}
			if notifyGenre != "" {
				fmt.Printf("📖 Genre: %s\n", notifyGenre)
			}
		} else {
			fmt.Printf("❌ Error: %s\n", string(body))
		}
	},
}

// View notification preferences
var preferencesCmd = &cobra.Command{
	Use:   "preferences",
	Short: "View your notification preferences",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		client := &http.Client{}
		req, _ := http.NewRequest("GET", "http://localhost:8080/notify/preferences", nil)
		req.Header.Set("Authorization", "Bearer "+string(tokenData))

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Error connecting to server:", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			var preferences map[string]interface{}
			json.Unmarshal(body, &preferences)

			fmt.Println("\n📋 Your Notification Preferences")
			fmt.Println(strings.Repeat("─", 50))

			if prefs, ok := preferences["preferences"].(map[string]interface{}); ok {
				for key, value := range prefs {
					// Skip keys we don't want to display
					if key == "subscribed_genres" || key == "user_id" {
						continue
					}
					// Format subscribed_mangas with commas when it's an array
					if key == "subscribed_mangas" {
						switch v := value.(type) {
						case string:
							fmt.Printf("  • %s: %s\n", key, v)
						case []interface{}:
							parts := make([]string, 0, len(v))
							for _, it := range v {
								parts = append(parts, fmt.Sprintf("%v", it))
							}
							fmt.Printf("  • %s: [%s]\n", key, strings.Join(parts, ", "))
						default:
							fmt.Printf("  • %s: %v\n", key, value)
						}
						continue
					}
					fmt.Printf("  • %s: %v\n", key, value)
				}
			} else {
				for key, value := range preferences {
					fmt.Printf("  • %s: %v\n", key, value)
				}
			}
			fmt.Println(strings.Repeat("─", 50))
		} else {
			fmt.Printf("❌ Error: %s\n", string(body))
		}
	},
}

// Test notification system
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the notification system with a sample notification",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		client := &http.Client{}
		req, _ := http.NewRequest("POST", "http://localhost:8080/notify/test", nil)
		req.Header.Set("Authorization", "Bearer "+string(tokenData))

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Error connecting to server:", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			json.Unmarshal(body, &result)
			fmt.Println("✅ Test notification sent successfully!")
			fmt.Println("📬 Check your notification preferences for more details")
		} else {
			fmt.Printf("❌ Error: %s\n", string(body))
		}
	},
}

func init() {
	// Register subscribe command
	NotifyCmd.AddCommand(subscribeCmd)
	subscribeCmd.Flags().StringVar(&notifyMangaID, "manga-id", "", "Manga ID to subscribe to (optional)")
	subscribeCmd.Flags().StringVar(&notifyGenre, "genre", "", "Genre to subscribe to (optional)")

	// Register unsubscribe command
	NotifyCmd.AddCommand(unsubscribeCmd)
	unsubscribeCmd.Flags().StringVar(&notifyMangaID, "manga-id", "", "Manga ID to unsubscribe from (optional)")
	unsubscribeCmd.Flags().StringVar(&notifyGenre, "genre", "", "Genre to unsubscribe from (optional)")

	// Register preferences command
	NotifyCmd.AddCommand(preferencesCmd)

	// Register test command
	NotifyCmd.AddCommand(testCmd)
}
