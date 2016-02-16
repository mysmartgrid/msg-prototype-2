module Common {
    export interface DeviceSensorMap<U> {
        [deviceId : string] : {
            [sensorId: string] : U;
        }
    }

    export function forEachSensor<U>(map : DeviceSensorMap<U>, f : {(deviceID : string, sensorID : string, data : U) : void}) {
        for(var deviceId in map) {
            for(var sensorId in map[deviceId]) {
                f(deviceId, sensorId, map[deviceId][sensorId]);
            }
        }
    }

    export function updateProperties<U>(target : U, source: U) : boolean {
        var wasUpdated = false;
        for(var prop in target) {
            if(target[prop] !== source[prop]) {
                target[prop] = source[prop];
                wasUpdated = true;
            }
        }

        return wasUpdated;
    }

    export function now() : number {
        return (new Date()).getTime();
    }
}
