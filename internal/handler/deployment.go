package handler

import (
	"net/http"

	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"

	"github.com/gin-gonic/gin"
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
	logger.LogInfo(c.Request.Context(), "getting user deployments")

	userPrincipal := c.GetHeader("x-ms-client-principal-name")

	deployments, err := d.deploymentService.GetUserDeployments(c.Request.Context(), userPrincipal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, deployments)
}

func (d *deploymentHandler) UpsertDeployment(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "upserting deployment")

	deployment := entity.Deployment{}
	if err := c.Bind(&deployment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userPrincipal := c.GetHeader("x-ms-client-principal-name")

	deployment.DeploymentId = userPrincipal + "-" + deployment.DeploymentWorkspace + "-" + deployment.DeploymentSubscriptionId
	deployment.DeploymentUserId = userPrincipal

	if err := d.deploymentService.UpsertDeployment(c.Request.Context(), deployment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (d *deploymentHandler) DeleteDeployment(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "deleting deployment")

	subscriptionId := c.Param("subscriptionId")
	workspace := c.Param("workspace")

	userPrincipal := c.GetHeader("x-ms-client-principal-name")

	if err := d.deploymentService.DeleteDeployment(c.Request.Context(), userPrincipal, workspace, subscriptionId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.Status(http.StatusNoContent)
}
