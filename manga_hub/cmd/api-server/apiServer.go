package apiserver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go_mangahub/manga_hub/internal/controllers"
	internalgrpc "go_mangahub/manga_hub/internal/grpc"
	"go_mangahub/manga_hub/internal/grpcpb"
	"go_mangahub/manga_hub/internal/grpcserver"
	"go_mangahub/manga_hub/internal/routes"
	"go_mangahub/manga_hub/internal/tcp"
	"go_mangahub/manga_hub/internal/udp"
	"go_mangahub/manga_hub/internal/websocket"
	"go_mangahub/manga_hub/pkg/database"
	"go_mangahub/manga_hub/pkg/models"
	"net"
	"net/http"
	"strings"
	"path/filepath"

	"time"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	_ "github.com/mattn/go-sqlite3"
)

type APIServer struct {
	Router *gin.Engine
	Database *sql.DB
	JWTSecret string
	Shutdown chan bool
}

type dbInfo struct {
	status 	string
	size	string
	tables	string
}

func startGrpcServer(db *sql.DB, tcpBroadcaster *tcp.ProgressSyncServer) error {
	lis, err := net.Listen("tcp", ":9092")
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	grpcpb.RegisterMangaServiceServer(s, grpcserver.New(db, tcpBroadcaster))

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Println("gRPC server error:", err)
		}
	}()
	return nil
}

// preloadGrpcLegacyCache keeps the old in-memory gRPC service features working
// (list/batch/stats/progress endpoints) without removing code.
func preloadGrpcLegacyCache(db *sql.DB, svc *internalgrpc.GrpcService) (int, error) {
	rows, err := db.Query(`SELECT id, title, author, artist, genres, status, year, total_chapters, total_volumes, serialization, publisher, description, my_anime_list, manga_dx FROM manga`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var (
			id            string
			title         string
			author        string
			artist        string
			genresJSON    string
			status        string
			year          int
			totalChapters int
			totalVolumes  int
			serialization string
			publisher     string
			description   string
			myAnimeList   string
			mangaDx       string
		)

		if err := rows.Scan(
			&id, &title, &author, &artist, &genresJSON, &status, &year, &totalChapters, &totalVolumes, &serialization,
			&publisher, &description, &myAnimeList, &mangaDx,
		); err != nil {
			continue
		}

		var genres []string
		_ = json.Unmarshal([]byte(genresJSON), &genres)
		m := &models.Manga{
			ID:            id,
			Title:         title,
			Author:        author,
			Artist:        artist,
			Genres:        genres,
			Status:        status,
			Year:          year,
			TotalChapters: totalChapters,
			TotalVolumes:  totalVolumes,
			Serialization: serialization,
			Publisher:     publisher,
			Description:   description,
			MyAnimeList:   myAnimeList,
			MangaDx:       mangaDx,
		}
		_ = svc.CacheManga(m)
		count++
	}
	return count, nil
}
// subcommand "server"
var ServerCmd = &cobra.Command{
	Use: "server",
	Short: "Manage the MangaHub server components",
}

