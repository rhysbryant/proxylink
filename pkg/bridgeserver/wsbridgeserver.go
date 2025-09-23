package bridgeserver

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
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rhysbryant/proxylink/pkg/httputils"
	"github.com/rhysbryant/proxylink/pkg/proxy"
	"github.com/rhysbryant/proxylink/pkg/wswrapper"
)

type BridgeServer struct {
	key []byte
}

func NewBridgeServer(key []byte) *BridgeServer {
	return &BridgeServer{key: key}
}

func (bs *BridgeServer) isAllowed(remoteAddr string) bool {
	//only allow external connections if a key is set
	if bs.key != nil {
		return true
	}

	ipAddr, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}

	parsedIPAddr := net.ParseIP(ipAddr)
	return parsedIPAddr.IsPrivate()
}

func (bs *BridgeServer) ProcessRequest(r *http.Request, w http.ResponseWriter) error {

	if !bs.isAllowed(r.RemoteAddr) {
		http.Error(w, "not allowed", http.StatusForbidden)
		return fmt.Errorf("connection from %s not allowed", r.RemoteAddr)
	}

	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	var rw io.ReadWriteCloser
	if bs.key != nil {
		rw = wswrapper.NewWSConnWithEncryption(conn, [32]byte(bs.key), false)
	} else {
		rw = wswrapper.NewWSConn(conn)
	}
	defer rw.Close()

	// first will come a http request from the client
	proxiedRequest, err := http.ReadRequest(bufio.NewReader(rw))
	if err != nil {
		return fmt.Errorf("failed to read request from websocket: %w", err)
	}

	logEntryContext := slog.With("target", proxiedRequest.URL.String(),
		"method", proxiedRequest.Method, "from", r.RemoteAddr)

	logEntryContext.Info("Processing tunneled request")

	if proxiedRequest.Method == http.MethodConnect {
		return proxy.NewDirectHTTPProxy().ProcessTunnelRequest(proxiedRequest, rw)
	}

	return proxy.NewDirectHTTPProxy().ProcessRequest(proxiedRequest, httputils.NewResponseWriter(rw))
}
