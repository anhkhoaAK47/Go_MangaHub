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
	"go_mangahub/manga_hub/pkg/database"
	"go_mangahub/manga_hub/pkg/models"
	"net"
	"net/http"
	"strings"

	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type APIServer struct {
	Router *gin.Engine
	Database *sql.DB
	JWTSecret string
	Shutdown chan bool
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
	
	err := godotenv.Load("")
	if err != nil {
		log.Println(err)
	}
		
	jwtSecret := os.Getenv("JWTSECRETKEY")
	fmt.Println("Starting MangaHub Server Components...")
	fmt.Println()

	if !tcpOnly {
	
	fmt.Println("[1/5] HTTP API Server")
	db, err := database.InitDB("./mangahub.db")
	
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	fmt.Println("  ✅ Database connection established")

	// Seed initial data
	err = database.SeedSampleManga(db)
	if err != nil {
		log.Println(err)
	}

	gin.SetMode(gin.ReleaseMode) // suppress gin debug logs
	server := &routes.APIServer{
		Router: gin.Default(),
		Database: db,
		JWTSecret: jwtSecret,
		Shutdown: make(chan bool),
	}

	fmt.Println("  ✅ JWT middleware loaded")

	// Provide DB handle to controllers
	controllers.SetDB(db)

	// ── UDP Notification Manager (in-process) ───────────────────────
	notifyManager := udp.NewNotificationManager(9091) // UDP port (per CLI manual/spec)
	if err := notifyManager.Start(); err != nil {
		fmt.Println("  ⚠ UDP notification service failed to start:", err)
	} else {
		controllers.InitializeNotifyManager(notifyManager)
		fmt.Println("  ✅ UDP notification service initialized (udp://localhost:9091)")
	}

	// ── Legacy in-memory gRPC service (kept for extra endpoints) ─────
	legacyGrpc := internalgrpc.NewGrpcService()
	controllers.InitializeGrpcService(legacyGrpc)
	if cached, err := preloadGrpcLegacyCache(db, legacyGrpc); err != nil {
		fmt.Println("  ⚠ Legacy gRPC cache preload failed:", err)
	} else {
		fmt.Printf("  ✅ Legacy gRPC cache ready (cached mangas: %d)\n", cached)
	}

	// ── gRPC Internal Service (real grpc-go server) ─────────────────
	if err := startGrpcServer(db, nil); err != nil {
		fmt.Println("  ⚠ gRPC service failed to start:", err)
	} else {
		if err := controllers.InitializeGrpcClient("localhost:9092"); err != nil {
			fmt.Println("  ⚠ gRPC client init failed:", err)
		} else {
			fmt.Println("  ✅ gRPC service initialized (grpc://localhost:9092)")
		}
	}

	routes.SetupRoutes(server)

	routesCount := len(server.Router.Routes())
	fmt.Printf("  ✅ %d routes registered\n", routesCount)

	if !httpOnly {
				// ── [2/5] TCP Sync Server ──────────────────────────────────
				fmt.Println()
				fmt.Println("[2/5] TCP Sync Server")

				tcpServer := tcp.NewProgressSyncServer("9090")
				if err := tcpServer.Start(); err != nil {
					fmt.Println("  ✗ Failed to start TCP server:", err)
				} else {
					fmt.Println("  ✓ Starting on tcp://localhost:9090")
					fmt.Println("  ✓ Connection pool initialized (max: 100)")
					fmt.Println("  ✓ Broadcast channels ready")
					fmt.Println("  Status: Listening for connections")
					controllers.SetTCPServer(tcpServer)
				}

				// ── [3/5] UDP Notification Server ─────────────────────────
				fmt.Println()
				fmt.Println("[3/5] UDP Notification Server")
				fmt.Println("  ✓ Running in-process")
				fmt.Println("  Status: Listening (udp://localhost:9091)")

				// ── [4/5] gRPC Internal Service ───────────────────────────
				fmt.Println()
				fmt.Println("[4/5] gRPC Internal Service")
				fmt.Println("  ✓ Running grpc-go server")
				fmt.Println("  Status: Operational")

				// ── [5/5] WebSocket Chat Server ───────────────────────────
				fmt.Println()
				fmt.Println("[5/5] WebSocket Chat Server")
				fmt.Println("  ⚠ Not yet implemented")
				fmt.Println("  Status: Skipped")
			}

			// Print summary
			fmt.Println()
			fmt.Println("─────────────────────────────────────────")
			if httpOnly {
				fmt.Println("  ✓ Starting on http://localhost:8080")
				fmt.Println("  Status: Running")
				fmt.Println()
				fmt.Println("HTTP server started successfully!")
			} else {
				fmt.Println("All available servers started successfully!")
			}
			fmt.Println()
			fmt.Println("Server URLs:")
			fmt.Println("  HTTP API:  http://localhost:8080")
			if !httpOnly {
				fmt.Println("  TCP Sync:  tcp://localhost:9090")
				fmt.Println("  UDP:       udp://localhost:9091")
				fmt.Println("  gRPC:      grpc://localhost:9092 (HTTP bridge: http://localhost:8080/grpc/*)")
				fmt.Println("  WebSocket: (not yet implemented)")
			}
			fmt.Println()
			fmt.Println("Stop: mangahub server stop")
			fmt.Println("─────────────────────────────────────────")

			// Start HTTP in background, block on shutdown
			go func() {
				if err := server.Router.Run(":8080"); err != nil {
					log.Println("HTTP server error:", err)
				}
			}()

			<-server.Shutdown
			log.Println("Shutting down server...")

		} else {
			// ── TCP only mode ──────────────────────────────────────────
			fmt.Println("[TCP] Starting TCP Sync Server only...")
			tcpServer := tcp.NewProgressSyncServer("9090")
			if err := tcpServer.Start(); err != nil {
				fmt.Println("  ✗ Failed to start TCP server:", err)
				return
			}
			fmt.Println("  ✓ Starting on tcp://localhost:9090")
			fmt.Println("  ✓ Connection pool initialized (max: 100)")
			fmt.Println("  ✓ Broadcast channels ready")
			fmt.Println("  Status: Listening for connections")
			fmt.Println()
			fmt.Println("Press Ctrl+C to stop.")

			// Block forever (TCP only mode)
			select {}
		} 
	},
}

var stopCmd = &cobra.Command{
	Use: "stop",
	Short: "Stop running all the MangaHub servers",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Stopping all servers...")

		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Run: mangahub auth login --username <username>")
			return
		}

		token := strings.TrimSpace(string(tokenData))

		// send POST request to server to shut down
		client := http.Client{}
		req, err := http.NewRequest("POST", "http://localhost:8080/server/stop", nil)
		if err != nil {
			fmt.Println("❌ Failed to create stop request:", err)
			return
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Failed to reach server. Is it running?")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Println("✅ Server shutdown successfully.")
		} else {
			fmt.Printf("❌ Server responded with status code: %d\n", resp.StatusCode)
		}
	},
}

func init() {
	// Add start command to the server command
	ServerCmd.AddCommand(startCmd) // 

	// Add stop command to the server command
	ServerCmd.AddCommand(stopCmd)

	// Register flags for selective startup
	startCmd.Flags().Bool("http-only", false, "Start only the HTTP API server")
	startCmd.Flags().Bool("tcp-only", false, "Start only the TCP sync server")
}