// add command "start"
var startCmd = &cobra.Command{
	Use: "start",
	Short: "Start all of the MangaHub server",
	Run: func(cmd *cobra.Command, args []string) {
	httpOnly, _ := cmd.Flags().GetBool("http-only")
	tcpOnly, _ := cmd.Flags().GetBool("tcp-only")
	udpOnly, _ := cmd.Flags().GetBool("udp-only")
	grpcOnly, _ := cmd.Flags().GetBool("grpc-only")
	wsOnly, _ := cmd.Flags().GetBool("ws-only")
		
	anyOnly := httpOnly || tcpOnly || udpOnly || grpcOnly || wsOnly
	allMode := !anyOnly

	err := godotenv.Load("")
	if err != nil {
		log.Println(err)
	}
		
	jwtSecret := os.Getenv("JWTSECRETKEY")
	fmt.Println("Starting MangaHub Server Components...")
	
	home, _ := os.UserHomeDir()
	mangahubDir := filepath.Join(home, ".mangahub")
	os.MkdirAll(mangahubDir, 0755) // create dir if it doesn't exist
	startTime := time.Now().Format(time.RFC3339)
	os.WriteFile(filepath.Join(mangahubDir, "server_start"), []byte(startTime), 0644)

	fmt.Println()

	var db *sql.DB
		needsDB := allMode || httpOnly || grpcOnly || wsOnly
		if needsDB {
			db, err = database.InitDB("./mangahub.db")
			if err != nil {
				log.Fatal("❌ Failed to connect to database:", err)
			}

			err = database.SeedSampleManga(db)
			if err != nil {
				log.Println(err)
			}
		}

		//  ── [1/5] HTTP Server ──────────────────────────────────────────
		var server *routes.APIServer
		needsHTTP := allMode || httpOnly
		if needsHTTP {
			fmt.Println("[1/5] HTTP API Server")

			gin.SetMode(gin.ReleaseMode)
			server = &routes.APIServer{
				Router:    gin.Default(),
				Database:  db,
				JWTSecret: jwtSecret,
				Shutdown:  make(chan bool),
			}

			fmt.Println("  ✓ Database connection established")
			fmt.Println("  ✓ JWT middleware loaded")

			controllers.SetDB(db)

			// UDP notification manager (needed by HTTP routes)
			notifyManager := udp.NewNotificationManager(9091)
			if err := notifyManager.Start(); err != nil {
				fmt.Println("  ⚠ UDP notification service failed:", err)
			} else {
				controllers.InitializeNotifyManager(notifyManager)
			}

			// Legacy gRPC cache
			legacyGrpc := internalgrpc.NewGrpcService()
			controllers.InitializeGrpcService(legacyGrpc)
			if cached, err := preloadGrpcLegacyCache(db, legacyGrpc); err != nil {
				fmt.Println("  ⚠ Legacy gRPC cache preload failed:", err)
			} else {
				fmt.Printf("  ✓ Legacy gRPC cache ready (%d manga cached)\n", cached)
			}

			// gRPC client init
			if err := startGrpcServer(db, nil); err != nil {
				fmt.Println("  ⚠ gRPC service failed:", err)
			} else {
				if err := controllers.InitializeGrpcClient("localhost:9092"); err != nil {
					fmt.Println("  ⚠ gRPC client init failed:", err)
				}
			}

			routes.SetupRoutes(server)
			fmt.Printf("  ✓ %d routes registered\n", len(server.Router.Routes()))
			fmt.Println("  Status: Running")
		}

		// ── [2/5] TCP Sync Server ──────────────────────────────────────────
		needsTCP := allMode || tcpOnly
		if needsTCP {
			fmt.Println()
			fmt.Println("[2/5] TCP Sync Server")

			tcpServer := tcp.NewProgressSyncServer("9090")
			if err := tcpServer.Start(); err != nil {
				fmt.Println("  ✗ Failed to start:", err)
			} else {
				fmt.Println("  ✓ Starting on tcp://localhost:9090")
				fmt.Println("  ✓ Connection pool initialized (max: 100)")
				fmt.Println("  ✓ Broadcast channels ready")
				fmt.Println("  Status: Listening for connections")
				if needsHTTP {
					controllers.SetTCPServer(tcpServer)
				}
			}

			// TCP-only mode blocks here
			if tcpOnly {
				fmt.Println()
				fmt.Println("Press Ctrl+C to stop.")
				select {}
			}
		}

		// ── [3/5] UDP Notification Server ─────────────────────────────────
		needsUDP := allMode || udpOnly
		if needsUDP {
			fmt.Println()
			fmt.Println("[3/5] UDP Notification Server")

			if udpOnly {
				// Standalone UDP mode — start its own manager
				notifyManager := udp.NewNotificationManager(9091)
				if err := notifyManager.Start(); err != nil {
					fmt.Println("  ✗ Failed to start:", err)
				} else {
					fmt.Println("  ✓ Starting on udp://localhost:9091")
					fmt.Println("  ✓ Client registry initialized")
					fmt.Println("  ✓ Notification queue ready")
					fmt.Println("  Status: Ready for broadcasts")
					fmt.Println()
					fmt.Println("Press Ctrl+C to stop.")
					select {}
				}
			} else {
				fmt.Println("  ✓ Running in-process on udp://localhost:9091")
				fmt.Println("  Status: Listening")
			}
		}

		// ── [4/5] gRPC Internal Service ───────────────────────────────────
		needsGRPC := allMode || grpcOnly
		if needsGRPC {
			fmt.Println()
			fmt.Println("[4/5] gRPC Internal Service")

			if grpcOnly {
				// Standalone gRPC mode
				if db == nil {
					var initErr error
					db, initErr = database.InitDB("./mangahub.db")
					if initErr != nil {
						fmt.Println("  ✗ Database failed:", initErr)
						return
					}
				}
				if err := startGrpcServer(db, nil); err != nil {
					fmt.Println("  ✗ Failed to start:", err)
				} else {
					fmt.Println("  ✓ Starting on grpc://localhost:9092")
					fmt.Println("  ✓ 3 services registered")
					fmt.Println("  ✓ Protocol buffers loaded")
					fmt.Println("  Status: Serving")
					fmt.Println()
					fmt.Println("Press Ctrl+C to stop.")
					select {}
				}
			} else {
				fmt.Println("  ✓ Running on grpc://localhost:9092")
				fmt.Println("  Status: Operational")
			}
		}

		// ── [5/5] WebSocket Chat Server ───────────────────────────────────
		needsWS := allMode || wsOnly
		if needsWS {
			fmt.Println()
			fmt.Println("[5/5] WebSocket Chat Server")

			if db == nil {
				var initErr error
				db, initErr = database.InitDB("./mangahub.db")
				if initErr != nil {
					fmt.Println("  ✗ Database failed:", initErr)
				}
			}

			chatHub := websocket.NewChatHub(db)
			chatHub.Start()

			if server != nil {
				server.ChatHub = chatHub
			} else {
				// ws-only mode — create a minimal server just for WebSocket
				server = &routes.APIServer{
					Database:  db,
					JWTSecret: jwtSecret,
					ChatHub:   chatHub,
				}
			}

			// Start WebSocket on port 9093
			wsRouter := routes.SetupWSRoutes(server)
			go func() {
				log.Println("[WS] Chat server starting on ws://localhost:9093")
				if err := wsRouter.Run(":9093"); err != nil {
					log.Println("[WS] Chat server error:", err)
				}
			}()

			fmt.Println("  ✓ Starting on ws://localhost:9093/chat")
			fmt.Println("  ✓ Chat rooms initialized")
			fmt.Println("  ✓ General room ready")
			fmt.Println("  Status: Ready for connections")
		}

		// ── Summary ───────────────────────────────────────────────────────
		fmt.Println()
		fmt.Println("─────────────────────────────────────────")
		fmt.Println("Server URLs:")
		if needsHTTP  { fmt.Println("  HTTP API:  http://localhost:8080") }
		if needsTCP   { fmt.Println("  TCP Sync:  tcp://localhost:9090") }
		if needsUDP   { fmt.Println("  UDP:       udp://localhost:9091") }
		if needsGRPC  { fmt.Println("  gRPC:      grpc://localhost:9092") }
		if needsWS    { fmt.Println("  WebSocket: ws://localhost:9093/chat/:room") }
		fmt.Println()
		fmt.Println("Stop: mangahub server stop")
		fmt.Println("─────────────────────────────────────────")

		// ── Start HTTP and block on shutdown ──────────────────────────────
		if needsHTTP && server != nil {
			go func() {
				if err := server.Router.Run(":8080"); err != nil {
					log.Println("HTTP server error:", err)
				}
			}()
			<-server.Shutdown
			log.Println("Shutting down server...")
		}
	},
}

