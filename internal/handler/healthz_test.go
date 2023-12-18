package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthzHandler_Healthz(t *testing.T) {
	// Set up the Gin router
	router := gin.Default()

	// Create the handler
	NewHealthzHandler(&router.RouterGroup)

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the status code is what we expect
	if w.Code != http.StatusOK {
		t.Errorf("Expected response code to be 200, got %d", w.Code)
	}
}
