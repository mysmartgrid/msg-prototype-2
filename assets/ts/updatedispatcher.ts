/// <reference path="utils.ts"/>
/// <reference path="common.ts"/>
/// <reference path="msg2socket.ts" />

import ExtArray = Utils.ExtArray;

module UpdateDispatcher  {

    // Map: DeviceIDs to Sensorslist and device metadata
    export interface DeviceMap {
        [deviceID : string] : DeviceMetadataWithSensors;
    }

    // Device metadata (currently just a name)
    export interface DeviceMetadata {
        name : string;
    }

    // Device metadata extende with a sensor list
    export interface DeviceMetadataWithSensors extends DeviceMetadata {
        sensors : SensorMap;
    }

    // Map: SensorIDs to sensor metadata
    export interface SensorMap {
        [sensorID : string] : SensorMetadata;
    }

    // Type alias, so we can change the type without refactoring
    export interface SensorMetadata extends Msg2Socket.SensorMetadata {};


    export interface SensorSpecifier {
        sensorID : string;
        deviceID : string;
    }

    export interface UnitSensorMap {
        [unit : string] : SensorSpecifier[];
    }


    // Interface for subscribers
    export interface Subscriber {
        // Called on new values and updates to old values
        updateValue(deviceID : string, sensorID : string, resolution : string, timestamp : number, value : number) : void;

        /**
         * Called in case device metadata chanced.
         * The subscriber can get the new metatdata from the dispatchers devices property
         */
        updateDeviceMetadata(deviceID : string) : void;

        /**
         * Called in case sensor metadata chanced.
         * The subscriber can get the new metatdata from the dispatchers devices property
         */
        updateSensorMetadata(deviceID : string, sensorID : string) : void;

        //TODO: Use if device deletion ever becomes a thing
        removeDevice(deviceID : string) : void;

        // Called to signal the removal of a sensor
        removeSensor(deviceID : string, sensorID : string) : void;
    }

    //IDEA: Rename to Subscription
    // Setting for a subscription
    interface SubscriberSettings {
        // True if start and end denote timestamps relative to the current timestamp
        slidingWindow : boolean;

        /**
         * Either the start timestamp of the subscription interval,
         * or how many milliseconds into the past the sliding window starts
         */
        start : number;

        /**
         * Either the end timestamp of the subscription interval,
         * or how many milliseconds into the past the sliding window ends.
         * If this attribute is 0 and slidingWindow is set all updates with a timestamp newer then
         * start milliseconds in the past will be forwarded to the subscriber.
         */
        end : number;

        // The subscriber object
        subscriber : Subscriber;
    }

    // Map: time resolution to Array of subscribers
    interface ResolutionSubscriberMap {
        [resolution : string] : ExtArray<SubscriberSettings>;
    }

    // Set of all supported time resolutions for faster sanity checks
    export const SupportedResolutions = new Set(["raw", "second", "minute", "hour", "day", "week", "month", "year"]);

    export const ResoltuionToMillisecs = {
        raw: 1000,
        second: 1000,
        minute: 60 * 1000,
        hour: 60 * 60 * 1000,
        day: 24 * 60 * 60 * 1000,
        week: 7 * 24 * 60 * 60 * 1000,
        month: 31 * 24 * 60 * 60 * 1000,
        year: 365 * 24 * 60 * 60 * 1000
    };

    // Angular factory function with injected dependencies
    export const UpdateDispatcherFactory = ["WSUserClient", "$interval",
                                            (wsClient : Msg2Socket.Socket, $interval : ng.IIntervalService) =>
                                                new UpdateDispatcher(wsClient, $interval)];

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
    export class UpdateDispatcher {

        // Flag the will be set to true as soons as metadata is avaiable
        private _hasInitialMetadata : boolean;

        // List of callbacks to call as soon as metadata becomes avaiable
        private _InitialCallbacks : (() => void)[];

        // Tree-Structure for storing device and sensor metadata
        private _devices : DeviceMap;

        // Device/Sensor Tuples sorted by Units
        private _sensorsByUnit : UnitSensorMap;

        // List of all avaiable units
        private _units : string[];

        // Accesor to prevent write access to the devices property
        public get devices() : DeviceMap {
            return this._devices;
        }

        // Pseudoproperty that contains all possible units
        public get units() : string[] {
            return this._units;
        }

