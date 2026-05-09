package mangahub

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

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

		// load user session for user_id
		userID := loadUserSession()
		if userID == "" {
			fmt.Println("❌ Could not load user session. Please login again.")
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

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ Error: %s\n", string(body))
			return
		}

		fmt.Println("✅ Successfully subscribed to notifications")
		if notifyMangaID != "" {
			fmt.Printf("📚 Manga ID: %s\n", notifyMangaID)
		}
		if notifyGenre != "" {
			fmt.Printf("📖 Genre: %s\n", notifyGenre)
		}

		// UDP registration
		udpServerAddr, err := net.ResolveUDPAddr("udp", "localhost:9091")
		if err != nil {
			fmt.Println("⚠️  Could not resolve UDP server address")
			return
		}

		// Bind a local UDP port — OS assigns a random available port
		localAddr, err := net.ResolveUDPAddr("udp", ":0")
		if err != nil {
			fmt.Println("⚠️  Could not bind local UDP port")
			return
		}

		conn, err := net.ListenUDP("udp", localAddr)
		if err != nil {
			fmt.Println("⚠️  Could not open local UDP port")
			return
		}

		// Get the port the OS assigned
		localPort := conn.LocalAddr().(*net.UDPAddr).Port


		// Send registration packet to server
		reg := map[string]string{
			"type": "register",
			"user_id": userID,
		}

		regData, _ := json.Marshal(reg)
		conn.WriteToUDP(regData, udpServerAddr)

		// wait for server ack
		conn.SetDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 256)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("⚠️  UDP server did not respond — notifications may not work")
			conn.Close()
			return
		}

		var ack map[string]interface{}
		json.Unmarshal(buf[:n], &ack)
		if ack["type"] == "registered" {
			fmt.Printf("✅ UDP registered on local port %d\n", localPort)
			fmt.Println("Run 'mangahub notify listen' to receive notifications")
		}

		// save the local port so listen command can use the same port
		os.WriteFile(".udp_port", []byte(fmt.Sprintf("%d", localPort)), 0644)
		conn.Close()
	},
}

var listenCmd = &cobra.Command{
	Use: "listen",
	Short: "Listen for incoming chapter release notifications",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}
		_ = tokenData

		userID := loadUserSession()
		if userID == "" {
			fmt.Println("❌ Could not load user session. Please login again.")
			return
		}

		// read saved port from subscribe
		var localPort int
		portData, err := os.ReadFile(".udp_port")
		if err == nil {
			fmt.Sscanf(strings.TrimSpace(string(portData)), "%d", &localPort)
		}

		// bind UDP listener
		listenAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", localPort))
		if err != nil {
			fmt.Println("❌ Could not resolve listen address")
			return
		}

		conn, err := net.ListenUDP("udp", listenAddr)
		if err != nil {
			// port might be in use so we bind a new one and re-register
			fmt.Println("Could not bind saved port, getting a new one...")
			conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 0})
			if err != nil {
				fmt.Println("❌ Could not open UDP port")
				return
			}
			localPort = conn.LocalAddr().(*net.UDPAddr).Port

			// Re-register with server on new port
			udpServerAddr, _ := net.ResolveUDPAddr("udp", "localhost:9091")
			reg := map[string]string{
				"type":    "register",
				"user_id": userID,
			}
			regData, _ := json.Marshal(reg)
			conn.WriteToUDP(regData, udpServerAddr)
			os.WriteFile(".udp_port", []byte(fmt.Sprintf("%d", localPort)), 0644)
		}
		defer conn.Close()
		
		fmt.Printf("📡 Listening for notifications on UDP port %d\n", localPort)

		// read loop
		buf := make([]byte, 4096)
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				break
			}

			var notif map[string]interface{}
			if err := json.Unmarshal(buf[:n], &notif); err != nil {
				continue
			}

			// Skip pong/ack packets
			if notif["type"] == "pong" || notif["type"] == "registered" {
				continue
			}

			//  Print notification 
			ts := time.Now().Format("15:04:05")
			fmt.Printf("[%s] New Chapter Alert!\n", ts)
			if title, ok := notif["manga_title"].(string); ok {
				fmt.Printf("   Manga:   %s\n", title)
			}
			if chapter, ok := notif["chapter_num"].(float64); ok {
				fmt.Printf("   Chapter: %.0f\n", chapter)
			}
			if chTitle, ok := notif["chapter_title"].(string); ok && chTitle != "" {
				fmt.Printf("   Title:   %s\n", chTitle)
			}
			if msg, ok := notif["message"].(string); ok {
				fmt.Printf("   %s\n", msg)
			}
			fmt.Println()
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

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ Error: %s\n", string(body))
			return
		}

		var result map[string]interface{}
		json.Unmarshal(body, &result)

		fmt.Println("✅ Test notification sent!")
		fmt.Println()


		// print notification results
		if notif, ok := result["notification"].(map[string]interface{}); ok {
			fmt.Println("Notification Details:")
			fmt.Printf("	Manga: %v\n", notif["manga_title"])
			fmt.Printf("	Chapter: %v\n", notif["chapter_num"])
			fmt.Printf("	Message: %v\n", notif["message"])
		}

		// Tell user if UDP delivery worked
		deliveredUDP, _ := result["delivered_over_udp"].(bool)
		fmt.Println()
		if deliveredUDP {
			fmt.Println("Delivered via UDP ✅")
			fmt.Println("   If you ran 'mangahub notify listen', you should see it there.")
		} else {
			fmt.Println("⚠️  UDP delivery failed.")
			fmt.Println("   Run 'mangahub notify subscribe --manga-id <id>' first,")
			fmt.Println("   then 'mangahub notify listen' in another terminal.")
			if udpErr, ok := result["udp_error"].(string); ok {
				fmt.Printf("   Reason: %s\n", udpErr)
			}
		}
	},
}

func loadUserSession() (string) {
	data, err := os.ReadFile(".session")
	if err != nil {
		return ""
	}

	var session map[string]string
	if err := json.Unmarshal(data, &session); err != nil {
		return ""
	}

	return session["user_id"]
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

	// Listen command to listen for notifications
	NotifyCmd.AddCommand(listenCmd)
}
