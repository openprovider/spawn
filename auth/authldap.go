package auth

import (
	"crypto/tls"
	"fmt"
	"time"

	"gopkg.in/ldap.v2"
)

// AuthLDAP contains LDAP connection parameters
type AuthLDAP struct {
	conn    *ldap.Conn
	config  *AuthConfig
	session map[string]*AuthInfo
}

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
	if al.conn == nil {
		var err error
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
	}
	stdlog.Println("LDAP Connection has opened")
	return nil
}

// Login create secure connection by username & password
func (al *AuthLDAP) Login(username, password string) (token string, err error) {
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
		al.conn.Close()
		al.conn = nil
		return
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
	al.session[token] = ai
	time.AfterFunc(al.config.ExpirationTime*time.Minute, func() {
		al.Logout(token)
	})

	stdlog.Println("user", ai.UID, "has logged in")

	return
}

// Logout resets current authentication
func (al *AuthLDAP) Logout(token string) error {
	if ai, exists := al.session[token]; exists {
		delete(al.session, token)
		stdlog.Println("user", ai.UID, "has logged out")
		return nil
	}
	return ErrNotLogged
}

// Close disconects from auth server and logout all users
func (al *AuthLDAP) Close() {
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
