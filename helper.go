// Copyright 2015 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

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

func decodeRecord(record interface{}, c *router.Control) bool {
	decoder := json.NewDecoder(bufio.NewReader(c.Request.Body))
	decoder.UseNumber()
	if err := decoder.Decode(&record); err != nil {
		c.Code(http.StatusBadRequest).Body(data{
			"success": false,
			"error":   http.StatusBadRequest,
			"message": "Could not recognize parameters",
			"info":    err.Error(),
		})
		errlog.Println(err)
		return false
	}

	return true
}

func preDecodeRecords(records interface{}, c *router.Control) (*bytes.Buffer, bool) {
	buffer := bytes.NewBuffer(make([]byte, 0))
	reader := io.TeeReader(bufio.NewReader(c.Request.Body), buffer)
	defer c.Request.Body.Close()
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	if err := decoder.Decode(&records); err != nil {
		c.Code(http.StatusBadRequest).Body(data{
			"success": false,
			"error":   http.StatusBadRequest,
			"message": "Could not recognize parameters",
			"info":    err.Error(),
		})
		errlog.Println(err)
		return buffer, false
	}
	return buffer, true
}

func postDecodeRecords(buffer *bytes.Buffer, records interface{}, c *router.Control) bool {
	decoder := json.NewDecoder(buffer)
	decoder.UseNumber()
	if err := decoder.Decode(&records); err != nil {
		c.Code(http.StatusBadRequest).Body(data{
			"success": false,
			"error":   http.StatusBadRequest,
			"message": "Could not recognize parameters",
			"info":    err.Error(),
		})
		errlog.Println(err)
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
	message := "The parameter '" + param + "' could not be zero value"
	c.Code(http.StatusBadRequest).Body(data{
		"success": false,
		"error":   http.StatusBadRequest,
		"message": message,
		"info":    "Please apply a non-zero value to the data",
	})
	errlog.Println(message)
}

func couldNotBeEmpty(param string, c *router.Control) {
	message := "The parameter '" + param + "' could not be empty"
	c.Code(http.StatusBadRequest).Body(data{
		"success": false,
		"error":   http.StatusBadRequest,
		"message": message,
		"info":    "Please apply a non-empty value to the data",
	})
	errlog.Println(message)
}

func notRecognizedParameterError(param string, err error, c *router.Control) {
	message := "Could not recognize " + strings.Trim(param, " ") + " parameter"
	c.Code(http.StatusBadRequest).Body(data{
		"success": false,
		"error":   http.StatusBadRequest,
		"message": message,
		"info":    err.Error(),
	})
	errlog.Println(message, err.Error())
}

func recordNotFound(c *router.Control) {
	message := "Record(s) not found"
	c.Code(http.StatusNotFound).Body(data{
		"success": false,
		"error":   http.StatusNotFound,
		"message": message,
		"info":    "Please add a record(s) before using",
	})
	errlog.Println(message)
}

func notFound(c *router.Control) {
	message := "Method not found for " + c.Request.URL.Path
	c.Code(http.StatusNotFound).Body(data{
		"Message": data{
			"Error":       message,
			"Information": "Please see list of the methods by using /list",
		},
	})
	errlog.Println(message)
}

func infoHandler(c *router.Control) {
	host, _ := os.Hostname()
	m := new(runtime.MemStats)
	runtime.ReadMemStats(m)
	c.Code(http.StatusOK).Body(data{
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
	remoteAddr := c.Request.Header.Get("X-Forwarded-For")
	if remoteAddr == "" {
		remoteAddr = c.Request.RemoteAddr
	}
	stdlog.Println(remoteAddr, c.Request.Method, c.Request.URL.Path)
}

func optionsHandler(c *router.Control) {
	c.Code(http.StatusOK)
}
