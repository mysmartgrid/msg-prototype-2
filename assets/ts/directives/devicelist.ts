interface DeviceProps {
    name : string;
    lan : any;
    wifi : any;
}

interface Device extends DeviceProps {
    sensors : Sensor[];
}

interface Sensor {
    name : string;
    confUrl : string;
    devId : string;
    sensId : string;
}

interface DeviceListScope extends ng.IScope {
    showSpinner : boolean;
    encodeURIComponent : (uriComponent : string) => string;

    deviceEditorSave : () => void;
    editedDeviceURL : string;
    editedDeviceProps : DeviceProps;
    editedDeviceId : string;

    errorSavingSettings : string;
    errorLoadingSettings : string;
    errorSavingSensor : string;

    devices : {[deviceID : string] : Device};

    editDev : ($event : Event) => void;
    remove : ($event : Event) => void;
    editSensor : ($event : Event) => void;
    saveSensor : () => void;

    error : string;

    editedSensor : Sensor;
}


class DeviceListController {
    public element : ng.IAugmentedJQuery;

    constructor(private $scope : DeviceListScope, private $interval : ng.IIntervalService, private $http : ng.IHttpService) {
        $scope.showSpinner = false;
        $scope.encodeURIComponent = encodeURIComponent;

        $scope.deviceEditorSave = function() {
            $http.post($scope.editedDeviceURL, $scope.editedDeviceProps)
                .success(function(data, status, headers, config) {
                    $scope.devices[$scope.editedDeviceId].name = $scope.editedDeviceProps.name;
                    $scope.devices[$scope.editedDeviceId].lan = $scope.editedDeviceProps.lan;
                    $scope.devices[$scope.editedDeviceId].wifi = $scope.editedDeviceProps.wifi;
                    $scope.editedDeviceId = undefined;
                    $scope.errorSavingSettings = null;
                    $("#deviceEditDialog").modal('hide');
                })
                .error(function(data, status, headers, config) {
                    $scope.errorSavingSettings = data;
                });
        };

        var flash = function(element) {
            element.removeClass("ng-hide");
            $interval(function() {
                element.addClass("ng-hide");
            }, 3000, 1);
        };

        $scope.editDev = function(e) {
            var id = $(e.target).parents("tr[data-device-id]").first().attr("data-device-id");
            var url = $(e.target).parents("tr[data-device-id]").first().attr("data-device-netconf-url");

            $scope.showSpinner = true;
            $http.get(url)
                .success(function(data : DeviceProps, status, headers, config) {
                    $scope.showSpinner = false;
                    $scope.errorLoadingSettings = null;
                    $scope.errorSavingSettings = null;

                    $scope.editedDeviceId = id;
                    $scope.editedDeviceURL = url;
                    $scope.editedDeviceProps = {
                        name: $scope.devices[id].name,
                        lan: data.lan || {},
                        wifi: data.wifi || {}
                    };
                    $("#deviceEditDialog").modal('show');
                })
                .error(function(data, status, headers, config) {
                    $scope.showSpinner = false;
                    $scope.errorLoadingSettings = data;
                });
        };

        $scope.remove = function(e) {
            var url = $(e.target).parents("tr[data-device-id]").first().attr("data-device-remove-url");
            var id = $(e.target).parents("tr[data-device-id]").first().attr("data-device-id");
            $scope.showSpinner = true;
            $http.delete(url)
                .success(function(data, status, headers, config) {
                    $scope.showSpinner = false;
                    delete $scope.devices[id];
                    flash($(e.target).parents(".device-list").first().find(".device-deleted"));
                })
                .error(function(data, status, headers, config) {
                    $scope.showSpinner = false;
                    $scope.error = data;
                });
        };

        $scope.editSensor = function(e) {
            var devId = $(e.target).parents("tr[data-device-id]").first().attr("data-device-id");
            var sensId = $(e.target).parents("tr[data-sensor-id]").first().attr("data-sensor-id");
            var url = $(e.target).parents("tr[data-sensor-conf-url]").first().attr("data-sensor-conf-url");

            $scope.errorSavingSensor = null;
            $scope.editedSensor = {
                name: $scope.devices[devId].sensors[sensId].name,
                confUrl: url,
                devId: devId,
                sensId: sensId,
            };
            $("#sensorEditDialog").modal('show');
        };

        $scope.saveSensor = function() {
            var props = {
                name: $scope.editedSensor.name
            };

            $scope.showSpinner = true;
            $http.post($scope.editedSensor.confUrl, props)
                .success(function(data, status, headers, config) {
                    $scope.showSpinner = false;
                    $scope.devices[$scope.editedSensor.devId].sensors[$scope.editedSensor.sensId].name = props.name;
                    $scope.editedSensor = null;
                    $("#sensorEditDialog").modal('hide');
                })
                .error(function(data, status, headers, config) {
                    $scope.showSpinner = false;
                    $scope.errorSavingSensor = data;
                });
        };
    }
}


class DeviceListDirective implements ng.IDirective {
    public require : string = "deviceList";
    public restrict = "A";
    public templateUrl = "/html/device-list.html";
    public scope = {
            devices: "="
        };

	public controller = ['$scope', '$interval', '$http', DeviceListController];

    // Implementing this as a method will not work as the this binding will break somewhere in angulars guts.
	public link:Function  = ($scope : DeviceListScope,
								element : ng.IAugmentedJQuery,
								attrs : ng.IAttributes,
								deviceList : DeviceListController) : void => {

		deviceList.element = element;
	}
}

export default function DeviceListFactory() : () => ng.IDirective {
	return () => new DeviceListDirective();
}
