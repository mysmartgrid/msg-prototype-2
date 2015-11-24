/// <reference path="angular.d.ts" />
"use strict";
var Msg2Socket;
(function (Msg2Socket) {
    var ApiVersion = "v2.user.msg";
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
        Socket.prototype.requestValues = function (since, until, resolution, withMetadata) {
            var cmd = {
                cmd: "getValues",
                args: {
                    since: since,
                    until: until,
                    resolution: resolution,
                    withMetadata: withMetadata
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
})(Common || (Common = {}));
/// <reference path="utils.ts"/>
/// <reference path="common.ts"/>
/// <reference path="msg2socket.ts" />
var ExtArray = Utils.ExtArray;
var UpdateDispatcher;
(function (UpdateDispatcher_1) {
    ;
    UpdateDispatcher_1.SupportedResolutions = new Set(["raw", "second", "minute", "hour", "day", "week", "month", "year"]);
    UpdateDispatcher_1.UpdateDispatcherFactory = ["WSUserClient",
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
            _wsClient.onOpen(function (error) {
                _wsClient.onMetadata(function (metadata) { return _this._updateMetadata(metadata); });
                _wsClient.onUpdate(function (data) { return _this._updateValues(data); });
                _this._hasInitialMetadata = false;
                _this._wsClient.requestValues(0, 0, "second", true);
            });
        }
        Object.defineProperty(UpdateDispatcher.prototype, "devices", {
            get: function () {
                return this._devices;
            },
            enumerable: true,
            configurable: true
        });
        UpdateDispatcher.prototype.subscribeSensor = function (deviceId, sensorId, resolution, start, end, subscriber) {
            if (this._devices[deviceId] === undefined) {
                throw new Error("Unknown device");
            }
            if (this._devices[deviceId] === undefined) {
                throw new Error("Unknown device");
            }
            if (!UpdateDispatcher_1.SupportedResolutions.has(resolution)) {
                throw new Error("Unsupported resolution");
            }
            if (this._subscribers[deviceId][sensorId][resolution] === undefined) {
                this._subscribers[deviceId][sensorId][resolution] = new ExtArray();
            }
            this._subscribers[deviceId][sensorId][resolution].push({ start: start, end: end, subscriber: subscriber });
            if (end === null) {
                var request = {};
                request[deviceId] = {};
                request[deviceId][resolution] = [sensorId];
                this._wsClient.requestRealtimeUpdates(request);
            }
        };
        UpdateDispatcher.prototype.unsubscribeSensor = function (deviceId, sensorId, resolution, subscriber, start, end) {
            if (this._devices[deviceId] === undefined) {
                throw new Error("Unknown device");
            }
            if (this._devices[deviceId] === undefined) {
                throw new Error("Unknown device");
            }
            if (this._subscribers[deviceId][sensorId][resolution] === undefined) {
                throw new Error("No subscribers for this resolution");
            }
            if (start === undefined && end === undefined) {
                this._subscribers[deviceId][sensorId][resolution].removeWhere(function (settings) { return settings.subscriber == subscriber; });
            }
            else if (start !== undefined && end !== undefined) {
                this._subscribers[deviceId][sensorId][resolution].removeWhere(function (settings) {
                    return settings.subscriber === subscriber &&
                        settings.start === start &&
                        settings.end === end;
                });
            }
            else {
                throw new Error("Either start or end missing");
            }
        };
        UpdateDispatcher.prototype.unsubscribeAll = function (subscriber) {
            var _this = this;
            Common.forEachSensor(this._subscribers, function (deviceId, sensorId, sensor) {
                for (var resolution in sensor) {
                    _this.unsubscribeSensor(deviceId, sensorId, resolution, subscriber);
                }
            });
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
            console.log(metadata);
            for (var deviceId in metadata.devices) {
                // Create device if necessary
                if (this._devices[deviceId] === undefined) {
                    this._devices[deviceId] = {
                        name: null,
                        sensors: {}
                    };
                }
                // Add space for subscribers if necessary
                if (this._subscribers[deviceId] === undefined) {
                    this._subscribers[deviceId] = {};
                }
                var deviceName = metadata.devices[deviceId].name;
                //TODO: Redo this check as soon as we have more device metadata
                if (deviceName !== undefined && this._devices[deviceId].name !== deviceName) {
                    console.log("Device name change '" + deviceName + "' '" + this._devices[deviceId].name + "'");
                    this._devices[deviceId].name = deviceName;
                    this._emitDeviceMetadataUpdate(deviceId);
                }
                // Add or update sensors
                for (var sensorId in metadata.devices[deviceId].sensors) {
                    // Add space for subscribers
                    if (this._subscribers[deviceId][sensorId] === undefined) {
                        this._subscribers[deviceId][sensorId] = {};
                    }
                    // Add empty entry to make updateProperties work
                    if (this._devices[deviceId].sensors[sensorId] === undefined) {
                        this._devices[deviceId].sensors[sensorId] = {
                            name: null,
                            unit: null,
                            port: null,
                        };
                    }
                    // Update metatdata and inform subscribers
                    var wasUpdated = Common.updateProperties(this._devices[deviceId].sensors[sensorId], metadata.devices[deviceId].sensors[sensorId]);
                    if (wasUpdated) {
                        this._emitSensorMetadataUpdate(deviceId, sensorId);
                    }
                }
                // Delete sensors
                for (var sensorId in metadata.devices[deviceId].deletedSensors) {
                    delete this._devices[deviceId].sensors[sensorId];
                    this._emitRemoveSensor(deviceId, sensorId);
                    delete this._subscribers[deviceId][sensorId];
                }
            }
            if (!this._hasInitialMetadata) {
                this._hasInitialMetadata = true;
                for (var _i = 0, _a = this._InitialCallbacks; _i < _a.length; _i++) {
                    var callback = _a[_i];
                    callback();
                }
            }
        };
        UpdateDispatcher.prototype._emitDeviceMetadataUpdate = function (deviceId) {
            // Notify every subscriber to the devices sensors once
            var notified = new Set();
            for (var sensorId in this._subscribers[deviceId]) {
                for (var resolution in this._subscribers[deviceId][sensorId]) {
                    for (var _i = 0, _a = this._subscribers[deviceId][sensorId][resolution]; _i < _a.length; _i++) {
                        var subscriber = _a[_i].subscriber;
                        if (!notified.has(subscriber)) {
                            subscriber.updateDeviceMetadata(deviceId);
                            notified.add(subscriber);
                        }
                    }
                }
            }
        };
        UpdateDispatcher.prototype._emitSensorMetadataUpdate = function (deviceId, sensorId) {
            // Notify every subscriber to the sensor once
            var notified = new Set();
            for (var resolution in this._subscribers[deviceId][sensorId]) {
                for (var _i = 0, _a = this._subscribers[deviceId][sensorId][resolution]; _i < _a.length; _i++) {
                    var subscriber = _a[_i].subscriber;
                    if (!notified.has(subscriber)) {
                        subscriber.updateSensorMetadata(deviceId, sensorId);
                        notified.add(subscriber);
                    }
                }
            }
        };
        UpdateDispatcher.prototype._emitRemoveSensor = function (deviceId, sensorId) {
            // Notify every subscriber to the sensor once
            var notified = new Set();
            for (var resolution in this._subscribers[deviceId][sensorId]) {
                for (var _i = 0, _a = this._subscribers[deviceId][sensorId][resolution]; _i < _a.length; _i++) {
                    var subscriber = _a[_i].subscriber;
                    if (!notified.has(subscriber)) {
                        subscriber.removeSensor(deviceId, sensorId);
                        notified.add(subscriber);
                    }
                }
            }
        };
        UpdateDispatcher.prototype._updateValues = function (data) {
            var resolution = data.resolution, values = data.values;
            for (var deviceId in values) {
                for (var sensorId in values[deviceId]) {
                    for (var _i = 0, _a = values[deviceId][sensorId]; _i < _a.length; _i++) {
                        var _b = _a[_i], timestamp = _b[0], value = _b[1];
                        this._emitValueUpdate(deviceId, sensorId, resolution, timestamp, value);
                    }
                }
            }
        };
        UpdateDispatcher.prototype._emitValueUpdate = function (deviceId, sensorId, resolution, timestamp, value) {
            if (this._subscribers[deviceId][sensorId][resolution] !== undefined) {
                for (var _i = 0, _a = this._subscribers[deviceId][sensorId][resolution]; _i < _a.length; _i++) {
                    var _b = _a[_i], start = _b.start, end = _b.end, subscriber = _b.subscriber;
                    if (start <= timestamp && (end >= timestamp || end === null)) {
                        subscriber.updateValue(deviceId, sensorId, resolution, timestamp, value);
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
        DummySubscriber.prototype.updateValue = function (deviceId, sensorId, resolution, timestamp, value) {
            console.log("Update for value " + deviceId + ":" + sensorId + " " + resolution + " " + timestamp + " " + value);
        };
        DummySubscriber.prototype.updateDeviceMetadata = function (deviceId) {
            console.log("Update for device metadata " + deviceId);
        };
        DummySubscriber.prototype.updateSensorMetadata = function (deviceId, sensorId) {
            console.log("Update for sensor metadata " + deviceId + ":" + sensorId);
        };
        DummySubscriber.prototype.removeDevice = function (deviceId) {
            console.log("Removed device " + deviceId);
        };
        DummySubscriber.prototype.removeSensor = function (deviceId, sensorId) {
            console.log("Remove sensor " + deviceId + ":" + sensorId);
        };
        return DummySubscriber;
    })();
    UpdateDispatcher_1.DummySubscriber = DummySubscriber;
})(UpdateDispatcher || (UpdateDispatcher = {}));
/// <reference path="es6-shim.d.ts" />
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
            this._interval = 5 * 60 * 1000;
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
        SensorValueStore.prototype.setInterval = function (interval) {
            this._interval = interval;
        };
        SensorValueStore.prototype.setTimeout = function (timeout) {
            this._timeout = timeout;
        };
        SensorValueStore.prototype.clampData = function () {
            var oldest = (new Date()).getTime() - this._interval;
            this._series.forEach(function (series) {
                series.data = series.data.filter(function (point) {
                    return point[0] >= oldest;
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
                //Check if we need to remove a timeout in the past
                if (pos > 0 && data[pos - 1][1] === null && timestamp - data[pos - 1][0] < this._timeout) {
                    data.splice(pos - 1, 1);
                    // We delete something bevor pos, so we should move pos
                    pos -= 1;
                }
                //Check if we need to remove a timeout in the future
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
        function SensorCollectionGraphController($scope, $interval, $timeout) {
            var _this = this;
            this.$scope = $scope;
            this.$interval = $interval;
            this.$timeout = $timeout;
            this.store = new Store.SensorValueStore();
            $scope.$watch('maxAgeMs', function (interval) { return _this.store.setInterval(interval); });
            $scope.$watch('assumeMissingAfterMs', function (timeout) { return _this.store.setTimeout(timeout); });
            $scope.$watch('sensors', function () { return _this.updateSensors(); });
            $interval(function () { return _this.store.clampData(); }, 1000);
        }
        SensorCollectionGraphController.prototype.updateSensors = function () {
            var labels = this.store.getLabels();
            for (var key in this.$scope.sensors) {
                var sensor = this.$scope.sensors[key];
                if (!this.store.hasSensor(sensor.deviceID, sensor.sensorID)) {
                    this.store.addSensor(sensor.deviceID, sensor.sensorID, sensor.name);
                }
                else if (labels[sensor.deviceID][sensor.sensorID] !== sensor.name) {
                    this.store.setLabel(sensor.deviceID, sensor.sensorID, sensor.name);
                }
                else {
                    delete labels[sensor.deviceID][sensor.sensorID];
                }
            }
            for (var deviceID in labels) {
                for (var sensorID in labels[deviceID]) {
                    this.store.removeSensor(deviceID, sensorID);
                }
            }
            this.$scope.sensorColors = this.store.getColors();
            //console.log(this.$scope.sensorColors);
        };
        SensorCollectionGraphController.prototype.updateValues = function (deviceID, sensorID, timestamp, value) {
            //console.log("Update: " + deviceID + ":" + sensorID + " " + timestamp + " " + value);
            this.store.addValue(deviceID, sensorID, timestamp, value);
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
            //this.graphOptions.resolution = Math.max(1.0, window.devicePixelRatio);
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
            this.controller = ["$scope", "$interval", "$timeout", SensorCollectionGraphController];
            // Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
            this.link = function ($scope, element, attrs, controllers) {
                var graphView = controllers[0];
                var sensorCollectionGraph = controllers[1];
                sensorCollectionGraph.createGraph(element);
                graphView.registerGraph($scope.unit, sensorCollectionGraph);
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
        function GraphViewController($scope, $timeout, wsclient, updateDispatcher) {
            var _this = this;
            this.$scope = $scope;
            this.$timeout = $timeout;
            this.wsclient = wsclient;
            this.graphs = {};
            this.$scope.sensors = {};
            this.realtimeUpdateTimeout = null;
            this.wsclient.onMetadata(function (meta) {
                for (var deviceID in meta.devices) {
                    var device = meta.devices[deviceID];
                    for (var sensorID in device.sensors) {
                        var sensorMetadata = device.sensors[sensorID];
                        _this.updateSensors(deviceID, sensorID, device.name, sensorMetadata);
                    }
                    for (var deletedID in device.deletedSensors) {
                    }
                }
                _this.requestRealtimeUpdates();
            });
            this.wsclient.onUpdate(function (update) {
                var values = update.values;
                for (var deviceID in values) {
                    for (var sensorID in values[deviceID]) {
                        var unit = _this.findUnit(deviceID, sensorID);
                        values[deviceID][sensorID].forEach(function (point) {
                            // We ignore updates we don't have metadata for
                            if (_this.graphs[unit] !== undefined) {
                                _this.graphs[unit].updateValues(deviceID, sensorID, point[0], point[1]);
                            }
                        });
                    }
                }
            });
            this.wsclient.onOpen(function (err) {
                if (err) {
                    return;
                }
                var now = (new Date()).getTime();
                _this.wsclient.requestValues(now - 120 * 1000, now, "second", true); //Results in Metadata update
            });
        }
        GraphViewController.prototype.updateSensors = function (deviceID, sensorID, deviceName, meta) {
            var unit = this.findUnit(deviceID, sensorID);
            if (unit === undefined) {
                var sensor = {
                    deviceID: deviceID,
                    sensorID: sensorID,
                    deviceName: deviceName,
                    name: meta.name,
                    port: meta.port,
                    unit: meta.unit
                };
                if (this.$scope.sensors[meta.unit] === undefined) {
                    this.$scope.sensors[meta.unit] = {};
                }
                this.$scope.sensors[meta.unit][sensorKey(deviceID, sensorID)] = sensor;
            }
            else {
                var sensor = this.$scope.sensors[unit][sensorKey(deviceID, sensorID)];
                sensor.deviceName = deviceName || sensor.deviceName;
                sensor.name = meta.name || sensor.name;
                sensor.port = meta.port || sensor.port;
                sensor.unit = meta.unit || sensor.unit;
            }
        };
        GraphViewController.prototype.requestRealtimeUpdates = function () {
            var _this = this;
            if (this.realtimeUpdateTimeout !== null) {
                this.$timeout.cancel(this.realtimeUpdateTimeout);
            }
            var sensors = {};
            for (var unit in this.$scope.sensors) {
                for (var key in this.$scope.sensors[unit]) {
                    var sensor = this.$scope.sensors[unit][key];
                    if (sensors[sensor.deviceID] === undefined) {
                        sensors[sensor.deviceID] = { raw: [] };
                    }
                    sensors[sensor.deviceID]['raw'].push(sensor.sensorID);
                }
            }
            this.wsclient.requestRealtimeUpdates(sensors);
            this.realtimeUpdateTimeout = this.$timeout(function () { return _this.requestRealtimeUpdates(); }, 30 * 1000);
        };
        GraphViewController.prototype.findUnit = function (deviceID, sensorID) {
            var _this = this;
            var units = Object.keys(this.$scope.sensors);
            var unit = units.filter(function (unit) { return _this.$scope.sensors[unit][sensorKey(deviceID, sensorID)] !== undefined; });
            if (unit.length > 1) {
                throw new Error("Multiple units for sensor " + sensorKey(deviceID, sensorID));
            }
            else if (unit.length === 0) {
                return undefined;
            }
            return unit[0];
        };
        GraphViewController.prototype.registerGraph = function (unit, graph) {
            this.graphs[unit] = graph;
            //console.log(this.graphs);
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
            this.controller = ["$scope", "$timeout", "WSUserClient", "UpdateDispatcher", GraphViewController];
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
    .controller("GraphPage", ["WSUserClient", "wsurl", "$http", "UpdateDispatcher", function (wsclient, wsurl, $http, dispatcher) {
        wsclient.connect(wsurl);
        dispatcher.onInitialMetadata(function () { return dispatcher.subscribeSensor("99a1f8639246d5ae3c3c4b24026ab20b", "010bd04b2dda7fe0823e1759906e5c56", "raw", 0, null, new UpdateDispatcher.DummySubscriber()); });
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