// Copyright 2015 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"io"
	"net/http"
)

// proxy contains request handler function which manage http requests/responses
type proxy struct {
	transport http.RoundTripper
}

// ServeHTTP implements http.Handler interface.
func (p *proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	response, err := p.transport.RoundTrip(req)
	if err != nil {
		errlog.Println(err)
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
