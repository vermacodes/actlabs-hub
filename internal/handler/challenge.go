package handler

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type challengeHandler struct {
	challengeService entity.ChallengeService
	appConfig        *config.Config
}

func NewChallengeHandler(r *gin.RouterGroup, service entity.ChallengeService, appConfig *config.Config) {
	handler := &challengeHandler{
		challengeService: service,
		appConfig:        appConfig,
	}

	r.GET("/challenge/labs", handler.GetAllLabsRedacted)
	r.GET("/challenge/labs/my", handler.GetMyChallengeLabsRedacted)
	r.GET("/challenge", handler.GetAllChallenges)
	r.GET("/challenge/my", handler.GetMyChallenges)
	r.GET("/challenge/lab/:labId", handler.GetChallengesByLabId)
	r.POST("/challenge", handler.UpsertChallenges)
	r.DELETE("/challenge/:challengeId", handler.DeleteChallenge)

	// requires super secret header.
	r.PUT("/challenge/:userId/:labId/:status", handler.UpdateChallenge)
}

func (ch *challengeHandler) GetAllLabsRedacted(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "get all labs redacted request")

	labs, err := ch.challengeService.GetAllLabsRedacted(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, labs)
}

func (ch *challengeHandler) GetMyChallengeLabsRedacted(c *gin.Context) {
	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userId, _ := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)

	logger.LogInfo(c.Request.Context(), "get my challenge labs redacted request",
		"user_id", userId,
	)

	labs, err := ch.challengeService.GetChallengesLabsRedactedByUserId(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, labs)
}

func (ch *challengeHandler) GetAllChallenges(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "get all challenges request")

	challenges, err := ch.challengeService.GetAllChallenges(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, challenges)
}

func (ch *challengeHandler) GetMyChallenges(c *gin.Context) {
	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userId, _ := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)

	logger.LogInfo(c.Request.Context(), "get my challenges request",
		"user_id", userId,
	)

	challenges, err := ch.challengeService.GetChallengesByUserId(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, challenges)
}

func (ch *challengeHandler) GetChallengesByLabId(c *gin.Context) {
	labId := c.Param("labId")

	logger.LogInfo(c.Request.Context(), "get challenges by lab id request",
		"lab_id", labId,
	)

	challenges, err := ch.challengeService.GetChallengesByLabId(c.Request.Context(), labId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, challenges)
}

func (ch *challengeHandler) UpsertChallenges(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "upsert challenges request")

	challenges := []entity.Challenge{}
	if err := c.BindJSON(&challenges); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ch.challengeService.UpsertChallenges(c.Request.Context(), challenges); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create/update one or more challenges"})
		return
	}

	c.Status(http.StatusOK)
}

func (ch *challengeHandler) UpdateChallenge(c *gin.Context) {
	userId := c.Param("userId")
	labId := c.Param("labId")
	status := c.Param("status")

	logger.LogInfo(c.Request.Context(), "update challenge request",
		"user_id", userId,
		"lab_id", labId,
		"status", status,
	)

	if err := ch.challengeService.UpdateChallenge(c.Request.Context(), userId, labId, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (ch *challengeHandler) DeleteChallenge(c *gin.Context) {
	challengeId := c.Param("challengeId")

	logger.LogInfo(c.Request.Context(), "delete challenge request",
		"challenge_id", challengeId,
	)

	challengeIds := []string{challengeId}

	if err := ch.challengeService.DeleteChallenges(c.Request.Context(), challengeIds); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, challengeId)
}
