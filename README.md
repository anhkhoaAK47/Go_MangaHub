# Go_MangaHub - Manga & Comic Tracking System

**Language: Go

A manga tracking system implemented using 5 core network protocols: HTTP, TCP, UDP, gRPC, and WebSocket

## Table of Content

- [Features](#features)
- [System Architecture](#system-architecture)
- [Requirements](#requirements)
- [Installation & Setup](#installation--setup)
- [How to Run](#how-to-run)
- [CLI Commands](#cli-commands)
  - [Server Management](#server-management)
  - [Authentication](#authentication)
  - [Manga](#manga)
  - [Library](#library)
  - [Progress Tracking](#progress-tracking)
  - [Network Protocol Features](#network-protocol-features)
  - [Chat System](#chat-system)
- [Project Structure](#project-structure)
- [Tech Stack](#tech-stack)


## Features

- JWT-based user authentication
- Manga search and discovery with filter
- Library management
- Reading progress tracking and sync across devices
- Real-time chapter release notifications
- Community chat
- Internal manga query service



## System Architecture

| Protocol  | Port | Purpose                        |
|-----------|------|--------------------------------|
| HTTP REST | 8080 | Core API, auth, manga, library |
| TCP       | 9090 | Progress sync across devices   |
| UDP       | 9091 | Chapter release notifications  |
| gRPC      | 9092 | Internal manga service         |
| WebSocket | 9093 | Real-time community chat       |
 
---

## Installation & Setup

### 1. Clone the repository

```bash
git clone https://github.com/anhkhoaAK47/Go_MangaHub.git

cd Go_MangaHub/manga_hub
```


### 2. Install dependencies

```bash
go mod tidy
```

### 3. Set up env variables
Create a `.env` file in the `manga_hub/` directory:
 
```env
JWTSECRETKEY=your_secret_key_here
```


### 4. Build the CLI library

```bash
go build -o mangahub.exe main.go
```

### 5. Add to system PATH (Windows)

Add the project directory to your system environment variable `Path`:

```
E.g.: "C:\Go_MangaHub\manga_hub"
```
---

## How to run


### Server commands

```bash
# Start all server components (HTTP, TCP, UDP, gRPC, WebSocket)
mangahub server start
 
# Stop all running servers (requires login)
mangahub server stop

# Check status of all servers
mangahub server status
 
# Run detailed health check
mangahub server health
 
# Ping all server components
mangahub server ping

```
---

### Authentication

```bash
# Register a new account
mangahub auth register --username <username>
 
# Login to your account
mangahub auth login --username <username>
 
# Logout and clear session
mangahub auth logout
 
# Check current login status
mangahub auth status
 
# Change your password
mangahub auth change-password
```
---

### Manga

```bash
# Search for manga
mangahub manga search "attack on titan"
 
# Search with filters
mangahub manga search "romance" --genre romance --status completed --limit 5
 
# View detailed info about a manga
mangahub manga info one-piece
 
# List all manga in the database
mangahub manga list
mangahub manga list --genre shounen
mangahub manga list --page 2 --limit 20
```
---

### Library

```bash
# Add manga to your library
mangahub library add --manga-id one-piece --status reading
mangahub library add --manga-id death-note --status completed --rating 9
 
# View your library
mangahub library list
mangahub library list --status reading
mangahub library list --sort-by last-updated --order desc
 
# Update a library entry
mangahub library update --manga-id one-piece --status completed --rating 10
 
# Remove manga from library
mangahub library remove --manga-id one-piece
```

### Progress-Tracking
```bash
# Update your reading progress
mangahub progress update --manga-id one-piece --chapter 1095
 
# View progress history for a manga
mangahub progress history --manga-id one-piece
 
# Manually sync progress with server
mangahub progress sync
 
# Check sync status
mangahub progress sync-status
```
 
---


### Network Protocol Features
 
#### TCP — Progress Sync
 
```bash
mangahub sync connect      # Connect to TCP sync server
mangahub sync disconnect   # Disconnect
mangahub sync status       # View connection info and stats
mangahub sync monitor      # Watch real-time sync updates live
```
 
#### UDP — Notifications
 
```bash
mangahub notify subscribe    # Subscribe to chapter release notifications
mangahub notify unsubscribe  # Unsubscribe
mangahub notify preferences  # View notification settings
mangahub notify test         # Test the notification system
```
 
#### gRPC — Internal Service
 
```bash
mangahub grpc manga get --id one-piece
mangahub grpc manga search --query "naruto"
mangahub grpc progress update --manga-id one-piece --chapter 1095
```
 
---


### Chat System
 
```bash
# Join general chat
mangahub chat join
 
# Join a manga-specific discussion room
mangahub chat join --manga-id one-piece
 
# Send a message
mangahub chat send "Great chapter!"
mangahub chat send "Loved this arc!" --manga-id one-piece
 
# View chat history
mangahub chat history
mangahub chat history --manga-id one-piece --limit 50
```

**In-chat commands:**
 
| Command | Description |
|---|---|
| `/help` | Show available commands |
| `/users` | List online users |
| `/pm <user> <msg>` | Send a private message |
| `/manga <id>` | Switch to a manga chat room |
| `/history` | Show recent messages |
| `/quit` | Leave the chat |

## Project Structure

```
mangahub/
├── cmd/
│   ├── api-server/         # HTTP server command (start/stop)
│   ├── mangahub/           # CLI commands (auth, manga, library, etc.)
│   └── root.go             # Root cobra command
├── internal/
│   ├── auth/               # Authentication handlers
│   ├── controllers/        # Manga & library controllers
│   ├── middleware/          # JWT middleware
│   ├── routes/             # HTTP route definitions
│   ├── tcp/                # TCP sync server
│   ├── udp/                # UDP notification server
│   ├── websocket/          # WebSocket chat server
│   └── grpc/               # gRPC service
├── pkg/
│   ├── database/           # SQLite setup & seeding
│   ├── models/             # Shared data structs
│   └── utils/              # Helper functions (JWT, validation)
├── proto/                  # Protocol Buffer definitions
├── data/                   # JSON manga data
├── docs/                   # Documentation
├── .env                    # Environment variables (not committed)
├── mangahub.db             # SQLite database (auto-created)
├── .token                  # Session token (auto-created on login)
├── go.mod
├── go.sum
└── main.go
```

---

## Tech Stack

| Tool | Purpose |
|---|---|
| [Gin](https://github.com/gin-gonic/gin) | HTTP web framework |
| [Cobra](https://github.com/spf13/cobra) | CLI framework |
| [golang-jwt/jwt](https://github.com/golang-jwt/jwt) | JWT authentication |
| [gorilla/websocket](https://github.com/gorilla/websocket) | WebSocket support |
| [go-sqlite3](https://github.com/mattn/go-sqlite3) | SQLite database driver |
| [google.golang.org/grpc](https://google.golang.org/grpc) | gRPC framework |
| [google.golang.org/protobuf](https://google.golang.org/protobuf) | Protocol Buffers |
| [godotenv](https://github.com/joho/godotenv) | `.env` file loading |
 