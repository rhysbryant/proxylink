package httputils

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
	"net/http"
	"strings"
)

// CopyRequest copies the header of an HTTP request from src to dst.
func CopyRequest(src *http.Request, dst *http.Request) {
	// Copy headers from the original request
	for key, values := range src.Header {
		for _, value := range values {
			dst.Header.Add(key, value)
		}
	}
	dst.Method = src.Method
	//dst.URL = src.URL
	dst.Body = src.Body
	dst.Host = src.Host
}

// CopyResponse copies the header of an HTTP response from src to dst.
func CopyResponse(src *http.Response, dst http.ResponseWriter) {
	// Copy headers from the original response
	for key, values := range src.Header {
		for _, value := range values {
			dst.Header().Add(key, value)
		}
	}
	dst.WriteHeader(src.StatusCode)
}

func GetTLSHostFromRequest(r *http.Request) string {
	host := r.URL.Host
	if !strings.Contains(host, ":") {
		host += ":443" // Default to port 443 for HTTPS
	}
	return host
}
