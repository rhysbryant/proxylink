package proxy

/*
 Copyright (c) 2025 Rhys Bryant

 This program is free software: you can redistribute it and/or modify
 it under the terms of the GNU General Public License as published by
 the Free Software Foundation, either version 3 of the License, or
 (at your option) any later version.

 This program is distributed in the hope that it will be useful,
 but WITHOUT ANY WARRANTY; without even the implied warranty of
 MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 GNU General Public License for more details.

 You should have received a copy of the GNU General Public License
 along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/rhysbryant/proxylink/pkg/httputils"
	"github.com/rhysbryant/proxylink/pkg/ioutils"
)

type DirectHTTPProxy struct {
}

func NewDirectHTTPProxy() *DirectHTTPProxy {
	return &DirectHTTPProxy{}

}

func (d *DirectHTTPProxy) writeHTTPResponse(w io.Writer, status int, message string) error {
	response := http.Response{}
	response.StatusCode = status
	response.Status = message
	response.ProtoMajor = 1
	response.ProtoMinor = 1
	response.ContentLength = -1
	if err := response.Write(w); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

func (d *DirectHTTPProxy) ProcessRequest(r *http.Request, w http.ResponseWriter) error {
	if r.Method == http.MethodConnect {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return fmt.Errorf("hijacking not supported")
		}

		clientConn, _, err := hijacker.Hijack()
		if err != nil {
			return fmt.Errorf("failed to hijack connection: %w", err)
		}
		defer clientConn.Close()
		return d.ProcessTunnelRequest(r, clientConn)
	} else {
		return d.ProcessPlainTextRequest(r, w)
	}

}

func (d *DirectHTTPProxy) ProcessTunnelRequest(r *http.Request, clientConn io.ReadWriteCloser) error {
	// Extract the host from the request
	host := httputils.GetTLSHostFromRequest(r)

	// Establish a connection to the target server
	destConn, err := net.Dial("tcp", host)
	if err != nil {
		d.writeHTTPResponse(clientConn, http.StatusGatewayTimeout, "Service Unavailable")
		return fmt.Errorf("failed to connect to target: %w", err)
	}
	defer destConn.Close()

	if err := d.writeHTTPResponse(clientConn, 200, "Connection Established"); err != nil {
		return fmt.Errorf("failed to write connection established response: %w", err)
	}

	return ioutils.ByoDirectionalCopy(destConn, clientConn)
}

// Extract the host from the request
func (DirectHTTPProxy) ProcessPlainTextRequest(r *http.Request, w http.ResponseWriter) error {

	// Create the request to the target URL
	targetURL := r.URL.String()
	if !strings.HasPrefix(targetURL, "http") {
		targetURL = "http://" + r.Host + r.URL.Path
	}

	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httputils.CopyRequest(r, req)

	// needs to not follow redirects, as we want to return the redirect to the client
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: nil, // disable storing cookies
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach the destination server", http.StatusBadGateway)
		return fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()
	httputils.CopyResponse(resp, w)

	// Copy the response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body: %w", err)
	}

	return err
}
