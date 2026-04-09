package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"

	"github.com/gin-gonic/gin"
)

// --- Mock ChallengeService ---

type mockChallengeService struct {
	labs       []entity.LabType
	challenges []entity.Challenge
	err        error
	// Track calls for verification
	lastDeletedIds []string
	lastUpserted   []entity.Challenge
	lastUpdateArgs struct {
		userId string
		labId  string
		status string
	}
}

func (m *mockChallengeService) GetAllLabsRedacted(ctx context.Context) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockChallengeService) GetChallengesLabsRedactedByUserId(ctx context.Context, userId string) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockChallengeService) GetAllChallenges(ctx context.Context) ([]entity.Challenge, error) {
	return m.challenges, m.err
}
func (m *mockChallengeService) GetChallengeByUserIdAndLabId(ctx context.Context, userId string, labId string) (entity.Challenge, error) {
	if len(m.challenges) > 0 {
		return m.challenges[0], m.err
	}
	return entity.Challenge{}, m.err
}
func (m *mockChallengeService) GetChallengesByLabId(ctx context.Context, labId string) ([]entity.Challenge, error) {
	return m.challenges, m.err
}
func (m *mockChallengeService) GetChallengesByUserId(ctx context.Context, userId string) ([]entity.Challenge, error) {
	return m.challenges, m.err
}
func (m *mockChallengeService) UpsertChallenges(ctx context.Context, challenges []entity.Challenge) error {
	m.lastUpserted = challenges
	return m.err
}
func (m *mockChallengeService) UpdateChallenge(ctx context.Context, userId string, labId string, status string) error {
	m.lastUpdateArgs.userId = userId
	m.lastUpdateArgs.labId = labId
	m.lastUpdateArgs.status = status
	return m.err
}
func (m *mockChallengeService) CreateChallenges(ctx context.Context, userIds []string, labIds []string, createdBy string) error {
	return m.err
}
func (m *mockChallengeService) DeleteChallenges(ctx context.Context, challengeIds []string) error {
	m.lastDeletedIds = challengeIds
	return m.err
}

// --- Helpers ---

// makeFakeJWT builds a fake JWT whose payload contains the given claims.
// The header and signature are dummy values; only the base64-encoded payload matters
// because auth.GetUserPrincipalFromToken does not verify signatures.
func makeFakeJWT(claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, _ := json.Marshal(claims)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	return fmt.Sprintf("%s.%s.fakesig", header, payloadB64)
}

func setupChallengeRouter(svc entity.ChallengeService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	appConfig := &config.Config{}
	NewChallengeHandler(router.Group("/"), svc, appConfig)
	NewChallengeAPIKeyHandler(router.Group("/"), svc, appConfig)
	return router
}

// --- Tests: GET /challenge/labs ---

