interface DeviceProps {
    name : string;
    lan : {
        enabled : boolean;
        protocol : string;
        ip : string;
        netmask : string;
        gateway : string;
        nameserver : string;
    };
    wifi : {
        enabled : boolean;
        protocol : string;
        essid : string;
        enc : string;
        psk : string;
        ip : string;
        netmask : string;
        gateway : string;
        nameserer : string;
    };
}

interface Device extends DeviceProps {
    sensors : {[sensorID : string] : Sensor};
}

interface SensorProps {
    name : string,
    unit : string,
    factor : number,
    port : number
}

interface Sensor extends SensorProps{
    devId : string;
    sensId : string;
}

interface DeviceListScope extends ng.IScope {
    showSpinner : boolean;
    encodeURIComponent : (uriComponent : string) => string;

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



interface DeviceEditorScope {
    showSpinner : boolean;
    deviceProps : DeviceProps;

    errorLoadingSettings : string;
    errorSavingSettings : string;

    ok : () => void;
    cancel : () => void;
}


class DeviceEditorController {

    private url : string;

    constructor(private $scope : DeviceEditorScope,
				private $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
                private $http : ng.IHttpService,
                private deviceID : string) {

        this.url = deviceConfigUrl(deviceID);

        $scope.showSpinner = true;
        $http.get(this.url)
            .success((data : DeviceProps, status, headers, config) => {
                $scope.showSpinner = false;
                $scope.errorLoadingSettings = null;
                $scope.errorSavingSettings = null;

                $scope.deviceProps = data;
            })
            .error((data, status, headers, config) => {
                $scope.showSpinner = false;
                $scope.errorLoadingSettings = data;
            });


        $scope.ok = () => this._saveConfig();
        $scope.cancel = () => this._close();
    }

    private _saveConfig() : void {
        this.$http.post(this.url, this.$scope.deviceProps)
            .success((data, status, headers, config) => {
                this.$scope.errorSavingSettings = null;
                this.$uibModalInstance.close(this.$scope.deviceProps);
            })
            .error((data, status, headers, config) => {
                this.$scope.errorSavingSettings = data;
            });
    }

    private _close() : void {
        this.$uibModalInstance.dismiss('cancel');
    }
}

const DeviceEditorControllerFactory = ["$scope", "$uibModalInstance", "$http", "deviceID",
                                        ($scope, $uibModalInstance, $http, deviceID) =>
                                            new DeviceEditorController($scope, $uibModalInstance, $http, deviceID)];





interface sensorEditorScope {
    showSpinner : boolean;
    sensorProps : SensorProps;

    errorLoadingSettings : string;
    errorSavingSettings : string;

    ok : () => void;
    cancel : () => void;
}

class SensorEditorController {
    private url : string;

    constructor(private $scope : sensorEditorScope,
				private $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
                private $http : ng.IHttpService,
                private deviceID : string,
                private sensorID : string) {

        this.url = sensorConfigUrl(deviceID, sensorID);

        $scope.showSpinner = true;
        $http.get(this.url)
            .success((data : SensorProps, status, headers, config) => {
                $scope.showSpinner = false;
                $scope.errorLoadingSettings = null;
                $scope.errorSavingSettings = null;

                $scope.sensorProps = data;
                console.log(data);
            })
            .error((data, status, headers, config) => {
                $scope.showSpinner = false;
                $scope.errorLoadingSettings = data;
            });


        $scope.ok = () => this._saveConfig();
        $scope.cancel = () => this._close();
    }

    private _saveConfig() : void {
        this.$http.post(this.url, this.$scope.sensorProps)
            .success((data, status, headers, config) => {
                this.$scope.errorSavingSettings = null;
                this.$uibModalInstance.close(this.$scope.sensorProps);
            })
            .error((data, status, headers, config) => {
                this.$scope.errorSavingSettings = data;
            });
    }

    private _close() : void {
        this.$uibModalInstance.dismiss('cancel');
    }
}

const SensorEditorControllerFactory = ["$scope", "$uibModalInstance", "$http", "deviceID", "sensorID",
                                        ($scope, $uibModalInstance, $http, deviceID, sensorID) =>
                                            new SensorEditorController($scope, $uibModalInstance, $http, deviceID, sensorID)];


class DeviceListController {
    public element : ng.IAugmentedJQuery;

    constructor(private $scope : DeviceListScope,
                private $interval : ng.IIntervalService,
                private $http : ng.IHttpService,
                private $uibModal : angular.ui.bootstrap.IModalService) {

        $scope.showSpinner = false;
        $scope.encodeURIComponent = encodeURIComponent;



        $scope.editDevice = (deviceID) => {
            var modalInstance = this.$uibModal.open({
                controller: DeviceEditorControllerFactory,
                size: "lg",
                templateUrl: "/html/device-edit-dialog.html",
                resolve: {
                        deviceID: () => deviceID,
                }
            });

            modalInstance.result.then((props : DeviceProps) : void => {
                $scope.devices[deviceID].name = props.name;
                $scope.devices[deviceID].lan = props.lan;
                $scope.devices[deviceID].wifi = props.wifi
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
            var modalInstance = this.$uibModal.open({
                controller: SensorEditorControllerFactory,
                size: "lg",
                templateUrl: "/html/sensor-edit-dialog.html",
                resolve: {
                        deviceID: () => deviceID,
                        sensorID: () => sensorID
                }
            });

            modalInstance.result.then((props : SensorProps) : void => {
                $scope.devices[deviceID].sensors[sensorID].name = props.name;
            });
        };

        $scope.saveSensor = () => {
            var props = {
                name: $scope.editedSensor.name
            };

            $scope.showSpinner = true;
            $http.post(sensorConfigUrl($scope.editedSensor.devId, $scope.editedSensor.sensId), props)
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

	public controller = ['$scope', '$interval', '$http', '$uibModal', DeviceListController];

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
