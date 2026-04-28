# UDP Notifications & gRPC Service Operations

This document describes the newly implemented UDP Notifications and gRPC Service Operations features for MangaHub.

## Overview

### UDP Notifications
The UDP Notifications system allows users to subscribe to chapter release notifications for specific manga titles or genres. Notifications are delivered via UDP protocol and can be customized based on user preferences.

### gRPC Service Operations
The gRPC Service provides high-performance operations for querying manga data and managing reading progress with built-in caching and query statistics.

---

## UDP Notifications

### Architecture
- **Service**: `internal/udp/notificationManager.go`
- **Controllers**: `internal/controllers/notify.go`
- **Commands**: `cmd/mangahub/notify.go`

### Features
1. **Subscribe to Notifications**: Subscribe to specific manga or genres
2. **Unsubscribe**: Remove subscriptions
3. **Manage Preferences**: View and manage notification settings
4. **Test System**: Send test notifications to verify the system works

### CLI Commands

#### Subscribe to Notifications
```bash
# Subscribe to a specific manga
mangahub notify subscribe --manga-id <manga-id>

# Subscribe to a genre
mangahub notify subscribe --genre <genre-name>

# Subscribe to both
mangahub notify subscribe --manga-id <manga-id> --genre <genre-name>
```

**Example:**
```bash
mangahub notify subscribe --manga-id "berserk-001"
mangahub notify subscribe --genre "action"
```

**Expected Output:**
```
✅ Successfully subscribed to notifications
📚 Manga ID: berserk-001
```

#### Unsubscribe from Notifications
```bash
# Unsubscribe from a specific manga
mangahub notify unsubscribe --manga-id <manga-id>

# Unsubscribe from a genre
mangahub notify unsubscribe --genre <genre-name>
```

**Example:**
```bash
mangahub notify unsubscribe --manga-id "berserk-001"
```

**Expected Output:**
```
✅ Successfully unsubscribed from notifications
📚 Manga ID: berserk-001
```

#### View Notification Preferences
```bash
mangahub notify preferences
```

**Expected Output:**
```
📋 Your Notification Preferences
──────────────────────────────────────────────────
  • subscribed_genres: [action, adventure]
  • subscribed_mangas: [berserk-001, chainsaw-man-002]
  • notifications_enabled: true
  • email_notifications: true
  • updated_at: 2026-04-28T10:30:00Z
──────────────────────────────────────────────────
```

#### Test Notification System
```bash
mangahub notify test
```

**Expected Output:**
```
✅ Test notification sent successfully!
📬 Check your notification preferences for more details
```

### HTTP API Endpoints

#### Subscribe Endpoint
```http
POST /notify/subscribe
Authorization: Bearer <token>
X-Manga-ID: <manga-id>
X-Genre: <genre>
```

**Response:**
```json
{
  "message": "Successfully subscribed to notifications",
  "status": "subscribed",
  "manga_id": "berserk-001"
}
```

#### Unsubscribe Endpoint
```http
POST /notify/unsubscribe
Authorization: Bearer <token>
X-Manga-ID: <manga-id>
X-Genre: <genre>
```

#### Get Preferences Endpoint
```http
GET /notify/preferences
Authorization: Bearer <token>
```

**Response:**
```json
{
  "preferences": {
    "user_id": "user-123",
    "subscribed_genres": ["action", "adventure"],
    "subscribed_mangas": ["berserk-001", "chainsaw-man-002"],
    "notifications_enabled": true,
    "email_notifications": true,
    "updated_at": "2026-04-28T10:30:00Z"
  }
}
```

#### Test Notification Endpoint
```http
POST /notify/test
Authorization: Bearer <token>
```

**Response:**
```json
{
  "status": "success",
  "message": "Test notification sent",
  "notification": {
    "id": "test-notification-001",
    "manga_title": "Test Manga Title",
    "chapter_num": 1,
    "chapter_title": "Chapter 1: Getting Started",
    "genre": "Adventure",
    "message": "This is a test notification to verify your notification system is working properly",
    "timestamp": "2026-04-28T10:30:00Z"
  }
}
```

---

## gRPC Service Operations

### Architecture
- **Service**: `internal/grpc/grpcService.go`
- **Controllers**: `internal/controllers/grpc.go`
- **Commands**: `cmd/mangahub/grpc.go`

### Features
1. **Query Manga**: Get detailed information about manga by ID
2. **Search Manga**: Search for manga by title or author
3. **Update Progress**: Update reading progress with chapter information
4. **Batch Operations**: Retrieve multiple manga in a single request
5. **Query Statistics**: Track and retrieve query statistics

### CLI Commands

#### Get Manga by ID
```bash
mangahub grpc manga get --id <manga-id>
```

**Example:**
```bash
mangahub grpc manga get --id "berserk-001"
```

