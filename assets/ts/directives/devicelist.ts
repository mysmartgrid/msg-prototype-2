import {DeviceAddControllerFactory,
        DeviceProps,
        SensorProps,
        DeviceEditorControllerFactory,
        SensorEditorControllerFactory} from '../controllers/deviceeditors';


interface Device extends DeviceProps {
    sensors : {[sensorID : string] : Sensor};
}


interface Sensor extends SensorProps{
    devId : string;
    sensId : string;
}

// TODO: Move to own file
export interface DeviceList {
    [deviceID : string] : Device;
}

interface DeviceListScope extends ng.IScope {
    showSpinner : boolean;
    encodeURIComponent : (uriComponent : string) => string;

    addDevice : () => void;

    errorSavingSettings : string;
    errorLoadingSettings : string;
    errorSavingSensor : string;

    devices : DeviceList;

    editDevice : (deviceID : string) => void;
    remove : (deviceID : string) => void;

    editSensor : (deviceID : string, sensorID : string) => void;

    error : string;

    editedSensor : Sensor;
}


function deviceRemoveUrl(deviceID : string) : string {
    return '/api/user/v1/device/' + encodeURIComponent(deviceID);
}



class DeviceListController {
    public element : ng.IAugmentedJQuery;

    constructor(private $scope : DeviceListScope,
                private $interval : ng.IIntervalService,
                private $http : ng.IHttpService,
                private $uibModal : angular.ui.bootstrap.IModalService) {

        $scope.showSpinner = false;
        $scope.encodeURIComponent = encodeURIComponent;

        $http.get('/api/user/v1/devices').success((data : {[deviceID : string] : Device}, status, headers, config) => {
    		$scope.devices = data;
    	});

        $scope.addDevice = () : void => {
            var modalInstance = $uibModal.open({
                controller: DeviceAddControllerFactory,
                size: "lg",
                templateUrl: "/html/add-device-dialog.html",
            });

            modalInstance.result.then((data) => {
                $scope.devices[data.deviceID] = data.data;
            });
        }


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
