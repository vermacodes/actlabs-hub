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
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	logger.SetupLogger()
	appConfig, err := config.NewConfig()
	if err != nil {
		logger.LogError(context.Background(), "Error initializing config", "error", err.Error())
		panic(err)
	}

	rdb, err := redis.NewRedisClient()
	if err != nil {
		logger.LogError(context.Background(), "Error initializing redis", "error", err.Error())
		panic(err)
	}

	auth, err := auth.NewAuth(appConfig)
	if err != nil {
		logger.LogError(context.Background(), "Error initializing auth", "error", err.Error())
		panic(err)
	}

	// mise
	miseServer := mise.Server{
		ContainerClient: miseadapter.NewMISEAdapter(http.DefaultClient, appConfig.MiseEndpoint),
		VerboseLogging:  appConfig.MiseVerboseLogging,
	}

	eventRepository, err := repository.NewEventRepository(auth)
	if err != nil {
		logger.LogError(context.Background(), "Error initializing event repository", "error", err.Error())
		panic(err)
	}

	serverRepository, err := repository.NewServerRepository(appConfig, auth, rdb)
	if err != nil {
		logger.LogError(context.Background(), "Error initializing server repository", "error", err.Error())
		panic(err)
	}
	labRepository, err := repository.NewLabRepository(auth, appConfig, rdb)
	if err != nil {
		logger.LogError(context.Background(), "Error initializing lab repository", "error", err.Error())
		panic(err)
	}
	assignmentRepository, err := repository.NewAssignmentRepository(auth, appConfig, rdb)
	if err != nil {
		logger.LogError(context.Background(), "Error initializing assignment repository", "error", err.Error())
		panic(err)
	}
	challengeRepository, err := repository.NewChallengeRepository(auth, appConfig, rdb)
	if err != nil {
		logger.LogError(context.Background(), "Error initializing challenge repository", "error", err.Error())
		panic(err)
	}
	authRepository, err := repository.NewAuthRepository(auth, appConfig, rdb)
	if err != nil {
		logger.LogError(context.Background(), "Error initializing auth repository", "error", err.Error())
		panic(err)
	}
	deploymentRepository, err := repository.NewDeploymentRepository(auth, rdb, appConfig)
	if err != nil {
		logger.LogError(context.Background(), "Error initializing deployment repository", "error", err.Error())
		panic(err)
	}

	eventService := service.NewEventService(eventRepository)
	serverService := service.NewServerService(serverRepository, appConfig, eventService)
	labService := service.NewLabService(labRepository)
	assignmentService := service.NewAssignmentService(assignmentRepository, labService)
	challengeService := service.NewChallengeService(challengeRepository, labService)
	authService := service.NewAuthService(authRepository)
	deploymentService := service.NewDeploymentService(deploymentRepository, serverService, eventService, appConfig)
	// autoRemediateService := service.NewAutoRemediateService(appConfig, auth)

	if appConfig.ActlabsHubMonitorAndAutoDestroyDeployments {
		logger.LogInfo(context.Background(), "Auto deploy of auto-destroyed servers to destroy pending deployments is ENABLED")
		go deploymentService.MonitorAndAutoDestroyDeployments(context.Background())
	}
	// go autoRemediateService.MonitorAndRemediate(context.Background())

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
