package handler

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"
	"log/slog"
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

	r.GET("/server", handler.GetServer)
	r.PUT("/server", handler.DeployServer)
	r.PUT("/server/update", handler.UpdateServer)
	r.DELETE("/server", handler.DestroyServer)

	r.PUT("/server/activity/:userPrincipalName", handler.UpdateActivityStatus)
}

func NewAdminServerHandler(r *gin.RouterGroup, serverService entity.ServerService) {
	handler := &serverHandler{
		serverService: serverService,
	}

	r.GET("/admin/servers", handler.AdminGetAllServers)
	r.DELETE("/admin/server/unregister/:userPrincipalName", handler.AdminUnregister)

	r.PUT("/admin/server", handler.AdminDeployServer)
	r.PUT("/admin/server/update", handler.AdminUpdateServer)
	r.DELETE("/admin/server/:userPrincipalName", handler.AdminDestroyServer)
}

func NewServerHandlerArmToken(r *gin.RouterGroup, serverService entity.ServerService) {
	handler := &serverHandler{
		serverService: serverService,
	}

	r.PUT("/arm/server/register", handler.RegisterSubscription)
}

func (h *serverHandler) RegisterSubscription(c *gin.Context) {
	server := entity.Server{}
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid object"})
		return
	}

	if err := h.serverService.RegisterSubscription(server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}

func (h *serverHandler) Unregister(c *gin.Context) {
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
	userPrincipalName := c.Param("userPrincipalName")

	if err := h.serverService.Unregister(c.Request.Context(), userPrincipalName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}

func (h *serverHandler) GetServer(c *gin.Context) {

	userPrincipalName, err := auth.GetUserPrincipalFromToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not authorized or invalid token"})
	}

	server, err := h.serverService.GetServer(userPrincipalName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, server)
}

func (h *serverHandler) AdminGetAllServers(c *gin.Context) {
	servers, err := h.serverService.GetAllServers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, servers)
}

func (h *serverHandler) UpdateServer(c *gin.Context) {
	server := entity.Server{}
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if server.UserPrincipalId == "" {
		slog.Info("UserPrincipalId not found in request, getting from token")
		userPrincipalId, err := auth.GetUserObjectIdFromToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not authorized or invalid token"})
		}
		server.UserPrincipalId = userPrincipalId

		// Updating server if user principal id is not present to record the user principal id
		if err := h.serverService.UpdateServer(server); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if !auth.VerifyUserObjectId(server.UserPrincipalId, c.GetHeader("Authorization")) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid request"})
		return
	}

	if err := h.serverService.UpdateServer(server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, server)
}

func (h *serverHandler) AdminUpdateServer(c *gin.Context) {
	server := entity.Server{}
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.serverService.UpdateServer(server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, server)
}

func (h *serverHandler) DeployServer(c *gin.Context) {
	server := entity.Server{}
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if server.UserPrincipalId == "" {
		slog.Info("UserPrincipalId not found in request, getting from token")
		userPrincipalId, err := auth.GetUserObjectIdFromToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not authorized or invalid token"})
		}
		server.UserPrincipalId = userPrincipalId

		// Updating server if user principal id is not present to record the user principal id
		if err := h.serverService.UpdateServer(server); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if !auth.VerifyUserObjectId(server.UserPrincipalId, c.GetHeader("Authorization")) {
		slog.Error("UserPrincipalId does not match token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid request"})
		return
	}

	server, err := h.serverService.DeployServer(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, server)
}

func (h *serverHandler) AdminDeployServer(c *gin.Context) {
	server := entity.Server{}
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.serverService.DeployServer(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, server)
}

func (h *serverHandler) DestroyServer(c *gin.Context) {
	userPrincipalName, err := auth.GetUserPrincipalFromToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not authorized or invalid token"})
	}

	err = h.serverService.DestroyServer(userPrincipalName, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}

func (h *serverHandler) AdminDestroyServer(c *gin.Context) {
	userPrincipalName := c.Param("userPrincipalName")

	err := h.serverService.DestroyServer(userPrincipalName, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}

func (h *serverHandler) UpdateActivityStatus(c *gin.Context) {
	userPrincipalName := c.Param("userPrincipalName")

	if !auth.VerifyUserPrincipalName(userPrincipalName, c.GetHeader("Authorization")) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid request"})
		return
	}

	if err := h.serverService.UpdateActivityStatus(userPrincipalName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}
