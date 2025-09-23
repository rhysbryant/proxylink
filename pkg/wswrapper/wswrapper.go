package wswrapper

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
	"time"

	"github.com/gorilla/websocket"
	stream "github.com/nknorg/encrypted-stream"
)

// WSConn is a wrapper around websocket.Conn to implement io.ReadWriteCloser
// this abstracts away the websocket message framing so the connection can be used as a Stream implementation of io.ReadWriteCloser
type WSConn struct {
	*websocket.Conn
	buf bytes.Buffer
}

func NewWSConn(conn *websocket.Conn) *WSConn {
	return &WSConn{Conn: conn}
}

// this wrapper adds encryption to the websocket messages using nknorg/encrypted-stream
func NewWSConnWithEncryption(conn *websocket.Conn, key [32]byte, initiator bool) io.ReadWriteCloser {
	c := NewWSConn(conn)
	encryptedConn, err := stream.NewEncryptedStream(c, &stream.Config{
		Cipher:          stream.NewXSalsa20Poly1305Cipher(&key),
		SequentialNonce: false,     // only when key is unique for every stream
		Initiator:       initiator, // only on the dialer side
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create encrypted stream: %v", err))
	}

	return encryptedConn
}

func (ws *WSConn) Write(data []byte) (int, error) {
	err := ws.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}
func (ws *WSConn) isGracefulClose(err error) bool {
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return true
	}
	if err == io.EOF {
		return true
	}
	return false
}

func (ws *WSConn) Read(data []byte) (int, error) {

	readLen := len(data)
	offset := 0

	if ws.buf.Len() > 0 {

		n, err := ws.buf.Read(data)
		if err != nil && err != io.EOF {
			return 0, err
		}
		offset += n
	}

	remaining := readLen - offset

	if remaining <= 0 {
		return offset, nil
	}

	_, payload, err := ws.ReadMessage()

	if err != nil {
		if ws.isGracefulClose(err) {
			return offset, io.EOF
		}
		return offset, fmt.Errorf("error reading from websocket: %w", err)
	}

	ws.buf.Write(payload)

	n, err := ws.buf.Read(data[offset:])
	if err != nil {
		return offset, fmt.Errorf("error reading from buffer after websocket read: %w", err)

	}
	offset += n

	return offset, nil

}

func (ws *WSConn) Close() error {
	if err := ws.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)); err != nil {
		return err
	}

	return ws.Conn.Close()
}
