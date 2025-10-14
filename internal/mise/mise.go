package mise

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"actlabs-hub/internal/logger"
	"actlabs-hub/internal/miseadapter"
)

type Server struct {
	ContainerClient miseadapter.Client
	VerboseLogging  bool
}

type ErrTokenValidation struct {
	StatusCode       int
	ErrorDescription []string
	WWWAuthenticate  []string
}

func (e *ErrTokenValidation) Error() string {
	return fmt.Sprintf("StatusCode: %d, ErrorDescription: %v, WWWAuthenticate: %v", e.StatusCode, e.ErrorDescription, e.WWWAuthenticate)
}

func (s Server) DelegateAuthToContainer(ctx context.Context, authHeader, uri, method, ipAddr string) (miseadapter.Result, error) {
	if s.VerboseLogging {
		logger.LogDebug(ctx, "delegating auth to mise container",
			"url", uri,
			"method", method,
			"ip", ipAddr,
		)
	}

	start := time.Now()

	// Create a background context from the request context for the external service call
	// This ensures the external call can complete even if the request is cancelled
	bgCtx := context.WithoutCancel(ctx)

	result, err := s.ContainerClient.ValidateRequest(bgCtx, miseadapter.Input{
		OriginalUri:    uri,
		OriginalMethod: method,

		OriginalIPAddress:   ipAddr,
		AuthorizationHeader: authHeader,

		// Replace SubjectClaimsToReturn with ReturnAllSubjectClaims
		// to return all claims in the subject token instead of just an allow list.
		//ReturnAllSubjectClaims: true,
		SubjectClaimsToReturn: []string{"preferred_username"},
	})

	end := time.Now()
	elapsed := end.Sub(start)

	if s.VerboseLogging {
		logger.LogDebug(ctx, "time elapsed in mise container validation", "duration_ms", elapsed.Milliseconds())
	}

	if err != nil {
		logger.LogError(ctx, "failed to validate token with mise adapter", "error", err)
		return miseadapter.Result{}, fmt.Errorf("error while validating token: %w", err)
	}

	if s.VerboseLogging {
		b, merr := json.MarshalIndent(result, "", "   ")
		if merr != nil {
			logger.LogError(ctx, "failed to marshal mise result for verbose logging", "error", merr)
		} else {
			logger.LogDebug(ctx, "mise validation result", "result", string(b))
		}
	}

	if result.StatusCode == http.StatusOK {
		return result, nil
	} else {
		return miseadapter.Result{}, &ErrTokenValidation{
			StatusCode:       result.StatusCode,
			ErrorDescription: result.ErrorDescription,
			WWWAuthenticate:  result.WWWAuthenticate,
		}
	}
}