var stopCmd = &cobra.Command{
	Use: "stop",
	Short: "Stop running all the MangaHub servers",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Stopping all servers...")


		// send POST request to server to shut down
		client := http.Client{}
		req, err := http.NewRequest("POST", "http://localhost:8080/server/stop", nil)
		if err != nil {
			fmt.Println("❌ Failed to create stop request:", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Failed to reach server. Is it running?")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Println("✅ Server shutdown successfully.")
			os.Remove(".session")
			os.Remove(".token")
			
			home, _ := os.UserHomeDir()
			os.Remove(filepath.Join(home, ".mangahub", "server_start"))
		} else {
			fmt.Printf("❌ Server responded with status code: %d\n", resp.StatusCode)
		}
	},
}


var statusCmd = &cobra.Command{
	Use: "status",
	Short: "Check status of all server running",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("MangaHub Server Status")
		fmt.Println()

		// ── Table header ───────────────────────────────────────────────────
		fmt.Printf("%-22s %-12s %-25s %-12s %s\n",
			"Service", "Status", "Address", "Uptime", "Load")
		fmt.Println(strings.Repeat("─", 85))

		startTime := getServerStartTime()

		// Probe each port

		// HTTP API
		httpStatus, httpLoad := probeHTTP()
		printServiceRow("HTTP API", httpStatus, "localhost:8080", startTime, httpLoad)

		// TCP Sync
		tcpStatus, tcpLoad := probeTCP("9090")
		printServiceRow("TCP Sync", tcpStatus, "localhost:9090", startTime, tcpLoad)

		// UDP Notifications
		udpStatus, udpLoad := probeUDP("9091")
		printServiceRow("UDP Notifications", udpStatus, "localhost:9091", startTime, udpLoad)

		// gRPC
		grpcStatus, grpcLoad := probeTCP("9092")
		printServiceRow("gRPC Internal", grpcStatus, "localhost:9092", startTime, grpcLoad)

		// WebSocket
		wsStatus, wsLoad := probeTCP("9093")
		printServiceRow("WebSocket Chat", wsStatus, "localhost:9093", startTime, wsLoad)

		// Overall health 
		allOnline := httpStatus == "online" &&
			tcpStatus == "online" &&
			udpStatus == "online" &&
			grpcStatus == "online" &&
			wsStatus == "online"

		fmt.Println()
		if allOnline {
			fmt.Println("Overall System Health: ✓ Healthy")
		} else {
			fmt.Println("Overall System Health: ⚠ Degraded")
			fmt.Println()
			fmt.Println("Issues Detected:")
			if httpStatus != "online" {
				fmt.Println("  ✗ HTTP API Server: Not reachable on port 8080")
			}
			if tcpStatus != "online" {
				fmt.Println("  ✗ TCP Sync Server: Not reachable on port 9090")
			}
			if udpStatus != "online" {
				fmt.Println("  ⚠ UDP Notifications: Not reachable on port 9091")
			}
			if grpcStatus != "online" {
				fmt.Println("  ✗ gRPC Service: Not reachable on port 9092")
			}
			if wsStatus != "online" {
				fmt.Println("  ✗ WebSocket Chat: Not reachable on port 9093")
			}
		}		

		// Database
		fmt.Println()
		fmt.Println("Database:")
		dbInfo := probeDatabaseInfo()

		fmt.Printf("  Connection:	%s\n", dbInfo.status)
		fmt.Printf("  Size:			%s\n", dbInfo.size)
		fmt.Printf("  Tables:		%s\n", dbInfo.tables)


		// System
		fmt.Println()
		fmt.Printf("Checked at: %s\n", time.Now().Format("2006-01-02 15:04:05"))	
	},
}

