package auth

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"gopkg.in/ldap.v2"
)

// AuthLDAP contains LDAP connection parameters
type AuthLDAP struct {
	mutex   sync.RWMutex
	conn    *ldap.Conn
	config  *AuthConfig
	session map[string]*AuthInfo
}

var DefaultExpiration = 60 * time.Minute

// NewAuthLDAP creates new LDAP connection
func NewAuthLDAP(config *AuthConfig) (*AuthLDAP, error) {
	al := &AuthLDAP{
		config:  config,
		session: make(map[string]*AuthInfo),
	}
	al.session["guest"] = &AuthInfo{
		UID: "guest",
	}
	return al, nil
}

// connect to LDAP server
func (al *AuthLDAP) connect() error {
	al.mutex.Lock()
	defer al.mutex.Unlock()
	if al.conn == nil {
		var err error
		ldap.DefaultTimeout = 15 * time.Second
		link := fmt.Sprintf("%s:%d", al.config.Host, al.config.Port)
		if al.config.Settings.UseSSL {
			al.conn, err = ldap.DialTLS("tcp", link, &tls.Config{InsecureSkipVerify: false})
			if err != nil {
				return err
			}
		} else {
			if al.conn, err = ldap.Dial("tcp", link); err != nil {
				return err
			}

			// Reconnect with TLS
			if err = al.conn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
				return err
			}
		}
		stdlog.Println("LDAP Connection has opened")
		// set expiration for LDAP connection
		time.AfterFunc(DefaultExpiration, func() {
			if al.conn != nil {
				al.conn.Close()
				al.conn = nil
				stdlog.Println("Closing of LDAP connection due to time expiration")
			}
		})
	}
	return nil
}

// Login create secure connection by username & password
func (al *AuthLDAP) Login(username, password string) (token string, err error) {
	defer func() {
		if recovery := recover(); recovery != nil {
			errlog.Println("Method 'Login' has been recovered:", recovery)
			err = fmt.Errorf("%s", recovery)
		}
	}()
	if err = al.connect(); err != nil {
		errlog.Println("Could not connect to LDAP server:", err)
		return
	}
	request := ldap.NewSearchRequest(
		al.config.Settings.Base,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 10, false,
		fmt.Sprintf(al.config.Settings.Filters.User, username),
		al.config.Settings.Attributes,
		nil,
	)
	result, err := al.conn.Search(request)
	if err != nil {
		al.mutex.Lock()
		al.conn.Close()
		al.conn = nil
		al.mutex.Unlock()
		stdlog.Println("LDAP Connection has been closed:", err)
		if err = al.connect(); err != nil {
			errlog.Println("Could not connect to LDAP server:", err)
			return
		}
		result, err = al.conn.Search(request)
		if err != nil {
			errlog.Println("Could not connect to LDAP server:", err)
			return
		}
	}
	if len(result.Entries) < 1 {
		errlog.Printf("Attempt to login as %s/%s", username, password)
		return "", ErrUserDoesNotExist
	}
	if len(result.Entries) > 1 {
		return "", ErrTooManyEntriesReturned
	}
	var v string
	var ai = &AuthInfo{
		UID: "guest",
	}
	if err = al.conn.Bind(result.Entries[0].DN, password); err != nil {
		return
	}
	for _, attr := range al.config.Settings.Attributes {
		v = result.Entries[0].GetAttributeValue(attr)
		switch attr {
		case "uid":
			ai.UID = v
		case "givenName":
			ai.Name.First = v
		case "sn":
			ai.Name.Last = v
		case "mail":
			ai.Email = v
		}
	}
	groupRequest := ldap.NewSearchRequest(
		al.config.Settings.Base,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 10, false,
		fmt.Sprintf(al.config.Settings.Filters.Group, username),
		[]string{"cn"},
		nil,
	)
	if result, err := al.conn.Search(groupRequest); err == nil {
		for _, entry := range result.Entries {
			ai.Groups = append(ai.Groups, entry.GetAttributeValue("cn"))
		}
	}
	token = GenerateSecureKey()
	al.mutex.Lock()
	al.session[token] = ai
	al.mutex.Unlock()
	time.AfterFunc(al.config.ExpirationTime*time.Minute, func() {
		al.Logout(token)
	})

	stdlog.Println("user", ai.UID, "has logged in")

	return
}

// Logout resets current authentication
func (al *AuthLDAP) Logout(token string) error {
	al.mutex.Lock()
	defer al.mutex.Unlock()
	if ai, exists := al.session[token]; exists {
		delete(al.session, token)
		stdlog.Println("user", ai.UID, "has logged out")
		return nil
	}
	return ErrNotLogged
}

// Close disconects from auth server and logout all users
func (al *AuthLDAP) Close() {
	al.mutex.Lock()
	defer func() {
		al.mutex.Unlock()
		if recovery := recover(); recovery != nil {
			errlog.Println("Method 'Close' has been recovered:", recovery)
		}
	}()
	for key := range al.session {
		delete(al.session, key)
	}
	if al.conn != nil {
		al.conn.Close()
		al.conn = nil
	}
	stdlog.Println("LDAP Connection has been closed")
}

// Info contains user detailed information
func (al *AuthLDAP) Info(token string) *AuthInfo {
	if info, exists := al.session[token]; exists {
		return info
	}
	return nil
}
