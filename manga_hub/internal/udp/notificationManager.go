package udp

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// NotificationPreferences stores user notification preferences
type NotificationPreferences struct {
	UserID               string
	SubscribedGenres     []string
	SubscribedMangas     []string
	NotificationsEnabled bool
	EmailNotifications   bool
	UpdatedAt            time.Time
}

// NotificationManager handles UDP-based notifications
type NotificationManager struct {
	preferences map[string]*NotificationPreferences
	mu          sync.RWMutex
	udpConn     *net.UDPConn
	port        int
	clients     map[string]*net.UDPAddr // userID -> client UDP address
}

// Notification represents a chapter release notification
type Notification struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	MangaID      string    `json:"manga_id"`
	MangaTitle   string    `json:"manga_title"`
	ChapterNum   int       `json:"chapter_num"`
	ChapterTitle string    `json:"chapter_title"`
	Genre        string    `json:"genre"`
	Timestamp    time.Time `json:"timestamp"`
	Message      string    `json:"message"`
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(port int) *NotificationManager {
	return &NotificationManager{
		preferences: make(map[string]*NotificationPreferences),
		clients:     make(map[string]*net.UDPAddr),
		port:        port,
	}
}

type registrationPacket struct {
	Type   string `json:"type"`    // "register" | "unregister" | "ping"
	UserID string `json:"user_id"` // required for register/unregister
}

// Start initializes the UDP notification service
func (nm *NotificationManager) Start() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", nm.port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %d: %v", nm.port, err)
	}

	nm.udpConn = conn
	fmt.Printf("🚀 UDP Notification Service started on port %d\n", nm.port)

	// Start listening for notifications
	go nm.listen()

	return nil
}

// listen waits for and processes incoming notifications
func (nm *NotificationManager) listen() {
	buffer := make([]byte, 4096)
	for {
		if nm.udpConn == nil {
			break
		}

		n, remoteAddr, err := nm.udpConn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("❌ Error reading from UDP: %v\n", err)
			continue
		}

		// Client registration packets (per spec UC-009)
		var pkt registrationPacket
		if err := json.Unmarshal(buffer[:n], &pkt); err != nil {
			fmt.Printf("📨 Received UDP payload from %s: %s\n", remoteAddr, string(buffer[:n]))
			continue
		}

		switch pkt.Type {
		case "register":
			if pkt.UserID == "" {
				nm.writeAck(remoteAddr, map[string]any{"type": "error", "error": "user_id is required"})
				continue
			}
			nm.mu.Lock()
			// copy address value
			addrCopy := *remoteAddr
			nm.clients[pkt.UserID] = &addrCopy
			nm.mu.Unlock()
			nm.writeAck(remoteAddr, map[string]any{"type": "registered", "user_id": pkt.UserID})

		case "unregister":
			if pkt.UserID == "" {
				nm.writeAck(remoteAddr, map[string]any{"type": "error", "error": "user_id is required"})
				continue
			}
			nm.mu.Lock()
			delete(nm.clients, pkt.UserID)
			nm.mu.Unlock()
			nm.writeAck(remoteAddr, map[string]any{"type": "unregistered", "user_id": pkt.UserID})

		case "ping":
			nm.writeAck(remoteAddr, map[string]any{"type": "pong"})

		default:
			nm.writeAck(remoteAddr, map[string]any{"type": "error", "error": "unknown packet type"})
		}
	}
}

