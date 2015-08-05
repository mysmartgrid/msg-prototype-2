/// <reference path="angular.d.ts" />
"use strict";
var Msg2Socket;
(function (Msg2Socket) {
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
                if (this.$rootScope.$$phase == "apply") {
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
            console.log(data.cmd);
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
            this._socket = new WebSocket(url, ["v1.user.msg"]);
            this._socket.onerror = this._emitError.bind(this);
            this._socket.onclose = this._emitClose.bind(this);
            this._socket.onopen = function (e) {
                if (_this._socket.protocol != "v1.user.msg") {
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
        Socket.prototype.close = function () {
            if (this._socket) {
                this._socket.close();
                this._socket = null;
                this._isOpen = false;
            }
        };
        ;
        Socket.prototype.requestValues = function (since, withMetadata) {
            var cmd = {
                "cmd": "getValues",
                "args": {
                    "since": since,
                    "withMetadata": withMetadata
                }
            };
            this._socket.send(JSON.stringify(cmd));
        };
        ;
        Socket.prototype.requestRealtimeUpdates = function (sensors) {
            var cmd = {
                "cmd": "requestRealtimeUpdates",
                "args": sensors
            };
            this._socket.send(JSON.stringify(cmd));
        };
        ;
        return Socket;
    })();
    Msg2Socket.Socket = Socket;
    ;
})(Msg2Socket || (Msg2Socket = {}));
/// <reference path="jquery.d.ts" />
/// <reference path="angular.d.ts" />
/// <reference path="bootstrap.d.ts" />
/// <reference path="msg2socket.ts" />
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
    .directive("sensorCollectionGraph", ["$interval", function ($interval) {
        return {
            require: "^graphView",
            restrict: "A",
            templateUrl: "/html/sensor-collection-graph.html",
            scope: {
                unit: "=",
                sensors: "=",
                maxAgeMs: "=",
                assumeMissingAfterMs: "=",
            },
            link: function (scope, element, attrs, graphView) {
                var graph = new Dygraph(element.find(".sensor-graph").get(0), [[new Date()]], {
                    labels: ["Time"],
                    connectSeparatedPoints: true
                });
                var maxAgeMs = undefined;
                var assumeMissingAfterMs = undefined;
                var columns = ["Time"];
                var data = [[new Date()]];
                var sensorToIndexMap = {};
                var lastUpdateOf = {};
                var valueMissingTimeouts = {};
                var valueMissing = function (sensorID) {
                    scope.mergeDataset({
                        sensorID: [[+new Date(), NaN]]
                    });
                };
                var restartValueMissingTimeout = function (sensorID) {
                    if (sensorID === void 0) { sensorID = undefined; }
                    if (sensorID !== undefined) {
                        if (sensorID in valueMissingTimeouts)
                            $interval.cancel(valueMissingTimeouts[sensorID]);
                        if (assumeMissingAfterMs !== undefined)
                            valueMissingTimeouts[sensorID] = $interval(valueMissing, assumeMissingAfterMs, 1, true, sensorID);
                    }
                    else {
                        Object.getOwnPropertyNames(valueMissingTimeouts).forEach(restartValueMissingTimeout);
                    }
                };
                var clampToMaxAge = function () {
                    if (maxAgeMs === undefined)
                        return;
                    var now = +new Date();
                    while (data.length > 0 && (now - data[0][0]) > maxAgeMs)
                        data.shift();
                    data.unshift([new Date((+new Date()) - maxAgeMs)]);
                    while (data[0].length < columns.length)
                        data[0].push(NaN);
                };
                scope.sensorColor = {};
                scope.mergeDataset = function (set, omitUpdate) {
                    //console.log(data);
                    //console.log(set);
                    var needsSorting = false;
                    data.pop();
                    Object.getOwnPropertyNames(set).forEach(function (sensor) {
                        for (var i = 0; i < set[sensor].length; i++) {
                            var item = set[sensor][i];
                            var at = new Date(Math.floor(item[0]));
                            if (data.length > 0 && at < data[data.length - 1][0])
                                needsSorting = true;
                            var line = [at];
                            while (line.length < columns.length)
                                line.push(null);
                            if (sensor in lastUpdateOf && +at - lastUpdateOf[sensor] >= assumeMissingAfterMs) {
                                var sep = [at];
                                while (sep.length < line.length)
                                    sep.push(null);
                                sep[sensorToIndexMap[sensor]] = NaN;
                                data.push(sep);
                            }
                            lastUpdateOf[sensor] = at;
                            line[sensorToIndexMap[sensor]] = item[1];
                            data.push(line);
                        }
                        restartValueMissingTimeout(sensor);
                    });
                    if (needsSorting) {
                        data.sort(function (a, b) { return a[0] - b[0]; });
                        lastUpdateOf = {};
                        var ids = Object.getOwnPropertyNames(sensorToIndexMap);
                        var wholeSet = data;
                        data = [[]];
                        for (var i = 0; i < wholeSet.length; i++) {
                            var line = wholeSet[i];
                            for (var j = 0; j < ids.length; j++) {
                                var sensor = ids[j];
                                var val = line[sensorToIndexMap[sensor]];
                                if (typeof val == "number" && !isNaN(val)) {
                                    var obj = {};
                                    obj[sensor] = [[line[0], val]];
                                    scope.mergeDataset(obj, true);
                                }
                            }
                        }
                        graph.updateOptions({
                            file: data
                        });
                        return;
                    }
                    clampToMaxAge();
                    data.push([new Date()]);
                    while (data[data.length - 1].length < columns.length)
                        data[data.length - 1].push(NaN);
                    if (!omitUpdate) {
                        graph.updateOptions({
                            file: data
                        });
                    }
                };
                scope.$watch(attrs.maxAgeMs, function (val) {
                    maxAgeMs = val === undefined ? undefined : +val;
                });
                scope.$watch(attrs.assumeMissingAfterMs, function (val) {
                    assumeMissingAfterMs = val === undefined ? undefined : +val;
                    restartValueMissingTimeout();
                });
                scope.$watchCollection(attrs.sensors, function (val) {
                    Object.getOwnPropertyNames(sensorToIndexMap).forEach(function (id) {
                        if (sensorToIndexMap[id] == 0 || id in val)
                            return;
                        var idx = sensorToIndexMap[id];
                        columns.splice(idx, 1);
                        for (var i = 0; i < data.length; i++) {
                            data[i].splice(idx, 1);
                        }
                        delete sensorToIndexMap[id];
                        Object.getOwnPropertyNames(sensorToIndexMap).forEach(function (id) {
                            if (sensorToIndexMap[id] >= idx)
                                sensorToIndexMap[id] -= 1;
                        });
                        graph.updateOptions({
                            labels: columns,
                            file: data
                        });
                    });
                    Object.getOwnPropertyNames(val).forEach(function (id) {
                        if (id in sensorToIndexMap)
                            return;
                        var idx = Object.getOwnPropertyNames(sensorToIndexMap).length;
                        sensorToIndexMap[id] = idx + 1;
                        columns.push(val[id].name);
                        for (var i = 0; i < data.length; i++) {
                            data[i].push(null);
                        }
                        graph.updateOptions({
                            labels: columns,
                            file: data
                        });
                    });
                    Object.getOwnPropertyNames(sensorToIndexMap).forEach(function (key) {
                        scope.sensorColor[key] = graph.getColors()[sensorToIndexMap[key] - 1];
                    });
                });
                graphView.registerGraph(scope.unit, scope);
            }
        };
    }])
    .directive("graphView", ["$interval", "WSUserClient", function ($interval, wsclient) {
        return {
            restrict: "A",
            templateUrl: "/html/graph-view.html",
            scope: {
                title: "@"
            },
            controller: function ($scope) {
                var graphInstances = $scope[".graphInstances"] = {};
                this.registerGraph = function (unit, graph) {
                    graphInstances[unit] = graph;
                };
            },
            link: function (scope, element, attrs) {
                scope.sensors = {};
                scope.sensorsByUnit = {};
                var graphInstances = scope[".graphInstances"];
                var sensorKey = function (devID, sensorID) {
                    return [devID.length, devID, sensorID.length, sensorID].join();
                };
                var addGraph = function (unit) {
                    scope.sensorsByUnit[unit] = unit;
                };
                var removeGraph = function (unit) {
                    delete scope.sensorsByUnit[unit];
                    if (graphInstances[unit] !== undefined) {
                        graphInstances[unit].destroy();
                        delete graphInstances[unit];
                    }
                };
                var getGraph = function (unit) {
                    return graphInstances[unit];
                };
                var realtimeUpdateTimeout;
                var requestRealtimeUpdates = function () {
                    if (realtimeUpdateTimeout !== undefined) {
                        $interval.cancel(realtimeUpdateTimeout);
                    }
                    var sensors = {};
                    Object.getOwnPropertyNames(scope.sensors).forEach(function (id) {
                        var sens = scope.sensors[id];
                        sensors[sens[".deviceID"]] = sensors[sens[".deviceID"]] || [];
                        sensors[sens[".deviceID"]].push(sens[".sensorID"]);
                    });
                    wsclient.requestRealtimeUpdates(sensors);
                    realtimeUpdateTimeout = $interval(requestRealtimeUpdates, 30 * 1000, 1, true);
                };
                wsclient.onMetadata(function (md) {
                    Object.getOwnPropertyNames(md.devices).forEach(function (devID) {
                        var mdev = md.devices[devID];
                        Object.getOwnPropertyNames(mdev.deletedSensors || {}).forEach(function (sensorID) {
                            var key = sensorKey(devID, sensorID);
                            var unit = scope.sensors[key].unit || "";
                            delete scope.sensorsByUnit[unit][key];
                            delete scope.sensors[key];
                        });
                        Object.getOwnPropertyNames(mdev.sensors || {}).forEach(function (sensorID) {
                            var msens = mdev.sensors[sensorID];
                            var key = sensorKey(devID, sensorID);
                            if (!(key in scope.sensors)) {
                                scope.sensors[key] = {};
                            }
                            var sens = scope.sensors[key];
                            sens.name = msens.name || sens.name || "";
                            sens.unit = msens.unit || sens.unit || "";
                            sens.port = msens.port || sens.port || -1;
                            sens[".deviceID"] = devID;
                            sens[".sensorID"] = sensorID;
                            sens.id = sensorID;
                            sens.key = key;
                            scope.sensorsByUnit[sens.unit] = scope.sensorsByUnit[sens.unit] || {};
                            scope.sensorsByUnit[sens.unit][key] = sens;
                        });
                    });
                    requestRealtimeUpdates();
                });
                wsclient.onUpdate(function (data) {
                    var updatesByUnit = {};
                    Object.getOwnPropertyNames(data).forEach(function (devID) {
                        Object.getOwnPropertyNames(data[devID]).forEach(function (sensorID) {
                            var key = sensorKey(devID, sensorID);
                            if (scope.sensors[key] !== undefined) {
                                var unit = scope.sensors[key].unit;
                                updatesByUnit[unit] = updatesByUnit[unit] || {};
                                updatesByUnit[unit][key] = data[devID][sensorID];
                            }
                        });
                    });
                    Object.getOwnPropertyNames(updatesByUnit).forEach(function (unit) {
                        getGraph(unit).mergeDataset(updatesByUnit[unit]);
                    });
                });
                var onError = function (e) {
                    scope.wsConnectionFailed = true;
                };
                wsclient.onClose(onError);
                wsclient.onError(onError);
                wsclient.onOpen(function (err) {
                    if (err)
                        return;
                    wsclient.requestValues(+new Date() - 120 * 1000, true);
                });
            }
        };
    }])
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
