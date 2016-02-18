package msg2api

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"time"
)

const pingTimeout = 5 * time.Minute

var badFrameType = errors.New("bad frame type")

type socketWrapper struct {
	socket *websocket.Conn

	sendQ    chan []byte
	sendErrQ chan error
}

func (w *socketWrapper) runQueues() {
	ping := time.NewTicker(pingTimeout / 3)
	defer ping.Stop()

	for {
		select {
		case msg, open := <-w.sendQ:
			if !open {
				return
			}

			w.sendErrQ <- w.socket.WriteMessage(websocket.TextMessage, msg)

		case <-ping.C:
			if err := w.socket.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (w *socketWrapper) receiveFrame() ([]byte, error) {
	msgType, msg, err := w.socket.ReadMessage()
	switch {
	case msgType == websocket.CloseMessage:
	case err == io.EOF:
		return nil, io.EOF
	case msgType == websocket.TextMessage:
		return msg, err
	}
	w.Close(websocket.CloseUnsupportedData, "")
	return nil, badFrameType
}

func (w *socketWrapper) Receive() (string, error) {
	msg, err := w.receiveFrame()
	if err != nil {
		return "", err
	}
	return string(msg), nil
}

func (w *socketWrapper) ReceiveJSON(value interface{}) error {
	msg, err := w.receiveFrame()
	if err != nil {
		return err
	}
	return json.Unmarshal(msg, value)
}

func (w *socketWrapper) Write(msg string) error {
	w.sendQ <- []byte(msg)
	return <-w.sendErrQ
}

func (w *socketWrapper) WriteJSON(value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	w.sendQ <- data
	return <-w.sendErrQ
}

func (w *socketWrapper) Close(reason int, msg string) {
	if w.sendQ != nil {
		close(w.sendQ)
	}
	w.sendQ = nil
	w.socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(reason, msg))
	w.socket.Close()
}

func wrapWebsocket(socket *websocket.Conn) *socketWrapper {
	result := &socketWrapper{socket, make(chan []byte), make(chan error)}

	socket.SetReadDeadline(time.Now().Add(pingTimeout))
	socket.SetPongHandler(func(string) error {
		socket.SetReadDeadline(time.Now().Add(pingTimeout))
		return nil
	})

	go result.runQueues()
	return result
}
