package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"golang.org/x/exp/slog"
)

func GetClaimFromToken(tokenString string, claim string) (string, error) {
	token, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	// Get the claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	value, ok := claims["oid"].(string)
	if !ok {
		return "", errors.New("not able to get oid from claims")
	}
	return value, nil
}

func ParseToken(tokenString string) (*jwt.Token, error) {
	// Drop the Bearer prefix if it exists
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.Split(tokenString, "Bearer ")[1]
	}

	keySet, err := jwk.Fetch(context.TODO(), "https://login.microsoftonline.com/common/discovery/v2.0/keys")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwa.RS256.String() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		keys, ok := keySet.LookupKeyID(kid)
		if !ok {
			return nil, fmt.Errorf("key %v not found", kid)
		}

		publicKey := &rsa.PublicKey{}
		err = keys.Raw(publicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key")
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		err := errors.New("token is not valid")
		return nil, err
	}

	return token, nil
}

func VerifyToken(tokenString string) (bool, error) {

	token, err := ParseToken(tokenString)
	if err != nil {
		return false, err
	}

	// Get the claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, errors.New("invalid claims")
	}

	// check the audience
	aud, ok := claims["aud"].(string)
	if !ok {
		return false, errors.New("not able to get audience from claims")
	}
	if aud != os.Getenv("AUTH_TOKEN_AUD") {
		return false, errors.New("unexpected audience, expected " + os.Getenv("AUTH_TOKEN_AUD") + " but got " + aud)
	}

	// Check the issuer
	iss, ok := claims["iss"].(string)
	if !ok {
		return false, errors.New("not able to get issuer from claims")
	}
	if iss != os.Getenv("AUTH_TOKEN_ISS") {
		return false, errors.New("unexpected issuer, expected " + os.Getenv("AUTH_TOKEN_ISS") + " but got " + iss)
	}

	// Check the expiration time
	exp, ok := claims["exp"].(float64)
	if !ok {
		return false, errors.New("invalid expiration time")
	}
	if time.Now().Unix() > int64(exp) {
		return false, errors.New("token has expired")
	}

	return true, nil
}

func GetTokenJSON(token string) (map[string]interface{}, error) {
	// Drop the Bearer prefix if it exists
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.Split(token, "Bearer ")[1]
	}

	// Split the token into its parts
	tokenParts := strings.Split(token, ".")
	if len(tokenParts) < 2 {
		err := errors.New("invalid token format")
		slog.Error("invalid token format", err)
		return map[string]interface{}{}, err
	}

	// Decode the token
	decodedToken, err := base64.URLEncoding.DecodeString(tokenParts[1] + strings.Repeat("=", (4-len(tokenParts[1])%4)%4))
	if err != nil {
		slog.Error("not able to decode token -> ", err)
		return map[string]interface{}{}, err
	}

	// Extract the user principal name from the decoded token
	var tokenJSON map[string]interface{}
	err = json.Unmarshal(decodedToken, &tokenJSON)
	if err != nil {
		slog.Error("not able to unmarshal token -> ", err)
		return map[string]interface{}{}, err
	}

	return tokenJSON, nil
}

func GetUserPrincipalFromToken(token string) (string, error) {

	// Split the token into its parts
	tokenJSON, err := GetTokenJSON(token)
	if err != nil {
		return "", err
	}

	userPrincipal, ok := tokenJSON["upn"].(string)
	if !ok {
		err := errors.New("user principal name not found in token")
		slog.Error("user principal name not found in token", err)
		return "", err
	}

	return userPrincipal, nil
}

func GetUserObjectIdFromToken(token string) (string, error) {

	// Split the token into its parts
	tokenJSON, err := GetTokenJSON(token)
	if err != nil {
		return "", err
	}

	userObjectId, ok := tokenJSON["oid"].(string)
	if !ok {
		err := errors.New("user object id not found in token")
		slog.Error("user object id not found in token", err)
		return "", err
	}

	return userObjectId, nil
}

func VerifyUserObjectId(userObjectId string, token string) bool {
	userObjectIdInToken, _ := GetUserObjectIdFromToken(token)
	return userObjectId == userObjectIdInToken
}

func VerifyUserPrincipalName(userPrincipalName string, token string) bool {
	userPrincipalNameInToken, _ := GetUserPrincipalFromToken(token)
	return userPrincipalName == userPrincipalNameInToken
}
