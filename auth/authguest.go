package auth

// AuthGuest contains guest parameters
type AuthGuest struct {
	session map[string]*AuthInfo
}

// NewAuthGuest creates new guest connection
func NewAuthGuest(config *AuthConfig) (*AuthGuest, error) {
	ag := new(AuthGuest)
	ag.session = make(map[string]*AuthInfo)
	return ag, nil
}

// Login create secure connection by username & password
func (ag *AuthGuest) Login(username, password string) (token string, err error) {
	token = GenerateSecureKey()
	if _, exists := ag.session[token]; !exists {
		ag.session[token] = &AuthInfo{
			UID: username,
		}
	}
	return
}

// Logout resets current authentication
func (ag *AuthGuest) Logout(token string) error {
	if _, exists := ag.session[token]; exists {
		delete(ag.session, token)
		return nil
	}
	return ErrNotLogged
}

// Close disconects from auth server and logout all users
func (ag *AuthGuest) Close() {
	for key := range ag.session {
		delete(ag.session, key)
	}
	return
}

// Info contains user detailed information
func (ag *AuthGuest) Info(token string) *AuthInfo {
	if info, exists := ag.session[token]; exists {
		return info
	}
	return nil
}
