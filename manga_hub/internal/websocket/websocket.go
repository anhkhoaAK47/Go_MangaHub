package websocket

import (
	"database/sql"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// manages all chat rooms
type ChatHub struct {
	Rooms			map[string]*ChatRoom
	mu				sync.RWMutex
	Broadcast 		chan ChatMessage
	Register		chan *ClientConnection
	Unregister		chan *websocket.Conn
	DB				*sql.DB
	quit			chan bool
}

// message structure
type ChatMessage struct {
	UserID		string `json:"user_id"`
	Username	string `json:"username"`
	Message		string `json:"message"`
	Timestamp	int64  `json:"timestamp"`
	RoomID		string `json:"room_id"`
	Type		string `json:"type"` // message, dm, join, leave, history
}

// holds websocket connection
type ClientConnection struct {
	Conn		*websocket.Conn
	UserID		string
	Username	string
	RoomID		string
}

// holds message history
type ChatRoom struct {
	ID		string
	Name	string
	Clients	map[*websocket.Conn]*ClientConnection
	History	[]ChatMessage
	mu		sync.RWMutex
}


// creates a chat hub
func NewChatHub(db *sql.DB) *ChatHub {
	hub := &ChatHub{
		Rooms: make(map[string]*ChatRoom),
		Broadcast: make(chan ChatMessage),
		Register: make(chan *ClientConnection),
		Unregister: make(chan *websocket.Conn),
		DB: db,
		quit: make(chan bool),
	}

	// default chat room
	hub.createRoom("general", "General Chat")
	return hub
}

func (h *ChatHub) createRoom(roomID, roomName string) *ChatRoom {
	h.mu.Lock()
	defer h.mu.Unlock()

	// return room if room already exists
	if room, exists := h.Rooms[roomID]; exists {
		return room
	}

	// load history from DB for this room
	history := h.loadHistory(roomID)

	room := &ChatRoom{
		ID: roomID,
		Name: roomName,
		Clients: make(map[*websocket.Conn]*ClientConnection),
		History: history,
	}
	// add new room into chatHub
	h.Rooms[roomID] = room

	// logs the new room in server
	log.Printf("[WS] Created room: %s (%s)\n", roomName, roomID)
	return room
}

func (h *ChatHub) loadHistory(roomID string) []ChatMessage {
	history := []ChatMessage{}

	if h.DB == nil {
		return history
	}

	rows, err := h.DB.Query(`
		SELECT user_id, username, message, timestamp, room_id
		FROM chat_messages
		WHERE room_id = ?
		ORDER BY timestamp ASC
		LIMIT 50
	`, roomID)
	if err != nil {
		return history
	}
	defer rows.Close()


	for rows.Next() {
		// create message placeholder
		var msg ChatMessage

		if err := rows.Scan(&msg.UserID, &msg.Username, &msg.Message, &msg.Timestamp, &msg.RoomID); err != nil {
			continue
		}
		msg.Type = "message"
		history = append(history, msg)
	}
	return history
}

// save messages to database
func (h *ChatHub) saveMessage(msg ChatMessage) {
	if h.DB == nil {
		return
	}
	h.DB.Exec(`
		INSERT INTO chat_messages (user_id, username, message, timestamp, room_id)
		VALUES (?, ?, ?, ?, ?)
	`, msg.UserID, msg.Username, msg.Message, msg.Timestamp, msg.RoomID)
}

// get room name and number of users
func (h *ChatHub) GetRoomInfo(roomID string) (string, int) {
	h.mu.RLock()
	room, exists := h.Rooms[roomID];
	h.mu.RUnlock()

	if !exists {
		return "", 0
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	return room.Name, len(room.Clients)
}

// chat join --manga-id command -
func (h *ChatHub) GetOrCreateMangaRoom(mangaID string, db *sql.DB) (string, string) {
	// look up manga title from DB
	var title string
	err := db.QueryRow(`SELECT title FROM manga WHERE id = ?`, mangaID).Scan(&title)

	if err != nil {
		title = mangaID
	}

	roomID := mangaID + "-room"
	roomName := title + " Chatroom"

	// get or create room here
	h.createRoom(roomID, roomName)
	return roomID, roomName
}