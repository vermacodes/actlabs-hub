package middleware

import (
	"actlabs-hub/internal/auth"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Debug("Auth Middleware")

		// if request is for the health check endpoint, skip auth
		if c.Request.URL.Path == "/healthz" {
			c.Next()
			return
		}

		accessToken := c.GetHeader("Authorization")
		if accessToken == "" {
			slog.Error("no auth token provided")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		err := handleAccessToken(c, accessToken)
		if err != nil {
			return
		}
		c.Next()
	}
}

func handleAccessToken(c *gin.Context, accessToken string) error {
	// body, _ := io.ReadAll(c.Request.Body)
	// server := entity.Server{}
	// if err := json.Unmarshal(body, &server); err != nil {
	// 	slog.Error("error binding json", slog.String("error", err.Error()))
	// 	c.AbortWithStatus(http.StatusBadRequest)
	// 	return err
	// }

	// // Reassign the body so it can be read again in the handler
	// c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	splitToken := strings.Split(accessToken, "Bearer ")
	if len(splitToken) < 2 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return errors.New("found something in the Authorization header, but it's not a bearer token")
	}

	ok, err := auth.VerifyToken(accessToken)
	if err != nil || !ok {
		slog.Error("token verification failed", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return err
	}

	return nil
}
