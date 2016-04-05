export interface DeviceProps {
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

export interface SensorProps {
    name : string,
    unit : string,
    factor : number,
    port : number
}


function deviceConfigUrl(deviceID : string) : string {
    return '/api/user/v1/device/' + encodeURIComponent(deviceID) + '/config';
}

function sensorConfigUrl(deviceID : string, sensorID : string) : string {
    return '/api/user/v1/sensor/' + encodeURIComponent(deviceID) + '/' + encodeURIComponent(sensorID) + '/props';
}

interface EditorScope {
    showSpinner : boolean;
    props : any;

    errorLoadingSettings : string;
    errorSavingSettings : string;

    ok : () => void;
    cancel : () => void;
}


abstract class EditorController {

    protected url : string;

    constructor(protected $scope : EditorScope,
				protected $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
                protected $http : ng.IHttpService) {

                    $scope.ok = () => this._saveConfig();
                    $scope.cancel = () => this._close();
                }


    protected _loadConfig() : void {
        this.$scope.showSpinner = true;
        this.$http.get(this.url)
            .success((data : DeviceProps, status, headers, config) => {
                this.$scope.showSpinner = false;
                this.$scope.errorLoadingSettings = null;
                this.$scope.errorSavingSettings = null;

                this.$scope.props = data;
            })
            .error((data, status, headers, config) => {
                this.$scope.showSpinner = false;
                if(data !== null) {
                    this.$scope.errorLoadingSettings = data;
                }
                else {
                    this.$scope.errorLoadingSettings = "Ooops something went terribly wrong.";
                }
            });
    }

    protected _saveConfig() : void {
        this.$http.post(this.url, this.$scope.props)
            .success((data, status, headers, config) => {
                this.$scope.errorSavingSettings = null;
                this.$uibModalInstance.close(this.$scope.props);
            })
            .error((data, status, headers, config) => {
                if(data !== null) {
                    this.$scope.errorLoadingSettings = data;
                }
                else {
                    this.$scope.errorLoadingSettings = "Ooops something went terribly wrong.";
                }
            });
    }

    private _close() : void {
        this.$uibModalInstance.dismiss('cancel');
    }
}




interface DeviceEditorScope extends EditorScope {
    props : DeviceProps;
}


class DeviceEditorController extends EditorController {

    constructor(protected $scope : DeviceEditorScope,
				protected $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
                protected $http : ng.IHttpService,
                protected deviceID : string) {

        super($scope, $uibModalInstance, $http);
        this.url = deviceConfigUrl(deviceID);

        this._loadConfig();
    }
}

export const DeviceEditorControllerFactory = ["$scope", "$uibModalInstance", "$http", "deviceID",
                                        ($scope, $uibModalInstance, $http, deviceID) =>
                                            new DeviceEditorController($scope, $uibModalInstance, $http, deviceID)];





interface sensorEditorScope extends EditorScope {
    props : SensorProps;
}

class SensorEditorController  extends EditorController{

    constructor(protected $scope : sensorEditorScope,
				protected $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
                protected $http : ng.IHttpService,
                protected deviceID : string,
                protected sensorID : string) {

        super($scope, $uibModalInstance, $http);

        this.url = sensorConfigUrl(deviceID, sensorID);

        this._loadConfig();
    }
}

export const SensorEditorControllerFactory = ["$scope", "$uibModalInstance", "$http", "deviceID", "sensorID",
                                        ($scope, $uibModalInstance, $http, deviceID, sensorID) =>
                                            new SensorEditorController($scope, $uibModalInstance, $http, deviceID, sensorID)];







const DeviceAddUrl = "/api/user/v1/device/";

interface DeviceAddScope {
    errorAddingDevice : string;
    deviceId : string;

    ok : () => void;
    cancel : () => void;
}

class DeviceAddController {

    constructor(protected $scope : DeviceAddScope,
				protected $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
                protected $http : ng.IHttpService) {

                    $scope.ok = () => this._addDevice();
                    $scope.cancel = () => this._close();
                }

    private _addDevice() : void {
        this.$http.post(DeviceAddUrl + encodeURIComponent(this.$scope.deviceId), null)
			.success((data, status, headers, config) => {
                this.$uibModalInstance.close({deviceID: this.$scope.deviceId, data: data});
			})
			.error((data, status, headers, config) => {
				this.$scope.errorAddingDevice = data;
			});
    }

    private _close() : void {
        this.$uibModalInstance.dismiss('cancel');
    }
}

export const DeviceAddControllerFactory = ["$scope", "$uibModalInstance", "$http",
                                        ($scope, $uibModalInstance, $http) =>
                                            new DeviceAddController($scope, $uibModalInstance, $http)];
