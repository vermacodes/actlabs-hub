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
	"context"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

func main() {
	logger.SetupLogger()
	appConfig, err := config.NewConfig()
	if err != nil {
		slog.Error("Error initializing config",
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	rdb, err := redis.NewRedisClient()
	if err != nil {
		slog.Error("Error initializing redis",
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	auth, err := auth.NewAuth(appConfig)
	if err != nil {
		slog.Error("Error initializing auth",
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	serverRepository, err := repository.NewServerRepository(appConfig, auth, rdb)
	if err != nil {
		slog.Error("Error initializing server repository",
			slog.String("error", err.Error()),
		)
		panic(err)
	}
	labRepository, err := repository.NewLabRepository(auth, appConfig, rdb)
	if err != nil {
		slog.Error("Error initializing lab repository",
			slog.String("error", err.Error()),
		)
		panic(err)
	}
	assignmentRepository, err := repository.NewAssignmentRepository(auth, appConfig, rdb)
	if err != nil {
		slog.Error("Error initializing assignment repository",
			slog.String("error", err.Error()),
		)
		panic(err)
	}
	challengeRepository, err := repository.NewChallengeRepository(auth, appConfig, rdb)
	if err != nil {
		slog.Error("Error initializing challenge repository",
			slog.String("error", err.Error()),
		)
		panic(err)
	}
	authRepository, err := repository.NewAuthRepository(auth, appConfig, rdb)
	if err != nil {
		slog.Error("Error initializing auth repository",
			slog.String("error", err.Error()),
		)
		panic(err)
	}
	deploymentRepository, err := repository.NewDeploymentRepository(auth, rdb)
	if err != nil {
		slog.Error("Error initializing deployment repository",
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	serverService := service.NewServerService(serverRepository, appConfig)
	labService := service.NewLabService(labRepository)
	assignmentService := service.NewAssignmentService(assignmentRepository, labService)
	challengeService := service.NewChallengeService(challengeRepository, labService)
	authService := service.NewAuthService(authRepository)
	autoDestroyService := service.NewAutoDestroyService(appConfig, serverRepository)
	deploymentService := service.NewDeploymentService(deploymentRepository, serverService, appConfig)

	go autoDestroyService.MonitorAndDestroyInactiveServers(context.Background())
	go deploymentService.MonitorAndDeployAutoDestroyedServersToDestroyPendingDeployments(context.Background())

	router := gin.Default()
	router.SetTrustedProxies(nil)

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:5173", "https://ashisverma.z13.web.core.windows.net", "https://actlabs.z13.web.core.windows.net", "https://actlabsbeta.z13.web.core.windows.net", "https://actlabs.azureedge.net", "https://*.azurewebsites.net"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Authorization", "Content-Type"}

	router.Use(cors.New(config))

	authRouter := router.Group("/")
	authRouter.Use(middleware.Auth())

	handler.NewHealthzHandler(authRouter.Group("/"))
	handler.NewServerHandler(authRouter.Group("/"), serverService)
	handler.NewAssignmentHandler(authRouter.Group("/"), assignmentService)
	handler.NewChallengeHandler(authRouter.Group("/"), challengeService)
	handler.NewAuthHandler(authRouter.Group("/"), authService)

	armAuthRouter := router.Group("/")
	armAuthRouter.Use(middleware.ARMTokenAuth())
	handler.NewDeploymentHandler(armAuthRouter.Group("/"), deploymentService)
	handler.NewServerHandlerArmToken(armAuthRouter.Group("/"), serverService)

	adminRouter := authRouter.Group("/")
	adminRouter.Use(middleware.AdminRequired(authService))
	handler.NewAdminAuthHandler(adminRouter, authService)

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8883"
	}
	router.Run(":" + port)
}