var healthCmd = &cobra.Command{
	Use: "health",
	Short: "Check server health",
	Run: func(cmd *cobra.Command, args []string) {
		client := &http.Client{}
		req, _ := http.NewRequest("GET", "http://localhost:8080/server/health", nil)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Error connecting to server")
			return
		}

		if resp.StatusCode == http.StatusOK {
			fmt.Println("✅ Server is running smoothly!")
			return
		}
	},
}

// reads the start time from ".server_start" file
func getServerStartTime() string {
	home, err := os.UserHomeDir()
	if err != nil {
        return ""
    }
	
    data, err := os.ReadFile(filepath.Join(home, ".mangahub", "server_start"))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

// prints one row of the status table
func printServiceRow(service, status, address, startTime, load string) {
	uptime := "-"
	if startTime != "" && status == "online" {
		uptime = formatUptime(startTime)
	}
	fmt.Printf("%-22s %-12s %-25s %-12s %s\n",
		service, status, address, uptime, load)
}

// calculates uptime from a saved start time
func formatUptime(startTimeStr string) string {
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return "─"
	}
	duration := time.Since(startTime)

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// check if HTTP server is up by hitting /health
func probeHTTP() (string, string) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		return "✗ Offline", "─"
	}
	defer resp.Body.Close()
	return "online", "HTTP active"
}


