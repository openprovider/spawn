// Copyright 2015 Igor Dolzhikov. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"

	"github.com/takama/router"
)

// data is a shortcut
type data map[string]interface{}

func isAlphaNumeric(str string) bool {
	isAlphaNum := true
	for _, b := range str {
		if !('0' <= b && b <= '9' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || b == '_' || b == '-' || b == '.') {
			isAlphaNum = false
			break
		}
	}

	return isAlphaNum
}

func decodeRecord(record interface{}, c *router.Control) (*[]byte, bool) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.Code(http.StatusBadRequest).Body(
			data{
				"success": false,
				"error":   http.StatusBadRequest,
				"message": "The body content is absent",
				"info":    err.Error(),
			})
		return &body, false
	}
	if err := json.Unmarshal(body, &record); err != nil {
		c.Code(http.StatusBadRequest).Body(
			data{
				"success": false,
				"error":   http.StatusBadRequest,
				"message": "Could not recognize parameters",
				"info":    err.Error(),
			})
		return &body, false
	}

	return &body, true
}

func decodeRecords(body *[]byte, records interface{}, c *router.Control) bool {
	if err := json.Unmarshal(*body, &records); err != nil {
		c.Code(http.StatusBadRequest).Body(
			data{
				"success": false,
				"error":   http.StatusBadRequest,
				"message": "Could not recognize parameters",
				"info":    err.Error(),
			})
		return false
	}

	return true
}

func decodeNumber(name string, c *router.Control) (uint64, bool) {
	number, err := strconv.ParseUint(c.Get(name), 10, 64)
	if err != nil {
		notRecognizedParameterError(name, err, c)
		return 0, false
	}
	if number == 0 {
		couldNotBeZero(name, c)
		return 0, false
	}

	return number, true
}

func decodeString(name string, c *router.Control) (string, bool) {
	str := c.Get(name)

	if !checkAlphaNumeric(str, c) {
		return "", false
	}
	if str == "" {
		couldNotBeEmpty(str, c)
		return "", false
	}

	return str, true
}

func checkAlphaNumeric(str string, c *router.Control) bool {
	if !isAlphaNumeric(str) {
		err := errors.New(str + " parameter is not alpha-numeric")
		notRecognizedParameterError(str, err, c)
		return false
	}

	return true
}

func couldNotBeZero(param string, c *router.Control) {
	c.Code(http.StatusBadRequest).Body(
		data{
			"success": false,
			"error":   http.StatusBadRequest,
			"message": "The parameter '" + param + "' could not be zero value",
			"info":    "Please apply a non-zero value to the data",
		})
}

func couldNotBeEmpty(param string, c *router.Control) {
	c.Code(http.StatusBadRequest).Body(
		data{
			"success": false,
			"error":   http.StatusBadRequest,
			"message": "The parameter '" + param + "' could not be empty",
			"info":    "Please apply a non-empty value to the data",
		})
}

func notRecognizedParameterError(param string, err error, c *router.Control) {
	c.Code(http.StatusBadRequest).Body(
		data{
			"success": false,
			"error":   http.StatusBadRequest,
			"message": "Could not recognize " + param + " parameter",
			"info":    err.Error(),
		})
}

func recordNotFound(c *router.Control) {
	c.Code(http.StatusNotFound).Body(
		data{
			"success": false,
			"error":   http.StatusNotFound,
			"message": "Record(s) not found",
			"info":    "Please add a record(s) before using",
		})
}

func notFound(c *router.Control) {
	c.Code(http.StatusNotFound).Body(
		data{
			"Message": data{
				"Error":       "Method not found",
				"Information": "Please see list of the methods by using /list",
			},
		})
}

func infoHandler(c *router.Control) {
	host, _ := os.Hostname()
	m := new(runtime.MemStats)
	runtime.ReadMemStats(m)
	c.Code(http.StatusOK).Body(
		data{
			"Spawn Sync Service": data{
				"Host": host,
				"Runtime": data{
					"Compiler": runtime.Version(),
					"CPU":      runtime.NumCPU(),
					"Memory":   fmt.Sprintf("%.2fMB", float64(m.Alloc)/(1<<(10*2))),
					"Threads":  runtime.NumGoroutine(),
				},
				"Release": data{
					"Number": VERSION,
					"Date":   DATE,
				},
			},
		})
}

func logger(c *router.Control) {
	stdlog.Println(c.Request.RemoteAddr, c.Request.Method, c.Request.URL.Path)
}

func baseHandler(handle router.Handle) router.Handle {
	return func(c *router.Control) {
		if c.Get("pretty") != "true" {
			c.CompactJSON(true)
		}
		if origin := c.Request.Header.Get("Origin"); origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		handle(c)
	}
}

func panicHandler(c *router.Control) {
	c.Code(http.StatusInternalServerError).Body(c.Request)
}
