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
