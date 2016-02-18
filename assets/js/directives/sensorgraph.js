define(["require", "exports", '../lib/utils', '../lib/common', '../lib/updatedispatcher', '../lib/sensorvaluestore'], function (require, exports, Utils, Common, UpdateDispatcher, Store) {
    "use strict";
    function sensorEqual(a, b) {
        return a.deviceID === b.deviceID && a.sensorID === b.sensorID;
    }
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
            $scope.resolutions['realtime'] = ['raw'];
            $scope.resolutions['slidingWindow'] = supportedResolutions.filter(function (res) { return res !== 'raw'; });
            $scope.resolutions['interval'] = supportedResolutions.filter(function (res) { return res !== 'raw'; });
            $scope.$watch("config.mode", function () {
                var mode = $scope.config.mode;
                if ($scope.resolutions[mode].indexOf($scope.config.resolution) === -1) {
                    $scope.config.resolution = $scope.resolutions[mode][0];
                }
                if (mode === 'realtime') {
                    $scope.config.resolution = 'raw';
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
    }());
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
                var addedSensors = Utils.difference(config.sensors, this._config.sensors, sensorEqual);
                var removedSensors = Utils.difference(this._config.sensors, config.sensors, sensorEqual);
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
    }());
    exports.SensorGraphController = SensorGraphController;
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
    }());
    function SensorGraphFactory() {
        return function () { return new SensorGraphDirective(); };
    }
    Object.defineProperty(exports, "__esModule", { value: true });
    exports.default = SensorGraphFactory;
});
//# sourceMappingURL=sensorgraph.js.map