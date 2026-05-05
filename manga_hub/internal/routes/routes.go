package routes

import (
	"database/sql"

	"go_mangahub/manga_hub/internal/auth"
	"go_mangahub/manga_hub/internal/controllers"
	"go_mangahub/manga_hub/internal/middleware"
	"go_mangahub/manga_hub/internal/websocket"

	"github.com/gin-gonic/gin"
)

type APIServer struct {
	Router    *gin.Engine
	Database  *sql.DB
	JWTSecret string
	Shutdown  chan bool
	ChatHub   *websocket.ChatHub
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

	// Manga routes (public routes)
	manga := s.Router.Group("/manga")
	{
		manga.GET("/", controllers.GetAllManga)
		manga.GET("/:id", controllers.GetMangaInfo)
	}

	// Users routes (protected routes)
	users := s.Router.Group("/users")
	{
		users.Use(middleware.ValidateMiddleware(s.JWTSecret))

		users.POST("/library", controllers.AddToLibrary)
		users.GET("/library", controllers.ListLibrary)
		users.DELETE("/library/:id", controllers.RemoveFromLibrary)
		users.PUT("/library/:id", controllers.UpdateLibraryEntry)
		users.PUT("/progress", controllers.UpdateProgress)
		users.GET("/progress/history", controllers.GetProgressHistory)
		users.POST("/progress/sync", controllers.SyncProgress)
		users.GET("/progress/sync-status", controllers.SyncProgressStatus)
	}

	// server routes (protected routes)
	servers := s.Router.Group("/server")
	{
		servers.Use(middleware.ValidateMiddleware(s.JWTSecret))

		servers.POST("/stop", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Server shutting down...",
			})

			go func() {
				s.Shutdown <- true
			}()
		})
	}

	// Notification routes (protected routes)
	notify := s.Router.Group("/notify")
	{
		notify.Use(middleware.ValidateMiddleware(s.JWTSecret))

		notify.POST("/subscribe", controllers.Subscribe)
		notify.POST("/unsubscribe", controllers.Unsubscribe)
		notify.GET("/preferences", controllers.GetPreferences)
		notify.POST("/test", controllers.TestNotification)
		notify.POST("/new-chapter", controllers.NotifyNewChapter)
	}

	// gRPC routes (protected routes)
	grpcRoutes := s.Router.Group("/grpc")
	{
		grpcRoutes.Use(middleware.ValidateMiddleware(s.JWTSecret))

		// Manga routes (specific routes before parameterized routes)
		grpcRoutes.GET("/manga/search", controllers.SearchManga)
		grpcRoutes.GET("/manga/list", controllers.ListAllManga)
		grpcRoutes.GET("/manga/stats/:id", controllers.GetMangaStats)
		grpcRoutes.GET("/manga/:id", controllers.GetMangaByID)
		grpcRoutes.POST("/manga/batch", controllers.BatchGetManga)

		// Progress routes
		grpcRoutes.POST("/progress/update", controllers.UpdateProgressGrpc)
		grpcRoutes.GET("/progress", controllers.GetProgress)

		// Service stats
		grpcRoutes.GET("/stats", controllers.GetServiceStats)
	}

	chatRoutes := s.Router.Group("/chat")
	{
		chatRoutes.Use(middleware.ValidateMiddleware(s.JWTSecret))
		chatRoutes.GET("/history", controllers.GetChatHistory)
	}	

}

func SetupWSRoutes(s *APIServer) *gin.Engine {
	wsRouter := gin.New()
	wsRouter.Use(gin.Recovery())

	wsRouter.GET("/chat/:room",
		middleware.ValidateMiddleware(s.JWTSecret),
		websocket.HandleChatRoom(s.ChatHub, s.Database),
	)

	return wsRouter
}