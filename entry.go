// Copyright 2016 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"bufio"
	"encoding/json"
	"net/http"

	"github.com/openprovider/spawn/auth"
	"github.com/takama/router"
)

type entryBundle struct {
	auth.Auth
}

// login and gets access through given token
func (entry *entryBundle) login(c *router.Control) {

	// Try to get username and password from params
	var username, password, info string

	if c.Request.Header.Get("Content-type") == "application/json" {
		params := make(map[string]string)
		if err := json.NewDecoder(bufio.NewReader(c.Request.Body)).Decode(&params); err == nil {
			if u, ok := params["username"]; ok && len(u) > 0 {
				username = u
			}
			if p, ok := params["password"]; ok && len(p) > 0 {
				password = p
			}
		}
	} else {
		if err := c.Request.ParseForm(); err == nil {
			username = c.Request.Form.Get("username")
			password = c.Request.Form.Get("password")
		}
	}
	if len(username) > 0 && len(password) > 0 {
		token, err := entry.Login(username, password)
		if err == nil {
			result := data{
				"success": true,
				"token":   token,
			}
			c.Code(http.StatusOK).Body(result)
			return
		} else {
			info = err.Error()
		}
	} else {
		info = "Username/Password is required"
	}
	result := data{
		"success": false,
		"error":   http.StatusUnauthorized,
		"message": "Not authorized",
		"info":    info,
	}
	c.Code(http.StatusUnauthorized).Body(result)
}

// info gets user info by token
func (entry *entryBundle) info(c *router.Control) {
	// Try to decode token
	token, ok := decodeString(":token", c)
	if !ok {
		return
	}
	info := entry.Info(token)
	if info != nil {
		result := data{
			"success": true,
			"info":    info,
		}
		c.Code(http.StatusOK).Body(result)
		return
	}
	result := data{
		"success": false,
		"error":   http.StatusUnauthorized,
		"message": "Not authorized",
		"info":    "Token is not valid",
	}
	c.Code(http.StatusUnauthorized).Body(result)
}

// logout user by the token
func (entry *entryBundle) logout(c *router.Control) {
	// Try to decode token
	token, ok := decodeString(":token", c)
	if !ok {
		return
	}
	err := entry.Logout(token)
	if err == nil {
		result := data{
			"success": true,
		}
		c.Code(http.StatusOK).Body(result)
		return
	}
	result := data{
		"success": false,
		"error":   http.StatusUnauthorized,
		"message": "Not authorized",
		"info":    err.Error(),
	}
	c.Code(http.StatusUnauthorized).Body(result)
}
