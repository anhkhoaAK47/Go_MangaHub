package apiserver

import (
	"database/sql"
	"fmt"
	"go_mangahub/manga_hub/internal/controllers"
	"go_mangahub/manga_hub/internal/routes"
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

	err := godotenv.Load("")
	if err != nil {
		log.Println(err)
	}
		
	jwtSecret := os.Getenv("JWTSECRETKEY")

		db, err := database.InitDB("./mangahub.db")
	
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Seed initial data
	err = database.SeedSampleManga(db)
	if err != nil {
		log.Println(err)
	}

	server := &routes.APIServer{
		Router: gin.Default(),
		Database: db,
		JWTSecret: jwtSecret,
		Shutdown: make(chan bool),
	}

	// Provide DB handle to controllers
	controllers.SetDB(db)

	routes.SetupRoutes(server)

	
	log.Println("Server running on http://localhost:8080")
	
	go func() {
		server.Router.Run(":8080")
	}()

	<-server.Shutdown
	log.Println("Shutting down server...")
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
}