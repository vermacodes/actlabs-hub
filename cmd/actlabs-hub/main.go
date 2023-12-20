package main

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/handler"
	"actlabs-hub/internal/logger"
	"actlabs-hub/internal/middleware"
	"actlabs-hub/internal/redis"
	"actlabs-hub/internal/repository"
	"actlabs-hub/internal/service"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

func main() {
	logger.SetupLogger()
	appConfig, err := config.NewConfig()
	if err != nil {
		slog.Error("Error initializing config", err)
		panic(err)
	}

	rdb, err := redis.NewRedisClient()
	if err != nil {
		slog.Error("Error initializing redis", err)
		panic(err)
	}

	auth, err := auth.NewAuth(appConfig)
	if err != nil {
		slog.Error("Error initializing auth", err)
		panic(err)
	}

	serverRepository, err := repository.NewServerRepository(appConfig, auth, rdb)
	if err != nil {
		slog.Error("Error initializing server repository", err)
		panic(err)
	}
	labRepository, err := repository.NewLabRepository(auth, appConfig, rdb)
	if err != nil {
		slog.Error("Error initializing lab repository", err)
		panic(err)
	}
	assignmentRepository, err := repository.NewAssignmentRepository(auth, appConfig, rdb)
	if err != nil {
		slog.Error("Error initializing assignment repository", err)
		panic(err)
	}
	challengeRepository, err := repository.NewChallengeRepository(auth, appConfig, rdb)
	if err != nil {
		slog.Error("Error initializing challenge repository", err)
		panic(err)
	}
	authRepository, err := repository.NewAuthRepository(auth, appConfig, rdb)
	if err != nil {
		slog.Error("Error initializing auth repository", err)
		panic(err)
	}

	serverService := service.NewServerService(serverRepository, appConfig)
	labService := service.NewLabService(labRepository)
	assignmentService := service.NewAssignmentService(assignmentRepository, labService)
	challengeService := service.NewChallengeService(challengeRepository, labService)
	authService := service.NewAuthService(authRepository)

	router := gin.Default()
	router.SetTrustedProxies(nil)

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:5173", "https://ashisverma.z13.web.core.windows.net", "https://actlabs.z13.web.core.windows.net", "https://actlabsbeta.z13.web.core.windows.net", "https://actlabs.azureedge.net", "https://*.azurewebsites.net"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Authorization", "Content-Type"}

	router.Use(cors.New(config))
	router.Use(middleware.Auth())

	handler.NewHealthzHandler(router.Group("/"))
	handler.NewServerHandler(router.Group("/"), serverService)
	handler.NewLabHandler(router.Group("/"), labService)
	handler.NewAssignmentHandler(router.Group("/"), assignmentService)
	handler.NewChallengeHandler(router.Group("/"), challengeService)
	handler.NewAuthHandler(router.Group("/"), authService)

	adminRouter := router.Group("/")
	adminRouter.Use(middleware.AdminRequired(authService))
	handler.NewAdminAuthHandler(adminRouter, authService)

	mentorRouter := router.Group("/")
	mentorRouter.Use(middleware.MentorRequired(authService))
	handler.NewLabHandlerMentorRequired(mentorRouter, labService)
	handler.NewAssignmentHandlerMentorRequired(mentorRouter, assignmentService)

	contributorRouter := router.Group("/")
	contributorRouter.Use(middleware.ContributorRequired(authService))
	handler.NewLabHandlerContributorRequired(contributorRouter, labService)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8883"
	}
	router.Run(":" + port)
}
