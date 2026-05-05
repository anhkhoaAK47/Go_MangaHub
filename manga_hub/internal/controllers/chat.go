package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ChatEntry struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	RoomID    string `json:"room_id"`
}

func GetChatHistory(c *gin.Context) {
    roomID := c.DefaultQuery("room_id", "general")
    limitStr := c.DefaultQuery("limit", "50")

    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit <= 0 {
        limit = 50
    }
    // cap at 100
    if limit > 100 {
        limit = 100
    }

    rows, err := db.Query(`
        SELECT user_id, username, message, timestamp, room_id
        FROM chat_messages
        WHERE room_id = ?
        ORDER BY timestamp ASC
        LIMIT ?
    `, roomID, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()


    messages := make([]ChatEntry, 0)
    for rows.Next() {
        var entry ChatEntry
        if err := rows.Scan(
            &entry.UserID,
            &entry.Username,
            &entry.Message,
            &entry.Timestamp,
            &entry.RoomID,
        ); err != nil {
            continue
        }
        messages = append(messages, entry)
    }

    c.JSON(http.StatusOK, messages)
}