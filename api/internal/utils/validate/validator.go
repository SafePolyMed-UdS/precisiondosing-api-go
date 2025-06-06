package validate

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

const (
	MinPasswordLength = int(8)
	MaxPasswordLength = int(64)

	MinNameLength = int(2)
	MaxNameLength = int(255)

	ServerTimeSkew = 5 * time.Minute
)

func Email(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return errors.New("invalid email address")
	}

	return nil
}

func Password(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", MinPasswordLength)
	}

	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password must be at most %d characters long", MaxPasswordLength)
	}

	return nil
}

func Name(name string) error {
	// Check for leading and trailing whitespace
	if strings.TrimSpace(name) != name {
		return errors.New("name cannot have leading or trailing whitespace")
	}

	// Check the length of the name
	nameLength := len(name)
	if nameLength < MinNameLength || nameLength > MaxNameLength {
		return fmt.Errorf("name must be between %d and %d characters long", MinNameLength, MaxNameLength)
	}

	// Check for valid characters
	for _, char := range name {
		if !unicode.IsLetter(char) && char != ' ' && char != '-' && char != '\'' {
			return errors.New("name can only contain alphabetic characters, spaces, hyphens, or apostrophes")
		}
	}

	return nil
}

func Organization(orgName string) error {
	// Check for leading and trailing whitespace
	if strings.TrimSpace(orgName) != orgName {
		return errors.New("organization name cannot have leading or trailing whitespace")
	}

	// Check the length of the organization name
	nameLength := len(orgName)
	if nameLength < MinNameLength || nameLength > MaxNameLength {
		return fmt.Errorf("organization name must be between %d and %d characters long", MinNameLength, MaxNameLength)
	}

	// Check for valid characters
	validChars := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsNumber(r) ||
			r == ' ' || r == '-' || r == '.' || r == '&' || r == ',' || r == '+'
	}

	for _, char := range orgName {
		if !validChars(char) {
			return errors.New("organization name contains invalid characters")
		}
	}

	return nil
}

func Access(requiredRole, userRole string) error {
	return CanSwitchToRole(requiredRole, userRole)
}

func CanSwitchToRole(requestRole string, dbRole string) error {
	var roleMap = map[string]int{
		"admin": 3,
		"debug": 2,
		"user":  1,
	}

	reqRoleValue := roleMap[requestRole]
	dbRoleValue := roleMap[dbRole]
	if reqRoleValue == 0 || dbRoleValue == 0 {
		return errors.New("invalid role")
	}

	if dbRoleValue < reqRoleValue {
		return errors.New("user role not sufficient")
	}

	return nil
}

func TokenExpiry(timeTokenExires time.Time) error {
	if time.Now().Add(ServerTimeSkew).After(timeTokenExires) {
		return errors.New("token expired")
	}

	return nil
}

func QueryRetry(lastTry time.Time, waitTime time.Duration) error {
	now := time.Now().Add(-ServerTimeSkew)
	minRetryTime := lastTry.Add(waitTime)
	retryAllowed := now.After(minRetryTime)

	if !retryAllowed {
		return errors.New("retry not allowed yet")
	}

	return nil
}
