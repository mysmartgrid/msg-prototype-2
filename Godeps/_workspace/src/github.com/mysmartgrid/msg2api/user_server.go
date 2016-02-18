package msg2api

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

// UserServer contains the websocket connection to the User and
// stores handler functions to handler user requests.
type UserServer struct {
	*apiBase

	// GetMetadata handles a request for all availiable metadata for the user.
	GetMetadata func() error

	// GetValues handles a request for measurements of a give resolution in a given timespan for a given set of sensors.
	// 'sensors' contains a mapping from device IDs to an array of sensor IDs.
	GetValues func(since, until time.Time, resolution string, sensors map[string][]string) error

	// RequestRealtimeUpdates handles a request for realtime update on a given set of sensors.
	// 'sensors' contains a mapping from device IDs to a resolution to an array of sensor IDs.
	RequestRealtimeUpdates func(sensors map[string][]string) error
}

// Run listens for incoming commands on the websocket and handles them.
func (u *UserServer) Run() error {
	for {
		var msg MessageIn

		if err := u.socket.ReceiveJSON(&msg); err != nil {
			u.socket.Close(websocket.CloseProtocolError, err.Error())
			return err
		}

		var opError *Error

		switch msg.Command {
		case "getMetadata":
			opError = u.doGetMetadata(&msg)

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
}

// SendUpdate sends a set of measuremnts to the users.
func (u *UserServer) SendUpdate(values UserEventUpdateArgs) error {
	now := time.Now().UnixNano() / 1e6
	return u.socket.WriteJSON(MessageOut{Command: "update", Now: &now, Args: values})
}

// SendMetadata sends a set of metadata descriptions to the user.
func (u *UserServer) SendMetadata(data UserEventMetadataArgs) error {
	now := time.Now().UnixNano() / 1e6
	return u.socket.WriteJSON(MessageOut{Command: "metadata", Now: &now, Args: data})
}

func (u *UserServer) doGetMetadata(cmd *MessageIn) *Error {
	var err error

	if u.GetMetadata == nil {
		return operationFailed("not supported")
	}

	err = u.GetMetadata()

	if err != nil {
		return operationFailed(err.Error())
	}
	return nil
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
		args.Sensors)

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

// NewUserServer returns a new UserServer running on a websocket on the given http connection.
func NewUserServer(w http.ResponseWriter, r *http.Request) (*UserServer, error) {
	base, err := initApiBaseFromHttp(w, r, []string{userApiProtocolV3})
	if err != nil {
		return nil, err
	}

	result := &UserServer{
		apiBase: base,
	}
	return result, nil
}
