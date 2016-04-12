package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// AuthType is type of authentication
type AuthType string

// List of aith methods
const (
	LDAP AuthType = "LDAP"
)

var (
	ErrAuthNotImplemented     = errors.New("Auth module is not implemented")
	ErrUserDoesNotExist       = errors.New("User does not exist")
	ErrNotLogged              = errors.New("User has not logged in")
	ErrTooManyEntriesReturned = errors.New("Too many entries returned")
)

// AuthInfo contains authentication information
type AuthInfo struct {
	UID  string
	Name struct {
		First string
		Last  string
	}
	Email  string
	Groups []string
}

// Auth is a interface which contains basic authentication methods
type Auth interface {
	Login(username, password string) (token string, err error)
	Logout(token string) error
	Info(token string) *AuthInfo
	Close()
}

// AuthConfig contains params to create new secure connection
type AuthConfig struct {
	Type AuthType `json:"type"`

	ExpirationTime time.Duration `json:"expiration"`

	Host string `json:"host"`
	Port int    `json:"port"`

	Settings struct {
		Base    string `json:"base"`
		UseSSL  bool   `json:"ssl"`
		Filters struct {
			User  string `json:"user"`
			Group string `json:"group"`
		} `json:"filters"`
		Attributes []string `json:"attributes"`
	} `json:"settings"`
}

// simplest logger, which initialized during starts of the application
var (
	stdlog = log.New(os.Stdout, "[AUTH]: ", log.LstdFlags)
	errlog = log.New(os.Stderr, "[AUTH:ERROR]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// NewAuth creates new type of authentication
func NewAuth(config *AuthConfig) (Auth, error) {
	switch config.Type {
	case LDAP:
		return NewAuthLDAP(config)
	default:
		stdlog.Println("Warning: authentication is not used")
		return NewAuthGuest(config)
	}
}

func GenerateSecureKey() string {
	k := make([]byte, 32)
	io.ReadFull(rand.Reader, k)
	return fmt.Sprintf("%x", k)
}
