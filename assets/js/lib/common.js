define(["require", "exports"], function (require, exports) {
    "use strict";
    function forEachSensor(map, f) {
        for (var deviceId in map) {
            for (var sensorId in map[deviceId]) {
                f(deviceId, sensorId, map[deviceId][sensorId]);
            }
        }
    }
    exports.forEachSensor = forEachSensor;
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
});
//# sourceMappingURL=common.js.map