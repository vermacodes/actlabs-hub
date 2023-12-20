package handler

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
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
	userPrincipal := c.Param("userPrincipal")

	// My roles
	if userPrincipal == "my" {

		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")

		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]

		userPrincipal, _ = auth.GetUserPrincipalFromToken(authToken)
	}

	profile, err := h.authService.GetProfile(userPrincipal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *AuthHandler) GetAllProfilesRedacted(c *gin.Context) {
	profiles, err := h.authService.GetAllProfilesRedacted()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

func (h *AuthHandler) GetAllProfiles(c *gin.Context) {
	profiles, err := h.authService.GetAllProfiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

func (h *AuthHandler) AddRole(c *gin.Context) {
	userPrincipal := c.Param("userPrincipal")
	role := c.Param("role")
	err := h.authService.AddRole(userPrincipal, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *AuthHandler) CreateProfile(c *gin.Context) {

	profile := entity.Profile{}

	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]

	userPrincipal, _ := auth.GetUserPrincipalFromToken(authToken)
	role := "user"

	// Ensure that the calling user is adding their own profile.
	if userPrincipal != profile.UserPrincipal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userPrincipal in the request body does not match the calling user"})
		return
	}

	// Ensure that the calling user is adding the user role.
	if !helper.Contains(profile.Roles, role) {
		profile.Roles = append(profile.Roles, role)
	}

	err := h.authService.CreateProfile(profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *AuthHandler) DeleteRole(c *gin.Context) {
	userPrincipal := c.Param("userPrincipal")
	role := c.Param("role")
	err := h.authService.DeleteRole(userPrincipal, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
