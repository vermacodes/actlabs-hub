package helper

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
)

var alphabet = []byte("abcdefghijklmnopqrstuvwxyz0123456789")

func Generate(length int) string {
	// Generate a alphanumeric string of length length.

	b := make([]byte, length)
	rand.Read(b)
	for i := 0; i < length; i++ {
		b[i] = alphabet[b[i]%byte(len(alphabet))]
	}
	return *(*string)(unsafe.Pointer(&b))
}

// Function to convert a slice of strings to a single string delimited by a comma
func SliceToString(s []string) string {
	return strings.Join(s, ",")
}

// Function to convert a string delimited by a comma to a slice of strings
func StringToSlice(s string) []string {
	return strings.Split(s, ",")
}

// Function to check if a string is in a slice of strings
func Contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// SlicesAreEqual checks if two slices are equal.
func SlicesAreEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Strings(a)
	sort.Strings(b)
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

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

func VerifyToken(tokenString string, userObjectId string) (bool, error) {

	token, err := ParseToken(tokenString)
	if err != nil {
		return false, err
	}

	// Get the claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, errors.New("invalid claims")
	}

	// check the oid
	oid, ok := claims["oid"].(string)
	if !ok {
		return false, errors.New("not able to get oid from claims")
	}
	if oid != userObjectId {
		return false, errors.New("unexpected oid, expected " + userObjectId + " but got " + oid)
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

// Return today's date in the format yyyy-mm-dd as string
func GetTodaysDateString() string {
	return time.Now().Format("2006-01-02")
}

// Return today's date and time in the format yyyy-mm-dd hh:mm:ss as string
func GetTodaysDateTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// Return today's date and time in ISO format as string
func GetTodaysDateTimeISOString() string {
	return time.Now().Format(time.RFC3339)
}

func UserAlias(userPrincipalName string) string {
	return strings.Split(userPrincipalName, "@")[0]
}