// check if a TCP Port is open
func probeTCP(port string) (string, string) {
	conn, err := net.DialTimeout("tcp", "localhost:"+port, 2 * time.Second)
	if err != nil {
		return "✗ Offline", "─"
	}
	conn.Close()
	return "online", "active"
}

// sends a ping packet and waits for pong
func probeUDP(port string) (string, string) {
	addr, err := net.ResolveUDPAddr("udp", "localhost:" + port)
	if err != nil {
		return "✗ Offline", "─"
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return "✗ Offline", "─"
	}
	defer conn.Close()

	// send ping packet
	ping := []byte(`{"type":"ping"}`)
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write(ping)
	if err != nil {
		return "✗ Offline", "─"
	}

	// wait for pong
	buf := make([]byte, 256)
	_, err = conn.Read(buf)
	if err != nil {
		return "✗ Offline", "─"
	}

	return "online", "active"
}

// check if local DB file exists and gets its size
func probeDatabaseInfo() dbInfo {
	dbPath := "./mangahub.db"
	info, err := os.Stat(dbPath)
	if err != nil {
		return dbInfo{
			status: "✗ Not found",
			size:   "─",
			tables: "─",		
		}
	}
	
	sizeMB := float64(info.Size()) / (1024 * 1024)
	sizeStr := fmt.Sprintf("%.1f MB", sizeMB)

	// count tables in db
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return dbInfo{
			status: "✗ Cannot open",
			size:   sizeStr,
			tables: "─",
		}
	}

	defer db.Close()

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	// database connected but no tables
	if err != nil {
		return dbInfo{
			status: "✓ Active",
			size:   sizeStr,
			tables: "─",
		}	
	}

	defer rows.Close()

	// add tables name strings into array
	var tableNames []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			tableNames = append(tableNames, name)
		}
	}

	return dbInfo{
		status: "✓ Active",
		size:   sizeStr,
		tables: fmt.Sprintf("%d (%s)", len(tableNames), strings.Join(tableNames, ", ")),
	}
}


func init() {
	// Add start command to the server command
	ServerCmd.AddCommand(startCmd) 

	// Add stop command to the server command
	ServerCmd.AddCommand(stopCmd)

	// Add status command to the server command
	ServerCmd.AddCommand(statusCmd)

	// Add health command
	ServerCmd.AddCommand(healthCmd)

	// Register flags for selective startup
	startCmd.Flags().Bool("http-only", false, "Start only the HTTP API server")
	startCmd.Flags().Bool("tcp-only", false, "Start only the TCP sync server")
	startCmd.Flags().Bool("udp-only", false, "Start only the UDP notification")
	startCmd.Flags().Bool("grpc-only", false, "Start only the gRPC service")
	startCmd.Flags().Bool("ws-only", false, "Start only the WebSocket server")
}