        // Accesor for _sensorsByUnit
        public get sensorsByUnit() : UnitSensorMap {
            return this._sensorsByUnit;
        }

        /**
         * Map for managing subscriptions
         * Structure [deviceID][sensorId][resolution] -> SubscriberSettings[]
         */
        private _subscribers : Common.DeviceSensorMap<ResolutionSubscriberMap>;

        /**
         * Construtor for UpdateDispatcher
         * Should not be called directly, use the factory to register an angular service instead
         *
         * Initalizes private members
         * Registers _updateMetadata and _updateValues as callbacks for the Msg2Socket
         * Requests initial metadata as soon as the socket is connected
         * Sets up an $interval instance for polling historical data using _pollHistoryData
         */
        constructor(private _wsClient : Msg2Socket.Socket, private $interval : ng.IIntervalService) {
            this._devices = {};
            this._subscribers = {};
            this._InitialCallbacks = new Array<() => void>();

            this._sensorsByUnit = {};
            this._units = [];

            _wsClient.onOpen((error : Msg2Socket.OpenError) => {
                _wsClient.onMetadata((metadata : Msg2Socket.MetadataUpdate) : void => this._updateMetadata(metadata));
                _wsClient.onUpdate((data : Msg2Socket.UpdateData) : void => this._updateValues(data));

                this._hasInitialMetadata = false;

                this._wsClient.requestMetadata();

                $interval(() => this._pollHistoryData(), 1 * 60 * 1000);
            });
        }


        /**
         * Subscribe for value updates with in a fixed interval from start to end.
         * Start and end are millisecond timestamps.
         */
        public subscribeInterval(deviceID : string,
                                sensorID : string,
                                resolution : string,
                                start : number,
                                end : number,
                                subscriber: Subscriber) {
            this._subscribeSensor(deviceID, sensorID, resolution, false, start, end, subscriber);
        }


        /**
         * Subscribe for value updates with in a slinding window from current_timestamp - start to current_timestamp  - end.
         * Start and end are in milliseconds.
         */
        public subscribeSlidingWindow(deviceID : string,
                                sensorID : string,
                                resolution : string,
                                start : number,
                                end : number,
                                subscriber: Subscriber) {
            this._subscribeSensor(deviceID, sensorID, resolution, true, start, end, subscriber);
        };


        /**
         * Subscribe for value updates with in a slinding window from current_timestamp - start to current_timestamp.
         * Subscribers using this subscrition also get forwarded realtime updates from the metering device
         * Start and end are in milliseconds.
         */
        public subscribeRealtimeSlidingWindow(deviceID : string,
                                sensorID : string,
                                resolution : string,
                                start : number,
                                subscriber: Subscriber) {
            this._subscribeSensor(deviceID, sensorID, resolution, true, start, 0, subscriber);
        };


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
        private _subscribeSensor(deviceID : string,
                                sensorID : string,
                                resolution : string,
                                slidingWindow : boolean,
                                start : number,
                                end : number,
                                subscriber: Subscriber) : void{

            if(this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }

            if(this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }

            if(!SupportedResolutions.has(resolution)) {
                throw new Error("Unsupported resolution");
            }

            if(slidingWindow && start < end) {
                throw new Error("Start should be bigger then end for sliding window mode");
            }
            else if(!slidingWindow && start > end) {
                throw new Error("End should be bigger then star for interval mode");
            }

            if(this._subscribers[deviceID][sensorID][resolution] === undefined) {
                this._subscribers[deviceID][sensorID][resolution] = new ExtArray<SubscriberSettings>();
            }

            this._subscribers[deviceID][sensorID][resolution].push({slidingWindow : slidingWindow,
                                                                    start: start,
                                                                    end: end,
                                                                    subscriber: subscriber});

            if(slidingWindow && end === 0) {
                var request : Msg2Socket.RequestRealtimeUpdateArgs = {};
                request[deviceID] = {};
                request[deviceID][resolution] = [sensorID];
                this._wsClient.requestRealtimeUpdates(request);
            }

            // Request history
            var now = Common.now();
            if(slidingWindow) {
                start = now - start;
                end = now - end;
            }

            var sensorsList : Msg2Socket.DeviceSensorList = {};
            sensorsList[deviceID] = [sensorID];
            this._wsClient.requestValues(start, end, resolution, sensorsList);
        }

