package middleware

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
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

		err := verifyAccessToken(c, accessToken)
		if err != nil {
			return
		}
		c.Next()
	}
}

func ARMTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Debug("ARMTokenAuth Middleware")

		accessToken := c.GetHeader("Authorization")
		if accessToken == "" {
			slog.Error("no auth token provided")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		err := verifyArmAccessToken(c, accessToken)
		if err != nil {
			return
		}

		c.Next()
	}
}

func AdminRequired(authService entity.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Debug("Middleware: AdminRequired")

		authToken := c.GetHeader("Authorization")

		callingUserPrincipal, err := auth.GetUserPrincipalFromToken(authToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Get the roles for the calling user
		profile, err := authService.GetProfile(callingUserPrincipal)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Check if the calling user has the admin role
		if !helper.Contains(profile.Roles, "admin") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user is not an admin"})
			return
		}

		c.Next()
	}
}

func MentorRequired(authService entity.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Debug("Middleware: MentorRequired")

		authToken := c.GetHeader("Authorization")

		callingUserPrincipal, err := auth.GetUserPrincipalFromToken(authToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Get the roles for the calling user
		profile, err := authService.GetProfile(callingUserPrincipal)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Check if the calling user has the mentor role
		if !helper.Contains(profile.Roles, "mentor") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user is not an mentor"})
			return
		}

		c.Next()
	}
}

func ContributorRequired(authService entity.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Debug("Middleware: contributorRequired")

		authToken := c.GetHeader("Authorization")

		callingUserPrincipal, err := auth.GetUserPrincipalFromToken(authToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Get the roles for the calling user
		profile, err := authService.GetProfile(callingUserPrincipal)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Check if the calling user has the mentor role
		if !helper.Contains(profile.Roles, "contributor") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user is not an contributor"})
			return
		}

		c.Next()
	}
}

func verifyAccessToken(c *gin.Context, accessToken string) error {
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

func verifyArmAccessToken(c *gin.Context, accessToken string) error {
	splitToken := strings.Split(accessToken, "Bearer ")
	if len(splitToken) < 2 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return errors.New("found something in the Authorization header, but it's not a bearer token")
	}

	ok, err := auth.VerifyArmToken(accessToken)
	if err != nil || !ok {
		slog.Error("token verification failed", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return err
	}

	return nil
}
