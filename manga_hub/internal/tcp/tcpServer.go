package tcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)


type ProgressUpdate struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Timestamp int64  `json:"timestamp"`
}

type Client struct {
	Conn net.Conn
	UserID string
	Username string
	Device string
}


type ProgressSyncServer struct {
	Port 		string
	clients 	map[string]*Client
	mu			sync.RWMutex
	Broadcast 	chan ProgressUpdate
	Register	chan *Client
	Unregister	chan string
	quit		chan bool
}

type RegisterMessage struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Device   string `json:"device"`
}


func NewProgressSyncServer(port string) *ProgressSyncServer {
	return &ProgressSyncServer {
		Port: 		port,
		clients: 	make(map[string]*Client),
		Broadcast:	make(chan ProgressUpdate, 100),
		Register:	make(chan *Client),
		Unregister: make(chan string),
		quit:		make(chan bool),			
	}
}


func (s *ProgressSyncServer) Start() error {
	listener, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		return fmt.Errorf("Failed to start TCP server on port: %s, %w", s.Port, err)
	}

	log.Printf("[TCP] Sync server listening on tcp://localhost:%s\n", s.Port)

	// run the hub (handle register/unregister/broadcast)
	go s.runHub()


	// accept incoming connections
	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <- s.quit:
					return
				default:
					log.Println("[TCP] Accept error:", err)
					continue
				}
			}
			go s.handleClient(conn)
		}
	}()

	return nil
}

func (s *ProgressSyncServer) Stop() {
	close(s.quit)
}

func (s *ProgressSyncServer) runHub() {
	for {
		select {
		case client := <-s.Register:
			s.mu.Lock()
			s.clients[client.UserID] = client
			s.mu.Unlock()
			log.Printf("[TCP] Client connected: %s (%s)\n", client.Username, client.Device)

		case userID := <-s.Unregister:
			s.mu.Lock()
			if client, ok := s.clients[userID]; ok {
				client.Conn.Close()
				delete(s.clients, userID)
				log.Printf("[TCP] Client disconnected: %s\n", userID)
			}
			s.mu.Unlock()

		case update := <-s.Broadcast:
			s.mu.RLock()
			for _, client := range s.clients {
				// Broadcast to all OTHER users (not the sender)
				if client.UserID == update.UserID {
					continue
				}
				data, _ := json.Marshal(update)
				data = append(data, '\n')
				client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if _, err := client.Conn.Write(data); err != nil {
					log.Printf("[TCP] Failed to write to %s: %v\n", client.Username, err)
				}
			}
			s.mu.RUnlock()

		case <-s.quit:
			return
		}
	}
}

// manages a single tcp connection lifecycle
func(s *ProgressSyncServer) handleClient(conn net.Conn) {
	defer conn.Close()

	log.Println("[TCP] New connection from:", conn.RemoteAddr()) 


	// scanner reads line by line (\n)
	scanner := bufio.NewScanner(conn)
	
	log.Println("[TCP] Waiting for registration message...") 
	
	// read registeration message in the first line
	if !scanner.Scan() {
		log.Println("[TCP] Client disconnected before registration")
		return
	}
	
	var reg RegisterMessage
	if err := json.Unmarshal(scanner.Bytes(), &reg); err != nil {
		log.Println("[TCP] Failed to parse registration:", err)
		log.Println("[TCP] Raw message:", scanner.Text())
		return
	}

	log.Println("[TCP] Registered:", reg.Username, reg.Device)	

	// create client placeholder
	client := &Client{
		Conn: conn,
		UserID: reg.UserID,
		Username: reg.Username,
		Device: reg.Device,
	}
	// register client
	s.Register <- client
	defer func() {
		s.Unregister <- client.UserID // unregister client in case things go wrong
	}()

	// send welcome acknowledgement
	ack := map[string]interface{}{
		"type":    "connected",
		"message": fmt.Sprintf("Welcome %s! Connected to MangaHub sync server.", reg.Username),
		"server":  "tcp://localhost:" + s.Port,
	}
	ackData, _ := json.Marshal(ack)
	ackData = append(ackData, '\n')
	conn.Write(ackData)


	// listen for progress update from this client
	for scanner.Scan(){
		var update ProgressUpdate
		if err := json.Unmarshal(scanner.Bytes(), &update); err != nil {
			// client disconnected
			return
		}

		update.UserID = client.UserID
		update.Username = client.Username
		update.Timestamp = time.Now().Unix()

		log.Printf("[TCP] Progress update from %s: %s → ch.%d\n",
			client.Username, update.MangaID, update.Chapter)
		
		// Broadcast to all other connected clients about manga progress update
		s.Broadcast <- update
	}
}

// called by HTTP controller to notify TCP clients
func (s *ProgressSyncServer) BroadcastUpdate(update ProgressUpdate) {
	select {
	case s.Broadcast <- update:
	default:
		log.Println("[TCP] Broadcast channel full, dropping update")
	}
}


// returns number of connected clients
func(s *ProgressSyncServer) ConnectedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}


// 