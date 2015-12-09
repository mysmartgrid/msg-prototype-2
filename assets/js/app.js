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
                    console.log("bad packet from server", data);
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
})(Utils || (Utils = {}));
/**
 * asasd
 */
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
    // Set of all supported time resolutions for faster sanity checks
    UpdateDispatcher_1.SupportedResolutions = new Set(["raw", "second", "minute", "hour", "day", "week", "month", "year"]);
    // Angular factory function with injected dependencies
    UpdateDispatcher_1.UpdateDispatcherFactory = ["WSUserClient", "$interval",
        function (wsClient, $interval) {
            return new UpdateDispatcher(wsClient, $interval);
        }];
    /**
     * Update dispatcher class
     *
     * This class provides three functions.
     *
     * Firstly it keeps all device and sensor metadata in its device property.
     * Device metadata can be accessed using devices[deviceID].
     * Sensor metadata is stored in devices[deviceID].sensors[SensorID].
     *
     * Secondly it allows subscribtions to metadata changes and value updates.
     * There are three types subscription a fixed interval in the past,
     * a sliding window between two points relative to the current timestamp,
     * and a sliding window from a point in the past to the current timestamp,
     * which will receive the latest values directly from the device using realtime updates.
     * Historical data will be updated by polling the backend in a regularl interval.
     *
     * It is ensured by the dispatcher that each subscribe only receives each update only once,
     * even if there are several overlapping subscriptions for the same sensor.
     * The dispatcher will also try to minimize the number of requests send to the backend,
     * by requesting only one large interval covering all subscriptions for a resolution.
     *
     * All subscribers are notified of metdata data changes for sensors they subscribed to,
     * as well as metdata changes for the devices these sensors are attached to.
     * It is the up to the subscriber to check the devices property for the updated metadata and
     * process it accordingly.
     *
     * Thirdly, since it is not possible to subscribe to any sensor before the dispatcher has received its initial metadata,
     * it provides a callback mechanism using the onInitialMetadata method to execute inital subscriptions,
     * as soon as the metadata is available.
     */
    var UpdateDispatcher = (function () {
        /**
         * Construtor for UpdateDispatcher
         * Should not be called directly, use the factory to register an angular service instead
         *
         * Initalizes private members
         * Registers _updateMetadata and _updateValues as callbacks for the Msg2Socket
         * Requests initial metadata as soon as the socket is connected
         * Sets up an $interval instance for polling historical data usinf _pollHistoryData
         */
        function UpdateDispatcher(_wsClient, $interval) {
            var _this = this;
            this._wsClient = _wsClient;
            this.$interval = $interval;
            this._devices = {};
            this._subscribers = {};
            this._InitialCallbacks = new Array();
            _wsClient.onOpen(function (error) {
                _wsClient.onMetadata(function (metadata) { return _this._updateMetadata(metadata); });
                _wsClient.onUpdate(function (data) { return _this._updateValues(data); });
                _this._hasInitialMetadata = false;
                _this._wsClient.requestMetadata();
                $interval(function () { return _this._pollHistoryData(); }, 5 * 60 * 1000);
            });
        }
        Object.defineProperty(UpdateDispatcher.prototype, "devices", {
            // Accesor to prevent write access to the metadata
            get: function () {
                return this._devices;
            },
            enumerable: true,
            configurable: true
        });
        /**
         * Subscribe for value updates with in a fixed interval from start to end.
         * Start and end are millisecond timestamps.
         */
        UpdateDispatcher.prototype.subscribeInterval = function (deviceID, sensorID, resolution, start, end, subscriber) {
            this._subscribeSensor(deviceID, sensorID, resolution, false, start, end, subscriber);
        };
        /**
         * Subscribe for value updates with in a slinding window from current_timestamp - start to current_timestamp  - end.
         * Start and end are in milliseconds.
         */
        UpdateDispatcher.prototype.subscribeSlidingWindow = function (deviceID, sensorID, resolution, start, end, subscriber) {
            this._subscribeSensor(deviceID, sensorID, resolution, true, start, end, subscriber);
        };
        ;
        /**
         * Subscribe for value updates with in a slinding window from current_timestamp - start to current_timestamp.
         * Subscribers using this subscrition also get forwarded realtime updates from the metering device
         * Start and end are in milliseconds.
         */
        UpdateDispatcher.prototype.subscribeRealtimeSlidingWindow = function (deviceID, sensorID, resolution, start, subscriber) {
            this._subscribeSensor(deviceID, sensorID, resolution, true, start, 0, subscriber);
        };
        ;
        /**
         * Internal handler for all types of subscrition.
         * There a three valid combinations of paramaters for this method.
         * Fixed Interval:
         *  slidingWindow: false,
         *  start: timestamp start,
         *  end: timestamp end
         *
         * Sliding window:
         *  slidingWindow: true,
         *  start: how many milliseconds back the window should start
         *  end: how many milliseconds back the window should end
         *
         * Sliding window for realtime updates:
         *  slidingWindow: true,
         *  start: how many milliseconds back the window should start
         *  end: 0 (window always end at the current timestamp)
         *
         */
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
            // Request history
            var now = Common.now();
            if (slidingWindow) {
                start = now - start;
                end = now - end;
            }
            var sensorsList = {};
            sensorsList[deviceID] = [sensorID];
            this._wsClient.requestValues(start, end, resolution, sensorsList);
        };
        // Shorthand to remove all subscribtions for a given subscriber
        UpdateDispatcher.prototype.unsubscribeAll = function (subscriber) {
            var _this = this;
            Common.forEachSensor(this._subscribers, function (deviceID, sensorID, sensor) {
                for (var resolution in sensor) {
                    _this.unsubscribeSensor(deviceID, sensorID, resolution, subscriber);
                }
            });
        };
        // Shorthand to remove all subscribtions to sensor and resoltion for a specific subscriber
        UpdateDispatcher.prototype.unsubscribeSensor = function (deviceID, sensorID, resolution, subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber);
        };
        /**
         * Unsubscribe for value updates with in a fixed interval from start to end.
         * Start and end are millisecond timestamps.
         */
        UpdateDispatcher.prototype.unsubscribeInterval = function (deviceID, sensorID, resolution, start, end, subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber, false, start, end);
        };
        /**
         * Unsubscribe for value updates with in a slinding window from current_timestamp - start to current_timestamp  - end.
         * Start and end are in milliseconds.
         */
        UpdateDispatcher.prototype.unsubscribeSlidingWindow = function (deviceID, sensorID, resolution, start, end, subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber, true, start, end);
        };
        ;
        /**
         * Unsubscribe for value updates with in a slinding window from current_timestamp - start to current_timestamp.
         * Start and end are in milliseconds.
         */
        UpdateDispatcher.prototype.unsubscribeRealtimeSlidingWindow = function (deviceID, sensorID, resolution, start, subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber, true, start, 0);
        };
        ;
        /**
         * Internal method to remove a subscribtion given by start, end, slidingWindow,
         * resolution and sensor for a specific subscriber.
         * If start, end and slidingWindow are missing all subscribtions for the sensor and resolution.
         */
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
        /**
         * Register callbacks which will be called as soon as metadata is avaiable.
         * Usefull for doing inital subscriptions.
         * If metadata is already avaiable the callback will be execute immediately.
         */
        UpdateDispatcher.prototype.onInitialMetadata = function (callback) {
            if (!this._hasInitialMetadata) {
                this._InitialCallbacks.push(callback);
            }
            else {
                callback();
            }
        };
        /**
         * Internal method which is called by the Msg2Socket in case of metadata updates.
         * Updates _devices and calls subscribers accordingly using _emitDeviceMetadataUpdate amd _emitSensorMetadataUpdate.
         */
        UpdateDispatcher.prototype._updateMetadata = function (metadata) {
            for (var deviceID in metadata.devices) {
                // Create device if necessary
                if (this._devices[deviceID] === undefined) {
                    this._devices[deviceID] = {
                        name: null,
                        sensors: {}
                    };
                }
                // Add space for subscribers if necessary
                if (this._subscribers[deviceID] === undefined) {
                    this._subscribers[deviceID] = {};
                }
                var deviceName = metadata.devices[deviceID].name;
                //TODO: Redo this check as soon as we have more device metadata
                if (deviceName !== undefined && this._devices[deviceID].name !== deviceName) {
                    this._devices[deviceID].name = deviceName;
                    this._emitDeviceMetadataUpdate(deviceID);
                }
                // Add or update sensors
                for (var sensorID in metadata.devices[deviceID].sensors) {
                    // Add space for subscribers
                    if (this._subscribers[deviceID][sensorID] === undefined) {
                        this._subscribers[deviceID][sensorID] = {};
                    }
                    // Add empty entry to make updateProperties work
                    if (this._devices[deviceID].sensors[sensorID] === undefined) {
                        this._devices[deviceID].sensors[sensorID] = {
                            name: null,
                            unit: null,
                            port: null,
                        };
                    }
                    // Update metatdata and inform subscribers
                    var wasUpdated = Common.updateProperties(this._devices[deviceID].sensors[sensorID], metadata.devices[deviceID].sensors[sensorID]);
                    if (wasUpdated) {
                        this._emitSensorMetadataUpdate(deviceID, sensorID);
                    }
                }
                // Delete sensors
                for (var sensorID in metadata.devices[deviceID].deletedSensors) {
                    delete this._devices[deviceID].sensors[sensorID];
                    this._emitRemoveSensor(deviceID, sensorID);
                    delete this._subscribers[deviceID][sensorID];
                }
            }
            // Excute the callbacks if this is the initial metadata update
            if (!this._hasInitialMetadata) {
                this._hasInitialMetadata = true;
                for (var _i = 0, _a = this._InitialCallbacks; _i < _a.length; _i++) {
                    var callback = _a[_i];
                    callback();
                }
            }
        };
        /**
         * Notify all subscribers to all sensors in all resolutions for this device of the update.
         * A set is used to ensure each subscriber is notified exactly once.
         */
        UpdateDispatcher.prototype._emitDeviceMetadataUpdate = function (deviceID) {
            // Notify every subscriber to the devices sensors once
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
        /**
         * Notify all subscribers to a sensors in all resolutions of the update.
         * A set is used to ensure each subscriber is notified exactly once.
         */
        UpdateDispatcher.prototype._emitSensorMetadataUpdate = function (deviceID, sensorID) {
            // Notify every subscriber to the sensor once
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
        /**
         * Notify all subscribers to a sensors in all resolutions.
         * A set is used to ensure each subscriber is notified exactly once.
         */
        UpdateDispatcher.prototype._emitRemoveSensor = function (deviceID, sensorID) {
            // Notify every subscriber to the sensor once
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
        /**
         * Request historical data for all subscriptions from the backend.
         * In order to minimize the number of requests to the backend,
         * only one reuqest per resoltion covering all subscribed sensors and intervals is generated.
         * The _updateValues method takes care of dropping unecessary values and dispatching the rest to the subscribers.
         */
        UpdateDispatcher.prototype._pollHistoryData = function () {
            var requests;
            requests = {};
            var now = Common.now();
            // Gather start, end and sensors for each resolution
            Common.forEachSensor(this._subscribers, function (deviceID, sensorID, map) {
                for (var resolution in map) {
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
                        //Adjust start and end of interval
                        requests[resolution].start = Math.min(start, requests[resolution].start);
                        requests[resolution].end = Math.max(end, requests[resolution].end);
                        if (requests[resolution].sensors[deviceID] === undefined) {
                            requests[resolution].sensors[deviceID] = new Set();
                        }
                        requests[resolution].sensors[deviceID].add(sensorID);
                    }
                }
            });
            // Send out the requests
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
        /**
         * Internal method which is called by the Msg2Socket in case of value updates.
         * Simply unpacks the update and calls _emitValueUpdate for each value.
         */
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
        /**
         * Internal methode called once from _updateValues for each value timestamp pair.
         * Matches the subscription interval of each subscripton for the sensor and resolution against the updates timestamps.
         * Also maintains a set of already notified subscribers to avoid notifying a subscriber twices in case of overlapping subscriptons.
         */
        UpdateDispatcher.prototype._emitValueUpdate = function (deviceID, sensorID, resolution, timestamp, value) {
            var now = Common.now();
            var notified = new Set();
            // Make sure we have subscribsers for this sensor
            if (this._subscribers[deviceID][sensorID][resolution] !== undefined) {
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
    // Dummy subscriber that dumps all updates to console.
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
            this._sensorLabels = {};
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
                //Series should not start or end with null after clamping
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
        SensorValueStore.prototype.addSensor = function (deviceId, sensorId, label) {
            if (this.hasSensor(deviceId, sensorId)) {
                throw new Error("Sensor has been added already");
            }
            var index = this._series.length;
            if (this._sensorMap[deviceId] === undefined) {
                this._sensorMap[deviceId] = {};
                this._sensorLabels[deviceId] = {};
            }
            this._sensorMap[deviceId][sensorId] = index;
            this._sensorLabels[deviceId][sensorId] = label;
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
            delete this._sensorLabels[deviceId][sensorId];
        };
        SensorValueStore.prototype.setLabel = function (deviceId, sensorId, label) {
            if (!this.hasSensor(deviceId, sensorId)) {
                throw new Error("No such sensor");
            }
            this._sensorLabels[deviceId][sensorId] = label;
        };
        SensorValueStore.prototype.addValue = function (deviceId, sensorId, timestamp, value) {
            var seriesIndex = this._getSensorIndex(deviceId, sensorId);
            if (seriesIndex === -1) {
                throw new Error("No such sensor");
            }
            // Find position for inserting
            var data = this._series[seriesIndex].data;
            var pos = data.findIndex(function (point) {
                return point[0] > timestamp;
            });
            if (pos === -1) {
                pos = data.length;
            }
            // Check if the value is an update for an existing timestamp
            if (data.length > 0 && pos === 0 && data[0][0] === timestamp) {
                // Update for the first tuple
                data[0][1] = value;
            }
            else if (data.length > 0 && pos > 0 && pos <= data.length && data[pos - 1][0] === timestamp) {
                //Update any other tuple including the last one
                data[pos - 1][1] = value;
            }
            else {
                // Insert
                data.splice(pos, 0, [timestamp, value]);
                //Check if we need to remove a null in the past
                if (pos > 0 && data[pos - 1][1] === null && timestamp - data[pos - 1][0] < this._timeout) {
                    data.splice(pos - 1, 1);
                    // We delete something bevor pos, so we should move pos
                    pos -= 1;
                }
                //Check if we need to remove a null in the future
                if (pos < data.length - 1 && data[pos + 1][1] === null && data[pos + 1][0] - timestamp < this._timeout) {
                    data.splice(pos + 1, 1);
                }
                //Check if a null in the past is needed
                if (pos > 0 && data[pos - 1][1] !== null && timestamp - data[pos - 1][0] >= this._timeout) {
                    data.splice(pos, 0, [timestamp - 1, null]);
                }
                //Check if a null in the future is needed
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
        SensorValueStore.prototype.getLabels = function () {
            var labels = {};
            for (var deviceId in this._sensorLabels) {
                labels[deviceId] = {};
                for (var sensorId in this._sensorLabels[deviceId]) {
                    labels[deviceId][sensorId] = this._sensorLabels[deviceId][sensorId];
                }
            }
            return labels;
        };
        return SensorValueStore;
    })();
    Store.SensorValueStore = SensorValueStore;
})(Store || (Store = {}));
/// <reference path="angular.d.ts" />
/// <reference path="msg2socket.ts" />
/// <reference path="sensorvaluestore.ts" />
/// <reference path="graphview.ts" />
"use strict";
var Directives;
(function (Directives) {
    var SensorCollectionGraphController = (function () {
        function SensorCollectionGraphController($scope, $interval, $timeout, _dispatcher) {
            var _this = this;
            this.$scope = $scope;
            this.$interval = $interval;
            this.$timeout = $timeout;
            this._dispatcher = _dispatcher;
            this.store = new Store.SensorValueStore();
            this.store.setSlidingWindowMode(true);
            this.store.setEnd(0);
            $scope.$watch('maxAgeMs', function (start) { return _this.store.setStart(start); });
            $scope.$watch('assumeMissingAfterMs', function (timeout) { return _this.store.setTimeout(timeout); });
            $scope.$watch('sensors', function () { return _this.updateSensors(); });
            $interval(function () { return _this.store.clampData(); }, 1000);
        }
        SensorCollectionGraphController.prototype.updateValue = function (deviceID, sensorID, resolution, timestamp, value) {
            this.store.addValue(deviceID, sensorID, timestamp, value);
        };
        SensorCollectionGraphController.prototype.updateDeviceMetadata = function (deviceID) { };
        ;
        SensorCollectionGraphController.prototype.updateSensorMetadata = function (deviceID, sensorID) { };
        ;
        SensorCollectionGraphController.prototype.removeDevice = function (deviceID) { };
        ;
        SensorCollectionGraphController.prototype.removeSensor = function (deviceID, sensorID) { };
        ;
        SensorCollectionGraphController.prototype.updateSensors = function () {
            var labels = this.store.getLabels();
            for (var _i = 0, _a = this.$scope.sensors; _i < _a.length; _i++) {
                var sensor = _a[_i];
                var metadata = this._dispatcher.devices[sensor.deviceID].sensors[sensor.sensorID];
                if (!this.store.hasSensor(sensor.deviceID, sensor.sensorID)) {
                    this.store.addSensor(sensor.deviceID, sensor.sensorID, metadata.name);
                    this._dispatcher.subscribeRealtimeSlidingWindow(sensor.deviceID, sensor.sensorID, "raw", this.$scope.maxAgeMs, this);
                }
                else if (labels[sensor.deviceID][sensor.sensorID] !== metadata.name) {
                    this.store.setLabel(sensor.deviceID, sensor.sensorID, metadata.name);
                }
                else {
                    delete labels[sensor.deviceID][sensor.sensorID];
                }
            }
            for (var deviceID in labels) {
                for (var sensorID in labels[deviceID]) {
                    this.store.removeSensor(deviceID, sensorID);
                    this._dispatcher.unsubscribeSensor(deviceID, sensorID, "raw", this);
                }
            }
            this.$scope.sensorColors = this.store.getColors();
        };
        SensorCollectionGraphController.prototype.createGraph = function (element) {
            this.graphOptions = {
                xaxis: {
                    mode: 'time',
                    timeMode: 'local',
                    title: 'Uhrzeit'
                },
                HtmlText: false,
                preventDefault: false,
                title: 'Messwerte [' + this.$scope.unit + ']',
                shadowSize: 0,
                lines: {
                    lineWidth: 2,
                }
            };
            this.graphNode = element.find(".sensor-graph").get(0);
            this.redrawGraph();
        };
        SensorCollectionGraphController.prototype.redrawGraph = function () {
            var _this = this;
            var time = (new Date()).getTime();
            this.graphOptions.xaxis.max = time - 1000;
            this.graphOptions.xaxis.min = time - this.$scope.maxAgeMs + 1000;
            var graph = Flotr.draw(this.graphNode, this.store.getData(), this.graphOptions);
            var delay = (this.$scope.maxAgeMs - 2000) / graph.plotWidth;
            this.$timeout(function () { return _this.redrawGraph(); }, delay);
        };
        return SensorCollectionGraphController;
    })();
    Directives.SensorCollectionGraphController = SensorCollectionGraphController;
    var SensorCollectionGraphDirective = (function () {
        function SensorCollectionGraphDirective() {
            this.require = ["^graphView", "sensorCollectionGraph"];
            this.restrict = "A";
            this.templateUrl = "/html/sensor-collection-graph.html";
            this.scope = {
                unit: "=",
                sensors: "=",
                maxAgeMs: "=",
                assumeMissingAfterMs: "=",
            };
            this.controller = ["$scope", "$interval", "$timeout", "UpdateDispatcher", SensorCollectionGraphController];
            // Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
            this.link = function ($scope, element, attrs, controllers) {
                var graphView = controllers[0];
                var sensorCollectionGraph = controllers[1];
                sensorCollectionGraph.createGraph(element);
            };
        }
        return SensorCollectionGraphDirective;
    })();
    function SensorCollectionGraphFactory() {
        return function () { return new SensorCollectionGraphDirective(); };
    }
    Directives.SensorCollectionGraphFactory = SensorCollectionGraphFactory;
})(Directives || (Directives = {}));
/// <reference path="angular.d.ts" />
/// <reference path="msg2socket.ts" />
/// <reference path="updatedispatcher.ts"/>
/// <reference path="sensorvaluestore.ts" />
/// <reference path="sensorcollectiongraph.ts" />
"use strict";
var Directives;
(function (Directives) {
    function sensorKey(deviceID, sensorID) {
        return deviceID + ':' + sensorID;
    }
    Directives.sensorKey = sensorKey;
    var GraphViewController = (function () {
        function GraphViewController($scope, $timeout, dispatcher) {
            var _this = this;
            this.$scope = $scope;
            this.$timeout = $timeout;
            this.dispatcher = dispatcher;
            this.$scope.sensorsByUnit = {};
            $scope.$watch('devices', function () { return _this._generateSensorByUnit(); });
            dispatcher.onInitialMetadata(function () { return _this.$scope.devices = dispatcher.devices; });
        }
        GraphViewController.prototype._generateSensorByUnit = function () {
            for (var deviceID in this.$scope.devices) {
                for (var sensorID in this.$scope.devices[deviceID].sensors) {
                    var sensor = this.$scope.devices[deviceID].sensors[sensorID];
                    if (this.$scope.sensorsByUnit[sensor.unit] === undefined) {
                        this.$scope.sensorsByUnit[sensor.unit] = [];
                    }
                    this.$scope.sensorsByUnit[sensor.unit].push({ deviceID: deviceID, sensorID: sensorID });
                }
            }
        };
        return GraphViewController;
    })();
    Directives.GraphViewController = GraphViewController;
    var GraphViewDirective = (function () {
        function GraphViewDirective() {
            this.restrict = "A";
            this.templateUrl = "/html/graph-view.html";
            this.scope = {
                title: "@"
            };
            // Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
            this.link = function ($scope, element, attrs, controller) {
            };
            this.controller = ["$scope", "$timeout", "UpdateDispatcher", GraphViewController];
        }
        ;
        return GraphViewDirective;
    })();
    function GraphViewFactory() {
        return function () { return new GraphViewDirective(); };
    }
    Directives.GraphViewFactory = GraphViewFactory;
})(Directives || (Directives = {}));
/// <reference path="jquery.d.ts" />
/// <reference path="angular.d.ts" />
/// <reference path="bootstrap.d.ts" />
/// <reference path="msg2socket.ts" />
/// <reference path="updatedispatcher.ts"/>
/// <reference path="sensorvaluestore.ts" />
/// <reference path="graphview.ts" />
/// <reference path="sensorcollectiongraph.ts" />
"use strict";
angular.module("msgp", [])
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
    .directive("sensorCollectionGraph", Directives.SensorCollectionGraphFactory())
    .directive("graphView", Directives.GraphViewFactory())
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
    .controller("GraphPage", ["WSUserClient", "wsurl", "$http", "UpdateDispatcher", function (wsclient, wsurl, $http) {
        wsclient.connect(wsurl);
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