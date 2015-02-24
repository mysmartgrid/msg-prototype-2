package ws

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"time"
)

const (
	pingTimeout = 60 * time.Second
)

type Dispatcher struct {
	Socket *websocket.Conn

	sendQ    chan []byte
	sendErrQ chan error
}

func (d *Dispatcher) runQueues() {
	ping := time.NewTicker(pingTimeout / 3)
	defer ping.Stop()

	for {
		select {
		case msg, open := <-d.sendQ:
			if !open {
				return
			}

			d.sendErrQ <- d.Socket.WriteMessage(websocket.TextMessage, msg)

		case <-ping.C:
			if err := d.Socket.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (d *Dispatcher) prepare() {
	if d.sendQ != nil {
		return
	}

	d.sendQ = make(chan []byte)
	d.sendErrQ = make(chan error)

	d.Socket.SetReadDeadline(time.Now().Add(pingTimeout))
	d.Socket.SetPongHandler(func(string) error {
		d.Socket.SetReadDeadline(time.Now().Add(pingTimeout))
		return nil
	})

	go d.runQueues()
}

func (d *Dispatcher) Receive() (int, []byte, error) {
	d.prepare()

	msgType, msg, err := d.Socket.ReadMessage()
	switch {
	case msgType == websocket.CloseMessage:
	case err == io.EOF:
		return -1, nil, io.EOF
	}
	return msgType, msg, err
}

func (d *Dispatcher) ReceiveJSON(value interface{}) error {
	_, msg, err := d.Receive()
	if err != nil {
		return err
	}
	return json.Unmarshal(msg, value)
}

func (d *Dispatcher) Write(msg string) error {
	d.prepare()

	d.sendQ <- []byte(msg)
	return <-d.sendErrQ
}

func (d *Dispatcher) WriteJSON(value interface{}) error {
	d.prepare()

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	d.sendQ <- data
	return <-d.sendErrQ
}

func (d *Dispatcher) CloseWith(reason int, msg string) {
	if d.sendQ != nil {
		close(d.sendQ)
	}
	d.Socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(reason, msg))
	d.Socket.Close()
}

func (d *Dispatcher) Close() {
	d.CloseWith(websocket.CloseNormalClosure, "")
}
