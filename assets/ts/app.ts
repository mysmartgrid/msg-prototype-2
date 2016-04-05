import * as Msg2Socket from './lib/msg2socket';
import * as UpdateDispatcher from './lib/updatedispatcher';

import NumberSpinnerFactory from './directives/ui-elements/numberspinner';
import TimeRangeSpinnerFactory from './directives/ui-elements/timerangespinner';
import DateTimePickerFactory from './directives/ui-elements/datetimepicker';
import SensorGraphFactory from './directives/sensorgraph';

import DeviceListFactory from './directives/devicelist';


angular.module("msgp", ['ui.bootstrap'])
.config(function($interpolateProvider) {
	$interpolateProvider.startSymbol("%%");
	$interpolateProvider.endSymbol("%%");
})
.factory("WSUserClient", ["$rootScope", function($rootScope : angular.IRootScopeService) {
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
.controller("DeviceListController", ["$scope", "$http", "devices", function($scope, $http, devices) {
	$scope.devices = devices;
	$scope.addDeviceId = "";

	$scope.openAddDeviceModal = () : void => {
		$scope.addDeviceId = "";
		$('#addDeviceDialog').modal();
	}

	$scope.addDevice = function(e) {
		var url = $(e.target).attr("data-add-device-prefix");
		$scope.errorAddingDevice = null;

		$http.post(url + encodeURIComponent($scope.addDeviceId))
			.success(function(data, status, headers, config) {
				$scope.devices[$scope.addDeviceId] = data;
				$scope.addDeviceId = null;
				$("#addDeviceDialog").modal('hide');
			})
			.error(function(data, status, headers, config) {
				$scope.errorAddingDevice = data;
			});
	};
}]);
