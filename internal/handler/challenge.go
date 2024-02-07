package handler

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
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
	labs, err := ch.challengeService.GetAllLabsRedacted()
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
	userId, _ := auth.GetUserPrincipalFromToken(authToken)

	labs, err := ch.challengeService.GetChallengesLabsRedactedByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, labs)
}

func (ch *challengeHandler) GetAllChallenges(c *gin.Context) {
	challenges, err := ch.challengeService.GetAllChallenges()
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
	userId, _ := auth.GetUserPrincipalFromToken(authToken)

	challenges, err := ch.challengeService.GetChallengesByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, challenges)
}

func (ch *challengeHandler) GetChallengesByLabId(c *gin.Context) {

	labId := c.Param("labId")

	challenges, err := ch.challengeService.GetChallengesByLabId(labId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, challenges)
}

func (ch *challengeHandler) UpsertChallenges(c *gin.Context) {
	challenges := []entity.Challenge{}
	if err := c.BindJSON(&challenges); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ch.challengeService.UpsertChallenges(challenges); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create/update one or more challenges"})
		return
	}

	c.Status(http.StatusOK)
}

func (ch *challengeHandler) UpdateChallenge(c *gin.Context) {
	userId := c.Param("userId")
	labId := c.Param("labId")
	status := c.Param("status")

	// get super secret from header
	protectedLabSecret := c.Request.Header.Get("ProtectedLabSecret")
	if protectedLabSecret != ch.appConfig.ProtectedLabSecret {
		slog.Error("invalid protected lab secret",
			slog.String("userId", userId),
			slog.String("labId", labId),
			slog.String("status", status),
			slog.String("protectedLabSecret", protectedLabSecret),
		)

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Protected lab secret is invalid."})
		return
	}

	if err := ch.challengeService.UpdateChallenge(userId, labId, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (ch *challengeHandler) DeleteChallenge(c *gin.Context) {
	challengeId := c.Param("challengeId")

	challengeIds := []string{challengeId}

	if err := ch.challengeService.DeleteChallenges(challengeIds); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, challengeId)
}
