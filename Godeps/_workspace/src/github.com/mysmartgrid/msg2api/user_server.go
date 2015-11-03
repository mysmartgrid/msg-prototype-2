package msg2api

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

type UserServer struct {
	*apiBase

	GetValues              func(since, until time.Time, resolution string, withMetadata bool) error
	RequestRealtimeUpdates func(sensors map[string]map[string][]string) error
}

func (u *UserServer) Run() error {
	for {
		var msg MessageIn

		if err := u.socket.ReceiveJSON(&msg); err != nil {
			u.socket.Close(websocket.CloseProtocolError, err.Error())
			return err
		}

		var opError *Error

		switch msg.Command {
		case "getValues":
			opError = u.doGetValues(&msg)

		case "requestRealtimeUpdates":
			opError = u.doRequestRealtimeUpdates(&msg)

		default:
			u.socket.WriteJSON(MessageOut{Error: badCommand(msg.Command)})
		}

		if opError != nil {
			u.socket.WriteJSON(MessageOut{Error: opError})
		}
	}

	return nil
}

func (u *UserServer) SendUpdate(values UserEventUpdateArgs) error {
	return u.socket.WriteJSON(MessageOut{Command: "update", Args: values})
}

func (u *UserServer) SendMetadata(data UserEventMetadataArgs) error {
	return u.socket.WriteJSON(MessageOut{Command: "metadata", Args: data})
}

func (u *UserServer) doGetValues(cmd *MessageIn) *Error {
	var args UserCmdGetValuesArgs
	var err error

	if err = json.Unmarshal(cmd.Args, &args); err != nil {
		return operationFailed(err.Error())
	}

	if u.GetValues == nil {
		return operationFailed("not supported")
	}

	err = u.GetValues(time.Unix(int64(args.SinceUnixMs/1000), int64(args.SinceUnixMs)%1000*1e6),
		time.Unix(int64(args.UntilUnixMs/1000), int64(args.UntilUnixMs)%1000*1e6),
		args.TimeResolution,
		args.WithMetadata)

	if err != nil {
		return operationFailed(err.Error())
	}
	return nil
}

func (u *UserServer) doRequestRealtimeUpdates(cmd *MessageIn) *Error {
	var args UserCmdRequestRealtimeUpdatesArgs
	var err error

	if err = json.Unmarshal(cmd.Args, &args); err != nil {
		return operationFailed(err.Error())
	}

	if u.RequestRealtimeUpdates == nil {
		return operationFailed("not supported")
	}

	err = u.RequestRealtimeUpdates(args)
	if err != nil {
		return operationFailed(err.Error())
	}
	return nil
}

func NewUserServer(w http.ResponseWriter, r *http.Request) (*UserServer, error) {
	base, err := initApiBaseFromHttp(w, r, []string{userApiProtocolV1})
	if err != nil {
		return nil, err
	}

	result := &UserServer{
		apiBase: base,
	}
	return result, nil
}