func TestGetAllLabsRedacted_Success(t *testing.T) {
	svc := &mockChallengeService{
		labs: []entity.LabType{
			{Id: "lab1", Name: "Challenge Lab 1"},
			{Id: "lab2", Name: "Challenge Lab 2"},
		},
	}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("GET", "/challenge/labs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var labs []entity.LabType
	if err := json.Unmarshal(w.Body.Bytes(), &labs); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(labs) != 2 {
		t.Errorf("expected 2 labs, got %d", len(labs))
	}
}

func TestGetAllLabsRedacted_Empty(t *testing.T) {
	svc := &mockChallengeService{labs: nil}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("GET", "/challenge/labs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetAllLabsRedacted_ServiceError(t *testing.T) {
	svc := &mockChallengeService{err: errors.New("storage unavailable")}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("GET", "/challenge/labs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Tests: GET /challenge ---

func TestGetAllChallenges_Success(t *testing.T) {
	svc := &mockChallengeService{
		challenges: []entity.Challenge{
			{ChallengeId: "user1@microsoft.com+lab1", UserId: "user1@microsoft.com", LabId: "lab1", Status: "challenged"},
			{ChallengeId: "user2@microsoft.com+lab2", UserId: "user2@microsoft.com", LabId: "lab2", Status: "accepted"},
		},
	}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("GET", "/challenge", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var challenges []entity.Challenge
	if err := json.Unmarshal(w.Body.Bytes(), &challenges); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(challenges) != 2 {
		t.Errorf("expected 2 challenges, got %d", len(challenges))
	}
	if challenges[0].Status != "challenged" {
		t.Errorf("expected status 'challenged', got %q", challenges[0].Status)
	}
}

func TestGetAllChallenges_ServiceError(t *testing.T) {
	svc := &mockChallengeService{err: errors.New("db error")}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("GET", "/challenge", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Tests: GET /challenge/lab/:labId ---

func TestGetChallengesByLabId_Success(t *testing.T) {
	svc := &mockChallengeService{
		challenges: []entity.Challenge{
			{ChallengeId: "user1@microsoft.com+lab1", UserId: "user1@microsoft.com", LabId: "lab1", Status: "challenged"},
		},
	}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("GET", "/challenge/lab/lab1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var challenges []entity.Challenge
	if err := json.Unmarshal(w.Body.Bytes(), &challenges); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(challenges) != 1 {
		t.Errorf("expected 1 challenge, got %d", len(challenges))
	}
}

func TestGetChallengesByLabId_ServiceError(t *testing.T) {
	svc := &mockChallengeService{err: errors.New("db error")}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("GET", "/challenge/lab/lab1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Tests: GET /challenge/my ---

func TestGetMyChallenges_Success(t *testing.T) {
	svc := &mockChallengeService{
		challenges: []entity.Challenge{
			{ChallengeId: "testuser@microsoft.com+lab1", UserId: "testuser@microsoft.com", LabId: "lab1", Status: "challenged"},
		},
	}
	router := setupChallengeRouter(svc)

	token := makeFakeJWT(map[string]interface{}{"upn": "testuser@microsoft.com"})
	req, _ := http.NewRequest("GET", "/challenge/my", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var challenges []entity.Challenge
	if err := json.Unmarshal(w.Body.Bytes(), &challenges); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(challenges) != 1 {
		t.Errorf("expected 1 challenge, got %d", len(challenges))
	}
}

func TestGetMyChallenges_ServiceError(t *testing.T) {
	svc := &mockChallengeService{err: errors.New("db error")}
	router := setupChallengeRouter(svc)

	token := makeFakeJWT(map[string]interface{}{"upn": "testuser@microsoft.com"})
	req, _ := http.NewRequest("GET", "/challenge/my", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Tests: GET /challenge/labs/my ---

func TestGetMyChallengeLabsRedacted_Success(t *testing.T) {
	svc := &mockChallengeService{
		labs: []entity.LabType{
			{Id: "lab1", Name: "My Challenge Lab"},
		},
	}
	router := setupChallengeRouter(svc)

	token := makeFakeJWT(map[string]interface{}{"upn": "testuser@microsoft.com"})
	req, _ := http.NewRequest("GET", "/challenge/labs/my", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var labs []entity.LabType
	if err := json.Unmarshal(w.Body.Bytes(), &labs); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(labs) != 1 || labs[0].Id != "lab1" {
		t.Errorf("unexpected labs: %+v", labs)
	}
}

func TestGetMyChallengeLabsRedacted_ServiceError(t *testing.T) {
	svc := &mockChallengeService{err: errors.New("service error")}
	router := setupChallengeRouter(svc)

	token := makeFakeJWT(map[string]interface{}{"upn": "testuser@microsoft.com"})
	req, _ := http.NewRequest("GET", "/challenge/labs/my", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Tests: POST /challenge ---

func TestUpsertChallenges_Success(t *testing.T) {
	svc := &mockChallengeService{}
	router := setupChallengeRouter(svc)

	challenges := []entity.Challenge{
		{ChallengeId: "user1@microsoft.com+lab1", UserId: "user1@microsoft.com", LabId: "lab1", Status: "challenged"},
	}
	body, _ := json.Marshal(challenges)

	req, _ := http.NewRequest("POST", "/challenge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(svc.lastUpserted) != 1 {
		t.Fatalf("expected 1 upserted challenge, got %d", len(svc.lastUpserted))
	}
	if svc.lastUpserted[0].ChallengeId != "user1@microsoft.com+lab1" {
		t.Errorf("unexpected upserted challenge id: %s", svc.lastUpserted[0].ChallengeId)
	}
}

func TestUpsertChallenges_InvalidJSON(t *testing.T) {
	svc := &mockChallengeService{}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("POST", "/challenge", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpsertChallenges_ServiceError(t *testing.T) {
	svc := &mockChallengeService{err: errors.New("upsert failed")}
	router := setupChallengeRouter(svc)

	challenges := []entity.Challenge{
		{ChallengeId: "user1@microsoft.com+lab1", UserId: "user1@microsoft.com", LabId: "lab1"},
	}
	body, _ := json.Marshal(challenges)

	req, _ := http.NewRequest("POST", "/challenge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpsertChallenges_EmptyBody(t *testing.T) {
	svc := &mockChallengeService{}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("POST", "/challenge", bytes.NewReader([]byte(`[]`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Tests: PUT /challenge/:userId/:labId/:status ---

func TestUpdateChallenge_Success(t *testing.T) {
	svc := &mockChallengeService{}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("PUT", "/challenge/user1@microsoft.com/lab1/accepted", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if svc.lastUpdateArgs.userId != "user1@microsoft.com" {
		t.Errorf("expected userId 'user1@microsoft.com', got %q", svc.lastUpdateArgs.userId)
	}
	if svc.lastUpdateArgs.labId != "lab1" {
		t.Errorf("expected labId 'lab1', got %q", svc.lastUpdateArgs.labId)
	}
	if svc.lastUpdateArgs.status != "accepted" {
		t.Errorf("expected status 'accepted', got %q", svc.lastUpdateArgs.status)
	}
}

func TestUpdateChallenge_ServiceError(t *testing.T) {
	svc := &mockChallengeService{err: errors.New("update failed")}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("PUT", "/challenge/user1@microsoft.com/lab1/accepted", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Tests: DELETE /challenge/:challengeId ---

func TestDeleteChallenge_Success(t *testing.T) {
	svc := &mockChallengeService{}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("DELETE", "/challenge/user1@microsoft.com+lab1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(svc.lastDeletedIds) != 1 || svc.lastDeletedIds[0] != "user1@microsoft.com+lab1" {
		t.Errorf("unexpected deleted ids: %v", svc.lastDeletedIds)
	}

	var result string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result != "user1@microsoft.com+lab1" {
		t.Errorf("expected response body to be the challenge id, got %q", result)
	}
}

func TestDeleteChallenge_ServiceError(t *testing.T) {
	svc := &mockChallengeService{err: errors.New("delete not allowed")}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("DELETE", "/challenge/user1@microsoft.com+lab1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Tests: Multiple challenges in batch ---

func TestUpsertChallenges_MultipleChallenges(t *testing.T) {
	svc := &mockChallengeService{}
	router := setupChallengeRouter(svc)

	challenges := []entity.Challenge{
		{ChallengeId: "user1@microsoft.com+lab1", UserId: "user1@microsoft.com", LabId: "lab1"},
		{ChallengeId: "user2@microsoft.com+lab2", UserId: "user2@microsoft.com", LabId: "lab2"},
		{ChallengeId: "user3@microsoft.com+lab1", UserId: "user3@microsoft.com", LabId: "lab1"},
	}
	body, _ := json.Marshal(challenges)

	req, _ := http.NewRequest("POST", "/challenge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(svc.lastUpserted) != 3 {
		t.Errorf("expected 3 upserted challenges, got %d", len(svc.lastUpserted))
	}
}

// --- Tests: Response content type ---

func TestGetAllChallenges_ReturnsJSON(t *testing.T) {
	svc := &mockChallengeService{
		challenges: []entity.Challenge{
			{ChallengeId: "user1@microsoft.com+lab1"},
		},
	}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("GET", "/challenge", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", contentType)
	}
}

// --- Tests: Route matching ---

func TestNonExistentRoute_Returns404(t *testing.T) {
	svc := &mockChallengeService{}
	router := setupChallengeRouter(svc)

	req, _ := http.NewRequest("GET", "/challenge/nonexistent/route", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestWrongHTTPMethod_Returns405(t *testing.T) {
	svc := &mockChallengeService{}
	router := setupChallengeRouter(svc)

	// PATCH is not registered
	req, _ := http.NewRequest("PATCH", "/challenge", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Gin returns 404 for unmatched method by default (unless HandleMethodNotAllowed is set)
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 404 or 405, got %d", w.Code)
	}
}
