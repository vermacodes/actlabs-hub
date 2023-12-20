package helper

import (
	"actlabs-hub/internal/entity"
	"crypto/rand"
	"sort"
	"strings"
	"time"
	"unsafe"
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

// ConvertProfileToRecord converts a Profile to a ProfileRecord.
func ConvertProfileToRecord(profile entity.Profile) entity.ProfileRecord {
	return entity.ProfileRecord{
		PartitionKey:  "actlabs",             // this is a static value.
		RowKey:        profile.UserPrincipal, // UserPrincipal is the unique identifier for the user.
		ObjectId:      profile.ObjectId,
		UserPrincipal: profile.UserPrincipal,
		DisplayName:   profile.DisplayName,
		ProfilePhoto:  profile.ProfilePhoto,
		Roles:         strings.Join(profile.Roles, ","),
	}
}

// ConvertRecordToProfile converts a ProfileRecord to a Profile.
func ConvertRecordToProfile(record entity.ProfileRecord) entity.Profile {
	return entity.Profile{
		ObjectId:      record.ObjectId,
		UserPrincipal: record.UserPrincipal,
		DisplayName:   record.DisplayName,
		ProfilePhoto:  record.ProfilePhoto,
		Roles:         strings.Split(record.Roles, ","),
	}
}
