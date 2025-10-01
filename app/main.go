package main

// PROPRIETARY AND CONFIDENTIAL
// This code contains trade secrets and confidential material of Finimen Sniper / FSC.
// Any unauthorized use, disclosure, or duplication is strictly prohibited.
// © 2025 Finimen Sniper / FSC. All rights reserved.

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"massager/app/config"
	"massager/internal/adapters"
	"massager/internal/handlers"
	"massager/internal/repositories"
	"massager/internal/services"
	websocket "massager/internal/websocet"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "massager/docs"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

func initRedis(cfg *config.RedisConfig) *redis.Client {
	var r = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return r
}

func initLogger(cfg *config.EnvironmentConfig) *slog.Logger {
	var logger *slog.Logger
	if cfg.Current == "development" {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	slog.SetDefault(logger)

	return logger
}

// @title Massager API
// @version 1.0
// @description Real-time messaging web service
// @termsOfService https://github.com/Finimen/SafeMassager

// @contact.name FinimenSniper
// @contact.url https://github.com/Finimen/SafeMassager
// @contact.email finimensniper@gmail.com

// @license.name MIT
// @license.url https://github.com/Finimen/SafeMassager

// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token in the format: Bearer <token>

// @schemes http

// main initializes and starts the messenger server
// @Summary Information about API
// @Description Returns basic information about the API
// @Tags info
// @Produce json
// @Success 200 {object} map[string]string
// @Router / [get]
func main() {
	var cfg, err = config.LoadConfig()
	if err != nil {
		log.Fatal(err)
		return
	}

	var logger = initLogger(&cfg.Environment)
	var redisClient = initRedis(&cfg.Redis)

	var repo *repositories.RepositoryAdapter
	repo, err = repositories.NewRepositoryAdapter(cfg.Database.Path)
	if err != nil {
		logger.Error("Repository initialize error",
			"error", err.Error())
		return
	}

	fmt.Println("IS NILL????", repo == nil)
	fmt.Println(repo.User == nil, repo.Chat == nil)

	var emailService = services.NewEmailService(cfg.Email, logger)
	var chatService = services.NewChatService(repo.Chat, repo.Message, repo.User, logger)

	wsHub := websocket.NewHub(chatService, logger)
	go wsHub.Run()

	chatService.SetWSHub(wsHub)

	var rateLimiter = NewRateLimiter(cfg.RateLimit.MaxRequests, cfg.RateLimit.Window)

	var authService = services.NewAuthService(repo.User, emailService, &services.BcryptHasher{}, adapters.NewRedisTokenRepository(redisClient), []byte(cfg.JWT.SecretKey), logger)

	ctx := context.Background()
	chatService.CreateChat(ctx, "General Chat", []string{"user1", "user2"})
	chatService.CreateChat(ctx, "Random Chat", []string{"user1", "user3"})

	var authHandler = handlers.NewAuthHandler(authService, logger)
	var chatHandler = handlers.NewChatHandler(chatService, logger)

	wsHandler := handlers.NewWebSocketHandler(wsHub, authService, logger)

	var eng = gin.Default()

	eng.Static("/static", "./static")
	eng.LoadHTMLGlob("static/*.html")

	api := eng.Group("/api")
	api.Use(RateLimitMiddleware(rateLimiter))
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/logout", authHandler.Logout)
			authGroup.GET("/verify-email", authHandler.VerifyEmail)
			authGroup.GET("/verification-token", authHandler.GetVerificationToken)
			authGroup.GET("/verification-status", authHandler.GetVerificationStatus)
		}

		chatsGroup := api.Group("/chats")
		chatsGroup.Use(authHandler.AuthMiddleware())
		{
			chatsGroup.POST("", chatHandler.CreateChat)
			chatsGroup.GET("", chatHandler.GetUserChats)
			chatsGroup.GET("/:chatId/messages", chatHandler.GetChatMessages)
			chatsGroup.DELETE("/:chatId", chatHandler.DeleteChat)
		}

		api.GET("/ws", wsHandler.HandleWebSocket)
	}

	eng.NoRoute(func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", nil)
	})

	var serv = &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: eng,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server is starting on http://localhost%s\n", serv.Addr)
		if err := serv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: http://localhost%s\n", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := serv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
