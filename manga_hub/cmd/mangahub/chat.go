package mangahub

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

		// set token in header
		header := http.Header{}
		header.Set("Authorization", "Bearer " + token)

		// dial the websocket
		conn, err := connectToRoom(roomParam, token, header)
		if err != nil {
			fmt.Println("❌ Failed to connect to chat server")
			fmt.Println("   Try: mangahub server start")
			return
		}
		defer conn.Close()

		
		fmt.Println("\n" + strings.Repeat("─", 60))
		fmt.Println("You are now in chat. Type your message and press Enter.")
		fmt.Println("Type /help for commands or /quit to leave.")
		fmt.Println()

		// create two gorountines - one for reading messages, one for reading user input
		// they run concurrently
		done := make(chan bool)

		// receive and print messages from other users
		go receiveMessages(conn, done)

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
			
			case input == "/users":
				if err := conn.WriteMessage(websocket.TextMessage, []byte("/users")); err != nil {
					fmt.Println("❌ Failed to request user list")
				}

			case strings.HasPrefix(input, "/pm"):
				parts := strings.Fields(input)
				if len(parts) < 3 {
					fmt.Println("Usage /pm <username> <message>")
				} else {
					if err := conn.WriteMessage(websocket.TextMessage, []byte(input)); err != nil {
						fmt.Println("❌ Failed to send private message")
					}
				}
			
			case strings.HasPrefix(input, "/manga"):
				parts := strings.Fields(input)
				if len(parts) < 2 {
					fmt.Println("Usage: /manga <manga-id>")
				} else {
					newMangaID := parts[1]
					
					newConn, err := switchRoom(conn, newMangaID, token, header, done)
					if err != nil {
						fmt.Printf("❌ Failed to switch to %s\n", newMangaID)
					} else {
						conn = newConn
						roomParam = newMangaID
						done = make(chan bool)
						go receiveMessages(conn, done)
					}
				}
				
			case input == "/history":
				fmt.Println("  Use /manga <id> to rejoin and see history again.")

			case input == "/status":
				fmt.Println("\n  Connection Status:")
				fmt.Printf("  Server:  ws://localhost:9093\n")
				fmt.Printf("  Room:    #%s\n", roomParam)
				fmt.Printf("  Status:  Online\n")
				fmt.Printf("  Time:    %s\n", time.Now().Format("2006-01-02 15:04:05"))
			
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

func connectToRoom(roomParam, token string, header http.Header) (*websocket.Conn, error) {
	wsURL := url.URL{
		Scheme: "ws",
		Host: "localhost:9093",
		Path: "/chat/" + roomParam,
		RawQuery: "token=" + token,
	}

	fmt.Printf("Connecting to WebSocket chat server at ws://localhost:9093...\n")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), header)
	if err != nil {
		return nil, err
	}


	// read history message
	_, historyData, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, err
	}

	var historyMsg map[string]interface{}
	json.Unmarshal(historyData, &historyMsg)
	
	roomName := "General Chat"
	roomTag := "#general"
	if roomParam != "general" {
		roomName = roomParam + " Chatroom"
		roomTag = "#" + roomParam
	}

	fmt.Printf("✓ Connected to %s\n\n", roomName)
	fmt.Printf("Chat Room: %s\n", roomTag)
	fmt.Printf("Your status: Online\n")

	// Print history
	if historyMsg["type"] == "history" {
		var messages []map[string]interface{}
		rawHistory, _ := historyMsg["message"].(string)
		if err := json.Unmarshal([]byte(rawHistory), &messages); err == nil && len(messages) > 0 {
			fmt.Println("\nRecent messages:")
			for _, m := range messages {
				ts := int64(m["timestamp"].(float64))
				t := time.Unix(ts, 0).Format("15:04")
				fmt.Printf("  [%s] %s: %s\n", t, m["username"], m["message"])
			}
		} else {
			fmt.Println("\nNo messages yet. Type something to be the first!")
		}
	}

	return conn, nil
}

func switchRoom(oldConn *websocket.Conn, newMangaID, token string, header http.Header, done chan bool) (*websocket.Conn, error){
	fmt.Printf("Switching to %s chatroom...\n", newMangaID)

	// close old connection
	oldConn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "switching rooms"),
	)
	oldConn.Close()

	select {
	case done <- true:
	default:
	}


	// connect to new room
	newConn, err := connectToRoom(newMangaID, token, header)
	if err != nil {
		return nil, err
	}

	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Printf("Switched to #%s\n", newMangaID)

	return newConn, nil
}

func receiveMessages(conn *websocket.Conn, done chan bool) {
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

				case "pm":
					ts := time.Now().Format("15:04")
					user, _ := msg["username"].(string)
					text, _ := msg["message"].(string)

					fmt.Printf("\r%s [PM FROM %s]: %s\n", ts, user, text)
					fmt.Printf("> ")
				case "system":
					text, _ := msg["message"].(string)
					fmt.Printf("\r [SYSTEM] %s\n", text)
					fmt.Print("> ")
				case "user_list":
				var users []struct {
					Username string `json:"Username"`
					RoomName string `json:"RoomName"`
				}
				
				rawJSON, _ := msg["message"].(string)
				if err := json.Unmarshal([]byte(rawJSON), &users); err == nil {
					fmt.Printf("\rOnline Users (%d):\n", len(users))
					for _, u := range users {
						fmt.Printf(" ● %s (%s)\n", u.Username, u.RoomName)
					}
				}
				fmt.Print("> ")
				}
			}	
}

var historyLimit int

var historyCmd = &cobra.Command{
	Use: "history",
	Short: "View recent chat messages",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Try: mangahub auth login --username <username>")
			return
		}
		token := strings.TrimSpace(string(tokenData))

		// ── Determine room ID ──────────────────────────────────────────────
		roomID := "general"
		roomName := "General Chat"
		if chatMangaID != "" {
			roomID = chatMangaID + "-room"
			roomName = chatMangaID + " Chatroom"
		}

		params := url.Values{}
		params.Set("room_id", roomID)
		if historyLimit > 0 {
			params.Set("limit", strconv.Itoa(historyLimit))
		}

		fullURL := "http://localhost:8080/chat/history?" + params.Encode()

		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			fmt.Println("❌ Failed to create request:", err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Server connection error. Is the server running?")
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ Failed to fetch history: %s\n", string(body))
			return
		}

		// ── Parse and print messages ───────────────────────────────────────
		var messages []map[string]interface{}
		if err := json.Unmarshal(body, &messages); err != nil {
			fmt.Println("❌ Failed to parse response")
			return
		}

		if len(messages) == 0 {
			fmt.Printf("No messages found in %s.\n", roomName)
			return
		}

		fmt.Printf("Chat History — %s (%d messages)\n", roomName, len(messages))
		fmt.Println(strings.Repeat("─", 60))

		for _, m := range messages {
			ts := int64(m["timestamp"].(float64))
			t := time.Unix(ts, 0).Format("2006-01-02 15:04")
			fmt.Printf("[%s] %s: %s\n", t, m["username"], m["message"])
		}

		fmt.Println(strings.Repeat("─", 60))
	},
}

func init() {
	// add join command
	ChatCmd.AddCommand(joinCmd)

	// join command flags
	joinCmd.Flags().StringVar(&chatMangaID, "manga-id", "", "Join a manga-specific chat room")


	// add chat history command
	ChatCmd.AddCommand(historyCmd)
	historyCmd.Flags().StringVar(&chatMangaID, "manga-id", "", "View history for a chat room")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 50, "Number of messages to show (max 100)")
}
