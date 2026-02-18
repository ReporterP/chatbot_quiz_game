package main

import (
	"log"
	"strconv"
	"time"

	"quiz-game-backend/internal/config"
	"quiz-game-backend/internal/database"
	"quiz-game-backend/internal/handlers"
	"quiz-game-backend/internal/middleware"
	"quiz-game-backend/internal/services"
	"quiz-game-backend/internal/telegram"
	"quiz-game-backend/internal/ws"

	_ "quiz-game-backend/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Quiz Game API
// @version         1.0
// @description     API for quiz game with host management and Telegram bot integration
// @host            localhost:8080
// @BasePath        /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer {token}"

func main() {
	cfg := config.Load()

	db := database.Connect(cfg)
	database.AutoMigrate(db)

	hub := ws.NewHub()

	authService := services.NewAuthService(db, cfg.JWTSecret)
	quizService := services.NewQuizService(db)
	scoringService := services.NewScoringService()
	sessionService := services.NewSessionService(db, scoringService)
	tgUserService := services.NewTelegramUserService(db)
	roomService := services.NewRoomService(db)

	aiService := services.NewAIGenerateService(cfg.QwenAPIKey, cfg.QwenAPIURL, cfg.QwenModel)

	authHandler := handlers.NewAuthHandler(authService)
	quizHandler := handlers.NewQuizHandler(quizService)
	questionHandler := handlers.NewQuestionHandler(quizService)
	sessionHandler := handlers.NewSessionHandler(sessionService, hub, db)
	participantHandler := handlers.NewParticipantHandler(sessionService, hub)
	settingsHandler := handlers.NewSettingsHandler(db)
	tgUserHandler := handlers.NewTelegramUserHandler(tgUserService)
	wsHandler := handlers.NewWSHandler(hub)
	aiHandler := handlers.NewAIGenerateHandler(quizService, aiService)
	roomHandler := handlers.NewRoomHandler(roomService, sessionService, hub)
	playHandler := handlers.NewPlayHandler(roomService, sessionService, hub)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Bot-API-Key"},
		AllowCredentials: true,
	}))

	r.Static("/uploads", "/uploads")
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/ws/session/:id", wsHandler.HandleWebSocket)
	r.GET("/ws/room/:code", playHandler.HandleRoomWebSocket)

	pollSec, _ := strconv.Atoi(cfg.PollInterval)
	if pollSec <= 0 {
		pollSec = 2
	}
	botManager := telegram.NewBotManager(
		db, sessionService, roomService, quizService, tgUserService, hub,
		cfg.WebhookBaseURL, cfg.BotAPIKey,
		time.Duration(pollSec)*time.Second,
		30*time.Second,
	)
	if cfg.WebhookBaseURL != "" {
		botManager.Start()
		defer botManager.Stop()
	} else {
		log.Println("WEBHOOK_BASE_URL not set, bot manager disabled")
	}
	r.POST("/webhook/bot/:secret", botManager.HandleWebhook)

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		settings := api.Group("/settings")
		settings.Use(middleware.JWTAuth(authService))
		{
			settings.GET("", settingsHandler.GetSettings)
			settings.PUT("", settingsHandler.UpdateSettings)
		}

		quizzes := api.Group("/quizzes")
		quizzes.Use(middleware.JWTAuth(authService))
		{
			quizzes.GET("/ai-status", aiHandler.CheckAI)
			quizzes.POST("/generate", aiHandler.Generate)
			quizzes.GET("", quizHandler.ListQuizzes)
			quizzes.POST("", quizHandler.CreateQuiz)
			quizzes.GET("/:id", quizHandler.GetQuiz)
			quizzes.PUT("/:id", quizHandler.UpdateQuiz)
			quizzes.DELETE("/:id", quizHandler.DeleteQuiz)
			quizzes.POST("/:id/questions", questionHandler.CreateQuestion)
			quizzes.POST("/:id/categories", questionHandler.CreateCategory)
			quizzes.PUT("/:id/reorder", questionHandler.ReorderQuiz)
			quizzes.GET("/:id/export", quizHandler.ExportQuiz)
			quizzes.POST("/:id/import", quizHandler.ImportQuiz)
		}

		questions := api.Group("/questions")
		questions.Use(middleware.JWTAuth(authService))
		{
			questions.PUT("/:id", questionHandler.UpdateQuestion)
			questions.DELETE("/:id", questionHandler.DeleteQuestion)
			questions.POST("/:id/images", questionHandler.AddQuestionImage)
		}

		categories := api.Group("/categories")
		categories.Use(middleware.JWTAuth(authService))
		{
			categories.PUT("/:id", questionHandler.UpdateCategory)
			categories.DELETE("/:id", questionHandler.DeleteCategory)
		}

		images := api.Group("/images")
		images.Use(middleware.JWTAuth(authService))
		{
			images.DELETE("/:id", questionHandler.DeleteQuestionImage)
		}

		upload := api.Group("/upload")
		upload.Use(middleware.JWTAuth(authService))
		{
			upload.POST("", questionHandler.UploadImage)
		}

		rooms := api.Group("/rooms")
		rooms.Use(middleware.JWTAuth(authService))
		{
			rooms.POST("", roomHandler.CreateRoom)
			rooms.GET("", roomHandler.ListActiveRooms)
			rooms.GET("/history", roomHandler.ListRoomHistory)
			rooms.GET("/:id", roomHandler.GetRoom)
			rooms.POST("/:id/close", roomHandler.CloseRoom)
			rooms.POST("/:id/start", roomHandler.StartQuizInRoom)
			rooms.POST("/:id/reveal", roomHandler.SessionReveal)
			rooms.POST("/:id/next", roomHandler.SessionNext)
			rooms.POST("/:id/finish", roomHandler.SessionFinish)
			rooms.GET("/:id/leaderboard", roomHandler.GetRoomLeaderboard)
		}

		play := api.Group("/play")
		{
			play.POST("/join", playHandler.Join)
			play.GET("/reconnect", playHandler.Reconnect)
			play.POST("/answer", playHandler.Answer)
			play.GET("/state", playHandler.GetState)
			play.PUT("/nickname", playHandler.UpdateNickname)
			play.POST("/leave", playHandler.Leave)
			play.GET("/my-result", playHandler.GetMyResult)
		}

		sessions := api.Group("/sessions")
		{
			sessions.GET("", middleware.JWTAuth(authService), sessionHandler.ListSessions)
			sessions.POST("", middleware.JWTAuth(authService), sessionHandler.CreateSession)
			sessions.GET("/:id", middleware.FlexAuth(authService, cfg.BotAPIKey), sessionHandler.GetSession)
			sessions.POST("/:id/reveal", middleware.JWTAuth(authService), sessionHandler.RevealAnswer)
			sessions.POST("/:id/next", middleware.JWTAuth(authService), sessionHandler.NextQuestion)
			sessions.POST("/:id/finish", middleware.JWTAuth(authService), sessionHandler.ForceFinish)
			sessions.GET("/:id/leaderboard", middleware.FlexAuth(authService, cfg.BotAPIKey), sessionHandler.GetLeaderboard)

			sessions.POST("/join", middleware.BotAuth(cfg.BotAPIKey), participantHandler.JoinSession)
			sessions.POST("/:id/answer", middleware.BotAuth(cfg.BotAPIKey), participantHandler.SubmitAnswer)
			sessions.GET("/:id/my-result", middleware.BotAuth(cfg.BotAPIKey), participantHandler.GetMyResult)
		}

		tgUsers := api.Group("/telegram-users")
		tgUsers.Use(middleware.BotAuth(cfg.BotAPIKey))
		{
			tgUsers.POST("", tgUserHandler.GetOrCreateUser)
			tgUsers.PUT("/:telegram_id/nickname", tgUserHandler.UpdateNickname)
			tgUsers.GET("/:telegram_id/history", tgUserHandler.GetHistory)
		}

		internal := api.Group("/internal")
		internal.Use(middleware.BotAuth(cfg.BotAPIKey))
		{
			internal.GET("/bot-tokens", settingsHandler.GetBotTokens)
		}
	}

	log.Printf("server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
