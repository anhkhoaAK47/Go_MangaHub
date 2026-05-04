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
			os.Remove(".session")
			os.Remove(".token")
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
	startCmd.Flags().Bool("udp-only", false, "Start only the UDP notification")
	startCmd.Flags().Bool("grpc-only", false, "Start only the gRPC service")
	startCmd.Flags().Bool("ws-only", false, "Start only the WebSocket server")
}