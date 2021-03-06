(function e(t,n,r){function s(o,u){if(!n[o]){if(!t[o]){var a=typeof require=="function"&&require;if(!u&&a)return a(o,!0);if(i)return i(o,!0);var f=new Error("Cannot find module '"+o+"'");throw f.code="MODULE_NOT_FOUND",f}var l=n[o]={exports:{}};t[o][0].call(l.exports,function(e){var n=t[o][1][e];return s(n?n:e)},l,l.exports,e,t,n,r)}return n[o].exports}var i=typeof require=="function"&&require;for(var o=0;o<r.length;o++)s(r[o]);return s})({1:[function(require,module,exports){
"use strict";
var Msg2Socket = require('./lib/msg2socket');
var UpdateDispatcher = require('./lib/updatedispatcher');
var numberspinner_1 = require('./directives/ui-elements/numberspinner');
var timerangespinner_1 = require('./directives/ui-elements/timerangespinner');
var datetimepicker_1 = require('./directives/ui-elements/datetimepicker');
var sensorgraph_1 = require('./directives/sensorgraph');
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
    .directive("numberSpinner", numberspinner_1.default())
    .directive("timeRangeSpinner", timerangespinner_1.default())
    .directive("dateTimePicker", datetimepicker_1.default())
    .directive("sensorGraph", sensorgraph_1.default())
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
        $scope.openAddDeviceModal = function () {
            $scope.addDeviceId = "";
            $('#addDeviceDialog').modal();
        };
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

},{"./directives/sensorgraph":2,"./directives/ui-elements/datetimepicker":3,"./directives/ui-elements/numberspinner":4,"./directives/ui-elements/timerangespinner":5,"./lib/msg2socket":8,"./lib/updatedispatcher":10}],2:[function(require,module,exports){
"use strict";
var __extends = (this && this.__extends) || function (d, b) {
    for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p];
    function __() { this.constructor = d; }
    d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
};
var Utils = require('../lib/utils');
var Store = require('../lib/sensorvaluestore');
var common_1 = require('../lib/common');
var Widget = require('./widget');
var SensorGraphSettingsFactory = ["$scope", "$uibModalInstance", "UpdateDispatcher", "config",
    function ($scope, $uibModalInstance, dispatcher, config) {
        return new SensorGraphSettingsController($scope, $uibModalInstance, dispatcher, config);
    }];
var SensorGraphSettingsController = (function (_super) {
    __extends(SensorGraphSettingsController, _super);
    function SensorGraphSettingsController($scope, $uibModalInstance, _dispatcher, config) {
        _super.call(this, $scope, $uibModalInstance, _dispatcher, config);
        this.$scope = $scope;
        this.$uibModalInstance = $uibModalInstance;
        this._dispatcher = _dispatcher;
        $scope.$watch("config.mode", function () {
            var mode = $scope.config.mode;
            if ($scope.resolutions[mode].indexOf($scope.config.resolution) === -1) {
                $scope.config.resolution = $scope.resolutions[mode][0];
            }
            if (mode === 'realtime') {
                $scope.config.resolution = 'raw';
            }
        });
    }
    SensorGraphSettingsController.prototype._checkConfig = function () {
        return true;
    };
    return SensorGraphSettingsController;
}(Widget.WidgetSettingsController));
var SensorGraphController = (function (_super) {
    __extends(SensorGraphController, _super);
    function SensorGraphController($interval, $timeout, $scope, $uibModal, _dispatcher) {
        var _this = this;
        _super.call(this, $scope, _dispatcher, $uibModal);
        this.$interval = $interval;
        this.$timeout = $timeout;
        this.$scope = $scope;
        this.$uibModal = $uibModal;
        this._dispatcher = _dispatcher;
        this._store = new Store.SensorValueStore();
        this._store.setSlidingWindowMode(true);
        this._store.setEnd(0);
        this._dispatcher.onInitialMetadata(function () {
            _this._setDefaultConfig();
            _this._redrawGraph();
        });
        this._settingsTemplate = 'sensor-graph-settings.html';
        this._settingsControllerFactory = SensorGraphSettingsFactory;
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
    SensorGraphController.prototype._setDefaultConfig = function () {
        this._applyConfig({
            unit: this._dispatcher.units[0],
            resolution: common_1.SupportedResolutions[0],
            sensors: [],
            mode: 'realtime',
            intervalStart: Utils.now() - 24 * 60 * 1000,
            intervalEnd: Utils.now(),
            windowStart: 5 * 60 * 1000,
            windowEnd: 0
        });
    };
    SensorGraphController.prototype._subscribeSensor = function (config, deviceID, sensorID) {
        if (config.mode === 'realtime') {
            this._dispatcher.subscribeRealtimeSlidingWindow(deviceID, sensorID, config.windowStart, this);
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
    SensorGraphController.prototype._applyConfig = function (config) {
        if (this._config !== undefined &&
            config.mode === this._config.mode &&
            config.resolution == this._config.resolution &&
            config.unit === this._config.unit &&
            config.windowStart === this._config.windowStart &&
            config.windowEnd === this._config.windowEnd &&
            config.intervalStart === this._config.intervalStart &&
            config.intervalEnd === this._config.intervalEnd) {
            var addedSensors = Utils.difference(config.sensors, this._config.sensors, common_1.sensorEqual);
            var removedSensors = Utils.difference(this._config.sensors, config.sensors, common_1.sensorEqual);
            console.log(addedSensors);
            console.log(removedSensors);
            for (var _i = 0, addedSensors_1 = addedSensors; _i < addedSensors_1.length; _i++) {
                var _a = addedSensors_1[_i], deviceID = _a.deviceID, sensorID = _a.sensorID;
                this._subscribeSensor(config, deviceID, sensorID);
                this._store.addSensor(deviceID, sensorID);
            }
            for (var _b = 0, removedSensors_1 = removedSensors; _b < removedSensors_1.length; _b++) {
                var _c = removedSensors_1[_b], deviceID = _c.deviceID, sensorID = _c.sensorID;
                this._dispatcher.unsubscribeSensor(deviceID, sensorID, config.resolution, this);
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
        this._store.setTimeout(common_1.ResoltuionToMillisecs[config.resolution] * 60);
        this._config = config;
        this.$scope.sensorColors = this._store.getColors();
        this.$scope.sensors = config.sensors;
        this._redrawGraph();
    };
    SensorGraphController.prototype._redrawGraph = function () {
        var _this = this;
        this.$timeout.cancel(this._timeout);
        var time = Utils.now();
        var graphOptions = {
            xaxis: {
                mode: 'time',
                timeMode: 'local',
                title: 'Time',
                noTicks: 15,
                minorTickFreq: 1
            },
            yaxis: {
                min: 0,
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
}(Widget.WidtgetController));
exports.SensorGraphController = SensorGraphController;
var SensorGraphDirective = (function () {
    function SensorGraphDirective() {
        this.require = "sensorGraph";
        this.restrict = "A";
        this.templateUrl = "/html/sensor-graph.html";
        this.scope = {};
        this.controller = ["$interval", "$timeout", "$scope", "$uibModal", "UpdateDispatcher", SensorGraphController];
        this.link = function ($scope, element, attrs, sensorGraph) {
            sensorGraph.graphNode = element;
        };
    }
    return SensorGraphDirective;
}());
function SensorGraphFactory() {
    return function () { return new SensorGraphDirective(); };
}
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = SensorGraphFactory;

},{"../lib/common":7,"../lib/sensorvaluestore":9,"../lib/utils":11,"./widget":6}],3:[function(require,module,exports){
"use strict";
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
        var pickerModes = {
            'raw': 'day',
            'second': 'day',
            'minute': 'day',
            'hour': 'day',
            'day': 'day',
            'week': 'day',
            'month': 'month',
            'year': 'year'
        };
        $scope.pickerOptions = {};
        $scope.disableTimepicker = false;
        $scope.$watch('resolution', function () {
            $scope.pickerOptions['datepickerMode'] = pickerModes[$scope.resolution];
            $scope.pickerOptions['minMode'] = pickerModes[$scope.resolution];
            $scope.disableTimepicker = !($scope.resolution == 'raw' || $scope.resolution == "second" || $scope.resolution == "hour");
        });
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
}());
function DateTimePickerFactory() {
    return function () { return new DateTimePickerDirective(); };
}
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = DateTimePickerFactory;
var DateTimePickerDirective = (function () {
    function DateTimePickerDirective() {
        this.restrict = "A";
        this.templateUrl = "/html/date-time-picker.html";
        this.scope = {
            ngModel: '=?',
            ngChange: '&',
            resolution: '=',
            min: '=?',
            max: '=?'
        };
        this.controller = ["$scope", DateTimePickerController];
        this.link = function ($scope, element, attrs, aateTimePicker) {
        };
    }
    return DateTimePickerDirective;
}());

},{}],4:[function(require,module,exports){
"use strict";
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
}());
function NumberSpinnerFactory() {
    return function () { return new NumberSpinnerDirective(); };
}
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = NumberSpinnerFactory;
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
}());

},{}],5:[function(require,module,exports){
"use strict";
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
            for (var _i = 0, TimeUnits_1 = TimeUnits; _i < TimeUnits_1.length; _i++) {
                var unit = TimeUnits_1[_i];
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
        for (var _i = 0, TimeUnits_2 = TimeUnits; _i < TimeUnits_2.length; _i++) {
            var unit = TimeUnits_2[_i];
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
}());
function TimeRangeSpinnerFactory() {
    return function () { return new TimeRangeSpinnerDirective(); };
}
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = TimeRangeSpinnerFactory;
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
}());

},{}],6:[function(require,module,exports){
"use strict";
var Utils = require('../lib/utils');
var common_1 = require('../lib/common');
;
var WidtgetController = (function () {
    function WidtgetController($scope, _dispatcher, $uibModal) {
        var _this = this;
        this.$scope = $scope;
        this._dispatcher = _dispatcher;
        this.$uibModal = $uibModal;
        $scope.devices = this._dispatcher.devices;
        $scope.units = _dispatcher.units;
        $scope.sensorsByUnit = _dispatcher.sensorsByUnit;
        $scope.openSettings = function () { return _this._openSettings(); };
    }
    WidtgetController.prototype._openSettings = function () {
        var _this = this;
        var modalInstance = this.$uibModal.open({
            controller: this._settingsControllerFactory,
            size: "lg",
            templateUrl: this._settingsTemplate,
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
    WidtgetController.prototype.updateValue = function (deviceID, sensorID, resolution, timestamp, value) { };
    ;
    WidtgetController.prototype.updateDeviceMetadata = function (deviceID) { };
    ;
    WidtgetController.prototype.updateSensorMetadata = function (deviceID, sensorID) { };
    ;
    WidtgetController.prototype.removeDevice = function (deviceID) { };
    ;
    WidtgetController.prototype.removeSensor = function (deviceID, sensorID) { };
    ;
    return WidtgetController;
}());
exports.WidtgetController = WidtgetController;
var WidgetSettingsController = (function () {
    function WidgetSettingsController($scope, $uibModalInstance, _dispatcher, config) {
        var _this = this;
        this.$scope = $scope;
        this.$uibModalInstance = $uibModalInstance;
        this._dispatcher = _dispatcher;
        $scope.devices = _dispatcher.devices;
        $scope.units = _dispatcher.units;
        $scope.sensorsByUnit = _dispatcher.sensorsByUnit;
        $scope.resolutions = common_1.ResolutionsPerMode;
        $scope.config = config;
        $scope.ok = function () { return _this._saveConfig(); };
        $scope.cancel = function () { return _this._close(); };
    }
    WidgetSettingsController.prototype._saveConfig = function () {
        if (this._checkConfig()) {
            this.$uibModalInstance.close(this.$scope.config);
        }
    };
    WidgetSettingsController.prototype._close = function () {
        this.$uibModalInstance.dismiss('cancel');
    };
    return WidgetSettingsController;
}());
exports.WidgetSettingsController = WidgetSettingsController;

},{"../lib/common":7,"../lib/utils":11}],7:[function(require,module,exports){
"use strict";
;
;
exports.SupportedResolutions = ["raw", "second", "minute", "hour", "day", "week", "month", "year"];
exports.ResolutionsPerMode = {
    "interval": exports.SupportedResolutions,
    "slidingWindow": exports.SupportedResolutions.filter(function (res) { return res !== "raw"; }),
    "realtime": ["raw"]
};
(function (ResoltuionToMillisecs) {
    ResoltuionToMillisecs[ResoltuionToMillisecs["raw"] = 1000] = "raw";
    ResoltuionToMillisecs[ResoltuionToMillisecs["second"] = 1000] = "second";
    ResoltuionToMillisecs[ResoltuionToMillisecs["minute"] = 60000] = "minute";
    ResoltuionToMillisecs[ResoltuionToMillisecs["hour"] = 3600000] = "hour";
    ResoltuionToMillisecs[ResoltuionToMillisecs["day"] = 86400000] = "day";
    ResoltuionToMillisecs[ResoltuionToMillisecs["week"] = 604800000] = "week";
    ResoltuionToMillisecs[ResoltuionToMillisecs["month"] = 2678400000] = "month";
    ResoltuionToMillisecs[ResoltuionToMillisecs["year"] = 31536000000] = "year";
})(exports.ResoltuionToMillisecs || (exports.ResoltuionToMillisecs = {}));
var ResoltuionToMillisecs = exports.ResoltuionToMillisecs;
;
function forEachSensor(map, f) {
    for (var deviceId in map) {
        for (var sensorId in map[deviceId]) {
            f(deviceId, sensorId, map[deviceId][sensorId]);
        }
    }
}
exports.forEachSensor = forEachSensor;
function sensorEqual(a, b) {
    return a.deviceID === b.deviceID && a.sensorID === b.sensorID;
}
exports.sensorEqual = sensorEqual;

},{}],8:[function(require,module,exports){
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

},{}],9:[function(require,module,exports){
"use strict";
var Utils = require('./utils');
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
            oldest = Utils.now() - this._start;
            newest = Utils.now() - this._end;
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
            lines: {
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
        for (deviceId in this._sensorMap) {
            for (sensorId in this._sensorMap[deviceId]) {
                if (this._sensorMap[deviceId][sensorId] > index) {
                    this._sensorMap[deviceId][sensorId] -= 1;
                }
            }
        }
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
                colors[deviceId][sensorId] = this._series[index].lines.color;
            }
        }
        return colors;
    };
    return SensorValueStore;
}());
exports.SensorValueStore = SensorValueStore;

},{"./utils":11}],10:[function(require,module,exports){
"use strict";
var __extends = (this && this.__extends) || function (d, b) {
    for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p];
    function __() { this.constructor = d; }
    d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
};
var Utils = require('./utils');
var common_1 = require('./common');
var SubscriptionMode;
(function (SubscriptionMode) {
    SubscriptionMode[SubscriptionMode["Realtime"] = 0] = "Realtime";
    SubscriptionMode[SubscriptionMode["SlidingWindow"] = 1] = "SlidingWindow";
    SubscriptionMode[SubscriptionMode["Interval"] = 2] = "Interval";
})(SubscriptionMode || (SubscriptionMode = {}));
var Subscription = (function () {
    function Subscription(_subscriber) {
        this._subscriber = _subscriber;
    }
    Subscription.prototype.inTimeRange = function (timestamp, now) {
        return this.getStart(now) <= timestamp && this.getEnd(now) >= timestamp;
    };
    Subscription.prototype.getSubscriber = function () {
        return this._subscriber;
    };
    ;
    return Subscription;
}());
var IntervalSubscription = (function (_super) {
    __extends(IntervalSubscription, _super);
    function IntervalSubscription(_start, _end, subscriber) {
        _super.call(this, subscriber);
        this._start = _start;
        this._end = _end;
        if (_start > _end) {
            throw new Error("Start should be less than end for IntervalSubscription");
        }
    }
    IntervalSubscription.prototype.getMode = function () {
        return SubscriptionMode.Interval;
    };
    IntervalSubscription.prototype.getStart = function (now) {
        return this._start;
    };
    IntervalSubscription.prototype.getEnd = function (now) {
        return this._end;
    };
    return IntervalSubscription;
}(Subscription));
var SlidingWindowSubscription = (function (_super) {
    __extends(SlidingWindowSubscription, _super);
    function SlidingWindowSubscription(_start, _end, subscriber) {
        _super.call(this, subscriber);
        this._start = _start;
        this._end = _end;
        if (_end > _start) {
            throw new Error("Start should be bigger than end for SlidingWindowSubscription");
        }
    }
    SlidingWindowSubscription.prototype.getMode = function () {
        return SubscriptionMode.SlidingWindow;
    };
    SlidingWindowSubscription.prototype.getStart = function (now) {
        return now - this._start;
    };
    SlidingWindowSubscription.prototype.getEnd = function (now) {
        return now - this._end;
    };
    return SlidingWindowSubscription;
}(Subscription));
var RealtimeSubscription = (function (_super) {
    __extends(RealtimeSubscription, _super);
    function RealtimeSubscription(_start, subscriber) {
        _super.call(this, subscriber);
        this._start = _start;
        if (_start <= 0) {
            throw new Error("Start should greater than zero for RealtimeSubscription");
        }
    }
    RealtimeSubscription.prototype.getMode = function () {
        return SubscriptionMode.Realtime;
    };
    RealtimeSubscription.prototype.getStart = function (now) {
        return now - this._start;
    };
    RealtimeSubscription.prototype.getEnd = function (now) {
        return now;
    };
    return RealtimeSubscription;
}(Subscription));
var RealtimeResoulution = 'raw';
exports.UpdateDispatcherFactory = ["WSUserClient", "$interval",
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
        var subscripton = new IntervalSubscription(start, end, subscriber);
        this._subscribeSensor(deviceID, sensorID, resolution, subscripton);
    };
    UpdateDispatcher.prototype.subscribeSlidingWindow = function (deviceID, sensorID, resolution, start, end, subscriber) {
        var subscripton = new SlidingWindowSubscription(start, end, subscriber);
        this._subscribeSensor(deviceID, sensorID, resolution, subscripton);
    };
    ;
    UpdateDispatcher.prototype.subscribeRealtimeSlidingWindow = function (deviceID, sensorID, start, subscriber) {
        var subscripton = new RealtimeSubscription(start, subscriber);
        this._subscribeSensor(deviceID, sensorID, RealtimeResoulution, subscripton);
        var request = {};
        request[deviceID] = [sensorID];
        this._wsClient.requestRealtimeUpdates(request);
    };
    ;
    UpdateDispatcher.prototype._subscribeSensor = function (deviceID, sensorID, resolution, subscription) {
        if (this._devices[deviceID] === undefined) {
            throw new Error("Unknown device");
        }
        if (this._devices[deviceID] === undefined) {
            throw new Error("Unknown device");
        }
        if (!Utils.contains(common_1.SupportedResolutions, resolution)) {
            throw new Error("Unsupported resolution: " + resolution);
        }
        if (this._subscribers[deviceID][sensorID][resolution] === undefined) {
            this._subscribers[deviceID][sensorID][resolution] = [];
        }
        this._subscribers[deviceID][sensorID][resolution].push(subscription);
        var now = Utils.now();
        var sensorsList = {};
        sensorsList[deviceID] = [sensorID];
        this._wsClient.requestValues(subscription.getStart(now), subscription.getEnd(now), resolution, sensorsList);
    };
    UpdateDispatcher.prototype.unsubscribeAll = function (subscriber) {
        var _this = this;
        common_1.forEachSensor(this._subscribers, function (deviceID, sensorID, sensor) {
            for (var resolution in sensor) {
                _this.unsubscribeSensor(deviceID, sensorID, resolution, subscriber);
            }
        });
    };
    UpdateDispatcher.prototype.unsubscribeSensor = function (deviceID, sensorID, resolution, subscriber) {
        if (this._devices[deviceID] === undefined) {
            throw new Error("Unknown device");
        }
        if (this._devices[deviceID] === undefined) {
            throw new Error("Unknown device");
        }
        if (this._subscribers[deviceID][sensorID][resolution] === undefined) {
            throw new Error("No subscribers for this resolution");
        }
        Utils.removeWhere(this._subscribers[deviceID][sensorID][resolution], function (subscripton) { return subscripton.getSubscriber() === subscriber; });
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
                var wasUpdated = Utils.updateProperties(this._devices[deviceID].sensors[sensorID], metadata.devices[deviceID].sensors[sensorID]);
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
    UpdateDispatcher.prototype._notifySubscribers = function (notify, deviceID, sensorID) {
        var notified = new Set();
        var subscriptionsByResolutions = [];
        if (sensorID === undefined) {
            for (var _sensorID in this._subscribers[deviceID]) {
                subscriptionsByResolutions.push(this._subscribers[deviceID][_sensorID]);
            }
        }
        else {
            subscriptionsByResolutions.push(this._subscribers[deviceID][sensorID]);
        }
        for (var _i = 0, subscriptionsByResolutions_1 = subscriptionsByResolutions; _i < subscriptionsByResolutions_1.length; _i++) {
            var subscriptionByResolution = subscriptionsByResolutions_1[_i];
            for (var resolution in subscriptionByResolution) {
                for (var _a = 0, _b = subscriptionByResolution[resolution]; _a < _b.length; _a++) {
                    var subscription = _b[_a];
                    var subscriber = subscription.getSubscriber();
                }
                if (!notified.has(subscriber)) {
                    notify(subscriber);
                    notified.add(subscriber);
                }
            }
        }
    };
    UpdateDispatcher.prototype._emitDeviceMetadataUpdate = function (deviceID) {
        this._notifySubscribers(function (subscriber) { return subscriber.updateDeviceMetadata(deviceID); }, deviceID);
    };
    UpdateDispatcher.prototype._emitSensorMetadataUpdate = function (deviceID, sensorID) {
        this._notifySubscribers(function (subscriber) { return subscriber.updateSensorMetadata(deviceID, sensorID); }, deviceID, sensorID);
    };
    UpdateDispatcher.prototype._emitRemoveSensor = function (deviceID, sensorID) {
        this._notifySubscribers(function (subscriber) { return subscriber.removeSensor(deviceID, sensorID); }, deviceID, sensorID);
    };
    UpdateDispatcher.prototype._pollHistoryData = function () {
        var requests;
        requests = {};
        var now = Utils.now();
        common_1.forEachSensor(this._subscribers, function (deviceID, sensorID, map) {
            for (var resolution in map) {
                if (resolution !== RealtimeResoulution) {
                    for (var _i = 0, _a = map[resolution]; _i < _a.length; _i++) {
                        var subscripton = _a[_i];
                        var start = subscripton.getStart(now);
                        var end = subscripton.getEnd(now);
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
        var hasRealtimeSubscriptions = false;
        common_1.forEachSensor(this._subscribers, function (deviceID, sensorID, map) {
            if (map[RealtimeResoulution] !== undefined) {
                if (request[deviceID] === undefined) {
                    request[deviceID] = [];
                }
                for (var _i = 0, _a = map[RealtimeResoulution]; _i < _a.length; _i++) {
                    var subscripton = _a[_i];
                    if (subscripton.getMode() == SubscriptionMode.Realtime) {
                        hasRealtimeSubscriptions = true;
                        if (request[deviceID].indexOf(sensorID) === -1) {
                            request[deviceID].push(sensorID);
                        }
                    }
                }
            }
        });
        if (hasRealtimeSubscriptions) {
            this._wsClient.requestRealtimeUpdates(request);
        }
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
        var now = Utils.now();
        var notified = new Set();
        if (this._subscribers[deviceID] !== undefined
            && this._subscribers[deviceID][sensorID] !== undefined
            && this._subscribers[deviceID][sensorID][resolution] !== undefined) {
            for (var _i = 0, _a = this._subscribers[deviceID][sensorID][resolution]; _i < _a.length; _i++) {
                var subscripton = _a[_i];
                var subscriber = subscripton.getSubscriber();
                if (subscripton.inTimeRange(timestamp, now) && !notified.has(subscriber)) {
                    subscriber.updateValue(deviceID, sensorID, resolution, timestamp, value);
                    notified.add(subscriber);
                }
            }
        }
    };
    return UpdateDispatcher;
}());
exports.UpdateDispatcher = UpdateDispatcher;
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
}());
exports.DummySubscriber = DummySubscriber;

},{"./common":7,"./utils":11}],11:[function(require,module,exports){
"use strict";
function contains(haystack, needle) {
    var i = haystack.indexOf(needle);
    return i !== -1;
}
exports.contains = contains;
function remove(haystack, needle) {
    var i = haystack.indexOf(needle);
    if (i !== -1) {
        haystack.splice(i, 1);
    }
}
exports.remove = remove;
function removeWhere(haystack, pred) {
    var i = haystack.findIndex(pred);
    while (i !== -1) {
        haystack.splice(i, 1);
        i = haystack.findIndex(pred);
    }
}
exports.removeWhere = removeWhere;
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
exports.updateProperties = updateProperties;
function now() {
    return (new Date()).getTime();
}
exports.now = now;
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
exports.deepCopyJSON = deepCopyJSON;
function difference(a, b, equals) {
    return a.filter(function (a_element) { return b.findIndex(function (b_element) { return equals(a_element, b_element); }) === -1; });
}
exports.difference = difference;

},{}]},{},[1]);
