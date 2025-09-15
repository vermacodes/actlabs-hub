package mise

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"actlabs-hub/internal/miseadapter"

	"golang.org/x/exp/slog"
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

func (s Server) DelegateAuthToContainer(authHeader, uri, method, ipAddr string) (miseadapter.Result, error) {
	if s.VerboseLogging {
		slog.Debug("Original request Information",
			slog.String("url", uri),
			slog.String("method", method),
			slog.String("ip", ipAddr),
		)
	}

	start := time.Now()

	result, err := s.ContainerClient.ValidateRequest(context.Background(), miseadapter.Input{
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

	slog.Debug("time elapsed in mise container + adapter", slog.Int64("ms", elapsed.Milliseconds()))

	if err != nil {
		slog.Error("error while validating token with mise adapter", slog.String("error", err.Error()))
		return miseadapter.Result{}, fmt.Errorf("error while validating token: %w", err)
	}

	if s.VerboseLogging {
		b, merr := json.MarshalIndent(result, "", "   ")
		if merr != nil {
			slog.Error("error marshalling json of result object", slog.String("error", merr.Error()))
		} else {
			slog.Debug("VERBOSE: result struct", slog.String("result", string(b)))
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
