package handler

import (
	"net/http"
	"strings"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

type assignmentHandler struct {
	assignmentService entity.AssignmentService
	appConfig         *config.Config
}

func NewAssignmentHandler(r *gin.RouterGroup, service entity.AssignmentService, appConfig *config.Config) {
	handler := &assignmentHandler{
		assignmentService: service,
		appConfig:         appConfig,
	}

	r.GET("/assignment/labs", handler.GetAllLabsRedacted)
	r.GET("/assignment/labs/my", handler.GetMyAssignedLabsRedacted)
	r.GET("/assignment/my", handler.GetMyAssignments)
	r.POST("/assignment/my", handler.CreateMyAssignments)
	r.DELETE("/assignment/my", handler.DeleteMyAssignments)

	// requires super secret header.
	r.PUT("/assignment/:userId/:labId/:status", handler.UpdateAssignment)
}

func NewAssignmentHandlerMentorRequired(r *gin.RouterGroup, service entity.AssignmentService) {
	handler := &assignmentHandler{
		assignmentService: service,
	}

	r.GET("/assignment", handler.GetAllAssignments)
	r.GET("/assignment/lab/:labId", handler.GetAssignmentsByLabId)
	r.GET("/assignment/user/:userId", handler.GetAssignmentsByUserId)
	r.POST("/assignment", handler.CreateAssignments)
	r.DELETE("/assignment", handler.DeleteAssignments)
}

func (a *assignmentHandler) GetAllAssignments(c *gin.Context) {
	assignments, err := a.assignmentService.GetAllAssignments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, assignments)
}

func (a *assignmentHandler) GetAllLabsRedacted(c *gin.Context) {

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")
	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	userId, _ := auth.GetUserPrincipalFromToken(authToken)

	labs, err := a.assignmentService.GetAllLabsRedacted(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, labs)
}

func (a *assignmentHandler) GetMyAssignedLabsRedacted(c *gin.Context) {

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userId, _ := auth.GetUserPrincipalFromToken(authToken)

	labs, err := a.assignmentService.GetAssignedLabsRedactedByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, labs)
}

func (a *assignmentHandler) GetAssignedLabsRedactedByUserId(c *gin.Context) {
	userId := c.Param("userId")
	if userId == "my" {
		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")

		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]
		//Get the user principal from the auth token
		userId, _ = auth.GetUserPrincipalFromToken(authToken)
	}

	labs, err := a.assignmentService.GetAssignedLabsRedactedByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, labs)
}

func (a *assignmentHandler) GetAssignmentsByLabId(c *gin.Context) {
	labId := c.Param("labId")
	assignments, err := a.assignmentService.GetAssignmentsByLabId(labId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, assignments)
}

func (a *assignmentHandler) GetAssignmentsByUserId(c *gin.Context) {
	userId := c.Param("userId")
	assignments, err := a.assignmentService.GetAssignmentsByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, assignments)
}

func (a *assignmentHandler) GetMyAssignments(c *gin.Context) {
	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userPrincipal, _ := auth.GetUserPrincipalFromToken(authToken)

	assignments, err := a.assignmentService.GetAssignmentsByUserId(userPrincipal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, assignments)
}

func (a *assignmentHandler) CreateMyAssignments(c *gin.Context) {
	bulkAssignment := entity.BulkAssignment{}
	if err := c.Bind(&bulkAssignment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userPrincipal, _ := auth.GetUserPrincipalFromToken(authToken)

	// Sanitizing to make sure that the user is not creating assignments for other users.
	for _, userId := range bulkAssignment.UserIds {
		if userId != userPrincipal {
			c.JSON(http.StatusBadRequest, gin.H{"error": "you can only create assignments for yourself"})
			return
		}
	}

	if err := a.assignmentService.CreateAssignments(bulkAssignment.UserIds, bulkAssignment.LabIds, userPrincipal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (a *assignmentHandler) CreateAssignments(c *gin.Context) {
	bulkAssignment := entity.BulkAssignment{}
	if err := c.Bind(&bulkAssignment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userPrincipal, _ := auth.GetUserPrincipalFromToken(authToken)

	if err := a.assignmentService.CreateAssignments(bulkAssignment.UserIds, bulkAssignment.LabIds, userPrincipal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (a *assignmentHandler) UpdateAssignment(c *gin.Context) {
	userId := c.Param("userId")
	labId := c.Param("labId")
	status := c.Param("status")

	// get super secret from header
	protectedLabSecret := c.Request.Header.Get("ProtectedLabSecret")
	if protectedLabSecret != a.appConfig.ProtectedLabSecret {
		slog.Error("invalid protected lab secret",
			slog.String("userId", userId),
			slog.String("labId", labId),
			slog.String("status", status),
			slog.String("protectedLabSecret", protectedLabSecret),
		)

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Protected lab secret is invalid."})
		return
	}

	if err := a.assignmentService.UpdateAssignment(userId, labId, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (a *assignmentHandler) DeleteMyAssignments(c *gin.Context) {
	assignments := []string{}
	if err := c.Bind(&assignments); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userPrincipal, _ := auth.GetUserPrincipalFromToken(authToken)

	// Sanitizing to make sure that the user is not deleting assignments for other users.
	for _, assignment := range assignments {
		if !strings.HasPrefix(assignment, userPrincipal) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "you can only delete your own assignments"})
			return
		}
	}

	if err := a.assignmentService.DeleteAssignments(assignments, userPrincipal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (a *assignmentHandler) DeleteAssignments(c *gin.Context) {
	assignments := []string{}
	if err := c.Bind(&assignments); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userPrincipal, _ := auth.GetUserPrincipalFromToken(authToken)

	if err := a.assignmentService.DeleteAssignments(assignments, userPrincipal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
