import {ExtArray} from './utils';

export interface DeviceMap<U> {
    [deviceID : string] : U;
}

export interface SensorMap<U> {
    [sensorID : string] : U;
}

export interface DeviceSensorMap<U> extends DeviceMap<SensorMap<U>> {};


// Device metadata (currently just a name)
export interface DeviceMetadata {
    name : string;
}

export interface SensorMetadata {
	name : string;
	unit : string;
	port : number;
}

// Device metadata extended with a sensor list
export interface DeviceWithSensors extends DeviceMetadata {
    sensors : SensorMap<SensorMetadata>;
}

export interface SensorSpecifier {
    sensorID : string;
    deviceID : string;
}

export interface SensorUnitMap {
    [unit : string] : SensorSpecifier[];
}

export interface MetadataTree extends DeviceMap<DeviceWithSensors> {};


export const SupportedResolutions = new ExtArray("raw", "second", "minute", "hour", "day", "week", "month", "year");

export var ResolutionsPerMode : {[resolution : string] : string[]} = {
    "interval" : SupportedResolutions,
    "slidingWindow" : SupportedResolutions.filter((res) => res !== "raw"),
    "realtime" : ["raw"]
}


export enum ResoltuionToMillisecs {
    raw = 1000,
    second = raw,
    minute = 60 * second,
    hour = 60 * minute,
    day = 24 * hour,
    week = 7 * day,
    month = 31 * day,
    year = 365 * day
};


export function forEachSensor<U>(map : DeviceSensorMap<U>, f : {(deviceID : string, sensorID : string, data : U) : void}) {
    for(var deviceId in map) {
        for(var sensorId in map[deviceId]) {
            f(deviceId, sensorId, map[deviceId][sensorId]);
        }
    }
}


export function sensorEqual(a : SensorSpecifier, b : SensorSpecifier) : boolean {
	return a.deviceID === b.deviceID && a.sensorID === b.sensorID;
}
