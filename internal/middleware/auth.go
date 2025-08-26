package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/mise"
)

func Auth(miseServer mise.Server) gin.HandlerFunc {
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

		err := verifyAccessToken(miseServer, c, accessToken)
		if err != nil {
			return
		}
		c.Next()
	}
}

// ARM Auth Token can be presented along with
// ProtectedLabSecret and x-ms-client-principal-name headers.
func ARMTokenAuth(appConfig *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Debug("ARMTokenAuth Middleware")

		// If request path includes /arm/server/register/ then skip verifyProtectedLabSecretAndUserPrincipalName
		if !strings.Contains(c.Request.URL.Path, "/arm/server/register") {
			err := verifyProtectedLabSecretAndUserPrincipalName(c, appConfig)
			if err != nil {
				return
			}
		}

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
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "user is not an contributor"},
			)
			return
		}

		c.Next()
	}
}

func verifyProtectedLabSecretAndUserPrincipalName(c *gin.Context, appConfig *config.Config) error {
	if c.GetHeader("ProtectedLabSecret") != appConfig.ProtectedLabSecret {
		slog.Error("ProtectedLabSecret header is missing or invalid")
		c.AbortWithStatus(http.StatusUnauthorized)
		return errors.New("ProtectedLabSecret header is missing or invalid")
	}

	if c.GetHeader("x-ms-client-principal-name") == "" {
		slog.Error("x-ms-client-principal-name header is missing")
		c.AbortWithStatus(http.StatusUnauthorized)
		return errors.New("x-ms-client-principal-name header is missing")
	}

	return nil
}

func verifyAccessToken(miseServer mise.Server, c *gin.Context, accessToken string) error {
	splitToken := strings.Split(accessToken, "Bearer ")
	if len(splitToken) < 2 {
		slog.Error("found something in the Authorization header, but it's not a bearer token")
		c.AbortWithStatus(http.StatusUnauthorized)
		return errors.New(
			"found something in the Authorization header, but it's not a bearer token",
		)
	}

	// MISE Implementation
	result, err := miseServer.DelegateAuthToContainer(accessToken, c.Request.URL.String(), c.Request.Method, c.ClientIP())
	if err != nil {
		var validationErr *mise.ErrTokenValidation
		if errors.As(err, &validationErr) {
			// can access validationErr.ErrorDescription, validationErr.WWWAuthenticate, validationErr.StatusCode
			slog.Error("token validation error", validationErr)
		} else {
			slog.Error("error while delegating auth to container", err)
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return err
	}

	userName, ok := result.SubjectClaims["preferred_username"]
	if !ok {
		slog.Error("preferred_username claim missing in subject claims")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing preferred username claim"})
		return errors.New("missing preferred username claim")
	}
	slog.Info("authenticated user", "user", userName)

	// Keeping the custom auth validation in place, just in case MISE isn't working as expected.
	ok, err = auth.VerifyToken(accessToken)
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
		return errors.New(
			"found something in the Authorization header, but it's not a bearer token",
		)
	}

	ok, err := auth.VerifyArmToken(accessToken)
	if err != nil || !ok {
		slog.Error("token verification failed", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return err
	}

	return nil
}
