package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"

	"github.com/gin-gonic/gin"
)

type labHandler struct {
	labService entity.LabService
	appConfig  *config.Config
}

// Authenticated user.
func NewLabHandler(r *gin.RouterGroup, labService entity.LabService, appConfig *config.Config) {
	handler := &labHandler{
		labService: labService,
		appConfig:  appConfig,
	}
	// all private lab operations.
	r.GET("/lab/private/:typeOfLab", handler.GetLabs)
	r.POST("/lab/private", handler.UpsertLab)
	r.DELETE("/lab/private/:typeOfLab/:labId", handler.DeleteLab)
	r.GET("/lab/private/versions/:typeOfLab/:labId", handler.GetLabVersions)

	// public lab read-only operations.
	r.GET("/lab/public/:typeOfLab", handler.GetLabs)
	r.GET("/lab/public/versions/:typeOfLab/:labId", handler.GetLabVersions)
}

// Authenticated with ARM token and ProtectedLabSecret.
func NewLabHandlerARMTokenWithProtectedLabSecret(r *gin.RouterGroup, labService entity.LabService, appConfig *config.Config) {
	handler := &labHandler{
		labService: labService,
		appConfig:  appConfig,
	}

	// protected lab read-only operations. requires ARM token and super secret header.
	r.GET("/lab/protected/:typeOfLab/:labId", handler.GetLabWithSecret)
}

// Authenticated user with 'contributor' role.
func NewLabHandlerContributorRequired(r *gin.RouterGroup, labService entity.LabService) {
	handler := &labHandler{
		labService: labService,
	}

	// public lab mutable operations.
	r.POST("/lab/public", handler.UpsertLab)
	r.DELETE("/lab/public/:typeOfLab/:labId", handler.DeleteLab)
}

// Authenticated user with 'mentor' role.
func NewLabHandlerMentorRequired(r *gin.RouterGroup, labService entity.LabService) {
	handler := &labHandler{
		labService: labService,
	}

	// all protected lab operations.
	r.POST("/lab/protected", handler.UpsertLab)
	r.POST("/lab/protected/withSupportingDocument", handler.UpsertLabWithSupportingDocument)
	r.GET("/lab/protected/:typeOfLab", handler.GetLabs)
	r.GET("/lab/protected/versions/:typeOfLab/:labId", handler.GetLabVersions)
	r.DELETE("/lab/protected/:typeOfLab/:labId", handler.DeleteLab)

	// supporting documents testing only
	r.POST("/lab/protected/supportingDocument", handler.UpsertSupportingDocument)
	r.DELETE("/lab/protected/supportingDocument/:supportingDocumentId", handler.DeleteSupportingDocument)
	r.GET("/lab/protected/supportingDocument/:supportingDocumentId", handler.GetSupportingDocument)
}

func (l *labHandler) GetLabWithSecret(c *gin.Context) {
	typeOfLab := c.Param("typeOfLab")
	labId := c.Param("labId")

	var lab entity.LabType
	var err error

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")
	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	userId, _ := auth.GetUserPrincipalFromToken(authToken)

	switch {
	case validateLabType(typeOfLab, entity.ProtectedLabs):
		lab, err = l.labService.GetProtectedLab(c.Request.Context(), typeOfLab, labId, userId, true)
	case validateLabType(typeOfLab, entity.PrivateLab):
		lab, err = l.labService.GetPrivateLab(c.Request.Context(), typeOfLab, labId)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab type: " + typeOfLab})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, lab)
}

func (l *labHandler) GetLab(c *gin.Context) {
	typeOfLab := c.Param("typeOfLab")
	labId := c.Param("labId")

	var lab entity.LabType
	var err error

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")
	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	userId, _ := auth.GetUserPrincipalFromToken(authToken)

	switch {
	case validateLabType(typeOfLab, entity.ProtectedLabs):
		lab, err = l.labService.GetProtectedLab(c.Request.Context(), typeOfLab, labId, userId, false)
	case validateLabType(typeOfLab, entity.PrivateLab):
		lab, err = l.labService.GetPrivateLab(c.Request.Context(), typeOfLab, labId)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab type: " + typeOfLab})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, lab)
}

func (l *labHandler) GetLabs(c *gin.Context) {
	typeOfLab := c.Param("typeOfLab")

	var labs []entity.LabType
	var err error

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")
	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	userId, _ := auth.GetUserPrincipalFromToken(authToken)

	switch {
	case validateLabType(typeOfLab, entity.PrivateLab):
		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")
		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]
		userId, _ := auth.GetUserPrincipalFromToken(authToken)
		labs, err = l.labService.GetPrivateLabs(c.Request.Context(), typeOfLab, userId)
	case validateLabType(typeOfLab, entity.PublicLab):
		labs, err = l.labService.GetPublicLabs(c.Request.Context(), typeOfLab)
	case validateLabType(typeOfLab, entity.ProtectedLabs):
		labs, err = l.labService.GetProtectedLabs(c.Request.Context(), typeOfLab, userId, false)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab type: " + typeOfLab})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, labs)
}

func (l *labHandler) UpsertLab(c *gin.Context) {
	// Bind the request body to the LabType struct
	var lab entity.LabType
	if err := c.Bind(&lab); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	var upsertErr error

	switch {
	case validateLabType(lab.Type, entity.PrivateLab):
		lab, upsertErr = l.labService.UpsertPrivateLab(c.Request.Context(), lab)
	case validateLabType(lab.Type, entity.PublicLab):
		lab, upsertErr = l.labService.UpsertPublicLab(c.Request.Context(), lab)
	case validateLabType(lab.Type, entity.ProtectedLabs):
		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")
		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]
		userId, _ := auth.GetUserPrincipalFromToken(authToken)
		lab, upsertErr = l.labService.UpsertProtectedLab(c.Request.Context(), lab, userId)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab type: " + lab.Type})
		return
	}

	if upsertErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": upsertErr.Error()})
		return
	}

	c.JSON(http.StatusOK, lab)
}

