// Copyright 2015 Igor Dolzhikov. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"io"
	"net/http"
)

// RequestHandler type is method which handle all requests
type RequestHandler func(request *http.Request) *http.Response

// proxy contains request handler method to manage http requests/responses
type proxy struct {
	handler RequestHandler
}

// ServeHTTP implements http.Handler interface.
func (p *proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	response := p.handler(req)
	if response == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(response.StatusCode)
	io.Copy(w, response.Body)
}