**Expected Output:**
```
📚 Manga Information (via gRPC)
──────────────────────────────────────────────────
  Title: Berserk
  Author: Kentaro Miura
  Artist: Kentaro Miura
  Status: Ongoing
  Total Chapters: 374
──────────────────────────────────────────────────
✅ Data retrieved from gRPC service
```

#### Search Manga
```bash
mangahub grpc manga search --query <search-term>
```

**Example:**
```bash
mangahub grpc manga search --query "dragon ball"
```

**Expected Output:**
```
🔍 Search Results for 'dragon ball' (via gRPC)
──────────────────────────────────────────────────
  1. Dragon Ball [Completed]
  2. Dragon Ball Z [Completed]
  3. Dragon Ball GT [Completed]
──────────────────────────────────────────────────
✅ Search completed via gRPC service
```

#### Update Progress via gRPC
```bash
mangahub grpc progress update --manga-id <id> --chapter <number>
```

**Optional Flags:**
- `--status <status>`: Set reading status (e.g., "reading", "completed", "paused")

**Example:**
```bash
mangahub grpc progress update --manga-id "berserk-001" --chapter 365
mangahub grpc progress update --manga-id "berserk-001" --chapter 365 --status "reading"
```

**Expected Output:**
```
✅ Progress Updated (via gRPC)
──────────────────────────────────────────────────
  Manga ID: berserk-001
  Chapter: 365
  Status: reading
──────────────────────────────────────────────────
✅ Update confirmed via gRPC service
```

### HTTP API Endpoints

#### Get Manga by ID
```http
GET /grpc/manga/<manga-id>
Authorization: Bearer <token>
```

**Response:**
```json
{
  "title": "Berserk",
  "author": "Kentaro Miura",
  "artist": "Kentaro Miura",
  "status": "Ongoing",
  "year": 1989,
  "totalChapters": 374,
  "totalVolumes": 41,
  "publisher": "Young Animal",
  "description": "Dark fantasy manga...",
  "genres": ["action", "dark", "fantasy", "supernatural"]
}
```

#### Search Manga
```http
GET /grpc/manga/search?q=<search-term>
Authorization: Bearer <token>
```

**Response:**
```json
{
  "count": 3,
  "mangas": [
    {
      "id": "dragon-ball-001",
      "title": "Dragon Ball",
      "author": "Akira Toriyama",
      "status": "Completed",
      "totalChapters": 519
    },
    {
      "id": "dragon-ball-z-001",
      "title": "Dragon Ball Z",
      "author": "Akira Toriyama",
      "status": "Completed",
      "totalChapters": 325
    }
  ]
}
```

#### Update Progress
```http
POST /grpc/progress/update
Authorization: Bearer <token>
Content-Type: application/json

{
  "manga_id": "berserk-001",
  "chapter": 365,
  "status": "reading"
}
```

**Response:**
```json
{
  "status": "success",
  "user_id": "user-123",
  "manga_id": "berserk-001",
  "chapter": 365,
  "reading_status": "reading",
  "rating": 0,
  "updated_at": "2026-04-28T10:30:00Z"
}
```

#### Get Progress
```http
GET /grpc/progress?manga_id=<manga-id>
Authorization: Bearer <token>
```

**Response:**
```json
{
  "user_id": "user-123",
  "manga_id": "berserk-001",
  "chapter": 365,
  "reading_status": "reading",
  "rating": 8.5,
  "started_reading": "2026-01-15T00:00:00Z",
  "updated_at": "2026-04-28T10:30:00Z"
}
```

#### List All Manga
```http
GET /grpc/manga/list
Authorization: Bearer <token>
```

**Response:**
```json
{
  "count": 150,
  "mangas": [
    {
      "id": "berserk-001",
      "title": "Berserk",
      "author": "Kentaro Miura",
      "status": "Ongoing",
      "totalChapters": 374
    },
    {
      "id": "chainsaw-man-002",
      "title": "Chainsaw Man",
      "author": "Tatsuki Fujimoto",
      "status": "Ongoing",
      "totalChapters": 125
    }
  ]
}
```

#### Batch Get Manga
```http
POST /grpc/manga/batch
Authorization: Bearer <token>
Content-Type: application/json

{
  "manga_ids": ["berserk-001", "chainsaw-man-002", "jjk-003"]
}
```

**Response:**
```json
{
  "count": 3,
  "mangas": [
    {
      "id": "berserk-001",
      "title": "Berserk",
      "author": "Kentaro Miura",
      "status": "Ongoing",
      "totalChapters": 374
    },
    {
      "id": "chainsaw-man-002",
      "title": "Chainsaw Man",
      "author": "Tatsuki Fujimoto",
      "status": "Ongoing",
      "totalChapters": 125
    },
    {
      "id": "jjk-003",
      "title": "Jujutsu Kaisen",
      "author": "Gege Akutami",
      "status": "Ongoing",
      "totalChapters": 247
    }
  ]
}
```

