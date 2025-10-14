package handler

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

type serverHandler struct {
	serverService entity.ServerService
}

func NewServerHandler(r *gin.RouterGroup, serverService entity.ServerService) {
	handler := &serverHandler{
		serverService: serverService,
	}

	r.PUT("/server/register/:subscriptionId", handler.RegisterSubscription)
	r.PUT("/server/unregister", handler.Unregister)

	r.PUT("/server/activity/:userPrincipalName", handler.UpdateActivityStatus)
}

func NewAdminServerHandler(r *gin.RouterGroup, serverService entity.ServerService) {
	handler := &serverHandler{
		serverService: serverService,
	}

	r.GET("/admin/servers", handler.AdminGetAllServers)
	r.DELETE("/admin/server/unregister/:userPrincipalName", handler.AdminUnregister)
}

func NewServerHandlerArmToken(r *gin.RouterGroup, serverService entity.ServerService) {
	handler := &serverHandler{
		serverService: serverService,
	}

	r.PUT("/arm/server/register", handler.RegisterSubscription)
	r.GET("/arm/server/:userPrincipalName", handler.ArmGetServer)
}

func (h *serverHandler) RegisterSubscription(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "registering server subscription")

	server := entity.Server{}
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid object"})
		return
	}

	if err := h.serverService.RegisterSubscription(c.Request.Context(), server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}

func (h *serverHandler) Unregister(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "unregistering server")

	userPrincipalName, err := auth.GetUserPrincipalFromToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not authorized or invalid token"})
	}

	if err := h.serverService.Unregister(c.Request.Context(), userPrincipalName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}

func (h *serverHandler) AdminUnregister(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "admin unregistering server", "requested_user_id", c.Param("userPrincipalName"))

	userPrincipalName := c.Param("userPrincipalName")

	if err := h.serverService.Unregister(c.Request.Context(), userPrincipalName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}

func (h *serverHandler) ArmGetServer(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "getting server for arm token", "requested_user_id", c.Param("userPrincipalName"))

	userPrincipalName := c.Param("userPrincipalName")
	if userPrincipalName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userPrincipalName is required"})
		return
	}

	server, err := h.serverService.GetServer(c.Request.Context(), userPrincipalName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"id": server.SubscriptionId})
}

func (h *serverHandler) AdminGetAllServers(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "getting all servers for admin")

	servers, err := h.serverService.GetAllServers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, servers)
}

func (h *serverHandler) UpdateActivityStatus(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "updating server activity status", "requested_user_id", c.Param("userPrincipalName"))

	userPrincipalName := c.Param("userPrincipalName")

	if !auth.VerifyUserPrincipalName(userPrincipalName, c.GetHeader("Authorization")) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid request"})
		return
	}

	if err := h.serverService.UpdateActivityStatus(c.Request.Context(), userPrincipalName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}
