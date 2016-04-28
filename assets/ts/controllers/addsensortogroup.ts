import {DeviceList} from '../directives/devicelist';


interface SensorAddScope {
    showSpinner : boolean;
    deviceId : string;
    sensorId : string;

    group : string;
    devices : DeviceList;

    errorAddingSensor: string;

    ok : () => void;
    cancel : () => void;
}


function deviceListUrl() {
    return '/api/user/v1/devices';
}

function addSensorToGroupUrl(group : string) {
    return '/api/user/v1/group/' + group + '/sensor/add';
}

export class AddSensorToGroupController {

    constructor(private $scope : SensorAddScope,
				private $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
                private $http : ng.IHttpService,
                private _group : string) {

                    this._loadDevices();

                    $scope.group = _group;

                    $scope.ok = () => this._addSensor();
                    $scope.cancel = () => this._close();
                }


    private _loadDevices() : void {
        this.$scope.showSpinner = true;
        this.$http.get(deviceListUrl())
            .success((data : DeviceList, status, headers, config) => {
                this.$scope.showSpinner = false;
                this.$scope.errorAddingSensor = null;

                this.$scope.devices = data;
            })
            .error((data, status, headers, config) => {
                this.$scope.showSpinner = false;
                if(data !== null) {
                    this.$scope.errorAddingSensor = data;
                }
                else {
                    this.$scope.errorAddingSensor = "Ooops something went terribly wrong.";
                }
            });
    }


    private _addSensor() : void {
        this.$http.post(addSensorToGroupUrl(this._group), {
            deviceId : this.$scope.deviceId,
            sensorId : this.$scope.sensorId
        })
        .success((data, status, headers, config) => {
            this.$scope.errorAddingSensor = null;
            this.$uibModalInstance.close();
        })
        .error((data, status, headers, config) => {
            if(data !== null) {
                this.$scope.errorAddingSensor = data;
            }
            else {
                this.$scope.errorAddingSensor = "Ooops something went terribly wrong.";
            }
        });
    }


    private _close() : void {
        this.$uibModalInstance.dismiss('cancel');
    }
}

export const AddSensorToGroupFactory = ["$scope", "$uibModalInstance", "$http", "group",
                    ($scope, $uibModalInstance, $http, group) => new AddSensorToGroupController($scope, $uibModalInstance, $http, group)]
