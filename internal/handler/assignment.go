package handler

import (
	"net/http"
	"strings"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"

	"github.com/gin-gonic/gin"
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

}

func NewAssignmentAPIKeyHandler(r *gin.RouterGroup, service entity.AssignmentService, appConfig *config.Config) {
	handler := &assignmentHandler{
		assignmentService: service,
		appConfig:         appConfig,
	}
	// requires api key.
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
	logger.LogInfo(c.Request.Context(), "Get all assignments request received",
		"endpoint", "GET /assignment",
	)

	assignments, err := a.assignmentService.GetAllAssignments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, assignments)
}

func (a *assignmentHandler) GetAllLabsRedacted(c *gin.Context) {

	logger.LogInfo(c.Request.Context(), "Get all labs redacted request received",
		"endpoint", "GET /assignment/labs",
	)

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")
	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	userId, _ := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)

	labs, err := a.assignmentService.GetAllLabsRedacted(c.Request.Context(), userId)
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
	userId, _ := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)

	labs, err := a.assignmentService.GetAssignedLabsRedactedByUserId(c.Request.Context(), userId)
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
		userId, _ = auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)
	}

	labs, err := a.assignmentService.GetAssignedLabsRedactedByUserId(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, labs)
}

func (a *assignmentHandler) GetAssignmentsByLabId(c *gin.Context) {
	labId := c.Param("labId")
	assignments, err := a.assignmentService.GetAssignmentsByLabId(c.Request.Context(), labId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, assignments)
}

func (a *assignmentHandler) GetAssignmentsByUserId(c *gin.Context) {
	userId := c.Param("userId")
	assignments, err := a.assignmentService.GetAssignmentsByUserId(c.Request.Context(), userId)
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
	userPrincipal, _ := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)

	assignments, err := a.assignmentService.GetAssignmentsByUserId(c.Request.Context(), userPrincipal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, assignments)
}

func (a *assignmentHandler) CreateMyAssignments(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "Create my assignments request received",
		"endpoint", "POST /assignment/my",
	)

	bulkAssignment := entity.BulkAssignment{}
	if err := c.Bind(&bulkAssignment); err != nil {
		logger.LogError(c.Request.Context(), "Invalid request payload for create my assignments",
			"validation_error", err.Error(),
			"endpoint", "POST /assignment/my",
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userPrincipal, _ := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)

	// Sanitizing to make sure that the user is not creating assignments for other users.
	for _, userId := range bulkAssignment.UserIds {
		if userId != userPrincipal {
			logger.LogError(c.Request.Context(), "Unauthorized assignment creation attempt",
				"error", "user can only create assignments for themselves",
				"requesting_user", userPrincipal,
				"target_user", userId,
				"endpoint", "POST /assignment/my",
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": "you can only create assignments for yourself"})
			return
		}
	}

	if err := a.assignmentService.CreateAssignments(c.Request.Context(), bulkAssignment.UserIds, bulkAssignment.LabIds, userPrincipal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (a *assignmentHandler) CreateAssignments(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "Create assignments request received",
		"endpoint", "POST /assignment",
	)

	bulkAssignment := entity.BulkAssignment{}
	if err := c.Bind(&bulkAssignment); err != nil {
		logger.LogError(c.Request.Context(), "Invalid request payload for create assignments",
			"validation_error", err.Error(),
			"endpoint", "POST /assignment",
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userPrincipal, _ := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)

	if err := a.assignmentService.CreateAssignments(c.Request.Context(), bulkAssignment.UserIds, bulkAssignment.LabIds, userPrincipal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (a *assignmentHandler) UpdateAssignment(c *gin.Context) {
	userId := c.Param("userId")
	labId := c.Param("labId")
	status := c.Param("status")

	logger.LogInfo(c.Request.Context(), "Update assignment request received",
		"endpoint", "PUT /assignment/:userId/:labId/:status",
		"userId", userId,
		"labId", labId,
		"status", status,
	)

	if err := a.assignmentService.UpdateAssignment(c.Request.Context(), userId, labId, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (a *assignmentHandler) DeleteMyAssignments(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "Delete my assignments request received",
		"endpoint", "DELETE /assignment/my",
	)

	assignments := []string{}
	if err := c.Bind(&assignments); err != nil {
		logger.LogError(c.Request.Context(), "Invalid request payload for delete my assignments",
			"validation_error", err.Error(),
			"endpoint", "DELETE /assignment/my",
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userPrincipal, _ := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)

	// Sanitizing to make sure that the user is not deleting assignments for other users.
	for _, assignment := range assignments {
		if !strings.HasPrefix(assignment, userPrincipal) {
			logger.LogError(c.Request.Context(), "Unauthorized assignment deletion attempt",
				"error", "user can only delete their own assignments",
				"requesting_user", userPrincipal,
				"target_assignment", assignment,
				"endpoint", "DELETE /assignment/my",
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": "you can only delete your own assignments"})
			return
		}
	}

	if err := a.assignmentService.DeleteAssignments(c.Request.Context(), assignments, userPrincipal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (a *assignmentHandler) DeleteAssignments(c *gin.Context) {
	logger.LogInfo(c.Request.Context(), "Delete assignments request received",
		"endpoint", "DELETE /assignment",
	)

	assignments := []string{}
	if err := c.Bind(&assignments); err != nil {
		logger.LogError(c.Request.Context(), "Invalid request payload for delete assignments",
			"validation_error", err.Error(),
			"endpoint", "DELETE /assignment",
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")

	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	//Get the user principal from the auth token
	userPrincipal, _ := auth.GetUserPrincipalFromToken(c.Request.Context(), authToken)

	if err := a.assignmentService.DeleteAssignments(c.Request.Context(), assignments, userPrincipal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
