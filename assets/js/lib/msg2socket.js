define(["require", "exports"], function (require, exports) {
    "use strict";
    var ApiVersion = "v5.user.msg";
    var Socket = (function () {
        function Socket($rootScope) {
            this.$rootScope = $rootScope;
            this._openHandlers = [];
            this._closeHandlers = [];
            this._errorHandlers = [];
            this._updateHandlers = [];
            this._metadataHandlers = [];
        }
        ;
        Object.defineProperty(Socket.prototype, "isOpen", {
            get: function () {
                return this._isOpen;
            },
            enumerable: true,
            configurable: true
        });
        Socket.prototype._callHandlers = function (handlers, param) {
            for (var _i = 0, handlers_1 = handlers; _i < handlers_1.length; _i++) {
                var handler = handlers_1[_i];
                if (this.$rootScope.$$phase === "apply" || this.$rootScope.$$phase === "$digest") {
                    handler(param);
                }
                else {
                    this.$rootScope.$apply(function (scope) {
                        handler(param);
                    });
                }
            }
        };
        Socket.prototype.onOpen = function (handler) {
            this._openHandlers.push(handler);
            if (this._isOpen) {
                this._callHandlers([handler], null);
            }
        };
        Socket.prototype._emitOpen = function (e) {
            this._callHandlers(this._openHandlers, e);
        };
        Socket.prototype.onClose = function (handler) {
            this._closeHandlers.push(handler);
        };
        Socket.prototype._emitClose = function (e) {
            this._callHandlers(this._closeHandlers, e);
        };
        Socket.prototype.onError = function (handler) {
            this._errorHandlers.push(handler);
        };
        Socket.prototype._emitError = function (e) {
            this._callHandlers(this._errorHandlers, e);
        };
        Socket.prototype.onUpdate = function (handler) {
            this._updateHandlers.push(handler);
        };
        Socket.prototype._emitUpdate = function (update) {
            this._callHandlers(this._updateHandlers, update);
        };
        Socket.prototype.onMetadata = function (handler) {
            this._metadataHandlers.push(handler);
        };
        Socket.prototype._emitMetadata = function (data) {
            this._callHandlers(this._metadataHandlers, data);
        };
        Socket.prototype._onMessage = function (msg) {
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
        };
        Socket.prototype.connect = function (url) {
            var _this = this;
            this._socket = new WebSocket(url, [ApiVersion]);
            this._socket.onerror = this._emitError.bind(this);
            this._socket.onclose = this._emitClose.bind(this);
            this._socket.onopen = function (e) {
                if (_this._socket.protocol !== ApiVersion) {
                    _this._emitOpen({ error: "protocol negotiation failed" });
                    _this._socket.close();
                    _this._socket = null;
                    return;
                }
                _this._isOpen = true;
                _this._socket.onmessage = _this._onMessage.bind(_this);
                _this._emitOpen(null);
            };
        };
        ;
        Socket.prototype._sendUserCommand = function (cmd) {
            if (!this._isOpen) {
                throw new Error("Websocket is not connected.");
            }
            this._socket.send(JSON.stringify(cmd));
        };
        Socket.prototype.close = function () {
            if (this._socket) {
                this._socket.close();
                this._socket = null;
                this._isOpen = false;
            }
        };
        ;
        Socket.prototype.requestMetadata = function () {
            var cmd = {
                cmd: "getMetadata"
            };
            this._sendUserCommand(cmd);
        };
        Socket.prototype.requestValues = function (since, until, resolution, sensors) {
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
        ;
        Socket.prototype.requestRealtimeUpdates = function (sensors) {
            var cmd = {
                cmd: "requestRealtimeUpdates",
                args: sensors
            };
            this._sendUserCommand(cmd);
        };
        ;
        return Socket;
    }());
    exports.Socket = Socket;
    ;
});
//# sourceMappingURL=msg2socket.js.map