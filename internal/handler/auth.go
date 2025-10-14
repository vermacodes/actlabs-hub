package handler

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService entity.AuthService
}

func NewAuthHandler(r *gin.RouterGroup, authService entity.AuthService) {
	handler := &AuthHandler{
		authService: authService,
	}

	r.GET("/profiles/:userPrincipal", handler.GetProfile)
	r.POST("/profiles", handler.CreateProfile)
	r.GET("/profilesRedacted", handler.GetAllProfilesRedacted)
}

func NewAdminAuthHandler(r *gin.RouterGroup, authService entity.AuthService) {
	handler := &AuthHandler{
		authService: authService,
	}

	r.GET("/profiles", handler.GetAllProfiles)
	r.POST("/profiles/:userPrincipal/:role", handler.AddRole)
	r.DELETE("/profiles/:userPrincipal/:role", handler.DeleteRole)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "get profile request",
		"user_principal", c.Param("userPrincipal"),
	)

	userPrincipal := c.Param("userPrincipal")

	// My roles
	if userPrincipal == "my" {

		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")

		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]

		userPrincipal, _ = auth.GetUserPrincipalFromToken(authToken)
	}

	profile, err := h.authService.GetProfile(c.Request.Context(), userPrincipal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *AuthHandler) GetAllProfilesRedacted(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "get all profiles redacted request")

	profiles, err := h.authService.GetAllProfilesRedacted(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

func (h *AuthHandler) GetAllProfiles(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "get all profiles request")

	profiles, err := h.authService.GetAllProfiles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

func (h *AuthHandler) AddRole(c *gin.Context) {
	userPrincipal := c.Param("userPrincipal")
	role := c.Param("role")

	logger.LogInfo(c.Request.Context(), "add role request",
		"user_principal", userPrincipal,
		"role", role,
	)

	err := h.authService.AddRole(c.Request.Context(), userPrincipal, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *AuthHandler) CreateProfile(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "create profile request")

	profile := entity.Profile{}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]

	userPrincipal, err := auth.GetUserPrincipalFromToken(authToken)
	if err != nil {
		logger.LogError(c.Request.Context(), "Failed to extract user principal from token",
			"endpoint", "POST /profiles",
			"error", err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not able to identify user"})
		return
	}

	role := "user"

	if err := c.ShouldBindJSON(&profile); err != nil {
		logger.LogError(c.Request.Context(), "Invalid request payload for create profile",
			"validation_error", err.Error(),
			"endpoint", "POST /profiles",
			"user_principal", userPrincipal,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure that the calling user is adding their own profile.
	if userPrincipal != profile.UserPrincipal {
		logger.LogError(c.Request.Context(), "Authorization violation: user attempting to create profile for different user",
			"error", "userPrincipal mismatch",
			"requesting_user", userPrincipal,
			"target_user", profile.UserPrincipal,
			"endpoint", "POST /profiles",
		)

		c.JSON(http.StatusBadRequest, gin.H{"error": "userPrincipal in the request body does not match the calling user"})
		return
	}

	// Ensure that the calling user is adding the user role.
	if !helper.Contains(profile.Roles, role) {
		profile.Roles = append(profile.Roles, role)
	}

	err = h.authService.CreateProfile(c.Request.Context(), profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *AuthHandler) DeleteRole(c *gin.Context) {
	userPrincipal := c.Param("userPrincipal")
	role := c.Param("role")

	logger.LogInfo(c.Request.Context(), "delete role request",
		"user_principal", userPrincipal,
		"role", role,
	)

	err := h.authService.DeleteRole(c.Request.Context(), userPrincipal, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
