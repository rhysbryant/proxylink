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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ResponseWriter struct {
	conn   io.ReadWriter
	header http.Header
	status int
}

func NewResponseWriter(conn io.ReadWriter) *ResponseWriter {
	return &ResponseWriter{conn: conn}
}

// Implement the Header method
func (w *ResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

// Implement the Write method
func (w *ResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.conn.Write(data)
}

// Implement the WriteHeader method
func (w *ResponseWriter) WriteHeader(statusCode int) {
	if w.status != 0 {
		return
	}
	w.status = statusCode
	response := bytes.Buffer{}
	fmt.Fprintf(&response, "HTTP/1.1 %d %s\r\n", statusCode, http.StatusText(statusCode))
	for key, values := range w.header {

		fmt.Fprintf(&response, "%s: %s\r\n", key, strings.Join(values, ","))
	}
	response.WriteString("\r\n")

	w.conn.Write(response.Bytes())
}
