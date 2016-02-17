/// <reference path="angular.d.ts" />
"use strict";
var Msg2Socket;
(function (Msg2Socket) {
    var ApiVersion = "v3.user.msg";
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
            for (var _i = 0; _i < handlers.length; _i++) {
                var handler = handlers[_i];
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
    })();
    Msg2Socket.Socket = Socket;
    ;
})(Msg2Socket || (Msg2Socket = {}));
var __extends = (this && this.__extends) || function (d, b) {
    for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p];
    function __() { this.constructor = d; }
    d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
};
var Utils;
(function (Utils) {
    var ExtArray = (function (_super) {
        __extends(ExtArray, _super);
        function ExtArray() {
            _super.apply(this, arguments);
        }
        ExtArray.prototype.contains = function (element) {
            var i = this.indexOf(element);
            return i !== -1;
        };
        ExtArray.prototype.remove = function (element) {
            var i = this.indexOf(element);
            if (i !== -1) {
                this.splice(i, 1);
            }
        };
        ExtArray.prototype.removeWhere = function (pred) {
            var i = this.findIndex(pred);
            while (i !== -1) {
                this.splice(i, 1);
                var i = this.findIndex(pred);
            }
        };
        return ExtArray;
    })(Array);
    Utils.ExtArray = ExtArray;
    function deepCopyJSON(src) {
        var dst = {};
        if (Array.isArray(src)) {
            dst = [];
        }
        for (var key in src) {
            if (src.hasOwnProperty(key)) {
                if (typeof (src[key]) === "object") {
                    dst[key] = deepCopyJSON(src[key]);
                }
                else {
                    dst[key] = src[key];
                }
            }
        }
        return dst;
    }
    Utils.deepCopyJSON = deepCopyJSON;
    function difference(a, b, equals) {
        return a.filter(function (a_element) { return b.findIndex(function (b_element) { return equals(a_element, b_element); }) === -1; });
    }
    Utils.difference = difference;
})(Utils || (Utils = {}));
var Common;
(function (Common) {
    function forEachSensor(map, f) {
        for (var deviceId in map) {
            for (var sensorId in map[deviceId]) {
                f(deviceId, sensorId, map[deviceId][sensorId]);
            }
        }
    }
    Common.forEachSensor = forEachSensor;
    function updateProperties(target, source) {
        var wasUpdated = false;
        for (var prop in target) {
            if (target[prop] !== source[prop]) {
                target[prop] = source[prop];
                wasUpdated = true;
            }
        }
        return wasUpdated;
    }
    Common.updateProperties = updateProperties;
    function now() {
        return (new Date()).getTime();
    }
    Common.now = now;
})(Common || (Common = {}));
/// <reference path="utils.ts"/>
/// <reference path="common.ts"/>
/// <reference path="msg2socket.ts" />
var ExtArray = Utils.ExtArray;
var UpdateDispatcher;
(function (UpdateDispatcher_1) {
    ;
    UpdateDispatcher_1.SupportedResolutions = new Set(["raw", "second", "minute", "hour", "day", "week", "month", "year"]);
    UpdateDispatcher_1.ResoltuionToMillisecs = {
        raw: 1000,
        second: 1000,
        minute: 60 * 1000,
        hour: 60 * 60 * 1000,
        day: 24 * 60 * 60 * 1000,
        week: 7 * 24 * 60 * 60 * 1000,
        month: 31 * 24 * 60 * 60 * 1000,
        year: 365 * 24 * 60 * 60 * 1000
    };
    UpdateDispatcher_1.UpdateDispatcherFactory = ["WSUserClient", "$interval",
        function (wsClient, $interval) {
            return new UpdateDispatcher(wsClient, $interval);
        }];
    var UpdateDispatcher = (function () {
        function UpdateDispatcher(_wsClient, $interval) {
            var _this = this;
            this._wsClient = _wsClient;
            this.$interval = $interval;
            this._devices = {};
            this._subscribers = {};
            this._InitialCallbacks = new Array();
            this._sensorsByUnit = {};
            this._units = [];
            this._hasInitialMetadata = false;
            _wsClient.onOpen(function (error) {
                _wsClient.onClose(function () {
                    _this._hasInitialMetadata = false;
                    _this._devices = {};
                    _this._subscribers = {};
                    _this._sensorsByUnit = {};
                    _this._units = [];
                });
                _wsClient.onMetadata(function (metadata) { return _this._updateMetadata(metadata); });
                _wsClient.onUpdate(function (data) { return _this._updateValues(data); });
                _this._wsClient.requestMetadata();
                $interval(function () { return _this._pollHistoryData(); }, 1 * 60 * 1000);
                $interval(function () { return _this._renewRealtimeRequests(); }, 30 * 1000);
            });
        }
        Object.defineProperty(UpdateDispatcher.prototype, "devices", {
            get: function () {
                return this._devices;
            },
            enumerable: true,
            configurable: true
        });
        Object.defineProperty(UpdateDispatcher.prototype, "units", {
            get: function () {
                return this._units;
            },
            enumerable: true,
            configurable: true
        });
        Object.defineProperty(UpdateDispatcher.prototype, "sensorsByUnit", {
            get: function () {
                return this._sensorsByUnit;
            },
            enumerable: true,
            configurable: true
        });
        UpdateDispatcher.prototype.subscribeInterval = function (deviceID, sensorID, resolution, start, end, subscriber) {
            this._subscribeSensor(deviceID, sensorID, resolution, false, start, end, subscriber);
        };
        UpdateDispatcher.prototype.subscribeSlidingWindow = function (deviceID, sensorID, resolution, start, end, subscriber) {
            this._subscribeSensor(deviceID, sensorID, resolution, true, start, end, subscriber);
        };
        ;
        UpdateDispatcher.prototype.subscribeRealtimeSlidingWindow = function (deviceID, sensorID, resolution, start, subscriber) {
            this._subscribeSensor(deviceID, sensorID, resolution, true, start, 0, subscriber);
        };
        ;
        UpdateDispatcher.prototype._subscribeSensor = function (deviceID, sensorID, resolution, slidingWindow, start, end, subscriber) {
            if (this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }
            if (this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }
            if (!UpdateDispatcher_1.SupportedResolutions.has(resolution)) {
                throw new Error("Unsupported resolution");
            }
            if (slidingWindow && start < end) {
                throw new Error("Start should be bigger then end for sliding window mode");
            }
            else if (!slidingWindow && start > end) {
                throw new Error("End should be bigger then star for interval mode");
            }
            if (this._subscribers[deviceID][sensorID][resolution] === undefined) {
                this._subscribers[deviceID][sensorID][resolution] = new ExtArray();
            }
            this._subscribers[deviceID][sensorID][resolution].push({ slidingWindow: slidingWindow,
                start: start,
                end: end,
                subscriber: subscriber });
            if (slidingWindow && end === 0) {
                var request = {};
                request[deviceID] = {};
                request[deviceID][resolution] = [sensorID];
                this._wsClient.requestRealtimeUpdates(request);
            }
            var now = Common.now();
            if (slidingWindow) {
                start = now - start;
                end = now - end;
            }
            var sensorsList = {};
            sensorsList[deviceID] = [sensorID];
            this._wsClient.requestValues(start, end, resolution, sensorsList);
        };
        UpdateDispatcher.prototype.unsubscribeAll = function (subscriber) {
            var _this = this;
            Common.forEachSensor(this._subscribers, function (deviceID, sensorID, sensor) {
                for (var resolution in sensor) {
                    _this.unsubscribeSensor(deviceID, sensorID, resolution, subscriber);
                }
            });
        };
        UpdateDispatcher.prototype.unsubscribeSensor = function (deviceID, sensorID, resolution, subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber);
        };
        UpdateDispatcher.prototype.unsubscribeInterval = function (deviceID, sensorID, resolution, start, end, subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber, false, start, end);
        };
        UpdateDispatcher.prototype.unsubscribeSlidingWindow = function (deviceID, sensorID, resolution, start, end, subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber, true, start, end);
        };
        ;
        UpdateDispatcher.prototype.unsubscribeRealtimeSlidingWindow = function (deviceID, sensorID, resolution, start, subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber, true, start, 0);
        };
        ;
        UpdateDispatcher.prototype._unsubscribeSensor = function (deviceID, sensorID, resolution, subscriber, slidingWindow, start, end) {
            if (this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }
            if (this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }
            if (this._subscribers[deviceID][sensorID][resolution] === undefined) {
                throw new Error("No subscribers for this resolution");
            }
            if (start === undefined && end === undefined && slidingWindow === undefined) {
                this._subscribers[deviceID][sensorID][resolution].removeWhere(function (settings) { return settings.subscriber == subscriber; });
            }
            else if (start !== undefined && end !== undefined && slidingWindow !== undefined) {
                this._subscribers[deviceID][sensorID][resolution].removeWhere(function (settings) {
                    return settings.subscriber === subscriber &&
                        settings.slidingWindow === slidingWindow &&
                        settings.start === start &&
                        settings.end === end;
                });
            }
            else {
                throw new Error("Either start or end missing");
            }
        };
        UpdateDispatcher.prototype.onInitialMetadata = function (callback) {
            if (!this._hasInitialMetadata) {
                this._InitialCallbacks.push(callback);
            }
            else {
                callback();
            }
        };
        UpdateDispatcher.prototype._updateMetadata = function (metadata) {
            for (var deviceID in metadata.devices) {
                if (this._devices[deviceID] === undefined) {
                    this._devices[deviceID] = {
                        name: null,
                        sensors: {}
                    };
                }
                if (this._subscribers[deviceID] === undefined) {
                    this._subscribers[deviceID] = {};
                }
                var deviceName = metadata.devices[deviceID].name;
                if (deviceName !== undefined && this._devices[deviceID].name !== deviceName) {
                    this._devices[deviceID].name = deviceName;
                    this._emitDeviceMetadataUpdate(deviceID);
                    console.log("Nameupdate: " + deviceName);
                }
                for (var sensorID in metadata.devices[deviceID].sensors) {
                    if (this._subscribers[deviceID][sensorID] === undefined) {
                        this._subscribers[deviceID][sensorID] = {};
                    }
                    if (this._devices[deviceID].sensors[sensorID] === undefined) {
                        this._devices[deviceID].sensors[sensorID] = {
                            name: null,
                            unit: null,
                            port: null,
                        };
                    }
                    var wasUpdated = Common.updateProperties(this._devices[deviceID].sensors[sensorID], metadata.devices[deviceID].sensors[sensorID]);
                    if (wasUpdated) {
                        this._emitSensorMetadataUpdate(deviceID, sensorID);
                    }
                }
                for (var sensorID in metadata.devices[deviceID].deletedSensors) {
                    delete this._devices[deviceID].sensors[sensorID];
                    this._emitRemoveSensor(deviceID, sensorID);
                    delete this._subscribers[deviceID][sensorID];
                }
            }
            this._updateSensorsByUnit();
            if (!this._hasInitialMetadata) {
                this._hasInitialMetadata = true;
                for (var _i = 0, _a = this._InitialCallbacks; _i < _a.length; _i++) {
                    var callback = _a[_i];
                    callback();
                }
            }
        };
        UpdateDispatcher.prototype._updateSensorsByUnit = function () {
            for (var index in this._sensorsByUnit) {
                delete this._sensorsByUnit[this._units[index]];
                delete this._units[index];
            }
            for (var deviceID in this._devices) {
                for (var sensorID in this._devices[deviceID].sensors) {
                    var unit = this._devices[deviceID].sensors[sensorID].unit;
                    if (this._sensorsByUnit[unit] === undefined) {
                        this._units.push(unit);
                        this._sensorsByUnit[unit] = [];
                    }
                    this._sensorsByUnit[unit].push({ deviceID: deviceID, sensorID: sensorID });
                }
            }
        };
        UpdateDispatcher.prototype._emitDeviceMetadataUpdate = function (deviceID) {
            var notified = new Set();
            for (var sensorID in this._subscribers[deviceID]) {
                for (var resolution in this._subscribers[deviceID][sensorID]) {
                    for (var _i = 0, _a = this._subscribers[deviceID][sensorID][resolution]; _i < _a.length; _i++) {
                        var subscriber = _a[_i].subscriber;
                        if (!notified.has(subscriber)) {
                            subscriber.updateDeviceMetadata(deviceID);
                            notified.add(subscriber);
                        }
                    }
                }
            }
        };
        UpdateDispatcher.prototype._emitSensorMetadataUpdate = function (deviceID, sensorID) {
            var notified = new Set();
            for (var resolution in this._subscribers[deviceID][sensorID]) {
                for (var _i = 0, _a = this._subscribers[deviceID][sensorID][resolution]; _i < _a.length; _i++) {
                    var subscriber = _a[_i].subscriber;
                    if (!notified.has(subscriber)) {
                        subscriber.updateSensorMetadata(deviceID, sensorID);
                        notified.add(subscriber);
                    }
                }
            }
        };
        UpdateDispatcher.prototype._emitRemoveSensor = function (deviceID, sensorID) {
            var notified = new Set();
            for (var resolution in this._subscribers[deviceID][sensorID]) {
                for (var _i = 0, _a = this._subscribers[deviceID][sensorID][resolution]; _i < _a.length; _i++) {
                    var subscriber = _a[_i].subscriber;
                    if (!notified.has(subscriber)) {
                        subscriber.removeSensor(deviceID, sensorID);
                        notified.add(subscriber);
                    }
                }
            }
        };
        UpdateDispatcher.prototype._pollHistoryData = function () {
            var requests;
            requests = {};
            var now = Common.now();
            Common.forEachSensor(this._subscribers, function (deviceID, sensorID, map) {
                for (var resolution in map) {
                    if (resolution !== 'raw') {
                        for (var _i = 0, _a = map[resolution]; _i < _a.length; _i++) {
                            var _b = _a[_i], start = _b.start, end = _b.end, slidingWindow = _b.slidingWindow;
                            if (slidingWindow) {
                                start = now - start;
                                end = now - end;
                            }
                            if (requests[resolution] === undefined) {
                                requests[resolution] = {
                                    start: start,
                                    end: end,
                                    sensors: {}
                                };
                            }
                            requests[resolution].start = Math.min(start, requests[resolution].start);
                            requests[resolution].end = Math.max(end, requests[resolution].end);
                            if (requests[resolution].sensors[deviceID] === undefined) {
                                requests[resolution].sensors[deviceID] = new Set();
                            }
                            requests[resolution].sensors[deviceID].add(sensorID);
                        }
                    }
                }
            });
            for (var resolution in requests) {
                var _a = requests[resolution], start = _a.start, end = _a.end, sensors = _a.sensors;
                var sensorList = {};
                for (var deviceID in sensors) {
                    sensorList[deviceID] = [];
                    sensors[deviceID].forEach(function (sensorID) { return sensorList[deviceID].push(sensorID); });
                }
                this._wsClient.requestValues(start, end, resolution, sensorList);
            }
        };
        UpdateDispatcher.prototype._renewRealtimeRequests = function () {
            var request = {};
            Common.forEachSensor(this._subscribers, function (deviceID, sensorID, map) {
                if (request[deviceID] === undefined) {
                    request[deviceID] = {};
                }
                for (var resolution in map) {
                    if (request[deviceID][resolution] === undefined) {
                        request[deviceID][resolution] = [];
                    }
                    for (var _i = 0, _a = map[resolution]; _i < _a.length; _i++) {
                        var _b = _a[_i], start = _b.start, end = _b.end, slidingWindow = _b.slidingWindow;
                        if (end === 0 && slidingWindow) {
                            if (request[deviceID][resolution].indexOf(sensorID) === -1) {
                                request[deviceID][resolution].push(sensorID);
                            }
                        }
                    }
                }
            });
            this._wsClient.requestRealtimeUpdates(request);
        };
        UpdateDispatcher.prototype._updateValues = function (data) {
            var resolution = data.resolution, values = data.values;
            for (var deviceID in values) {
                for (var sensorID in values[deviceID]) {
                    for (var _i = 0, _a = values[deviceID][sensorID]; _i < _a.length; _i++) {
                        var _b = _a[_i], timestamp = _b[0], value = _b[1];
                        this._emitValueUpdate(deviceID, sensorID, resolution, timestamp, value);
                    }
                }
            }
        };
        UpdateDispatcher.prototype._emitValueUpdate = function (deviceID, sensorID, resolution, timestamp, value) {
            var now = Common.now();
            var notified = new Set();
            if (this._subscribers[deviceID] !== undefined
                && this._subscribers[deviceID][sensorID] !== undefined
                && this._subscribers[deviceID][sensorID][resolution] !== undefined) {
                for (var _i = 0, _a = this._subscribers[deviceID][sensorID][resolution]; _i < _a.length; _i++) {
                    var _b = _a[_i], start = _b.start, end = _b.end, slidingWindow = _b.slidingWindow, subscriber = _b.subscriber;
                    if (slidingWindow) {
                        start = now - start;
                        end = now - end;
                    }
                    if (start <= timestamp && end >= timestamp && !notified.has(subscriber)) {
                        subscriber.updateValue(deviceID, sensorID, resolution, timestamp, value);
                        notified.add(subscriber);
                    }
                }
            }
        };
        return UpdateDispatcher;
    })();
    UpdateDispatcher_1.UpdateDispatcher = UpdateDispatcher;
    var DummySubscriber = (function () {
        function DummySubscriber() {
        }
        DummySubscriber.prototype.updateValue = function (deviceID, sensorID, resolution, timestamp, value) {
            var date = new Date(timestamp);
            console.log("Update for value " + deviceID + ":" + sensorID + " " + resolution + " " + date + " " + value);
        };
        DummySubscriber.prototype.updateDeviceMetadata = function (deviceID) {
            console.log("Update for device metadata " + deviceID);
        };
        DummySubscriber.prototype.updateSensorMetadata = function (deviceID, sensorID) {
            console.log("Update for sensor metadata " + deviceID + ":" + sensorID);
        };
        DummySubscriber.prototype.removeDevice = function (deviceID) {
            console.log("Removed device " + deviceID);
        };
        DummySubscriber.prototype.removeSensor = function (deviceID, sensorID) {
            console.log("Remove sensor " + deviceID + ":" + sensorID);
        };
        return DummySubscriber;
    })();
    UpdateDispatcher_1.DummySubscriber = DummySubscriber;
})(UpdateDispatcher || (UpdateDispatcher = {}));
/// <reference path="es6-shim.d.ts" />
/// <reference path="common.ts"/>
/// <reference path="msg2socket.ts" />
"use strict";
var Store;
(function (Store) {
    var ColorScheme = ['#00A8F0', '#C0D800', '#CB4B4B', '#4DA74D', '#9440ED'];
    var SensorValueStore = (function () {
        function SensorValueStore() {
            this._series = [];
            this._sensorMap = {};
            this._timeout = 2.5 * 60 * 1000;
            this._start = 5 * 60 * 1000;
            this._end = 0;
            this._slidingWindow = true;
            this._colorIndex = 0;
        }
        ;
        SensorValueStore.prototype._pickColor = function () {
            var color = ColorScheme[this._colorIndex];
            this._colorIndex = (this._colorIndex + 1) % ColorScheme.length;
            return color;
        };
        SensorValueStore.prototype._getSensorIndex = function (deviceId, sensorId) {
            if (this._sensorMap[deviceId] !== undefined && this._sensorMap[deviceId][sensorId] !== undefined) {
                return this._sensorMap[deviceId][sensorId];
            }
            return -1;
        };
        SensorValueStore.prototype.setStart = function (start) {
            this._start = start;
        };
        SensorValueStore.prototype.setEnd = function (end) {
            this._end = end;
        };
        SensorValueStore.prototype.setSlidingWindowMode = function (mode) {
            this._slidingWindow = mode;
        };
        SensorValueStore.prototype.setTimeout = function (timeout) {
            this._timeout = timeout;
        };
        SensorValueStore.prototype.clampData = function () {
            var oldest = this._start;
            var newest = this._end;
            if (this._slidingWindow) {
                oldest = Common.now() - this._start;
                newest = Common.now() - this._end;
            }
            this._series.forEach(function (series) {
                series.data = series.data.filter(function (point) {
                    return point[0] >= oldest && point[0] <= newest;
                });
                if (series.data.length > 0) {
                    if (series.data[0][1] === null) {
                        series.data.splice(0, 1);
                    }
                    if (series.data[series.data.length - 1][1] === null) {
                        series.data.splice(series.data.length - 1, 1);
                    }
                }
            });
        };
        SensorValueStore.prototype.addSensor = function (deviceId, sensorId) {
            if (this.hasSensor(deviceId, sensorId)) {
                throw new Error("Sensor has been added already");
            }
            var index = this._series.length;
            if (this._sensorMap[deviceId] === undefined) {
                this._sensorMap[deviceId] = {};
            }
            this._sensorMap[deviceId][sensorId] = index;
            this._series.push({
                line: {
                    color: this._pickColor(),
                },
                data: []
            });
        };
        SensorValueStore.prototype.hasSensor = function (device, sensor) {
            return this._getSensorIndex(device, sensor) !== -1;
        };
        SensorValueStore.prototype.removeSensor = function (deviceId, sensorId) {
            var index = this._getSensorIndex(deviceId, sensorId);
            if (index === -1) {
                throw new Error("No such sensor");
            }
            this._series.splice(index, 1);
            delete this._sensorMap[deviceId][sensorId];
        };
        SensorValueStore.prototype._findInsertionPos = function (data, timestamp) {
            for (var pos = 0; pos < data.length; pos++) {
                if (data[pos][0] > timestamp) {
                    return pos;
                }
            }
            return data.length;
        };
        SensorValueStore.prototype.addValue = function (deviceId, sensorId, timestamp, value) {
            var seriesIndex = this._getSensorIndex(deviceId, sensorId);
            if (seriesIndex === -1) {
                throw new Error("No such sensor");
            }
            var data = this._series[seriesIndex].data;
            var pos = this._findInsertionPos(data, timestamp);
            if (data.length > 0 && pos === 0 && data[0][0] === timestamp) {
                data[0][1] = value;
            }
            else if (data.length > 0 && pos > 0 && pos <= data.length && data[pos - 1][0] === timestamp) {
                data[pos - 1][1] = value;
            }
            else {
                data.splice(pos, 0, [timestamp, value]);
                if (pos > 0 && data[pos - 1][1] === null && timestamp - data[pos - 1][0] < this._timeout) {
                    data.splice(pos - 1, 1);
                    pos -= 1;
                }
                if (pos < data.length - 1 && data[pos + 1][1] === null && data[pos + 1][0] - timestamp < this._timeout) {
                    data.splice(pos + 1, 1);
                }
                if (pos > 0 && data[pos - 1][1] !== null && timestamp - data[pos - 1][0] >= this._timeout) {
                    data.splice(pos, 0, [timestamp - 1, null]);
                }
                if (pos < data.length - 1 && data[pos + 1][1] !== null && data[pos + 1][0] - timestamp >= this._timeout) {
                    data.splice(pos + 1, 0, [timestamp + 1, null]);
                }
            }
        };
        SensorValueStore.prototype.getData = function () {
            return this._series;
        };
        SensorValueStore.prototype.getColors = function () {
            var colors = {};
            for (var deviceId in this._sensorMap) {
                colors[deviceId] = {};
                for (var sensorId in this._sensorMap[deviceId]) {
                    var index = this._sensorMap[deviceId][sensorId];
                    colors[deviceId][sensorId] = this._series[index].line.color;
                }
            }
            return colors;
        };
        return SensorValueStore;
    })();
    Store.SensorValueStore = SensorValueStore;
})(Store || (Store = {}));
/// <reference path="../angular.d.ts" />
var Directives;
(function (Directives) {
    var UserInterface;
    (function (UserInterface) {
        var NumberSpinnerController = (function () {
            function NumberSpinnerController($scope) {
                var _this = this;
                this.$scope = $scope;
                $scope.change = function () {
                    _this._enforceLimits();
                    $scope.ngChange();
                };
                $scope.increment = function () {
                    $scope.ngModel++;
                    $scope.change();
                };
                $scope.decrement = function () {
                    $scope.ngModel--;
                    $scope.change();
                };
            }
            NumberSpinnerController.prototype._enforceLimits = function () {
                if (this.$scope.ngModel !== undefined && this.$scope.ngModel !== null) {
                    console.log("Enforcing limits");
                    if (this.$scope.ngModel !== Math.round(this.$scope.ngModel)) {
                        this.$scope.ngModel = Math.round(this.$scope.ngModel);
                    }
                    if (this.$scope.ngModel > this.$scope.max) {
                        this.$scope.ngModel = this.$scope.max;
                        console.log("Overflow");
                        this.$scope.overflow();
                    }
                    if (this.$scope.ngModel < this.$scope.min) {
                        this.$scope.ngModel = this.$scope.min;
                        this.$scope.underflow();
                    }
                    console.log("Limits enforced");
                }
            };
            NumberSpinnerController.prototype._onMouseWheel = function (event) {
                if (event.originalEvent !== undefined) {
                    event = event.originalEvent;
                }
                var delta = event.wheelDelta;
                if (delta === undefined) {
                    delta = -event.deltaY;
                }
                if (Math.abs(delta) > 10) {
                    if (delta > 0) {
                        this.$scope.increment();
                    }
                    else {
                        this.$scope.decrement();
                    }
                }
                event.preventDefault();
            };
            NumberSpinnerController.prototype.setupEvents = function (element) {
                var _this = this;
                var input = element.find(".numberSpinner");
                input.bind("mouse wheel", function (event) { return _this._onMouseWheel(event); });
            };
            return NumberSpinnerController;
        })();
        UserInterface.NumberSpinnerController = NumberSpinnerController;
        function NumberSpinnerFactory() {
            return function () { return new NumberSpinnerDirective(); };
        }
        UserInterface.NumberSpinnerFactory = NumberSpinnerFactory;
        var NumberSpinnerDirective = (function () {
            function NumberSpinnerDirective() {
                this.require = "numberSpinner";
                this.restrict = "A";
                this.templateUrl = "/html/number-spinner.html";
                this.scope = {
                    ngModel: '=',
                    ngChange: '&',
                    overflow: '&',
                    underflow: '&',
                    min: '=',
                    max: '='
                };
                this.controller = ["$scope", NumberSpinnerController];
                this.link = function ($scope, element, attrs, numberSpinner) {
                    numberSpinner.setupEvents(element);
                };
            }
            return NumberSpinnerDirective;
        })();
    })(UserInterface = Directives.UserInterface || (Directives.UserInterface = {}));
})(Directives || (Directives = {}));
/// <reference path="../angular.d.ts" />
/// <reference path="../common.ts"/>
var Directives;
(function (Directives) {
    var UserInterface;
    (function (UserInterface) {
        var TimeUnits = ["years", "days", "hours", "minutes"];
        var UnitsToMillisecs = {
            "years": 365 * 24 * 60 * 60 * 1000,
            "days": 24 * 60 * 60 * 1000,
            "hours": 60 * 60 * 1000,
            "minutes": 60 * 1000
        };
        var TimeRangeSpinnerController = (function () {
            function TimeRangeSpinnerController($scope) {
                var _this = this;
                this.$scope = $scope;
                $scope.time = {
                    years: 0,
                    days: 0,
                    hours: 0,
                    minutes: 0
                };
                if ($scope.ngModel !== undefined) {
                    $scope.$watch("ngModel", function () { return _this._setFromMilliseconds($scope.ngModel); });
                }
                console.log($scope);
                $scope.change = function () { return _this._change(); };
                $scope.increment = function (unit) { return _this._increment(unit); };
                $scope.decrement = function (unit) { return _this._decrement(unit); };
            }
            TimeRangeSpinnerController.prototype._increment = function (unit) {
                if (this.$scope.time[unit] !== undefined) {
                    this.$scope.time[unit] += 1;
                }
                this.$scope.change();
            };
            TimeRangeSpinnerController.prototype._decrement = function (unit) {
                if (this.$scope.time[unit] !== undefined) {
                    this.$scope.time[unit] -= 1;
                }
                this.$scope.change();
            };
            TimeRangeSpinnerController.prototype._change = function () {
                var _this = this;
                var editDone = TimeUnits.every(function (unit) { return (_this.$scope.time[unit] !== null && _this.$scope.time[unit] !== undefined); });
                if (editDone) {
                    var milliseconds = 0;
                    for (var _i = 0; _i < TimeUnits.length; _i++) {
                        var unit = TimeUnits[_i];
                        milliseconds += this.$scope.time[unit] * UnitsToMillisecs[unit];
                    }
                    if (this.$scope.min !== undefined) {
                        milliseconds = Math.max(this.$scope.min, milliseconds);
                    }
                    if (this.$scope.max !== undefined) {
                        milliseconds = Math.min(this.$scope.max, milliseconds);
                    }
                    this._setFromMilliseconds(milliseconds);
                    if (this.$scope.ngModel !== undefined) {
                        this.$scope.ngModel = milliseconds;
                    }
                    this.$scope.ngChange();
                }
            };
            TimeRangeSpinnerController.prototype._setFromMilliseconds = function (milliseconds) {
                var remainder = milliseconds;
                for (var _i = 0; _i < TimeUnits.length; _i++) {
                    var unit = TimeUnits[_i];
                    this.$scope.time[unit] = Math.floor(remainder / UnitsToMillisecs[unit]);
                    remainder = remainder % UnitsToMillisecs[unit];
                }
            };
            TimeRangeSpinnerController.prototype.setupScrollEvents = function (element) {
                var _this = this;
                element.find("input[type='number']").each(function (index, element) {
                    var field = $(element);
                    field.bind("mouse wheel", function (jqEvent) {
                        if (jqEvent.originalEvent === undefined) {
                            return;
                        }
                        var event = jqEvent.originalEvent;
                        var delta = event.wheelDelta;
                        if (delta === undefined) {
                            delta = -event.deltaY;
                        }
                        if (delta > 0) {
                            _this.$scope.increment(field.attr('name'));
                        }
                        else {
                            _this.$scope.decrement(field.attr('name'));
                        }
                        jqEvent.preventDefault();
                    });
                });
            };
            return TimeRangeSpinnerController;
        })();
        UserInterface.TimeRangeSpinnerController = TimeRangeSpinnerController;
        function TimeRangeSpinnerFactory() {
            return function () { return new TimeRangeSpinnerDirective(); };
        }
        UserInterface.TimeRangeSpinnerFactory = TimeRangeSpinnerFactory;
        var TimeRangeSpinnerDirective = (function () {
            function TimeRangeSpinnerDirective() {
                this.restrict = "A";
                this.templateUrl = "/html/time-range-spinner.html";
                this.scope = {
                    ngModel: '=?',
                    ngChange: '&',
                    min: '=?',
                    max: '=?'
                };
                this.controller = ["$scope", TimeRangeSpinnerController];
                this.link = function ($scope, element, attrs, controller) {
                    controller.setupScrollEvents(element);
                };
            }
            return TimeRangeSpinnerDirective;
        })();
    })(UserInterface = Directives.UserInterface || (Directives.UserInterface = {}));
})(Directives || (Directives = {}));
/// <reference path="../angular.d.ts" />
/// <reference path="../common.ts" />
var Directives;
(function (Directives) {
    var UserInterface;
    (function (UserInterface) {
        var DateTimePickerController = (function () {
            function DateTimePickerController($scope) {
                var _this = this;
                this.$scope = $scope;
                if ($scope.ngModel !== undefined) {
                    $scope.$watch("ngModel", function () {
                        if (_this.$scope.ngModel !== _this._dateToMillisecs()) {
                            _this._millisecsToDate($scope.ngModel);
                        }
                    });
                }
                $scope.change = function () { return _this._change(); };
            }
            DateTimePickerController.prototype._millisecsToDate = function (millisecs) {
                this.$scope.date = new Date(millisecs);
            };
            DateTimePickerController.prototype._dateToMillisecs = function () {
                var result = new Date(this.$scope.date);
                return result.getTime();
            };
            DateTimePickerController.prototype._change = function () {
                if (this.$scope.date !== null) {
                    var millisecs = this._dateToMillisecs();
                    if (this.$scope.min !== undefined) {
                        millisecs = Math.max(millisecs, this.$scope.min);
                    }
                    if (this.$scope.max !== undefined) {
                        millisecs = Math.min(millisecs, this.$scope.max);
                    }
                    this.$scope.ngModel = millisecs;
                    this.$scope.ngChange();
                }
            };
            return DateTimePickerController;
        })();
        UserInterface.DateTimePickerController = DateTimePickerController;
        function DateTimePickerFactory() {
            return function () { return new DateTimePickerDirective(); };
        }
        UserInterface.DateTimePickerFactory = DateTimePickerFactory;
        var DateTimePickerDirective = (function () {
            function DateTimePickerDirective() {
                this.restrict = "A";
                this.templateUrl = "/html/date-time-picker.html";
                this.scope = {
                    ngModel: '=?',
                    ngChange: '&',
                    min: '=?',
                    max: '=?'
                };
                this.controller = ["$scope", DateTimePickerController];
                this.link = function ($scope, element, attrs, aateTimePicker) {
                };
            }
            return DateTimePickerDirective;
        })();
    })(UserInterface = Directives.UserInterface || (Directives.UserInterface = {}));
})(Directives || (Directives = {}));
/// <reference path="angular.d.ts" />
/// <reference path="angular-ui-bootstrap.d.ts" />
/// <reference path="common.ts"/>
/// <reference path="msg2socket.ts" />
/// <reference path="sensorvaluestore.ts" />
"use strict";
var Directives;
(function (Directives) {
    function sensorEqual(a, b) {
        return a.deviceID === b.deviceID && a.sensorID === b.sensorID;
    }
    Directives.sensorEqual = sensorEqual;
    var SensorGraphSettingsFactory = ["$scope", "$uibModalInstance", "UpdateDispatcher", "config",
        function ($scope, $uibModalInstance, dispatcher, config) {
            return new SensorGraphSettingsController($scope, $uibModalInstance, dispatcher, config);
        }];
    var SensorGraphSettingsController = (function () {
        function SensorGraphSettingsController($scope, $uibModalInstance, _dispatcher, config) {
            this.$scope = $scope;
            this.$uibModalInstance = $uibModalInstance;
            this._dispatcher = _dispatcher;
            $scope.devices = _dispatcher.devices;
            var supportedResolutions = Array.from(UpdateDispatcher.SupportedResolutions.values());
            $scope.resolutions = {};
            $scope.resolutions['realtime'] = supportedResolutions.filter(function (res) { return res !== 'second'; });
            $scope.resolutions['slidingWindow'] = supportedResolutions.filter(function (res) { return res !== 'raw'; });
            $scope.resolutions['interval'] = supportedResolutions.filter(function (res) { return res !== 'raw'; });
            $scope.$watch("config.mode", function () {
                var mode = $scope.config.mode;
                if ($scope.resolutions[mode].indexOf($scope.config.resolution) === -1) {
                    $scope.config.resolution = $scope.resolutions[mode][0];
                }
            });
            $scope.units = _dispatcher.units;
            $scope.sensorsByUnit = _dispatcher.sensorsByUnit;
            $scope.config = config;
            $scope.pickerModes = {
                raw: 'day',
                second: 'day',
                minute: 'day',
                hour: 'day',
                day: 'day',
                week: 'day',
                month: 'month',
                year: 'year'
            };
            $scope.ok = function () {
                $uibModalInstance.close($scope.config);
            };
            $scope.cancel = function () {
                $uibModalInstance.dismiss('cancel');
            };
        }
        return SensorGraphSettingsController;
    })();
    var SensorGraphController = (function () {
        function SensorGraphController($scope, $interval, $timeout, $uibModal, _dispatcher) {
            var _this = this;
            this.$scope = $scope;
            this.$interval = $interval;
            this.$timeout = $timeout;
            this.$uibModal = $uibModal;
            this._dispatcher = _dispatcher;
            this._store = new Store.SensorValueStore();
            this._store.setSlidingWindowMode(true);
            this._store.setEnd(0);
            this.$scope.devices = this._dispatcher.devices;
            this._dispatcher.onInitialMetadata(function () {
                _this._setDefaultConfig();
                _this._redrawGraph();
            });
            this.$scope.openSettings = function () {
                var modalInstance = $uibModal.open({
                    controller: SensorGraphSettingsFactory,
                    size: "lg",
                    templateUrl: 'sensor-graph-settings.html',
                    resolve: {
                        config: function () {
                            return Utils.deepCopyJSON(_this._config);
                        }
                    }
                });
                modalInstance.result.then(function (config) {
                    _this._applyConfig(config);
                });
            };
            $interval(function () { return _this._store.clampData(); }, 60 * 1000);
        }
        Object.defineProperty(SensorGraphController.prototype, "graphNode", {
            set: function (element) {
                this._graphNode = element.find(".sensor-graph").get(0);
            },
            enumerable: true,
            configurable: true
        });
        SensorGraphController.prototype.updateValue = function (deviceID, sensorID, resolution, timestamp, value) {
            this._store.addValue(deviceID, sensorID, timestamp, value);
        };
        SensorGraphController.prototype.updateDeviceMetadata = function (deviceID) { };
        ;
        SensorGraphController.prototype.updateSensorMetadata = function (deviceID, sensorID) {
        };
        ;
        SensorGraphController.prototype.removeDevice = function (deviceID) { };
        ;
        SensorGraphController.prototype.removeSensor = function (deviceID, sensorID) { };
        ;
        SensorGraphController.prototype._setDefaultConfig = function () {
            this._applyConfig({
                unit: this._dispatcher.units[0],
                resolution: UpdateDispatcher.SupportedResolutions.values().next().value,
                sensors: [],
                mode: 'realtime',
                intervalStart: Common.now() - 24 * 60 * 1000,
                intervalEnd: Common.now(),
                windowStart: 5 * 60 * 1000,
                windowEnd: 0
            });
        };
        SensorGraphController.prototype._subscribeSensor = function (config, deviceID, sensorID) {
            if (config.mode === 'realtime') {
                this._dispatcher.subscribeRealtimeSlidingWindow(deviceID, sensorID, config.resolution, config.windowStart, this);
            }
            else if (config.mode === 'slidingWindow') {
                this._dispatcher.subscribeSlidingWindow(deviceID, sensorID, config.resolution, config.windowStart, config.windowEnd, this);
            }
            else if (config.mode === 'interval') {
                this._dispatcher.subscribeInterval(deviceID, sensorID, config.resolution, config.intervalStart, config.intervalEnd, this);
            }
            else {
                throw new Error("Unknown mode:" + config.mode);
            }
        };
        SensorGraphController.prototype._unsubscribeSensor = function (config, deviceID, sensorID) {
            if (config.mode === 'realtime') {
                this._dispatcher.unsubscribeRealtimeSlidingWindow(deviceID, sensorID, config.resolution, config.windowStart, this);
            }
            else if (config.mode === 'slidingWindow') {
                this._dispatcher.unsubscribeSlidingWindow(deviceID, sensorID, config.resolution, config.windowStart, config.windowEnd, this);
            }
            else if (config.mode === 'interval') {
                this._dispatcher.unsubscribeInterval(deviceID, sensorID, config.resolution, config.intervalStart, config.intervalEnd, this);
            }
            else {
                throw new Error("Unknown mode:" + config.mode);
            }
        };
        SensorGraphController.prototype._applyConfig = function (config) {
            if (this._config !== undefined &&
                config.mode === this._config.mode &&
                config.resolution == this._config.resolution &&
                config.unit === this._config.unit &&
                config.windowStart === this._config.windowStart &&
                config.windowEnd === this._config.windowEnd &&
                config.intervalStart === this._config.intervalStart &&
                config.intervalEnd === this._config.intervalEnd) {
                var addedSensors = Utils.difference(config.sensors, this._config.sensors, sensorEqual);
                var removedSensors = Utils.difference(this._config.sensors, config.sensors, sensorEqual);
                for (var _i = 0; _i < addedSensors.length; _i++) {
                    var _a = addedSensors[_i], deviceID = _a.deviceID, sensorID = _a.sensorID;
                    this._subscribeSensor(config, deviceID, sensorID);
                    this._store.addSensor(deviceID, sensorID);
                }
                for (var _b = 0; _b < removedSensors.length; _b++) {
                    var _c = removedSensors[_b], deviceID = _c.deviceID, sensorID = _c.sensorID;
                    this._unsubscribeSensor(this._config, deviceID, sensorID);
                    this._store.removeSensor(deviceID, sensorID);
                }
            }
            else {
                this._dispatcher.unsubscribeAll(this);
                this._store = new Store.SensorValueStore();
                if (config.mode === 'realtime') {
                    this._store.setSlidingWindowMode(true);
                    this._store.setStart(config.windowStart);
                    this._store.setEnd(0);
                }
                else if (config.mode === 'slidingWindow') {
                    this._store.setSlidingWindowMode(true);
                    this._store.setStart(config.windowStart);
                    this._store.setEnd(config.windowEnd);
                }
                else if (config.mode === 'interval') {
                    this._store.setSlidingWindowMode(false);
                    this._store.setStart(config.intervalStart);
                    this._store.setEnd(config.intervalEnd);
                }
                for (var _d = 0, _e = config.sensors; _d < _e.length; _d++) {
                    var _f = _e[_d], deviceID = _f.deviceID, sensorID = _f.sensorID;
                    this._subscribeSensor(config, deviceID, sensorID);
                    this._store.addSensor(deviceID, sensorID);
                }
            }
            this._store.setTimeout(UpdateDispatcher.ResoltuionToMillisecs[config.resolution] * 60);
            this._config = config;
            this.$scope.sensorColors = this._store.getColors();
            this.$scope.sensors = config.sensors;
            this._redrawGraph();
        };
        SensorGraphController.prototype._redrawGraph = function () {
            var _this = this;
            this.$timeout.cancel(this._timeout);
            var time = Common.now();
            var graphOptions = {
                xaxis: {
                    mode: 'time',
                    timeMode: 'local',
                    title: 'Time',
                    noTicks: 15,
                    minorTickFreq: 1
                },
                HtmlText: false,
                preventDefault: false,
                title: 'Messwerte',
                shadowSize: 0,
                lines: {
                    lineWidth: 2,
                }
            };
            graphOptions.title = 'Values [' + this._config.unit + ']';
            var delay;
            if (this._config.mode === "slidingWindow" || this._config.mode === "realtime") {
                graphOptions.xaxis.min = time - this._config.windowStart;
                if (this._config.mode === "realtime") {
                    graphOptions.xaxis.max = time;
                    delay = this._config.windowStart;
                }
                else {
                    graphOptions.xaxis.max = time - this._config.windowEnd;
                    delay = this._config.windowStart - this._config.windowEnd;
                }
            }
            else {
                graphOptions.xaxis.min = this._config.intervalStart;
                graphOptions.xaxis.max = this._config.intervalEnd;
                delay = this._config.intervalStart - this._config.intervalEnd;
            }
            var graph = Flotr.draw(this._graphNode, this._store.getData(), graphOptions);
            delay = delay / graph.plotWidth;
            delay = Math.min(10000, delay);
            this._timeout = this.$timeout(function () { return _this._redrawGraph(); }, delay);
        };
        return SensorGraphController;
    })();
    Directives.SensorGraphController = SensorGraphController;
    var SensorGraphDirective = (function () {
        function SensorGraphDirective() {
            this.require = "sensorGraph";
            this.restrict = "A";
            this.templateUrl = "/html/sensor-graph.html";
            this.scope = {};
            this.controller = ["$scope", "$interval", "$timeout", "$uibModal", "UpdateDispatcher", SensorGraphController];
            this.link = function ($scope, element, attrs, sensorGraph) {
                sensorGraph.graphNode = element;
            };
        }
        return SensorGraphDirective;
    })();
    function SensorGraphFactory() {
        return function () { return new SensorGraphDirective(); };
    }
    Directives.SensorGraphFactory = SensorGraphFactory;
})(Directives || (Directives = {}));
/// <reference path="jquery.d.ts" />
/// <reference path="angular.d.ts" />
/// <reference path="bootstrap.d.ts" />
/// <reference path="msg2socket.ts" />
/// <reference path="updatedispatcher.ts"/>
/// <reference path="sensorvaluestore.ts" />
/// <reference path="ui-elements/numberspinner.ts"/>
/// <reference path="ui-elements/timerangespinner.ts"/>
/// <reference path="ui-elements/datetimepicker.ts"/>
/// <reference path="sensorgraph.ts"/>
"use strict";
angular.module("msgp", ['ui.bootstrap'])
    .config(function ($interpolateProvider) {
    $interpolateProvider.startSymbol("%%");
    $interpolateProvider.endSymbol("%%");
})
    .factory("WSUserClient", ["$rootScope", function ($rootScope) {
        if (!window["WebSocket"])
            throw "websocket support required";
        return new Msg2Socket.Socket($rootScope);
    }])
    .factory("UpdateDispatcher", UpdateDispatcher.UpdateDispatcherFactory)
    .directive("numberSpinner", Directives.UserInterface.NumberSpinnerFactory())
    .directive("timeRangeSpinner", Directives.UserInterface.TimeRangeSpinnerFactory())
    .directive("dateTimePicker", Directives.UserInterface.DateTimePickerFactory())
    .directive("sensorGraph", Directives.SensorGraphFactory())
    .directive("deviceEditor", [function () {
        return {
            restrict: "A",
            templateUrl: "/html/device-editor.html",
            scope: {
                device: "="
            },
            link: function (scope, element, attrs) {
            }
        };
    }])
    .directive("deviceList", ["$http", "$interval", function ($http, $interval) {
        return {
            restrict: "A",
            templateUrl: "/html/device-list.html",
            scope: {
                devices: "="
            },
            link: function (scope, element, attrs) {
                scope.showSpinner = false;
                scope.encodeURIComponent = encodeURIComponent;
                scope.deviceEditorSave = function () {
                    $http.post(scope.editedDeviceURL, scope.editedDeviceProps)
                        .success(function (data, status, headers, config) {
                        scope.devices[scope.editedDeviceId].name = scope.editedDeviceProps.name;
                        scope.devices[scope.editedDeviceId].lan = scope.editedDeviceProps.lan;
                        scope.devices[scope.editedDeviceId].wifi = scope.editedDeviceProps.wifi;
                        scope.editedDeviceId = undefined;
                        scope.errorSavingSettings = null;
                        $("#deviceEditDialog").modal('hide');
                    })
                        .error(function (data, status, headers, config) {
                        scope.errorSavingSettings = data;
                    });
                };
                var flash = function (element) {
                    element.removeClass("ng-hide");
                    $interval(function () {
                        element.addClass("ng-hide");
                    }, 3000, 1);
                };
                scope.editDev = function (e) {
                    var id = $(e.target).parents("tr[data-device-id]").first().attr("data-device-id");
                    var url = $(e.target).parents("tr[data-device-id]").first().attr("data-device-netconf-url");
                    scope.showSpinner = true;
                    $http.get(url)
                        .success(function (data, status, headers, config) {
                        scope.showSpinner = false;
                        scope.errorLoadingSettings = null;
                        scope.errorSavingSettings = null;
                        scope.editedDeviceId = id;
                        scope.editedDeviceURL = url;
                        scope.editedDeviceProps = {
                            name: scope.devices[id].name,
                            lan: data.lan || {},
                            wifi: data.wifi || {}
                        };
                        $("#deviceEditDialog").modal('show');
                    })
                        .error(function (data, status, headers, config) {
                        scope.showSpinner = false;
                        scope.errorLoadingSettings = data;
                    });
                };
                scope.remove = function (e) {
                    var url = $(e.target).parents("tr[data-device-id]").first().attr("data-device-remove-url");
                    var id = $(e.target).parents("tr[data-device-id]").first().attr("data-device-id");
                    scope.showSpinner = true;
                    $http.delete(url)
                        .success(function (data, status, headers, config) {
                        scope.showSpinner = false;
                        delete scope.devices[id];
                        flash($(e.target).parents(".device-list-").first().find(".device-deleted-"));
                    })
                        .error(function (data, status, headers, config) {
                        scope.showSpinner = false;
                        scope.error = data;
                    });
                };
                scope.editSensor = function (e) {
                    var devId = $(e.target).parents("tr[data-device-id]").first().attr("data-device-id");
                    var sensId = $(e.target).parents("tr[data-sensor-id]").first().attr("data-sensor-id");
                    var url = $(e.target).parents("tr[data-sensor-conf-url]").first().attr("data-sensor-conf-url");
                    scope.errorSavingSensor = null;
                    scope.editedSensor = {
                        name: scope.devices[devId].sensors[sensId].name,
                        confUrl: url,
                        devId: devId,
                        sensId: sensId,
                    };
                    $("#sensorEditDialog").modal('show');
                };
                scope.saveSensor = function () {
                    var props = {
                        name: scope.editedSensor.name
                    };
                    scope.showSpinner = true;
                    $http.post(scope.editedSensor.confUrl, props)
                        .success(function (data, status, headers, config) {
                        scope.showSpinner = false;
                        scope.devices[scope.editedSensor.devId].sensors[scope.editedSensor.sensId].name = props.name;
                        scope.editedSensor = null;
                        $("#sensorEditDialog").modal('hide');
                    })
                        .error(function (data, status, headers, config) {
                        scope.showSpinner = false;
                        scope.errorSavingSensor = data;
                    });
                };
            }
        };
    }])
    .controller("GraphPage", ["WSUserClient", "wsurl", "$http", "$timeout", "$uibModal", function (wsclient, wsurl, $http, $timeout, $uibModal) {
        wsclient.connect(wsurl);
        var modalInstance = null;
        wsclient.onClose(function () {
            if (modalInstance === null) {
                modalInstance = $uibModal.open({
                    size: "lg",
                    keyboard: false,
                    backdrop: 'static',
                    templateUrl: 'connection-lost.html',
                });
            }
            $timeout(function () { return wsclient.connect(wsurl); }, 1000);
        });
        wsclient.onOpen(function () {
            if (modalInstance !== null) {
                modalInstance.close();
            }
        });
    }])
    .controller("DeviceListController", ["$scope", "$http", "devices", function ($scope, $http, devices) {
        $scope.devices = devices;
        $scope.addDeviceId = "";
        $scope.addDevice = function (e) {
            var url = $(e.target).attr("data-add-device-prefix");
            $scope.errorAddingDevice = null;
            $http.post(url + encodeURIComponent($scope.addDeviceId))
                .success(function (data, status, headers, config) {
                $scope.devices[$scope.addDeviceId] = data;
                $scope.addDeviceId = null;
                $("#addDeviceDialog").modal('hide');
            })
                .error(function (data, status, headers, config) {
                $scope.errorAddingDevice = data;
            });
        };
    }]);
//# sourceMappingURL=app.js.map