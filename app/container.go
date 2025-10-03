package main

import (
	"context"
	"log/slog"
	"massager/app/config"
	"massager/internal/adapters"
	"massager/internal/handlers"
	"massager/internal/repositories"
	"massager/internal/services"
	websocket "massager/internal/websocet"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"

	otelgin "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

type Container struct {
	isShuttingDown bool

	GinEngine   *gin.Engine
	Config      *config.Config
	Redis       *redis.Client
	RateLimiter *RateLimiter

	Metrics        *Metrics
	Logger         *slog.Logger
	TracerProvider *tracesdk.TracerProvider
	Tracer         trace.Tracer

	Server *http.Server

	Repository *repositories.RepositoryAdapter

	AuthService  *services.AuthService
	EmailService *services.EmailService
	ChatService  *services.ChatService

	AuthHandler      *handlers.AuthHandler
	ChatHandler      *handlers.ChatHandler
	WebSocketHandler *handlers.WebsocetHandler

	WsHub *websocket.Hub
}

func NewContainer() (*Container, error) {
	container := &Container{}

	if err := container.initCore(); err != nil {
		return nil, err
	}

	if err := container.initProductionFeatures(); err != nil {
		return nil, err
	}

	return container, nil
}

func (c *Container) initCore() error {
	var cfg, err = config.LoadConfig()
	if err != nil {
		return err
	}
	c.Config = &cfg

	c.Logger = c.initLogger()
	c.Redis = c.initRedis()

	if err = c.initTracing(); err != nil {
		return err
	}

	c.Repository, err = repositories.NewRepositoryAdapter(cfg.Database, cfg.DatabaseConnections, c.Logger)
	if err != nil {
		c.Logger.Error("Repository initialize error", "error", err.Error())
		return err
	}

	var emailService = services.NewEmailService(cfg.Email, c.Logger)
	var chatService = services.NewChatService(c.Repository.Chat, c.Repository.Message, c.Repository.User, c.Logger)

	c.WsHub = websocket.NewHub(chatService, c.Logger)
	go c.WsHub.Run()

	chatService.SetWSHub(c.WsHub)

	c.RateLimiter = NewRateLimiter(cfg.RateLimit.MaxRequests, cfg.RateLimit.Window)

	c.AuthService = services.NewAuthService(c.Repository.User, emailService, &services.BcryptHasher{}, adapters.NewRedisTokenRepository(c.Redis), []byte(cfg.JWT.SecretKey), c.Logger)

	c.AuthHandler = handlers.NewAuthHandler(c.AuthService, c.Logger, c.Tracer)
	c.ChatHandler = handlers.NewChatHandler(chatService, c.Logger, c.Tracer)

	c.WebSocketHandler = handlers.NewWebSocketHandler(c.WsHub, c.AuthService, c.Logger, c.Tracer)

	c.Server = c.initServer()
	c.GinEngine = c.initGinEngine()
	c.Server.Handler = c.GinEngine

	return nil
}

func (c *Container) initProductionFeatures() error {
	c.initMetrics()

	c.initHealthRoutes(c.GinEngine)

	c.GinEngine.Use(services.SecurityMiddleware())
	c.GinEngine.Use(services.RequestIDMiddleware())

	return nil
}

func (c *Container) initMetrics() {
	c.Metrics = &Metrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "HTTP request duration",
			},
			[]string{"method", "endpoint"},
		),
	}
	prometheus.MustRegister(c.Metrics.RequestsTotal, c.Metrics.RequestDuration)
}

func (c *Container) initTracing() error {
	if !c.Config.Tracing.Enabled {
		c.Logger.Info("tracing disabled")
		return nil
	}

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(c.Config.Tracing.Endpoint)))
	if err != nil {
		return err
	}

	c.TracerProvider = tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(c.Config.Tracing.ServiceName),
			attribute.String("environment", c.Config.Environment.Current),
		)),
	)

	otel.SetTracerProvider(c.TracerProvider)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	c.Tracer = c.TracerProvider.Tracer("massager-app")

	c.Logger.Info("tracing initialized", "endpoint", c.Config.Tracing.Endpoint)
	return nil
}

func (c *Container) initHealthRoutes(eng *gin.Engine) {
	eng.GET("/health", func(ctx *gin.Context) {
		health := map[string]string{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		if err := c.Repository.HealthCheck(ctx); err != nil {
			health["database"] = "unhealthy"
			health["status"] = "degraded"
			ctx.JSON(503, health)
			return
		}

		if err := c.Redis.Ping().Err(); err != nil {
			health["redis"] = "unhealthy"
			health["status"] = "degraded"
			ctx.JSON(503, health)
			return
		}

		health["database"] = "healthy"
		health["redis"] = "healthy"
		ctx.JSON(200, health)
	})

	eng.GET("/ready", func(ctx *gin.Context) {
		if c.isShuttingDown {
			ctx.JSON(503, gin.H{"status": "shutting down"})
			return
		}
		ctx.JSON(200, gin.H{"status": "ready"})
	})

	eng.GET("/live", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "live"})
	})
}

func (c *Container) initGinEngine() *gin.Engine {
	var eng = gin.Default()

	eng.Use(otelgin.Middleware(c.Config.Tracing.ServiceName))

	eng.Static("/static", "./static")
	eng.LoadHTMLGlob("static/*.html")

	api := eng.Group("/api")

	api.Use(RateLimitMiddleware(c.RateLimiter))
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", c.AuthHandler.Register)
			authGroup.POST("/login", c.AuthHandler.Login)
			authGroup.POST("/logout", c.AuthHandler.Logout)
			authGroup.GET("/verify-email", c.AuthHandler.VerifyEmail)
			authGroup.GET("/verification-token", c.AuthHandler.GetVerificationToken)
			authGroup.GET("/verification-status", c.AuthHandler.GetVerificationStatus)
		}

		chatsGroup := api.Group("/chats")
		chatsGroup.Use(c.AuthHandler.AuthMiddleware())
		{
			chatsGroup.POST("", c.ChatHandler.CreateChat)
			chatsGroup.GET("", c.ChatHandler.GetUserChats)
			chatsGroup.GET("/:chatId/messages", c.ChatHandler.GetChatMessages)
			chatsGroup.DELETE("/:chatId", c.ChatHandler.DeleteChat)
		}

		api.GET("/ws", c.WebSocketHandler.HandleWebSocket)
	}

	eng.NoRoute(func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", nil)
	})

	return eng
}

func (c *Container) initLogger() *slog.Logger {
	var logger *slog.Logger
	if c.Config.Environment.Current == "development" {
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

func (c *Container) initRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     c.Config.Redis.Addr,
		Password: c.Config.Redis.Password,
		DB:       c.Config.Redis.DB,
	})
}

func (c *Container) initServer() *http.Server {
	return &http.Server{
		Addr:         ":" + c.Config.Server.Port,
		ReadTimeout:  time.Duration(c.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(c.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(c.Config.Server.IdleTimeout) * time.Second,
	}
}

func (c *Container) Close() error {
	c.isShuttingDown = true

	if c.Redis != nil {
		return c.Redis.Close()
	}

	if c.Repository != nil {
		c.Close()
	}

	if c.TracerProvider != nil {
		if err := c.TracerProvider.Shutdown(context.Background()); err != nil {
			c.Logger.Error("failed to shutdown tracer provider", "error", err)
		}
	}

	return nil
}
