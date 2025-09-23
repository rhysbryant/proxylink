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

/**

* A proxy that bridges HTTP requests over a WebSocket connection to another proxy server.

raw HHTTP 1.1 requests are sent over the WebSocket connection to the next proxy server, which processes them and sends back the responses.

*/
import (
	"bufio"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rhysbryant/proxylink/pkg/httputils"
	"github.com/rhysbryant/proxylink/pkg/ioutils"
	"github.com/rhysbryant/proxylink/pkg/wswrapper"
)

type WSBridgeProxyClient struct {
	nextProxyServer string
	key             []byte
}

func NewWSBridgeProxyClient(nextProxyAddress string, key []byte) *WSBridgeProxyClient {
	return &WSBridgeProxyClient{nextProxyServer: nextProxyAddress, key: key}
}

func (b *WSBridgeProxyClient) ProcessRequest(r *http.Request, w http.ResponseWriter) error {

	nextProxyConn, resp, err := websocket.DefaultDialer.Dial(b.nextProxyServer, nil)
	if err != nil {
		if err == websocket.ErrBadHandshake {
			if resp.StatusCode == http.StatusProxyAuthRequired {
				http.Error(w, resp.Status, http.StatusProxyAuthRequired)
				return fmt.Errorf("proxy authentication required")
			}
		}
		http.Error(w, "failed to connect to next proxy", http.StatusBadGateway)
		return fmt.Errorf("failed to connect to websocket proxy: %w", err)
	}

	var destConn io.ReadWriteCloser
	if b.key != nil {

		destConn = wswrapper.NewWSConnWithEncryption(nextProxyConn, [32]byte(b.key), true)
	} else {
		destConn = wswrapper.NewWSConn(nextProxyConn)
	}

	defer destConn.Close()

	//write the original http request to the websocket connection
	if err := r.Write(destConn); err != nil {
		return fmt.Errorf("failed to write request to websocket proxy: %w", err)
	}

	//read back the response from the websocket connection
	wsProxiedResponse, err := http.ReadResponse(bufio.NewReader(destConn), r)
	if err != nil {
		http.Error(w, "invalid response from next proxy", http.StatusBadGateway)
		return fmt.Errorf("failed to read response from websocket proxy: %w", err)
	}

	// write the proxied response back to the client
	httputils.CopyResponse(wsProxiedResponse, w)

	// if this is a tunnel request, we need to hijack the connection to allow for raw TLS traffic
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

		if err := ioutils.ByoDirectionalCopy(destConn, clientConn); err != nil {
			return fmt.Errorf("failed to copy data between client and websocket proxy: %w", err)
		}
	} else {
		// Copy the response body
		_, err = io.Copy(w, wsProxiedResponse.Body)
		if err != nil {
			return fmt.Errorf("failed to copy response body from websocket proxy: %w", err)
		}
	}

	return nil
}
