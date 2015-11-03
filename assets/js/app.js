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
            for (var i in handlers) {
                if (this.$rootScope.$$phase === "apply" || this.$rootScope.$$phase === "$digest") {
                    handlers[i](param);
                }
                else {
                    this.$rootScope.$apply(function (scope) {
                        handlers[i](param);
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
            console.log(pos);
            console.log(data);
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
            console.log(this.$scope.sensorColors);
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
        function GraphViewController($scope, $timeout, wsclient) {
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
            console.log(this.graphs);
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
            this.controller = ["$scope", "$timeout", "WSUserClient", GraphViewController];
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
    .controller("GraphPage", ["WSUserClient", "wsurl", "$http", function (wsclient, wsurl, $http) {
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