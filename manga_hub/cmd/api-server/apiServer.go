package apiserver

import (
	"database/sql"
	"go_mangahub/manga_hub/internal/controllers"
	"go_mangahub/manga_hub/internal/routes"
	"go_mangahub/manga_hub/pkg/database"

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

	db, err := database.InitDB("D:/DatabaseSQLite/mangahub.db")
	
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	err = database.SeedSampleManga(db)
	if err != nil {
		log.Println(err)
	}

	server := &routes.APIServer{
		Router: gin.Default(),
		Database: db,
		JWTSecret: jwtSecret,
	}

	// Seed initial data
	database.SeedSampleManga(db)

	// Provide DB handle to controllers
	controllers.SetDB(db)

	routes.SetupRoutes(server)

	
	log.Println("Server running on http://localhost:8080")
	server.Router.Run(":8080")
	}, 
}

func init() {
	// Add start command to the server command
	ServerCmd.AddCommand(startCmd) // 
}