package mise

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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

func (s Server) DelegateAuthToContainer(authHeader, uri, method, ipAddr string) (miseadapter.Result, error) {
	if s.VerboseLogging {
		log.Default().Printf(
			"VERBOSE: Original request Information:\nURL:%s, method:%s, original IP address:%s",
			uri,
			method,
			ipAddr,
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

	log.Default().Printf("time elapsed in mise container + adapter: %d", elapsed.Milliseconds())

	if err != nil {
		return miseadapter.Result{}, fmt.Errorf("error while validating token: %w", err)
	}

	if s.VerboseLogging {
		json, err := json.MarshalIndent(result, "", "   ")
		if err != nil {
			log.Default().Printf("error marshalling json of result object err=%v\n", err)
		} else {
			log.Default().Printf("VERBOSE: result struct:\n%s", string(json))
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
