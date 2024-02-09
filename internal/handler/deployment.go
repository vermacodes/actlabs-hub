package handler

import (
	"context"
	"errors"
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
	userPrincipal, err := GetUserPrincipalFromToken(c.Request.Context(), d.deploymentService, c.GetHeader("Authorization"), c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
		return
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

	userPrincipal, err := GetUserPrincipalFromToken(c.Request.Context(), d.deploymentService, c.GetHeader("Authorization"), c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
		return
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

	userPrincipal, err := GetUserPrincipalFromToken(c.Request.Context(), d.deploymentService, c.GetHeader("Authorization"), c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
		return
	}

	if err := d.deploymentService.DeleteDeployment(c.Request.Context(), userPrincipal, workspace, subscriptionId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.Status(http.StatusNoContent)
}

func GetUserPrincipalFromToken(ctx context.Context, d entity.DeploymentService, token string, c *gin.Context) (string, error) {
	userPrincipal, err := auth.GetUserPrincipalFromToken(token)
	if err != nil && userPrincipal == "" {
		slog.Debug("no user principal found in token checking if it is an MSI or SP token")

		oid, err := auth.GetUserObjectIdFromToken(token)
		if err != nil {
			return "", errors.New("invalid access token")
		}

		if oid == "dbe22174-3ecc-4cfb-be36-76ed132ef90c" {
			slog.Debug("service principal request, checking for x-ms-client-principal-name header")
			userPrincipal := c.GetHeader("x-ms-client-principal-name")
			if userPrincipal == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
				return "", errors.New("missing header in request")
			}

			return userPrincipal, nil
		}

		userPrincipal, err := d.GetUserPrincipalNameByMSIPrincipalID(ctx, oid)
		if err != nil {
			return "", errors.New("invalid access token")
		}

		if userPrincipal == "" {
			return "", errors.New("invalid access token")
		}

		return userPrincipal, nil
	}

	return userPrincipal, nil
}
