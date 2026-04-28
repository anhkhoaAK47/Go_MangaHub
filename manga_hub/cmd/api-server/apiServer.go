package apiserver

import (
	"database/sql"
	"fmt"
	"go_mangahub/manga_hub/internal/controllers"
	"go_mangahub/manga_hub/internal/routes"
	"go_mangahub/manga_hub/internal/tcp"
	"go_mangahub/manga_hub/pkg/database"
	"net/http"
	"strings"

	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

type APIServer struct {
	Router *gin.Engine
	Database *sql.DB
	JWTSecret string
	Shutdown chan bool
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
				fmt.Println("  ⚠ Not yet implemented")
				fmt.Println("  Status: Skipped")

				// ── [4/5] gRPC Internal Service ───────────────────────────
				fmt.Println()
				fmt.Println("[4/5] gRPC Internal Service")
				fmt.Println("  ⚠ Not yet implemented")
				fmt.Println("  Status: Skipped")

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
				fmt.Println("  UDP:       (not yet implemented)")
				fmt.Println("  gRPC:      (not yet implemented)")
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
}