func (nm *NotificationManager) writeAck(addr *net.UDPAddr, payload any) {
	if nm.udpConn == nil || addr == nil {
		return
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = nm.udpConn.WriteToUDP(b, addr)
}

// Stop shuts down the UDP notification service
func (nm *NotificationManager) Stop() error {
	if nm.udpConn != nil {
		return nm.udpConn.Close()
	}
	return nil
}

// Subscribe adds a user subscription
func (nm *NotificationManager) Subscribe(userID string, mangaID string, genre string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	prefs, exists := nm.preferences[userID]
	if !exists {
		prefs = &NotificationPreferences{
			UserID:               userID,
			SubscribedGenres:     []string{},
			SubscribedMangas:     []string{},
			NotificationsEnabled: true,
			EmailNotifications:   true,
			UpdatedAt:            time.Now(),
		}
		nm.preferences[userID] = prefs
	}

	// Add manga subscription
	if mangaID != "" {
		for _, m := range prefs.SubscribedMangas {
			if m == mangaID {
				return fmt.Errorf("already subscribed to manga %s", mangaID)
			}
		}
		prefs.SubscribedMangas = append(prefs.SubscribedMangas, mangaID)
	}

	// Add genre subscription
	if genre != "" {
		for _, g := range prefs.SubscribedGenres {
			if g == genre {
				return fmt.Errorf("already subscribed to genre %s", genre)
			}
		}
		prefs.SubscribedGenres = append(prefs.SubscribedGenres, genre)
	}

	prefs.UpdatedAt = time.Now()
	return nil
}

// Unsubscribe removes a user subscription
func (nm *NotificationManager) Unsubscribe(userID string, mangaID string, genre string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	prefs, exists := nm.preferences[userID]
	if !exists {
		return fmt.Errorf("user %s has no preferences", userID)
	}

	// Remove manga subscription
	if mangaID != "" {
		for i, m := range prefs.SubscribedMangas {
			if m == mangaID {
				prefs.SubscribedMangas = append(prefs.SubscribedMangas[:i], prefs.SubscribedMangas[i+1:]...)
				break
			}
		}
	}

	// Remove genre subscription
	if genre != "" {
		for i, g := range prefs.SubscribedGenres {
			if g == genre {
				prefs.SubscribedGenres = append(prefs.SubscribedGenres[:i], prefs.SubscribedGenres[i+1:]...)
				break
			}
		}
	}

	prefs.UpdatedAt = time.Now()
	return nil
}

// GetPreferences retrieves user notification preferences
func (nm *NotificationManager) GetPreferences(userID string) (*NotificationPreferences, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	prefs, exists := nm.preferences[userID]
	if !exists {
		return nil, fmt.Errorf("no preferences found for user %s", userID)
	}

	return prefs, nil
}

// SendNotification broadcasts a notification to subscribed users
func (nm *NotificationManager) SendNotification(notification *Notification) error {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if nm.udpConn == nil {
		return fmt.Errorf("UDP connection not initialized")
	}

	// Check if user is subscribed
	prefs, exists := nm.preferences[notification.UserID]
	if !exists || !prefs.NotificationsEnabled {
		return fmt.Errorf("user %s not subscribed to notifications", notification.UserID)
	}

	clientAddr, hasClient := nm.clients[notification.UserID]
	if !hasClient {
		return fmt.Errorf("user %s has no registered UDP client", notification.UserID)
	}

	// Verify subscription match
	isSubscribed := false
	for _, m := range prefs.SubscribedMangas {
		if m == notification.MangaID {
			isSubscribed = true
			break
		}
	}
	if !isSubscribed {
		for _, g := range prefs.SubscribedGenres {
			if g == notification.Genre {
				isSubscribed = true
				break
			}
		}
	}

	if !isSubscribed {
		return fmt.Errorf("user %s not subscribed to manga or genre", notification.UserID)
	}

	// Send notification to the registered UDP client
	b, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	_, err = nm.udpConn.WriteToUDP(b, clientAddr)
	return err
}

// BroadcastNotification broadcasts a notification to all registered clients (UC-010).
// This is intentionally simple: it does not filter by prefs (use SendNotification for that).
func (nm *NotificationManager) BroadcastNotification(notification *Notification) error {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if nm.udpConn == nil {
		return fmt.Errorf("UDP connection not initialized")
	}
	b, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	for _, addr := range nm.clients {
		if addr == nil {
			continue
		}
		_, _ = nm.udpConn.WriteToUDP(b, addr)
	}
	return nil
}

// TestNotification sends a test notification
func (nm *NotificationManager) TestNotification(userID string) (*Notification, error) {
	nm.mu.RLock()
	_, exists := nm.preferences[userID]
	nm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("user %s has no preferences", userID)
	}

	testNotif := &Notification{
		ID:           "test-notification-001",
		UserID:       userID,
		MangaID:      "test-manga-123",
		MangaTitle:   "Test Manga Title",
		ChapterNum:   1,
		ChapterTitle: "Chapter 1: Getting Started",
		Genre:        "Adventure",
		Timestamp:    time.Now(),
		Message:      "This is a test notification to verify your notification system is working properly",
	}

	return testNotif, nil
}

// NotifyNewChapter proactively pushes a "new chapter released" event to all
// registered UDP clients whose preferences match the manga or genre.
// It simulates a server-side push (UC-010 style) but with preference filtering.
func (nm *NotificationManager) NotifyNewChapter(mangaID, mangaTitle string, chapterNum int, chapterTitle, genre string) (int, error) {
	if mangaID == "" {
		return 0, fmt.Errorf("mangaID is required")
	}
	if chapterNum <= 0 {
		return 0, fmt.Errorf("chapterNum must be > 0")
	}

	nm.mu.RLock()
	conn := nm.udpConn
	if conn == nil {
		nm.mu.RUnlock()
		return 0, fmt.Errorf("UDP connection not initialized")
	}

	type target struct {
		userID string
		addr   *net.UDPAddr
	}
	targets := make([]target, 0, len(nm.clients))
	for userID, prefs := range nm.preferences {
		if prefs == nil || !prefs.NotificationsEnabled {
			continue
		}
		addr := nm.clients[userID]
		if addr == nil {
			continue
		}
		if !matchesSubscription(prefs, mangaID, genre) {
			continue
		}
		// Copy address value to avoid aliasing a map value that can be replaced.
		addrCopy := *addr
		targets = append(targets, target{userID: userID, addr: &addrCopy})
	}
	nm.mu.RUnlock()

	if len(targets) == 0 {
		return 0, nil
	}

	// Bound potential blocking in case of OS buffer pressure.
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	defer func() { _ = conn.SetWriteDeadline(time.Time{}) }()

	sent := 0
	var lastErr error
	now := time.Now()

	for _, t := range targets {
		notif := &Notification{
			ID:           fmt.Sprintf("notif-%d-%s-%s-%d", now.UnixNano(), t.userID, mangaID, chapterNum),
			UserID:       t.userID,
			MangaID:      mangaID,
			MangaTitle:   mangaTitle,
			ChapterNum:   chapterNum,
			ChapterTitle: chapterTitle,
			Genre:        genre,
			Timestamp:    now,
			Message:      fmt.Sprintf("New chapter released: %s - Chapter %d", mangaTitle, chapterNum),
		}

		b, err := json.Marshal(notif)
		if err != nil {
			lastErr = err
			continue
		}
		if _, err := conn.WriteToUDP(b, t.addr); err != nil {
			lastErr = err
			continue
		}
		sent++
	}

	if sent == 0 && lastErr != nil {
		return 0, lastErr
	}
	return sent, lastErr
}

func matchesSubscription(prefs *NotificationPreferences, mangaID, genre string) bool {
	if prefs == nil {
		return false
	}
	for _, m := range prefs.SubscribedMangas {
		if m == mangaID {
			return true
		}
	}
	if genre == "" {
		return false
	}
	for _, g := range prefs.SubscribedGenres {
		if g == genre {
			return true
		}
	}
	return false
}
