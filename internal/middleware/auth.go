package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"
	"actlabs-hub/internal/mise"
)

func Auth(miseServer mise.Server, config config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := GetContextFromGin(c)
		logger.LogDebug(ctx, "Auth Middleware")

		// if request is for the health check endpoint, skip auth
		if c.Request.URL.Path == "/healthz" {
			c.Next()
			return
		}

		accessToken := c.GetHeader("Authorization")
		if accessToken == "" {
			logger.LogError(ctx, "no auth token provided")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		err := verifyAccessToken(miseServer, c, accessToken, config)
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
		ctx := GetContextFromGin(c)
		logger.LogDebug(ctx, "ARMTokenAuth Middleware")

		// If request path includes /arm/server/register/ then skip verifyProtectedLabSecretAndUserPrincipalName
		if !strings.Contains(c.Request.URL.Path, "/arm/server/register") {
			err := verifyProtectedLabSecretAndUserPrincipalName(c, appConfig)
			if err != nil {
				return
			}
		}

		accessToken := c.GetHeader("Authorization")
		if accessToken == "" {
			logger.LogError(ctx, "no auth token provided")
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
		ctx := GetContextFromGin(c)
		logger.LogDebug(ctx, "Middleware: AdminRequired")

		authToken := c.GetHeader("Authorization")

		callingUserPrincipal, err := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Get the roles for the calling user
		profile, err := authService.GetProfile(c.Request.Context(), callingUserPrincipal)
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
		ctx := GetContextFromGin(c)
		logger.LogDebug(ctx, "Middleware: MentorRequired")

		authToken := c.GetHeader("Authorization")

		callingUserPrincipal, err := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Get the roles for the calling user
		profile, err := authService.GetProfile(c.Request.Context(), callingUserPrincipal)
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
		ctx := GetContextFromGin(c)
		logger.LogDebug(ctx, "Middleware: contributorRequired")

		authToken := c.GetHeader("Authorization")

		callingUserPrincipal, err := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Get the roles for the calling user
		profile, err := authService.GetProfile(c.Request.Context(), callingUserPrincipal)
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
	ctx := GetContextFromGin(c)
	if c.GetHeader("ProtectedLabSecret") != appConfig.ProtectedLabSecret {
		logger.LogError(ctx, "ProtectedLabSecret header is missing or invalid")
		c.AbortWithStatus(http.StatusUnauthorized)
		return errors.New("ProtectedLabSecret header is missing or invalid")
	}

	if c.GetHeader("x-ms-client-principal-name") == "" {
		logger.LogError(ctx, "x-ms-client-principal-name header is missing")
		c.AbortWithStatus(http.StatusUnauthorized)
		return errors.New("x-ms-client-principal-name header is missing")
	}

	return nil
}

func verifyAccessToken(miseServer mise.Server, c *gin.Context, accessToken string, config config.Config) error {
	ctx := GetContextFromGin(c)
	splitToken := strings.Split(accessToken, "Bearer ")
	if len(splitToken) < 2 {
		logger.LogError(ctx, "found something in the Authorization header, but it's not a bearer token")
		c.AbortWithStatus(http.StatusUnauthorized)
		return errors.New(
			"found something in the Authorization header, but it's not a bearer token",
		)
	}

	if config.AuthVerifyMode == "MISE" {
		// MISE Implementation
		result, err := miseServer.DelegateAuthToContainer(accessToken, c.Request.URL.String(), c.Request.Method, c.ClientIP())
		if err != nil {
			var validationErr *mise.ErrTokenValidation
			if errors.As(err, &validationErr) {
				// can access validationErr.ErrorDescription, validationErr.WWWAuthenticate, validationErr.StatusCode
				logger.LogError(ctx, "token validation error", "error", validationErr)
			} else {
				logger.LogError(ctx, "error while delegating auth to container", "error", err)
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
			return err
		}

		userName, ok := result.SubjectClaims["preferred_username"]
		if !ok {
			logger.LogError(ctx, "preferred_username claim missing in subject claims")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing preferred username claim"})
			return errors.New("missing preferred username claim")
		}

		// Extract user principal from the slice
		var userPrincipal string
		if len(userName) > 0 {
			userPrincipal = userName[0] // take the first one
		}

		if userPrincipal == "" {
			logger.LogError(ctx, "preferred_username claim is empty")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "preferred_username claim is empty"})
			return errors.New("preferred_username claim is empty")
		}

		// Set user ID in context for tracing and user-specific operations
		SetUserIDInGin(c, userPrincipal)
		ctx = GetContextFromGin(c) // Get updated context with user ID

		logger.LogDebug(ctx, "authenticated user", "user", userPrincipal)

		return nil
	}

	// Keeping the custom auth validation in place, just in case MISE isn't working as expected.
	// Always defaults to Custom
	ok, err := auth.VerifyToken(c.Request.Context(), accessToken)
	if err != nil || !ok {
		logger.LogError(ctx, "token verification failed", "error", err.Error())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return err
	}

	// For custom auth, extract user principal from token
	token := splitToken[1]
	userPrincipal, err := auth.GetUserPrincipalFromToken(c.Request.Context(), token)
	if err != nil {
		logger.LogError(ctx, "failed to get user principal from token", "error", err.Error())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return err
	}

	// Set user ID in context for tracing and user-specific operations
	SetUserIDInGin(c, userPrincipal)
	ctx = GetContextFromGin(c) // Get updated context with user ID

	logger.LogDebug(ctx, "authenticated user using custom auth", "user", userPrincipal)

	return nil
}

func verifyArmAccessToken(c *gin.Context, accessToken string) error {
	ctx := GetContextFromGin(c)
	splitToken := strings.Split(accessToken, "Bearer ")
	if len(splitToken) < 2 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return errors.New(
			"found something in the Authorization header, but it's not a bearer token",
		)
	}

	ok, err := auth.VerifyArmToken(c.Request.Context(), accessToken)
	if err != nil || !ok {
		logger.LogError(ctx, "token verification failed", "error", err.Error())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return err
	}

	SetUserIDInGin(c, c.GetHeader("x-ms-client-principal-name"))
	ctx = GetContextFromGin(c)

	logger.LogDebug(ctx, "authenticated user using arm token auth", "user", c.GetHeader("x-ms-client-principal-name"))

	return nil
}
