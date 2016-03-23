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
    deviceEditorDismiss : () => void;
    editedDeviceURL : string;
    editedDeviceProps : DeviceProps;
    editedDeviceId : string;

    errorSavingSettings : string;
    errorLoadingSettings : string;
    errorSavingSensor : string;

    devices : {[deviceID : string] : Device};

    editDevice : (deviceID : string) => void;
    remove : (deviceID : string) => void;
    editSensor : (deviceID : string, sensorID : string) => void;
    saveSensor : () => void;
    dismissSensor : () => void;

    error : string;

    editedSensor : Sensor;
}


function deviceConfigUrl(deviceID : string) : string {
    return '/api/user/v1/device/' + encodeURIComponent(deviceID) + '/config';
}


function deviceRemoveUrl(deviceID : string) : string {
    return '/api/user/v1/device/' + encodeURIComponent(deviceID);
}


function sensorConfigUrl(deviceID : string, sensorID : string) : string {
    return '/api/user/v1/sensor/' + encodeURIComponent(deviceID) + '/' + encodeURIComponent(sensorID) + '/props';
}

class DeviceListController {
    public element : ng.IAugmentedJQuery;

    constructor(private $scope : DeviceListScope, private $interval : ng.IIntervalService, private $http : ng.IHttpService) {
        $scope.showSpinner = false;
        $scope.encodeURIComponent = encodeURIComponent;

        $scope.deviceEditorSave = () => {
            $http.post($scope.editedDeviceURL, $scope.editedDeviceProps)
                .success((data, status, headers, config) => {
                    $scope.devices[$scope.editedDeviceId].name = $scope.editedDeviceProps.name;
                    $scope.devices[$scope.editedDeviceId].lan = $scope.editedDeviceProps.lan;
                    $scope.devices[$scope.editedDeviceId].wifi = $scope.editedDeviceProps.wifi;
                    $scope.editedDeviceId = undefined;
                    $scope.errorSavingSettings = null;
                    $("#deviceEditDialog").modal('hide');
                })
                .error((data, status, headers, config) => {
                    $scope.errorSavingSettings = data;
                });
        };

        $scope.deviceEditorDismiss = () => {
            $("#deviceEditDialog").modal('hide');
        }

        $scope.editDevice = (deviceID) => {
            var url = deviceConfigUrl(deviceID);

            $scope.showSpinner = true;
            $http.get(url)
                .success((data : DeviceProps, status, headers, config) => {
                    $scope.showSpinner = false;
                    $scope.errorLoadingSettings = null;
                    $scope.errorSavingSettings = null;

                    $scope.editedDeviceId = deviceID;
                    $scope.editedDeviceURL = url;
                    $scope.editedDeviceProps = {
                        name: $scope.devices[deviceID].name,
                        lan: data.lan || {},
                        wifi: data.wifi || {}
                    };
                    $("#deviceEditDialog").modal('show');
                })
                .error((data, status, headers, config) => {
                    $scope.showSpinner = false;
                    $scope.errorLoadingSettings = data;
                });
        };

        $scope.remove = (deviceID) => {
            var url = deviceRemoveUrl(deviceID);
            $scope.showSpinner = true;
            $http.delete(url)
                .success((data, status, headers, config) => {
                    $scope.showSpinner = false;

                    this.flash(this.element.find(".device-deleted"));

                    delete $scope.devices[deviceID];
                })
                .error((data, status, headers, config) => {
                    $scope.showSpinner = false;
                    $scope.error = data;
                });
        };

        $scope.editSensor = (deviceID, sensorID) => {
            var url = sensorConfigUrl(deviceID, sensorID);

            $scope.errorSavingSensor = null;
            $scope.editedSensor = {
                name: $scope.devices[deviceID].sensors[sensorID].name,
                confUrl: url,
                devId: deviceID,
                sensId: sensorID,
            };
            $("#sensorEditDialog").modal('show');
        };

        $scope.saveSensor = () => {
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

        $scope.dismissSensor = () => {
            $("#sensorEditDialog").modal('hide');
        }
    }

    private flash(element : ng.IAugmentedJQuery) : void {
        element.removeClass("ng-hide");
        this.$interval(() => element.addClass("ng-hide"), 3000, 1);
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
