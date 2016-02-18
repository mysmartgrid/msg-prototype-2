var __extends = (this && this.__extends) || function (d, b) {
    for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p];
    function __() { this.constructor = d; }
    d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
};
define(["require", "exports", './common', '../lib/utils'], function (require, exports, Common, utils_1) {
    "use strict";
    console.log('Dispatcher');
    ;
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
    exports.SupportedResolutions = new Set(["raw", "second", "minute", "hour", "day", "week", "month", "year"]);
    exports.ResoltuionToMillisecs = {
        raw: 1000,
        second: 1000,
        minute: 60 * 1000,
        hour: 60 * 60 * 1000,
        day: 24 * 60 * 60 * 1000,
        week: 7 * 24 * 60 * 60 * 1000,
        month: 31 * 24 * 60 * 60 * 1000,
        year: 365 * 24 * 60 * 60 * 1000
    };
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
            if (!exports.SupportedResolutions.has(resolution)) {
                throw new Error("Unsupported resolution");
            }
            if (this._subscribers[deviceID][sensorID][resolution] === undefined) {
                this._subscribers[deviceID][sensorID][resolution] = new utils_1.ExtArray();
            }
            this._subscribers[deviceID][sensorID][resolution].push(subscription);
            var now = Common.now();
            var sensorsList = {};
            sensorsList[deviceID] = [sensorID];
            this._wsClient.requestValues(subscription.getStart(now), subscription.getEnd(now), resolution, sensorsList);
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
            if (this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }
            if (this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }
            if (this._subscribers[deviceID][sensorID][resolution] === undefined) {
                throw new Error("No subscribers for this resolution");
            }
            this._subscribers[deviceID][sensorID][resolution].removeWhere(function (subscripton) { return subscripton.getSubscriber() === subscriber; });
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
            var now = Common.now();
            Common.forEachSensor(this._subscribers, function (deviceID, sensorID, map) {
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
            Common.forEachSensor(this._subscribers, function (deviceID, sensorID, map) {
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
            var now = Common.now();
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
});
//# sourceMappingURL=updatedispatcher.js.map