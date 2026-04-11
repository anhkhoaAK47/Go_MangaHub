package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"go_mangahub/mangahub/internal/auth"
	"go_mangahub/mangahub/internal/controllers"
	"go_mangahub/mangahub/internal/middleware"
)

type APIServer struct {
	Router *gin.Engine
	Database *sql.DB
	JWTSecret string
}

func SetupRoutes(s *APIServer) {
	// Auth routes (non protected routes)
	authGroup := s.Router.Group("/auth")
	{
		authGroup.POST("/register", func(c *gin.Context) {
			auth.HandleRegister(c, s.Database)
		})
		authGroup.POST("/login", func(c *gin.Context) {
			auth.HandleLogin(c, s.Database, s.JWTSecret)
		})
	}

	// Manga routes (protected routes)
	manga := s.Router.Group("/manga")
	{
		manga.Use(middleware.ValidateMiddleware(s.JWTSecret))

		manga.GET("/", controllers.GetAllManga)
		manga.GET("/:id", controllers.GetMangaInfo)
	}

	// Users routes (protected routes)
	users := s.Router.Group("/users")
	{
		users.Use(middleware.ValidateMiddleware(s.JWTSecret))

		users.POST("/library", )
		users.GET("/library",)
		users.PUT("/progress", )
	}
}