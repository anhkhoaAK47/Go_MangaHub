package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func (h *ChatHub) Start() {
	go h.runHub()
	log.Println("[WS] Chat hub started")
}

func (h* ChatHub) Stop() {
	close(h.quit)
}

// event loop - handles register, unregister, broadcast
func (h *ChatHub) runHub() {
	for {
		select {
		
		// client joins a room
		case client := <- h.Register:
			h.mu.Lock()
			room, exists := h.Rooms[client.RoomID]
			h.mu.Unlock()

			// if room dont exists
			if !exists {
				log.Printf("[WS] Room %s not found for client %s\n", client.RoomID, client.Username)
				continue
			}

			// add client to room if does
			room.mu.Lock()
			room.Clients[client.Conn] = client
			userCount := len(room.Clients)
			room.mu.Unlock()

			log.Printf("[WS %s joined %s (%d users)\n", client.Username, room.Name, userCount)

			// send chat history to new client
			room.mu.RLock()

			// converts message history to JSON
			history, _ := json.Marshal(room.History)
			historyMsg := ChatMessage {
				Type: "history",
				RoomID: client.RoomID,
				Message: string(history), // convert to string
			}
			room.mu.RUnlock()
			sendToClient(client.Conn, historyMsg)

			// broadcast join notification to everyone
			joinMsg := ChatMessage{
				Type: "join",
				UserID: client.UserID,
				Username: client.Username,
				RoomID: client.RoomID,
				Message: client.Username + " joined the chat",
				Timestamp: time.Now().Unix(),
			}
			h.broadcastToRoom(client.RoomID, joinMsg, nil)

		// client leaves a room
		case conn := <- h.Unregister:
			h.mu.RLock()
			rooms := h.Rooms
			h.mu.RUnlock()

			// finds room and remove connection from it
			for _, room := range rooms {
				room.mu.Lock()
				client, exists := room.Clients[conn]
				if exists {
					delete(room.Clients, conn)
					username := client.Username
					roomID := client.RoomID
					room.mu.Unlock()
					
					// broadcast to everyone that client left the room
					leaveMsg := ChatMessage {
						Type: "leave",
						Username: username,
						RoomID: roomID,
						Message: username + " left the chat",
						Timestamp: time.Now().Unix(),
					}
					h.broadcastToRoom(roomID, leaveMsg, nil)
					log.Printf("[WS] %s left %s\n", username, room.Name)
				} else {
					room.mu.Unlock()
				}
			}
		
		// broadcast a message to a room
		case msg := <- h.Broadcast:
			// save message to DB
			h.saveMessage(msg)

			h.mu.RLock()
			room, exists := h.Rooms[msg.RoomID]
			h.mu.RUnlock()

			// save to history
			if exists {
				room.mu.Lock()
				room.History = append(room.History, msg)

				// Keep last 100 messages in memory
				if len(room.History) > 100 {
					room.History = room.History[len(room.History) - 100:]
				}
				room.mu.Unlock()

				// broadcast message to everyone in the room
				h.broadcastToRoom(msg.RoomID, msg, nil)
			}

		// user sends private message
		case pm := <- h.privateMsg:
			h.handlePrivateMessage(pm)

		
		case req := <- h.userListReq:
			h.handleUserList(req)
		
		case <- h.quit:
			return
		}
	}
}

// sends a message to all clients in a room, skipConns skips the sender
func (h *ChatHub) broadcastToRoom(roomID string, msg ChatMessage, skipConn *websocket.Conn) {
	h.mu.RLock()
	room, exists := h.Rooms[roomID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	// loop through every client
	for conn := range room.Clients {
		// skip broadcast to sender
		if conn == skipConn {
			continue
		}
		sendToClient(conn, msg)
	}
}

// sends a message to a single websocket connection
func sendToClient(conn *websocket.Conn, msg ChatMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[WS] Write error: %v\n", err)
	}
}

func (h* ChatHub) handlePrivateMessage(pm *PrivateMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	found := false
	for _, room := range h.Rooms {
		room.mu.RLock()
		for conn, client := range room.Clients {
			if client.Username == pm.Recipient {
				// create "pm" type message
				privateMsg := ChatMessage{
					Type: "pm",
					Username: pm.Sender.Username,
					Message: string(pm.Message),
					Timestamp: time.Now().Unix(),
				}
				sendToClient(conn, privateMsg)
				found = true
			}
		}
		room.mu.RUnlock()
		if found {
			break
		}
	}
	if !found {
		// error handling
		sendToClient(pm.Sender.Conn, ChatMessage{
			Type: "system",
			Message: "User " + pm.Recipient + " not found.",
		})
	}
}


func (h *ChatHub) handleUserList(req *UserListRequest) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var allUsers []UserInfo

	// loop through every room in the hub
	for _, room := range h.Rooms {
		room.mu.RLock()
		// loop through every client in that specific room
		for _, client := range room.Clients {
			allUsers = append(allUsers, UserInfo{
				Username: client.Username,
				RoomName: room.Name,
			})
		}
		room.mu.RUnlock()
	}

	// send the full list back to the requester
	req.Response <- allUsers
}