package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"go_mangahub/manga_hub/internal/auth"
	"go_mangahub/manga_hub/internal/controllers"
	"go_mangahub/manga_hub/internal/middleware"
)

type APIServer struct {
	Router    *gin.Engine
	Database  *sql.DB
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

		// PROTECTED ROUTES: require token
		// FIXED: Added middleware to logout
		authGroup.POST("/logout", middleware.ValidateMiddleware(s.JWTSecret), func(c *gin.Context) {
			auth.HandleLogout(c)
		})

		// requires token (protected route)
		authGroup.GET("/check", middleware.ValidateMiddleware(s.JWTSecret), func(c *gin.Context) {
			auth.CheckStatus(c, s.Database)
		})

		authGroup.PUT("/change-password", middleware.ValidateMiddleware(s.JWTSecret), func(c *gin.Context) {
			auth.ChangePassword(c, s.Database)
		})

	}

	// Manga routes (partially protected routes)
	manga := s.Router.Group("/manga")
	{
		manga.GET("/", middleware.ValidateMiddleware(s.JWTSecret), controllers.GetAllManga)
		manga.GET("/:id", middleware.OptionalValidateMiddleware(s.JWTSecret), controllers.GetMangaInfo)
	}

	// Users routes (protected routes)
	users := s.Router.Group("/users")
	{
		users.Use(middleware.ValidateMiddleware(s.JWTSecret))

		users.POST("/library", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Not implemented yet"})
		})
		users.GET("/library", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Not implemented yet"})
		})
		users.PUT("/progress", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Not implemented yet"})
		})
	}
}