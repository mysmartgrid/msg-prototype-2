import {DeviceSensorMap, DeviceMap, SensorMap, SensorMetadata} from './common';

const ApiVersion : string = "v5.user.msg";

/*
 * Handlers
 */
export interface OpenError {
	error : string;
}

export interface OpenHandler {
	(e : OpenError) : void;
}

export interface CloseHandler {
	(e : CloseEvent) : void;
}

export interface ErrorHandler {
	(e : Event) : void;
}

export interface UpdateHandler {
	(update : UpdateData) : void;
}

/*
 * Messages
 */

export interface UpdateData {
	resolution: string;
	values: DeviceSensorMap<[number, number][]>;
}

export interface MetadataUpdate {
	devices : DeviceMap<DeviceMetadataUpdate>;
}

export interface DeviceMetadataUpdate {
	name : string;
	sensors : SensorMap<SensorMetadata>;
	deletedSensors : SensorMap<string>;
}

export interface MetadataHandler {
	(metadata : MetadataUpdate) : void;
}

export interface UserCommand {
	cmd : string;
	args? : RequestRealtimeUpdateArgs | GetValuesArgs;
}

export interface RequestRealtimeUpdateArgs {
	[deviceID : string] : string[];
}

export interface DeviceSensorList {
	[deviceID : string] : string[]
}

export interface GetValuesArgs {
	since : number;
	until : number;
	resolution : string;
	sensors : DeviceSensorList;
}

export class Socket {
	constructor(private $rootScope : angular.IRootScopeService) {
		this._openHandlers = [];
		this._closeHandlers = [];
		this._errorHandlers = [];
		this._updateHandlers = [];
		this._metadataHandlers = [];
	};

	private _socket : WebSocket;
	private _isOpen : boolean;

	get isOpen() : boolean {
		return this._isOpen;
	}

	private _callHandlers<U>(handlers : ((p : U) => void)[], param : U) {
		for(var handler of handlers) {
			if(this.$rootScope.$$phase === "apply" || this.$rootScope.$$phase === "$digest") {
				handler(param);
			} else {
				this.$rootScope.$apply(function(scope : angular.IScope) : any {
					handler(param);
				});
			}
		}
	}

	private _openHandlers : OpenHandler[];

	public onOpen(handler : OpenHandler) {
		this._openHandlers.push(handler);

		if(this._isOpen) {
			this._callHandlers([handler], null);
		}
	}

	private _emitOpen(e : OpenError) : void {
		this._callHandlers(this._openHandlers, e);
	}

	private _closeHandlers : CloseHandler[];

	public onClose(handler : CloseHandler) {
		this._closeHandlers.push(handler);
	}

	private _emitClose(e : CloseEvent) : void {
		this._callHandlers(this._closeHandlers, e);
	}

	private _errorHandlers : ErrorHandler[];

	public onError(handler : ErrorHandler) {
		this._errorHandlers.push(handler);
	}

	private _emitError(e : Event) : void {
		this._callHandlers(this._errorHandlers, e);
	}

	private _updateHandlers : UpdateHandler[];

	public onUpdate(handler : UpdateHandler) {
		this._updateHandlers.push(handler);
	}

	private _emitUpdate(update : UpdateData) : void {
		this._callHandlers(this._updateHandlers, update);
	}

	private _metadataHandlers : MetadataHandler[];

	public onMetadata(handler : MetadataHandler) {
		this._metadataHandlers.push(handler);
	}

	private _emitMetadata(data : MetadataUpdate) : void {
		this._callHandlers(this._metadataHandlers, data);
	}

	private _onMessage(msg : MessageEvent) : void {
		var data = JSON.parse(msg.data);

        switch (data.cmd) {
        case "update":
            this._emitUpdate(data.args);
            break;

        case "metadata":
            this._emitMetadata(data.args);
            break;

        default:
			console.error("bad packet from server", data);
            this.close();
			break;
        }


	}

	public connect(url : string) {
		this._socket = new WebSocket(url, [ApiVersion]);

		this._socket.onerror = this._emitError.bind(this);
		this._socket.onclose = this._emitClose.bind(this);

		this._socket.onopen = (e : Event) => {
			if (this._socket.protocol !== ApiVersion) {
				this._emitOpen({error: "protocol negotiation failed"});
				this._socket.close();
				this._socket = null;
				return;
			}

			this._isOpen = true;

			this._socket.onmessage = this._onMessage.bind(this);
			this._emitOpen(null);
		};
	};

	private _sendUserCommand(cmd : UserCommand) {
		if(!this._isOpen) {
			throw new Error("Websocket is not connected.");
		}

		this._socket.send(JSON.stringify(cmd));
	}

	public close() : void {
		if (this._socket) {
			this._socket.close();
			this._socket = null;
			this._isOpen = false;
		}
	};

	public requestMetadata() {
		var cmd = {
			cmd: "getMetadata"
		}

		this._sendUserCommand(cmd);
	}

	public requestValues(since : number, until : number, resolution: string, sensors : DeviceSensorList) : void {
        var cmd = {
            cmd: "getValues",
            args: {
                since: since,
				until: until,
				resolution: resolution,
				sensors: sensors
            }
        };
        this._sendUserCommand(cmd);
    };

    public requestRealtimeUpdates(sensors : RequestRealtimeUpdateArgs) : void {
        var cmd = {
            cmd: "requestRealtimeUpdates",
            args: sensors
        };
    	this._sendUserCommand(cmd);
	};
};
