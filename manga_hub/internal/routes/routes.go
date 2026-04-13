package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"go_mangahub/manga_hub/internal/auth"
	"go_mangahub/manga_hub/internal/controllers"
	"go_mangahub/manga_hub/internal/middleware"
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
		// requires token (protected route)
		authGroup.GET("/check", middleware.ValidateMiddleware(s.JWTSecret),func(c *gin.Context) {
			auth.CheckStatus(c, s.Database)
		})
		authGroup.POST("/change-password", middleware.ValidateMiddleware(s.JWTSecret), func(c *gin.Context) {
			auth.ChangePassword(c, s.Database)
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