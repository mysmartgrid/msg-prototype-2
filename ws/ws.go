package ws

import (
	"github.com/gorilla/websocket"
	"io"
	"time"
)

const (
	pingTimeout = 60 * time.Second
)

type ReadHandler func(d *Dispatcher, msgType int, message string) error
type CloseHandler func(d *Dispatcher)

type Dispatcher struct {
	Socket *websocket.Conn

	OnRead  ReadHandler
	OnClose CloseHandler

	sendQ    chan string
	sendErrQ chan error
}

func (d *Dispatcher) Run() error {
	d.sendQ = make(chan string)
	d.sendErrQ = make(chan error)

	d.Socket.SetReadDeadline(time.Now().Add(pingTimeout))
	d.Socket.SetPongHandler(func(string) error {
		d.Socket.SetReadDeadline(time.Now().Add(pingTimeout))
		return nil
	})

	defer func() {
		if d.OnClose != nil {
			d.OnClose(d)
		}
		d.Socket.Close()
	}()

	go func() {
		ping := time.NewTicker(pingTimeout / 3)
		defer ping.Stop()

		for {
			select {
			case msg, open := <-d.sendQ:
				if !open {
					d.Socket.WriteMessage(websocket.CloseMessage, []byte{})
					d.Socket.Close()
					return
				}

				d.sendErrQ <- d.Socket.WriteMessage(websocket.TextMessage, []byte(msg))

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
		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case d.OnRead != nil:
			d.OnRead(d, msgType, string(msg))
		}
	}
}

func (d *Dispatcher) Write(msg string) error {
	d.sendQ <- msg
	return <-d.sendErrQ
}

func (d *Dispatcher) Close() {
	close(d.sendQ)
}
