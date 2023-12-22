package middleware

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

func ChallengeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// if calling method is GET or DELETE, then no need to update credits
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete {
			c.Next()
			return
		}

		// if calling method is not for /lab endpoint, then no need to update credits
		if !strings.Contains(c.Request.URL.Path, "/challenge") {
			c.Next()
			return
		}

		slog.Debug("Middleware: ChallengeMiddleware")

		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")

		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]

		callingUserPrincipal, err := auth.GetUserPrincipalFromToken(authToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		//get challenge from the payload
		challenges := []entity.Challenge{}
		if err := c.Bind(&challenges); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updatedChallenges := []entity.Challenge{}

		// update credits
		for _, challenge := range challenges {
			if challenge.ChallengeId == "" {
				challenge.CreatedBy = callingUserPrincipal
				challenge.CreatedOn = helper.GetTodaysDateTimeISOString()
			}
			updatedChallenges = append(updatedChallenges, challenge)
		}

		// update request payload
		marshaledChallenges, err := json.Marshal(updatedChallenges)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Create a new request based on existing request
		newRequest := c.Request.Clone(c.Request.Context())
		newRequest.Body = io.NopCloser(bytes.NewReader(marshaledChallenges))
		newRequest.ContentLength = int64(len(marshaledChallenges))

		// Replace request with newRequest
		c.Request = newRequest
		c.Next()
	}
}
