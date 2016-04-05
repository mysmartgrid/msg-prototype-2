import * as Msg2Socket from './lib/msg2socket';
import * as UpdateDispatcher from './lib/updatedispatcher';

import NumberSpinnerFactory from './directives/ui-elements/numberspinner';
import TimeRangeSpinnerFactory from './directives/ui-elements/timerangespinner';
import DateTimePickerFactory from './directives/ui-elements/datetimepicker';
import SensorGraphFactory from './directives/sensorgraph';

import DeviceListFactory from './directives/devicelist';
import {DeviceAddControllerFactory} from './controllers/deviceeditors';


angular.module("msgp", ['ui.bootstrap'])
.config(["$interpolateProvider", ($interpolateProvider : ng.IInterpolateProvider) => {
	$interpolateProvider.startSymbol("%%");
	$interpolateProvider.endSymbol("%%");
}])
.factory("WSUserClient", ["$rootScope", ($rootScope : angular.IRootScopeService) => {
	if (!window["WebSocket"])
		throw "websocket support required";
	return new Msg2Socket.Socket($rootScope);
}])
.factory("UpdateDispatcher", UpdateDispatcher.UpdateDispatcherFactory)
.directive("numberSpinner", NumberSpinnerFactory())
.directive("timeRangeSpinner", TimeRangeSpinnerFactory())
.directive("dateTimePicker", DateTimePickerFactory())
.directive("sensorGraph", SensorGraphFactory())
.directive("deviceList", DeviceListFactory())
.controller("GraphPage", ["WSUserClient", "wsurl", "$http", "$timeout", "$uibModal", function(wsclient, wsurl, $http, $timeout : ng.ITimeoutService, $uibModal) {
	wsclient.connect(wsurl);

	var modalInstance = null;

	wsclient.onClose(() : void => {
		if(modalInstance === null) {
			modalInstance = $uibModal.open({
				size: "lg",
				keyboard: false,
				backdrop : 'static',
				templateUrl: 'connection-lost.html',
			});
		}

		$timeout(() : void => wsclient.connect(wsurl), 1000);
	});

	wsclient.onOpen(() : void => {
		if(modalInstance !== null) {
			modalInstance.close();
		}
	});
}])
.controller("DeviceListController", ["$scope", "$uibModal", "$http", ($scope, $uibModal, $http) => {
	$http.get('/api/user/v1/devices').success((data, status, headers, config) => {
		$scope.devices = data;
	});

	$scope.openAddDeviceModal = () : void => {
		var modalInstance = $uibModal.open({
			controller: DeviceAddControllerFactory,
			size: "lg",
			templateUrl: "/html/add-device-dialog.html",
		});

		modalInstance.result.then((data) => {
			$scope.devices[data.deviceID] = data.data;
		});
	}
}]);

console.log('MSGP loaded');
