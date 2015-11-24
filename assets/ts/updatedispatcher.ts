/// <reference path="utils.ts"/>
/// <reference path="common.ts"/>
/// <reference path="msg2socket.ts" />

import ExtArray = Utils.ExtArray;

module UpdateDispatcher  {

    export interface DeviceMap {
        [deviceId : string] : DeviceMetadataWithSensors;
    }

    export interface DeviceMetadata {
        name : string;
    }

    export interface DeviceMetadataWithSensors extends DeviceMetadata {
        sensors : SensorMap;
    }

    export interface SensorMap {
        [sensorId : string] : SensorMetadata;
    }

    export interface SensorMetadata extends Msg2Socket.SensorMetadata {};

    export interface Subscriber {
        updateValue(deviceId : string, sensorId : string, resolution : string, timestamp : number, value : number) : void;

        updateDeviceMetadata(deviceId : string) : void;
        updateSensorMetadata(deviceId : string, sensorId : string) : void;

        //IDEA:Use if device deletion ever becomes a thing
        removeDevice(deviceId : string) : void;
        removeSensor(deviceId : string, sensorId : string) : void;
    }

    interface SubscriberSettings {
        start : number;
        end : number;
        subscriber : Subscriber;
    }

    interface ResolutionSubscriberMap {
        [resolution : string] : ExtArray<SubscriberSettings>;
    }

    export const SupportedResolutions = new Set(["raw", "second", "minute", "hour", "day", "week", "month", "year"]);

    export const UpdateDispatcherFactory = ["WSUserClient",
                                            (wsClient : Msg2Socket.Socket, $interval : ng.IIntervalService) =>
                                                new UpdateDispatcher(wsClient, $interval)];

    export class UpdateDispatcher {

        private _hasInitialMetadata : boolean;
        private _InitialCallbacks : (() => void)[];

        private _devices : DeviceMap;

        public get devices() : DeviceMap {
            return this._devices;
        }

        private _subscribers : Common.DeviceSensorMap<ResolutionSubscriberMap>;

        constructor(private _wsClient : Msg2Socket.Socket, private $interval : ng.IIntervalService) {
            this._devices = {};
            this._subscribers = {};
            this._InitialCallbacks = new Array<() => void>();

            _wsClient.onOpen((error : Msg2Socket.OpenError) => {
                _wsClient.onMetadata((metadata : Msg2Socket.MetadataUpdate) : void => this._updateMetadata(metadata));
                _wsClient.onUpdate((data : Msg2Socket.UpdateData) : void => this._updateValues(data));

                this._hasInitialMetadata = false;

                this._wsClient.requestValues(0, 0, "second", true);
            });
        }

        public subscribeSensor(deviceId : string,
                                sensorId : string,
                                resolution : string,
                                start : number,
                                end : number,
                                subscriber: Subscriber) : void{

            if(this._devices[deviceId] === undefined) {
                throw new Error("Unknown device");
            }

            if(this._devices[deviceId] === undefined) {
                throw new Error("Unknown device");
            }

            if(!SupportedResolutions.has(resolution)) {
                throw new Error("Unsupported resolution");
            }

            if(this._subscribers[deviceId][sensorId][resolution] === undefined) {
                this._subscribers[deviceId][sensorId][resolution] = new ExtArray<SubscriberSettings>();
            }

            this._subscribers[deviceId][sensorId][resolution].push({start: start, end: end, subscriber: subscriber});

            if(end === null) {
                var request : Msg2Socket.RequestRealtimeUpdateArgs = {};
                request[deviceId] = {};
                request[deviceId][resolution] = [sensorId];
                this._wsClient.requestRealtimeUpdates(request);
            }
        }

        public unsubscribeSensor(deviceId : string,
                                    sensorId : string,
                                    resolution : string,
                                    subscriber: Subscriber) : void;

        public unsubscribeSensor(deviceId : string,
                                    sensorId : string,
                                    resolution : string,
                                    subscriber: Subscriber,
                                    start? : number,
                                    end? : number) : void {
            if(this._devices[deviceId] === undefined) {
                throw new Error("Unknown device");
            }

            if(this._devices[deviceId] === undefined) {
                throw new Error("Unknown device");
            }

            if(this._subscribers[deviceId][sensorId][resolution] === undefined) {
                throw new Error("No subscribers for this resolution");
            }

            if(start === undefined && end === undefined) {
                this._subscribers[deviceId][sensorId][resolution].removeWhere((settings) => settings.subscriber == subscriber);
            }
            else if(start !== undefined && end !== undefined) {
                this._subscribers[deviceId][sensorId][resolution].removeWhere((settings) =>
                                                                                settings.subscriber === subscriber &&
                                                                                settings.start === start &&
                                                                                settings.end === end);
            }
            else {
                throw new Error("Either start or end missing");
            }
        }

        public unsubscribeAll(subscriber: Subscriber) {
            Common.forEachSensor(this._subscribers, (deviceId, sensorId, sensor) : void => {
                for(var resolution in sensor) {
                    this.unsubscribeSensor(deviceId, sensorId, resolution, subscriber);
                }
            });
        }

        public onInitialMetadata(callback : () => void) {
            if(!this._hasInitialMetadata) {
                this._InitialCallbacks.push(callback);
            }
            else {
                callback();
            }
        }

