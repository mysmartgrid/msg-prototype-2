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

type ReadHandler func(d *Dispatcher, msgType int, message []byte) error

type Dispatcher struct {
	Socket *websocket.Conn

	OnRead ReadHandler

	sendQ    chan []byte
	sendErrQ chan error
}

func (d *Dispatcher) Run() error {
	d.sendQ = make(chan []byte)
	d.sendErrQ = make(chan error)

	d.Socket.SetReadDeadline(time.Now().Add(pingTimeout))
	d.Socket.SetPongHandler(func(string) error {
		d.Socket.SetReadDeadline(time.Now().Add(pingTimeout))
		return nil
	})

	defer d.Socket.Close()

	go func() {
		ping := time.NewTicker(pingTimeout / 3)
		defer ping.Stop()

		for {
			select {
			case msg, open := <-d.sendQ:
				if !open {
					d.Socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					return
				}

				d.sendErrQ <- d.Socket.WriteMessage(websocket.TextMessage, msg)

			case <-ping.C:
				if err := d.Socket.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					return
				}
			}
		}
	}()

	for {
		msgType, msg, err := d.Socket.ReadMessage()
		switch {
		case msgType == websocket.CloseMessage:
			return nil

		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case d.OnRead != nil:
			if err := d.OnRead(d, msgType, msg); err != nil {
				d.Socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseProtocolError, ""))
				return err
			}
		}
	}
}

func (d *Dispatcher) Write(msg string) error {
	d.sendQ <- []byte(msg)
	return <-d.sendErrQ
}

func (d *Dispatcher) WriteJSON(value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	d.sendQ <- data
	return <-d.sendErrQ
}

func (d *Dispatcher) Close() {
	close(d.sendQ)
}