#### Get Service Statistics
```http
GET /grpc/stats
Authorization: Bearer <token>
```

**Response:**
```json
{
  "service": "grpc",
  "manga_count": 150,
  "progress_count": 45,
  "status": "operational"
}
```

#### Get Manga Query Statistics
```http
GET /grpc/manga/stats/<manga-id>
Authorization: Bearer <token>
```

**Response:**
```json
{
  "manga_id": "berserk-001",
  "times_queried": 42,
  "last_queried": "2026-04-28T10:30:00Z"
}
```

---

## Integration Notes

### Required Initialization
Before using these services, the following initialization is needed:

```go
// In your API server startup code:
import (
    "go_mangahub/manga_hub/internal/udp"
    "go_mangahub/manga_hub/internal/grpc"
    "go_mangahub/manga_hub/internal/controllers"
)

// Initialize UDP Notification Manager
notifyManager := udp.NewNotificationManager(5555) // UDP port
if err := notifyManager.Start(); err != nil {
    log.Fatal(err)
}
controllers.InitializeNotifyManager(notifyManager)

// Initialize gRPC Service
grpcService := grpc.NewGrpcService()
controllers.InitializeGrpcService(grpcService)
```

### File Structure Created
```
internal/
├── udp/
│   └── notificationManager.go    # UDP notification service
├── grpc/
│   └── grpcService.go            # gRPC service implementation
├── controllers/
│   ├── notify.go                 # Notification handlers
│   └── grpc.go                   # gRPC handlers
└── routes/
    └── routes.go                 # Updated with new endpoints

cmd/mangahub/
├── notify.go                     # Notify CLI commands
└── grpc.go                       # gRPC CLI commands
```

---

## Error Handling

### Common Error Responses

**Not Authenticated:**
```json
{
  "error": "Unauthorized"
}
```

**Service Not Available:**
```json
{
  "error": "Notification service not available"
}
```

**Resource Not Found:**
```json
{
  "error": "manga with ID xyz not found in cache"
}
```

**Invalid Request:**
```json
{
  "error": "Either manga-id or genre is required"
}
```

---

## Future Enhancements

1. **Real-time UDP Broadcasting**: Implement actual UDP packet broadcasting for notifications
2. **Email Integration**: Send email notifications for subscribed users
3. **WebSocket Support**: Real-time notification delivery via WebSocket
4. **gRPC Proto Definitions**: Define proper .proto files for gRPC services
5. **Caching Layer**: Implement Redis caching for frequently queried manga
6. **Notification History**: Store and retrieve notification history
7. **Advanced Search**: Fuzzy matching and advanced filtering for manga search

---

## Testing

### Manual Testing via CLI
```bash
# Test notification flow
mangahub auth login --username testuser
mangahub notify subscribe --manga-id "test-001" --genre "action"
mangahub notify preferences
mangahub notify test
mangahub notify unsubscribe --manga-id "test-001"

# Test gRPC flow
mangahub grpc manga get --id "test-001"
mangahub grpc manga search --query "dragon"
mangahub grpc progress update --manga-id "test-001" --chapter 10
```

### Manual Testing via HTTP API
```bash
# Get auth token
TOKEN=$(curl -X POST http://localhost:8080/auth/login -d '{"username":"testuser"}' | jq -r '.token')

# Test notifications
curl -H "Authorization: Bearer $TOKEN" -X POST http://localhost:8080/notify/subscribe -H "X-Manga-ID: test-001"
curl -H "Authorization: Bearer $TOKEN" -X GET http://localhost:8080/notify/preferences

# Test gRPC
curl -H "Authorization: Bearer $TOKEN" -X GET http://localhost:8080/grpc/manga/test-001
curl -H "Authorization: Bearer $TOKEN" -X GET http://localhost:8080/grpc/manga/search?q=dragon
```

---

## Performance Considerations

### gRPC Service Caching
- Manga data is cached in memory for fast retrieval
- Each query is tracked for statistics
- Batch operations reduce API calls

### UDP Notification Performance
- Non-blocking subscription management
- Thread-safe preference storage using sync.RWMutex
- Efficient genre/manga filtering

---

## Security Notes

- All endpoints require JWT authentication (except public manga endpoints)
- User preferences are isolated per user ID
- Query statistics are maintained per manga (not per user)
- Sensitive data is not logged or cached unnecessarily

---

## Support & Troubleshooting

### Issue: "Service not available" error
- Ensure the API server is running
- Verify the gRPC and UDP managers are initialized

### Issue: "Not logged in" error in CLI
- Run `mangahub auth login --username <username>` first
- Verify the `.token` file exists in the current directory

### Issue: Commands not recognized
- Run `mangahub --help` to see all available commands
- Use `mangahub notify --help` or `mangahub grpc --help` for subcommand help

---

For more information, see the main project README.md
