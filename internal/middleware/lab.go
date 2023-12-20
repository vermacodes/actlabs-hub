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

func UpdateCredits() gin.HandlerFunc {
	return func(c *gin.Context) {

		// if calling method is GET or DELETE, then no need to update credits
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete {
			c.Next()
			return
		}

		// if calling method is not for /lab endpoint, then no need to update credits
		if !strings.Contains(c.Request.URL.Path, "/lab") {
			c.Next()
			return
		}

		slog.Debug("Middleware: UpdateCredits")

		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")

		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]

		callingUserPrincipal, err := auth.GetUserPrincipalFromToken(authToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// get lab from the payload
		lab := entity.LabType{}
		if err := c.Bind(&lab); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// update credits
		if lab.Id == "" {
			lab.CreatedBy = callingUserPrincipal
			lab.CreatedOn = helper.GetTodaysDateString()
		} else {
			lab.UpdatedBy = callingUserPrincipal
			lab.UpdatedOn = helper.GetTodaysDateString()
		}

		// update request payload
		marshaledLab, err := json.Marshal(lab)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Create a new request based on the existing request
		newRequest := c.Request.Clone(c.Request.Context())
		newRequest.Body = io.NopCloser(bytes.NewReader(marshaledLab))
		newRequest.ContentLength = int64(len(marshaledLab))

		// Replace the current request with the new request
		c.Request = newRequest
		c.Next()
	}
}
