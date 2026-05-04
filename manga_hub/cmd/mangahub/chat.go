package mangahub

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var chatMangaID string

var ChatCmd = &cobra.Command{
	Use: "chat",
	Short: "Join manga discussion chat roomns",
}

var joinCmd = &cobra.Command{
	Use: "join",
	Short: "Join a chat room",
	Run: func(cmd *cobra.Command, args []string) {

		// check if user logged in
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Try: mangahub auth login --username <username>")
			return
		}
		token := strings.TrimSpace(string(tokenData))

		// Determine which room to join
		roomParam := "general"
		if chatMangaID != "" {
			roomParam = chatMangaID
		}

		// Connect to WebSocket server
		wsURL := url.URL{
			Scheme: "ws",
			Host: "localhost:9093",
			Path: "/chat/" + roomParam,
			RawQuery: "token=" + token,
		}

		fmt.Printf("Connecting to WebSocket chat server at ws://localhost:9093...\n")

		// set token in header
		header := http.Header{}
		header.Set("Authorization", "Bearer " + token)

		// dial the websocket
		conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), header)
		if err != nil {
			fmt.Println("❌ Failed to connect to chat server")
			fmt.Println("	Try: manga server start")
			return
		}
		defer conn.Close()

		// waits for history message
		_, historyData, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("❌ Failed to receive room info")
			return
		}

		var historyMsg map[string]interface{}
		json.Unmarshal(historyData, &historyMsg)
		
		// get room display name
		roomName := "General Chat" // default
		roomTag := "#general"
		if chatMangaID != "" {
			roomName = chatMangaID + " Chatroom"
			roomTag = "#" + chatMangaID
		}

		// print room header
		fmt.Printf("✓ Connected to %s\n\n", roomName)
		fmt.Printf("Chat Room: %s\n", roomTag)
		fmt.Printf("Your status: Online\n")

		// print history message
		if historyMsg["type"] == "history" {
			var messages []map[string]interface{}
			rawHistory := historyMsg["message"].(string)
			if err := json.Unmarshal([]byte(rawHistory), &messages); err == nil && len(messages) > 0 {
				fmt.Println("\nRecent messages:")
				for _, m := range messages {
					ts := int64(m["timestamp"].(float64))
					t := time.Unix(ts, 0).Format("15:04")
					fmt.Printf("	[%s] %s: %s\n", t, m["username"], m["message"])	
				}
			} else {
				fmt.Println("\nNo messages yet. Type something to be the first in chat!")
			}
		}
		fmt.Println("\n" + strings.Repeat("─", 60))
		fmt.Println("You are now in chat. Type your message and press Enter.")
		fmt.Println("Type /help for commands or /quit to leave.")
		fmt.Println()

		// create two gorountines - one for reading messages, one for reading user input
		// they run concurrently
		done := make(chan bool)

		// receive and print messages from other users
		go func() {
			for {
				_, rawMsg, err := conn.ReadMessage()
				if err != nil {
					done <- true
					return
				}

				var msg map[string]interface{}
				if err := json.Unmarshal(rawMsg, &msg); err != nil {
					continue
				}

				msgType, _ := msg["type"].(string)
				switch msgType {
				case "message":
					ts := int64(msg["timestamp"].(float64))
					t := time.Unix(ts, 0).Format("15:04")
					fmt.Printf("\r[%s] %s: %s\n", t, msg["username"], msg["message"])
				
				case "join":
					fmt.Printf("\r %s joined the room\n", msg["username"])
				
				case "leave":
					fmt.Printf("\r %s left the room\n", msg["username"])
				}
			}
		}()

		// read user input
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("> ")

		for scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				fmt.Print("> ")
				continue
			}

			// chat commands handler
			switch {
			case input == "/quit":
				fmt.Printf("Leaving chat...")
				return
			case input == "/help":
				fmt.Println("  /help    - Show this help")
				fmt.Println("  /users	- List online users")
				fmt.Println("  /quit    - Leave chat")		
				fmt.Println("  /pm <user> <msg>	- Private message")
				fmt.Println("  /manga <id>	- Switch to manga chat")
				fmt.Println("  /history		- Show recent history")
				fmt.Println("  /status		- Connection status")	
			
			default:
				if err := conn.WriteMessage(websocket.TextMessage, []byte(input)); err != nil {
					fmt.Println("❌ Failed to send message")
					return
				}
			}

			select {
			case <- done:
				fmt.Println("✓ Disconnected from chat server")
				return
			default:
				fmt.Print("> ")
			}
		}
	},
}

func init() {
	// add join command
	ChatCmd.AddCommand(joinCmd)

	// join command flags
	joinCmd.Flags().StringVar(&chatMangaID, "manga-id", "", "Join a manga-specific chat room")
}
