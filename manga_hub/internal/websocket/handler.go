package websocket

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// converts HTTP connection into WebSocket connection
var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}


// chat room handler
func HandleChatRoom(hub *ChatHub, db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomParam := c.Param("room") // "general" or "one-piece"

		// get user info from JWT middleware
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")

		userIDStr, _ := userID.(string)
		usernameStr, _ := username.(string)

		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized, Not logged in",
			})
			return
		}

		// get roomID and roomName
		var roomID, roomName string // placeholder
		if roomParam == "general" || roomParam == "" {
			roomID = "general"
			roomName = "General Chat"
			hub.createRoom(roomID, roomName)
		} else { // if manga room
			roomID, roomName = hub.GetOrCreateMangaRoom(roomParam, db)
		}

		// upgrade HTTP to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("[WS] Upgrade error:", err)
			return
		}

		// create client
		client := &ClientConnection{
			Conn: conn,
			UserID: userIDStr,
			Username: usernameStr,
			RoomID: roomID,
		}
		// register with hub
		hub.Register <- client

		// handle incoming messages from this client
		go handleMessages(hub, client, conn)

		log.Printf("[WS] %s connected to %s\n", usernameStr, roomName)
	}
}

func handleMessages(hub *ChatHub, client *ClientConnection, conn *websocket.Conn) {
	defer func() {
		hub.Unregister <- conn
		conn.Close()
	}()

	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			// client disconnected
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Unexpected close from %s: %v\n", client.Username, err)
			}
			break
		}

		text := strings.TrimSpace(string(rawMsg))
		log.Printf("[WS] Received from %s: '%s'\n", client.Username, text)

		if text == "" {
			continue
		}

		// /pm command
		if strings.HasPrefix(text, "/pm") {
			parts := strings.Fields(text)
			if len(parts) >= 3 {
				hub.privateMsg <- &PrivateMessage{
					Sender: client,
					Recipient: parts[1],
					Message: []byte(strings.Join(parts[2:], " ")),
				}
			}
			continue
		}

		// /users command
		if text == "/users" {
			respChan := make(chan []UserInfo)

			// send request to the hub
			hub.userListReq <- &UserListRequest{
				Response: respChan,
			}

			// wait for answer and send to client
			users := <- respChan
			userData, _ := json.Marshal(users)

			sendToClient(conn, ChatMessage{
				Type:    "user_list",
				Message: string(userData),
			})
			continue
		}

		// msg placeholder
		msg := ChatMessage {
			Type: "message",
			UserID: client.UserID,
			Username: client.Username,
			Message: text,
			Timestamp: time.Now().Unix(),
			RoomID: client.RoomID,
		}

		// broadcast message
		hub.Broadcast <- msg
	}
}