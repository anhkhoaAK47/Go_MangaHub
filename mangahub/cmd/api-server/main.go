package main

import (
	"database/sql"
	"go_mangahub/mangahub/pkg/database"
	"go_mangahub/mangahub/internal/routes"
	"go_mangahub/mangahub/internal/controllers"

	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)
type APIServer struct {
	Router *gin.Engine
	Database *sql.DB
	JWTSecret string
}


func NewAPIServer(datapath string, jwtSecret string) *routes.APIServer {
	db, err := database.InitDB(datapath)
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
	
	return server
}

func main() {
	err := godotenv.Load("")
	if err != nil {
		log.Println(err)
	}


	jwtSecret := os.Getenv("JWTSECRETKEY")
	server := NewAPIServer("D:/DatabaseSQLite/mangahub.db", jwtSecret)

	log.Println("Server running on http://localhost:8080")
	server.Router.Run(":8080")
}