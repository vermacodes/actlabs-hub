package handler

import (
	"net/http"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

type deploymentHandler struct {
	deploymentService entity.DeploymentService
}

func NewDeploymentHandler(r *gin.RouterGroup, service entity.DeploymentService) {
	handler := &deploymentHandler{
		deploymentService: service,
	}

	r.GET("/deployments", handler.GetUserDeployments)
	r.PUT("/deployments", handler.UpsertDeployment)
	r.DELETE("/deployments/:subscriptionId/:workspace", handler.DeleteDeployment)
}

func (d *deploymentHandler) GetUserDeployments(c *gin.Context) {
	userPrincipal, err := auth.GetUserPrincipalFromToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
	}

	deployments, err := d.deploymentService.GetUserDeployments(c.Request.Context(), userPrincipal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, deployments)
}

func (d *deploymentHandler) UpsertDeployment(c *gin.Context) {
	deployment := entity.Deployment{}
	if err := c.Bind(&deployment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userPrincipal, err := auth.GetUserPrincipalFromToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
	}

	deployment.DeploymentId = userPrincipal + "-" + deployment.DeploymentWorkspace + "-" + deployment.DeploymentSubscriptionId
	deployment.DeploymentUserId = userPrincipal

	if err := d.deploymentService.UpsertDeployment(c.Request.Context(), deployment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (d *deploymentHandler) DeleteDeployment(c *gin.Context) {
	subscriptionId := c.Param("subscriptionId")
	workspace := c.Param("workspace")

	userPrincipal, err := auth.GetUserPrincipalFromToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
	}

	if err := d.deploymentService.DeleteDeployment(c.Request.Context(), userPrincipal, workspace, subscriptionId); err != nil {
		slog.Error("error deleting deployment ", err)
	}

	c.Status(http.StatusNoContent)
}
