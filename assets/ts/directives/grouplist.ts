import {AddSensorToGroupFactory} from '../controllers/addsensortogroup';


interface SharedSensor {
    deviceId : string,
    sensorId : string,
    deviceName : string,
    sensorName : string,
    owner : string
}

interface Group {
    members : {[user : string] : boolean};
    sensors : SharedSensor[];
}

interface GroupResponse {
    user : string,
    groups : {[name : string] : Group};
}

interface GroupListScope extends ng.IScope {
    user : string;
    groups : {[name : string] : Group};

    addSensorToGroup : (group : string) => void;

    removeSensor : (group : string, deviceId : string, sensorId : string) => void;

    message : string;
    error : string;
}


function getGroupsUrl() : string {
    return '/api/user/v1/groups'
}

function addGroupSensorUrl(groupName : string) : string {
    return '/api/user/v1/group/' + encodeURIComponent(groupName) + '/sensor/add';
}

function removeSensorfromGroupUrl(groupName : string, deviceId : string, sensorId : string) : string {
    return '/api/user/v1/group/' + encodeURIComponent(groupName) + '/sensor/' + encodeURIComponent(deviceId) + '/' + encodeURIComponent(sensorId);
}


class GroupListController {
    public element : ng.IAugmentedJQuery;

    private _timeout : ng.IPromise<any>;

    constructor(private $scope : GroupListScope,
                private $timeout : ng.ITimeoutService,
                private $http : ng.IHttpService,
                private $uibModal : angular.ui.bootstrap.IModalService) {



        $scope.message = null;
        $scope.error = null;

        this._updateData();


        $scope.addSensorToGroup = (group) => {
            var modalInstance = $uibModal.open({
                controller: AddSensorToGroupFactory,
                resolve: {
                    group: () => group,
                },
                size: "lg",
                templateUrl: "/html/add-sensor-to-group-dialog.html",
            });

            modalInstance.result.then(() => {
                this._updateData();
            });
        }

        $scope.removeSensor = (group, deviceId, sensorId) => {
            this.$http.delete(removeSensorfromGroupUrl(group, deviceId, sensorId))
            .success((data, status, headers, config) => {
                this._showMessage("Successfully removed sensor.");
                this._updateData();
            })
            .error((data, status, headers, config) => {
                if(data !== null) {
                    this.$scope.error = data;
                }
                else {
                    this.$scope.error = "Ooops something went terribly wrong.";
                }
            });
        }
    }

    private _showMessage(message : string) {
        this.$scope.message = message;

        if(this._timeout !== undefined) {
            this.$timeout.cancel(this._timeout);
        }

        this._timeout = this.$timeout(() => {
            this.$scope.message = null;
            this._timeout = undefined;
        }, 2000);
    }

    private _updateData() : void {
        this.$http.get(getGroupsUrl()).success((data : GroupResponse, status, headers, config) => {
            this.$scope.user = data.user;
            this.$scope.groups = data.groups;
        }).error((data, status, headers, config) => {
            if(data !== null) {
                this.$scope.error = data;
            }
            else {
                this.$scope.error = "Ooops something went terribly wrong.";
            }
        });
    }
}


class GroupListDirective implements ng.IDirective {
    public require : string = "groupList";
    public restrict = "E";
    public templateUrl = "/html/group-list.html";
    public scope = {};

	public controller = ['$scope', '$timeout', '$http', '$uibModal', GroupListController];

    // Implementing this as a method will not work as the this binding will break somewhere in angulars guts.
	public link:Function  = ($scope : GroupListScope,
								element : ng.IAugmentedJQuery,
								attrs : ng.IAttributes,
								deviceList : GroupListController) : void => {

		deviceList.element = element;
        console.log("link");
	}
}

export default function GroupListFactory() : () => ng.IDirective {
	return () => new GroupListDirective();
}
