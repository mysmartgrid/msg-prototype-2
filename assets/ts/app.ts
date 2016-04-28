import {Msg2SocketFactory} from './lib/msg2socket';
import {ServerTimeFactory, ServerTime} from './lib/servertime';
import {UpdateDispatcherFactory} from './lib/updatedispatcher';

import NumberSpinnerFactory from './directives/ui-elements/numberspinner';
import TimeRangeSpinnerFactory from './directives/ui-elements/timerangespinner';
import DateTimePickerFactory from './directives/ui-elements/datetimepicker';
import SensorGraphFactory from './directives/sensorgraph';

import DeviceListFactory from './directives/devicelist';
import GroupListFactory from './directives/grouplist';


angular.module("msgp", ['ui.bootstrap', 'treasure-overlay-spinner'])
.config(["$interpolateProvider", ($interpolateProvider : ng.IInterpolateProvider) => {
	$interpolateProvider.startSymbol("%%");
	$interpolateProvider.endSymbol("%%");
}])
.factory("WSUserClient", Msg2SocketFactory)
.factory("UpdateDispatcher", UpdateDispatcherFactory)
.factory("ServerTime", ServerTimeFactory)
.directive("numberSpinner", NumberSpinnerFactory())
.directive("timeRangeSpinner", TimeRangeSpinnerFactory())
.directive("dateTimePicker", DateTimePickerFactory())
.directive("sensorGraph", SensorGraphFactory())
.directive("deviceList", DeviceListFactory())
.directive("groupList", GroupListFactory())
.controller("GraphPage", ["WSUserClient", "wsurl", "$http", "$timeout", "$uibModal",
	(wsclient, wsurl, $http, $timeout : ng.ITimeoutService, $uibModal) => {
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
.controller("NavbarServerTime", ["ServerTime", "$scope", "$interval", (serverTime : ServerTime, $scope : any, $interval : ng.IIntervalService) => {
	function displayTime() {
		$scope.time = serverTime.now();
	}

	$interval(displayTime, 1000);
	displayTime();
}]);


console.log('MSGP loaded');