        private _updateMetadata(metadata : Msg2Socket.MetadataUpdate) : void {
            console.log(metadata);
            for(var deviceId in metadata.devices) {
                // Create device if necessary
                if(this._devices[deviceId] === undefined) {
                    this._devices[deviceId] = {
                        name : null,  //Leave empty to emit update
                        sensors : {}
                    };
                }

                // Add space for subscribers if necessary
                if(this._subscribers[deviceId] === undefined) {
                    this._subscribers[deviceId] = {};
                }

                var deviceName = metadata.devices[deviceId].name;

                //TODO: Redo this check as soon as we have more device metadata
                if(deviceName !== undefined && this._devices[deviceId].name !== deviceName) {
                    console.log("Device name change '" + deviceName + "' '" + this._devices[deviceId].name + "'");
                    this._devices[deviceId].name = deviceName;
                    this._emitDeviceMetadataUpdate(deviceId);
                }



                // Add or update sensors
                for(var sensorId in metadata.devices[deviceId].sensors) {
                    // Add space for subscribers
                    if(this._subscribers[deviceId][sensorId] === undefined) {
                        this._subscribers[deviceId][sensorId] = {};
                    }

                    // Add empty entry to make updateProperties work
                    if(this._devices[deviceId].sensors[sensorId] === undefined) {
                        this._devices[deviceId].sensors[sensorId] = {
                            name: null,
                            unit: null,
                            port: null,
                        };
                    }

                    // Update metatdata and inform subscribers
                    var wasUpdated = Common.updateProperties(this._devices[deviceId].sensors[sensorId],
                                                                metadata.devices[deviceId].sensors[sensorId]);
                    if(wasUpdated) {
                        this._emitSensorMetadataUpdate(deviceId, sensorId);
                    }
                }

                // Delete sensors
                for(var sensorId in metadata.devices[deviceId].deletedSensors) {
                    delete this._devices[deviceId].sensors[sensorId];
                    this._emitRemoveSensor(deviceId, sensorId);
                    delete this._subscribers[deviceId][sensorId];
                }
            }

            if(!this._hasInitialMetadata) {
                this._hasInitialMetadata = true;
                for(var callback of this._InitialCallbacks) {
                    callback();
                }
            }
        }

        private _emitDeviceMetadataUpdate(deviceId : string) : void {
            // Notify every subscriber to the devices sensors once
            var notified = new Set<Subscriber>();
            for(var sensorId in this._subscribers[deviceId]) {
                for(var resolution in this._subscribers[deviceId][sensorId]) {
                    for(var {subscriber: subscriber} of this._subscribers[deviceId][sensorId][resolution])
                        if(!notified.has(subscriber)) {
                            subscriber.updateDeviceMetadata(deviceId);
                            notified.add(subscriber);
                        }
                }
            }
        }

        private _emitSensorMetadataUpdate(deviceId : string, sensorId : string) : void {
            // Notify every subscriber to the sensor once
            var notified = new Set<Subscriber>();
            for(var resolution in this._subscribers[deviceId][sensorId]) {
                for(var {subscriber: subscriber} of this._subscribers[deviceId][sensorId][resolution])
                    if(!notified.has(subscriber)) {
                        subscriber.updateSensorMetadata(deviceId, sensorId);
                        notified.add(subscriber);
                    }
            }
        }

        private _emitRemoveSensor(deviceId : string, sensorId : string) : void {
            // Notify every subscriber to the sensor once
            var notified = new Set<Subscriber>();
            for(var resolution in this._subscribers[deviceId][sensorId]) {
                for(var {subscriber: subscriber} of this._subscribers[deviceId][sensorId][resolution])
                    if(!notified.has(subscriber)) {
                        subscriber.removeSensor(deviceId, sensorId);
                        notified.add(subscriber);
                    }
            }
        }

        private _updateValues(data : Msg2Socket.UpdateData) : void {
            var {resolution, values} = data;
            for(var deviceId in values) {
                for(var sensorId in values[deviceId]) {
                    for(var [timestamp, value] of values[deviceId][sensorId]) {
                        this._emitValueUpdate(deviceId, sensorId, resolution, timestamp, value);
                    }
                }
            }
        }

        private _emitValueUpdate(deviceId : string, sensorId : string, resolution : string, timestamp : number, value : number) : void {
            if(this._subscribers[deviceId][sensorId][resolution] !== undefined) {
                for(var {start: start, end: end, subscriber: subscriber} of this._subscribers[deviceId][sensorId][resolution]) {
                    if(start <= timestamp && (end >= timestamp || end === null)) {
                        subscriber.updateValue(deviceId, sensorId, resolution, timestamp, value);
                    }
                }
            }
        }

    }


    export class DummySubscriber implements Subscriber {
        public updateValue(deviceId : string, sensorId : string, resolution : string, timestamp : number, value : number) : void {
            console.log("Update for value " + deviceId + ":" + sensorId + " " +  resolution + " " + timestamp + " " + value);
        }

        public updateDeviceMetadata(deviceId : string) : void {
            console.log("Update for device metadata " + deviceId);
        }

        public updateSensorMetadata(deviceId : string, sensorId : string) : void {
            console.log("Update for sensor metadata " + deviceId + ":" + sensorId);
        }


        public removeDevice(deviceId : string) : void {
            console.log("Removed device " + deviceId);
        }

        public removeSensor(deviceId : string, sensorId : string) :void {
            console.log("Remove sensor " + deviceId + ":" + sensorId);
        }

    }
}