        // Shorthand to remove all subscribtions for a given subscriber
        public unsubscribeAll(subscriber: Subscriber) {
            Common.forEachSensor(this._subscribers, (deviceID, sensorID, sensor) : void => {
                for(var resolution in sensor) {
                    this.unsubscribeSensor(deviceID, sensorID, resolution, subscriber);
                }
            });
        }

        // Shorthand to remove all subscribtions to sensor and resoltion for a specific subscriber
        public unsubscribeSensor(deviceID : string,
                                    sensorID : string,
                                    resolution : string,
                                    subscriber: Subscriber) : void {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber);
        }

        /**
         * Unsubscribe for value updates with in a fixed interval from start to end.
         * Start and end are millisecond timestamps.
         */
        public unsubscribeInterval(deviceID : string,
                                sensorID : string,
                                resolution : string,
                                start : number,
                                end : number,
                                subscriber: Subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber, false, start, end);
        }

        /**
         * Unsubscribe for value updates with in a slinding window from current_timestamp - start to current_timestamp  - end.
         * Start and end are in milliseconds.
         */
        public unsubscribeSlidingWindow(deviceID : string,
                                sensorID : string,
                                resolution : string,
                                start : number,
                                end : number,
                                subscriber: Subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber, true, start, end);
        };

        /**
         * Unsubscribe for value updates with in a slinding window from current_timestamp - start to current_timestamp.
         * Start and end are in milliseconds.
         */
        public unsubscribeRealtimeSlidingWindow(deviceID : string,
                                sensorID : string,
                                resolution : string,
                                start : number,
                                subscriber: Subscriber) {
            this._unsubscribeSensor(deviceID, sensorID, resolution, subscriber, true, start, 0);
        };

        /**
         * Internal method to remove a subscribtion given by start, end, slidingWindow,
         * resolution and sensor for a specific subscriber.
         * If start, end and slidingWindow are missing all subscribtions for the sensor and resolution.
         */
        private _unsubscribeSensor(deviceID : string,
                                    sensorID : string,
                                    resolution : string,
                                    subscriber: Subscriber,
                                    slidingWindow?: boolean,
                                    start? : number,
                                    end? : number) : void {
            if(this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }

            if(this._devices[deviceID] === undefined) {
                throw new Error("Unknown device");
            }

            if(this._subscribers[deviceID][sensorID][resolution] === undefined) {
                throw new Error("No subscribers for this resolution");
            }

            if(start === undefined && end === undefined && slidingWindow === undefined) {
                this._subscribers[deviceID][sensorID][resolution].removeWhere((settings) => settings.subscriber == subscriber);
            }
            else if(start !== undefined && end !== undefined && slidingWindow !== undefined) {
                this._subscribers[deviceID][sensorID][resolution].removeWhere((settings) =>
                                                                                settings.subscriber === subscriber &&
                                                                                settings.slidingWindow === slidingWindow &&
                                                                                settings.start === start &&
                                                                                settings.end === end);
            }
            else {
                throw new Error("Either start or end missing");
            }
        }

        /**
         * Register callbacks which will be called as soon as metadata is avaiable.
         * Usefull for doing inital subscriptions.
         * If metadata is already avaiable the callback will be execute immediately.
         */
        public onInitialMetadata(callback : () => void) {
            if(!this._hasInitialMetadata) {
                this._InitialCallbacks.push(callback);
            }
            else {
                callback();
            }
        }

        /**
         * Internal method which is called by the Msg2Socket in case of metadata updates.
         * Updates _devices and calls subscribers accordingly using _emitDeviceMetadataUpdate amd _emitSensorMetadataUpdate.
         */
        private _updateMetadata(metadata : Msg2Socket.MetadataUpdate) : void {
            for(var deviceID in metadata.devices) {
                // Create device if necessary
                if(this._devices[deviceID] === undefined) {
                    this._devices[deviceID] = {
                        name : null,  //Leave empty to emit update
                        sensors : {}
                    };
                }

                // Add space for subscribers if necessary
                if(this._subscribers[deviceID] === undefined) {
                    this._subscribers[deviceID] = {};
                }

                var deviceName = metadata.devices[deviceID].name;

                //TODO: Redo this check as soon as we have more device metadata
                if(deviceName !== undefined && this._devices[deviceID].name !== deviceName) {
                    this._devices[deviceID].name = deviceName;
                    this._emitDeviceMetadataUpdate(deviceID);
                    console.log("Nameupdate: " + deviceName);
                }



                // Add or update sensors
                for(var sensorID in metadata.devices[deviceID].sensors) {
                    // Add space for subscribers
                    if(this._subscribers[deviceID][sensorID] === undefined) {
                        this._subscribers[deviceID][sensorID] = {};
                    }

                    // Add empty entry to make updateProperties work
                    if(this._devices[deviceID].sensors[sensorID] === undefined) {
                        this._devices[deviceID].sensors[sensorID] = {
                            name: null,
                            unit: null,
                            port: null,
                        };
                    }

                    // Update metatdata and inform subscribers
                    var wasUpdated = Common.updateProperties(this._devices[deviceID].sensors[sensorID],
                                                                metadata.devices[deviceID].sensors[sensorID]);
                    if(wasUpdated) {
                        this._emitSensorMetadataUpdate(deviceID, sensorID);
                    }
                }

                // Delete sensors
                for(var sensorID in metadata.devices[deviceID].deletedSensors) {
                    delete this._devices[deviceID].sensors[sensorID];
                    this._emitRemoveSensor(deviceID, sensorID);
                    delete this._subscribers[deviceID][sensorID];
                }

                //TODO: Handle Device deletion as well
            }

            this._updateSensorsByUnit();

            // Excute the callbacks if this is the initial metadata update
            if(!this._hasInitialMetadata) {
                this._hasInitialMetadata = true;
                for(var callback of this._InitialCallbacks) {
                    callback();
                }
            }
        }

        private _updateSensorsByUnit() : void {
            for(var index in this._sensorsByUnit) {
                delete this._sensorsByUnit[this._units[index]];
                delete this._units[index];
            }

            for(var deviceID in this._devices) {
                for(var sensorID in this._devices[deviceID].sensors) {
                    var unit = this._devices[deviceID].sensors[sensorID].unit;

                    if(this._sensorsByUnit[unit] === undefined) {
                        this._units.push(unit);
                        this._sensorsByUnit[unit] = [];
                    }

                    this._sensorsByUnit[unit].push({deviceID: deviceID, sensorID: sensorID});
                }
            }
        }

        /**
         * Notify all subscribers to all sensors in all resolutions for this device of the update.
         * A set is used to ensure each subscriber is notified exactly once.
         */
        private _emitDeviceMetadataUpdate(deviceID : string) : void {
            // Notify every subscriber to the devices sensors once
            var notified = new Set<Subscriber>();
            for(var sensorID in this._subscribers[deviceID]) {
                for(var resolution in this._subscribers[deviceID][sensorID]) {
                    for(var {subscriber: subscriber} of this._subscribers[deviceID][sensorID][resolution])
                        if(!notified.has(subscriber)) {
                            subscriber.updateDeviceMetadata(deviceID);
                            notified.add(subscriber);
                        }
                }
            }
        }

        /**
         * Notify all subscribers to a sensors in all resolutions of the update.
         * A set is used to ensure each subscriber is notified exactly once.
         */
        private _emitSensorMetadataUpdate(deviceID : string, sensorID : string) : void {
            // Notify every subscriber to the sensor once
            var notified = new Set<Subscriber>();
            for(var resolution in this._subscribers[deviceID][sensorID]) {
                for(var {subscriber: subscriber} of this._subscribers[deviceID][sensorID][resolution])
                    if(!notified.has(subscriber)) {
                        subscriber.updateSensorMetadata(deviceID, sensorID);
                        notified.add(subscriber);
                    }
            }
        }

        /**
         * Notify all subscribers to a sensors in all resolutions.
         * A set is used to ensure each subscriber is notified exactly once.
         */
        private _emitRemoveSensor(deviceID : string, sensorID : string) : void {
            // Notify every subscriber to the sensor once
            var notified = new Set<Subscriber>();
            for(var resolution in this._subscribers[deviceID][sensorID]) {
                for(var {subscriber: subscriber} of this._subscribers[deviceID][sensorID][resolution])
                    if(!notified.has(subscriber)) {
                        subscriber.removeSensor(deviceID, sensorID);
                        notified.add(subscriber);
                    }
            }
        }

        /**
         * Request historical data for all subscriptions from the backend.
         * In order to minimize the number of requests to the backend,
         * only one reuqest per resoltion covering all subscribed sensors and intervals is generated.
         * The _updateValues method takes care of dropping unecessary values and dispatching the rest to the subscribers.
         */
        private _pollHistoryData() : void {
            var requests : {[resolution : string] : {start :number, end: number, sensors: {[deviceID : string] : Set<string>}}};
            requests = {};
            var now = Common.now();

            // Gather start, end and sensors for each resolution
            Common.forEachSensor<ResolutionSubscriberMap>(this._subscribers, (deviceID, sensorID, map) => {
                for(var resolution in map) {
                    for(var {start: start, end: end, slidingWindow: slidingWindow} of map[resolution]) {

                        if(slidingWindow) {
                            start = now - start;
                            end = now - end;
                        }

                        if(requests[resolution] === undefined) {
                            requests[resolution] = {
                                start: start,
                                end: end,
                                sensors: {}
                            };
                        }

                        //Adjust start and end of interval
                        requests[resolution].start = Math.min(start, requests[resolution].start);
                        requests[resolution].end = Math.max(end, requests[resolution].end);


                        if(requests[resolution].sensors[deviceID] === undefined) {
                            requests[resolution].sensors[deviceID] = new Set<string>();
                        }

                        requests[resolution].sensors[deviceID].add(sensorID);
                    }
                }
            });

            // Send out the requests
            for(var resolution in requests) {
                var {start: start, end: end, sensors: sensors} = requests[resolution];

                var sensorList : Msg2Socket.DeviceSensorList = {};
                for(var deviceID in sensors) {
                    sensorList[deviceID] = [];
                    sensors[deviceID].forEach((sensorID) => sensorList[deviceID].push(sensorID));
                }

                this._wsClient.requestValues(start, end, resolution, sensorList);
            }
        }

        /**
         * Internal method which is called by the Msg2Socket in case of value updates.
         * Simply unpacks the update and calls _emitValueUpdate for each value.
         */
        private _updateValues(data : Msg2Socket.UpdateData) : void {
            var {resolution, values} = data;
            for(var deviceID in values) {
                for(var sensorID in values[deviceID]) {
                    for(var [timestamp, value] of values[deviceID][sensorID]) {
                        this._emitValueUpdate(deviceID, sensorID, resolution, timestamp, value);
                    }
                }
            }
        }

        /**
         * Internal methode called once from _updateValues for each value timestamp pair.
         * Matches the subscription interval of each subscripton for the sensor and resolution against the updates timestamps.
         * Also maintains a set of already notified subscribers to avoid notifying a subscriber twices in case of overlapping subscriptons.
         */
        private _emitValueUpdate(deviceID : string, sensorID : string, resolution : string, timestamp : number, value : number) : void {
            var now = Common.now();
            var notified = new Set<Subscriber>();

            // Make sure we have subscribsers for this sensor
            if(this._subscribers[deviceID] !== undefined
                && this._subscribers[deviceID][sensorID] !== undefined
                && this._subscribers[deviceID][sensorID][resolution] !== undefined) {
                for(var {start: start, end: end, slidingWindow: slidingWindow, subscriber: subscriber} of this._subscribers[deviceID][sensorID][resolution]) {

                    if(slidingWindow) {
                        start = now - start;
                        end = now - end;
                    }

                    if(start <= timestamp && end >= timestamp && !notified.has(subscriber)) {
                        subscriber.updateValue(deviceID, sensorID, resolution, timestamp, value);
                        notified.add(subscriber);
                    }
                }
            }
        }

    }


    // Dummy subscriber that dumps all updates to console.
    export class DummySubscriber implements Subscriber {
        public updateValue(deviceID : string, sensorID : string, resolution : string, timestamp : number, value : number) : void {
            var date = new Date(timestamp);
            console.log("Update for value " + deviceID + ":" + sensorID + " " +  resolution + " " + date + " " + value);
        }

        public updateDeviceMetadata(deviceID : string) : void {
            console.log("Update for device metadata " + deviceID);
        }

        public updateSensorMetadata(deviceID : string, sensorID : string) : void {
            console.log("Update for sensor metadata " + deviceID + ":" + sensorID);
        }


        public removeDevice(deviceID : string) : void {
            console.log("Removed device " + deviceID);
        }

        public removeSensor(deviceID : string, sensorID : string) :void {
            console.log("Remove sensor " + deviceID + ":" + sensorID);
        }

    }
}