func (l *labHandler) UpsertLabWithSupportingDocument(c *gin.Context) {
	// Parse the multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max memory
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
		return
	}

	// Get the lab field
	labField := c.Request.FormValue("lab")
	if labField == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lab field is required"})
		return
	}

	// Unmarshal the lab field into the LabType struct
	lab := entity.LabType{}
	if err := json.Unmarshal([]byte(labField), &lab); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab field: " + err.Error()})
		return
	}

	// Get the supportingDocument field (optional)
	supportingDocument, _, err := c.Request.FormFile("supportingDocument")
	if err != nil && err != http.ErrMissingFile {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving supporting document: " + err.Error()})
		return
	}

	if supportingDocument != nil {
		defer supportingDocument.Close()

		supportingDocumentId, err := l.labService.UpsertSupportingDocument(c.Request.Context(), supportingDocument)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		lab.SupportingDocumentId = supportingDocumentId
	}

	var upsertErr error

	switch {
	case validateLabType(lab.Type, entity.PrivateLab):
		lab, upsertErr = l.labService.UpsertPrivateLab(c.Request.Context(), lab)
	case validateLabType(lab.Type, entity.PublicLab):
		lab, upsertErr = l.labService.UpsertPublicLab(c.Request.Context(), lab)
	case validateLabType(lab.Type, entity.ProtectedLabs):
		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")
		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]
		userId, _ := auth.GetUserPrincipalFromToken(authToken)
		lab, upsertErr = l.labService.UpsertProtectedLab(c.Request.Context(), lab, userId)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab type: " + lab.Type})
		return
	}

	if upsertErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": upsertErr.Error()})
		return
	}

	c.JSON(http.StatusOK, lab)
}

func (l *labHandler) DeleteLab(c *gin.Context) {
	typeOfLab := c.Param("typeOfLab")
	labId := c.Param("labId")

	var err error

	// Get the auth token from the request header
	authToken := c.GetHeader("Authorization")
	// Remove Bearer from the authToken
	authToken = strings.Split(authToken, "Bearer ")[1]
	userId, _ := auth.GetUserPrincipalFromToken(authToken)

	switch {
	case validateLabType(typeOfLab, entity.PrivateLab):
		err = l.labService.DeletePrivateLab(c.Request.Context(), typeOfLab, labId, userId)
	case validateLabType(typeOfLab, entity.PublicLab):
		err = l.labService.DeletePublicLab(c.Request.Context(), typeOfLab, labId, userId)
	case validateLabType(typeOfLab, entity.ProtectedLabs):
		err = l.labService.DeleteProtectedLab(c.Request.Context(), typeOfLab, labId)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab type: " + typeOfLab})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (l *labHandler) GetLabVersions(c *gin.Context) {
	typeOfLab := c.Param("typeOfLab")
	labId := c.Param("labId")

	var labs []entity.LabType
	var err error

	switch {
	case validateLabType(typeOfLab, entity.PrivateLab):
		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")
		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]
		userId, _ := auth.GetUserPrincipalFromToken(authToken)
		labs, err = l.labService.GetPrivateLabVersions(c.Request.Context(), typeOfLab, labId, userId)
	case validateLabType(typeOfLab, entity.PublicLab):
		labs, err = l.labService.GetPublicLabVersions(c.Request.Context(), typeOfLab, labId)
	case validateLabType(typeOfLab, entity.ProtectedLabs):
		labs, err = l.labService.GetProtectedLabVersions(c.Request.Context(), typeOfLab, labId)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab type: " + typeOfLab})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, labs)
}

func validateLabType(typeOfLab string, validTypes []string) bool {
	for _, t := range validTypes {
		if typeOfLab == t {
			return true
		}
	}
	return false
}

// Supporting Documents
func (l *labHandler) UpsertSupportingDocument(c *gin.Context) {
	// Parse the multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max memory
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
		return
	}

	// Get the supportingDocument field
	supportingDocument, _, err := c.Request.FormFile("supportingDocument")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving supporting document: " + err.Error()})
		return
	}
	defer supportingDocument.Close()

	// Process the file as needed
	// For example, you can save the file to disk or process it in memory

	supportingDocumentId, err := l.labService.UpsertSupportingDocument(c.Request.Context(), supportingDocument)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"supportingDocumentId": supportingDocumentId})
}

func (l *labHandler) DeleteSupportingDocument(c *gin.Context) {
	supportingDocumentId := c.Param("supportingDocumentId")

	err := l.labService.DeleteSupportingDocument(c.Request.Context(), supportingDocumentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (l *labHandler) GetSupportingDocument(c *gin.Context) {
	supportingDocumentId := c.Param("supportingDocumentId")

	supportingDocumentReader, err := l.labService.GetSupportingDocument(c.Request.Context(), supportingDocumentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	defer supportingDocumentReader.Close()

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.pdf\"", supportingDocumentId))
	c.Header("Content-Type", "application/pdf")

	// Stream the file content to the response writer
	if _, err := io.Copy(c.Writer, supportingDocumentReader); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}
