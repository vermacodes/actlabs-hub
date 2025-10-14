package main

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/handler"
	"actlabs-hub/internal/logger"
	"actlabs-hub/internal/middleware"
	"actlabs-hub/internal/mise"
	"actlabs-hub/internal/miseadapter"
	"actlabs-hub/internal/redis"
	"actlabs-hub/internal/repository"
	"actlabs-hub/internal/service"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Create root application context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	setupGracefulShutdown(cancel)

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		slog.WarnContext(ctx, "Error loading .env file")
	}

	// Load environment variables from .env.local file (overrides .env)
	err = godotenv.Load(".env.local")
	if err != nil {
		slog.WarnContext(ctx, "No .env.local file found or error loading it")
	}

	logger.SetupLogger(ctx)
	appConfig, err := config.NewConfig(ctx)
	if err != nil {
		logger.LogError(ctx, "error initializing config", "error", err)
		panic(err)
	}

	rdb, err := redis.NewRedisClient(ctx)
	if err != nil {
		logger.LogError(ctx, "error initializing redis", "error", err)
		panic(err)
	}

	auth, err := auth.NewAuth(ctx, appConfig)
	if err != nil {
		logger.LogError(ctx, "error initializing auth", "error", err)
		panic(err)
	}

	// mise
	miseServer := mise.Server{
		ContainerClient: miseadapter.NewMISEAdapter(ctx, http.DefaultClient, appConfig.MiseEndpoint),
		VerboseLogging:  appConfig.MiseVerboseLogging,
	}

	eventRepository, err := repository.NewEventRepository(ctx, auth)
	if err != nil {
		logger.LogError(ctx, "error initializing event repository", "error", err)
		panic(err)
	}

	serverRepository, err := repository.NewServerRepository(appConfig, auth, rdb)
	if err != nil {
		logger.LogError(ctx, "error initializing server repository", "error", err)
		panic(err)
	}
	labRepository, err := repository.NewLabRepository(auth, appConfig, rdb)
	if err != nil {
		logger.LogError(ctx, "error initializing lab repository", "error", err)
		panic(err)
	}
	assignmentRepository, err := repository.NewAssignmentRepository(auth, appConfig, rdb)
	if err != nil {
		logger.LogError(ctx, "error initializing assignment repository", "error", err)
		panic(err)
	}
	challengeRepository, err := repository.NewChallengeRepository(auth, appConfig, rdb)
	if err != nil {
		logger.LogError(ctx, "error initializing challenge repository", "error", err)
		panic(err)
	}
	authRepository, err := repository.NewAuthRepository(auth, appConfig, rdb)
	if err != nil {
		logger.LogError(ctx, "error initializing auth repository", "error", err)
		panic(err)
	}
	deploymentRepository, err := repository.NewDeploymentRepository(auth, rdb, appConfig)
	if err != nil {
		logger.LogError(ctx, "error initializing deployment repository", "error", err)
		panic(err)
	}

	eventService := service.NewEventService(eventRepository)
	serverService := service.NewServerService(serverRepository, appConfig, eventService)
	labService := service.NewLabService(labRepository)
	assignmentService := service.NewAssignmentService(assignmentRepository, labService)
	challengeService := service.NewChallengeService(challengeRepository, labService)
	authService := service.NewAuthService(authRepository)
	deploymentService := service.NewDeploymentService(deploymentRepository, serverService, eventService, appConfig)

	if appConfig.ActlabsHubMonitorAndAutoDestroyDeployments {
		logger.LogInfo(ctx, "auto deploy of auto-destroyed servers to destroy pending deployments is enabled")
		go deploymentService.MonitorAndAutoDestroyDeployments(ctx)
	}

	// Disable Gin's default logging since we use structured logging
	middleware.DisableGinDefaultLogging()

	router := gin.New()

	// Add recovery middleware (since we're using gin.New() instead of gin.Default())
	router.Use(gin.Recovery())

	router.SetTrustedProxies(nil)

	config := cors.DefaultConfig()
	config.AllowOrigins = strings.Split(appConfig.CorsAllowOrigins, ",")
	config.AllowMethods = strings.Split(appConfig.CorsAllowMethods, ",")
	config.AllowHeaders = strings.Split(appConfig.CorsAllowHeaders, ",")

	router.Use(cors.New(config))

	// Add context middleware to generate trace IDs and manage context
	router.Use(middleware.ContextMiddleware())

	// Add structured logging middleware
	router.Use(middleware.GinLoggerWithTraceID())

	authRouter := router.Group("/")
	authRouter.Use(middleware.Auth(miseServer, *appConfig))

	handler.NewHealthzHandler(router.Group("/"))
	handler.NewServerHandler(authRouter.Group("/"), serverService)
	handler.NewAssignmentHandler(authRouter.Group("/"), assignmentService, appConfig)
	handler.NewChallengeHandler(authRouter.Group("/"), challengeService, appConfig)
	handler.NewAuthHandler(authRouter.Group("/"), authService)

	armAuthRouter := router.Group("/")
	armAuthRouter.Use(middleware.ARMTokenAuth(appConfig))
	handler.NewDeploymentHandler(armAuthRouter.Group("/"), deploymentService)
	handler.NewServerHandlerArmToken(armAuthRouter.Group("/"), serverService)

	adminRouter := authRouter.Group("/")
	adminRouter.Use(middleware.AdminRequired(authService))
	handler.NewAdminAuthHandler(adminRouter, authService)
	handler.NewAdminServerHandler(adminRouter, serverService)

	mentorRouter := authRouter.Group("/")
	mentorRouter.Use(middleware.MentorRequired(authService))
	handler.NewAssignmentHandlerMentorRequired(mentorRouter, assignmentService)

	mentorRouter.Use(middleware.UpdateCredits())
	handler.NewLabHandlerMentorRequired(mentorRouter, labService)

	labRouter := authRouter.Group("/")
	labRouter.Use(middleware.UpdateCredits())
	handler.NewLabHandler(labRouter, labService, appConfig)

	contributorRouter := labRouter.Group("/")
	contributorRouter.Use(middleware.ContributorRequired(authService)).Use(middleware.UpdateCredits())
	handler.NewLabHandlerContributorRequired(contributorRouter, labService)

	handler.NewLabHandlerARMTokenWithProtectedLabSecret(armAuthRouter, labService, appConfig)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8883"
	}
	router.Run(":" + port)
}

// setupGracefulShutdown sets up signal handling for graceful shutdown
func setupGracefulShutdown(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.LogInfo(context.Background(), "received shutdown signal, initiating graceful shutdown", "signal", sig.String())
		cancel() // Cancel the root context to stop all background services

		// Give background services time to shut down gracefully
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}()
}
