import * as Msg2Socket from './lib/msg2socket';
import * as UpdateDispatcher from './lib/updatedispatcher';

import NumberSpinnerFactory from './directives/ui-elements/numberspinner';
import TimeRangeSpinnerFactory from './directives/ui-elements/timerangespinner';
import DateTimePickerFactory from './directives/ui-elements/datetimepicker';
import SensorGraphFactory from './directives/sensorgraph';



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
.directive("deviceEditor", [function() {
	return {
		restrict: "A",
		templateUrl: "/html/device-editor.html",
		scope: {
			device: "="
		},
		link: function(scope, element, attrs) {
		}
	};
}])
.directive("deviceList", ["$http", "$interval", function($http, $interval) {
	return {
		restrict: "A",
		templateUrl: "/html/device-list.html",
		scope: {
			devices: "="
		},
		link: function(scope, element, attrs) {
			scope.showSpinner = false;
			scope.encodeURIComponent = encodeURIComponent;

			scope.deviceEditorSave = function() {
				$http.post(scope.editedDeviceURL, scope.editedDeviceProps)
					.success(function(data, status, headers, config) {
						scope.devices[scope.editedDeviceId].name = scope.editedDeviceProps.name;
						scope.devices[scope.editedDeviceId].lan = scope.editedDeviceProps.lan;
						scope.devices[scope.editedDeviceId].wifi = scope.editedDeviceProps.wifi;
						scope.editedDeviceId = undefined;
						scope.errorSavingSettings = null;
						$("#deviceEditDialog").modal('hide');
					})
					.error(function(data, status, headers, config) {
						scope.errorSavingSettings = data;
					});
			};

			var flash = function(element) {
				element.removeClass("ng-hide");
				$interval(function() {
					element.addClass("ng-hide");
				}, 3000, 1);
			};

			scope.editDev = function(e) {
				var id = $(e.target).parents("tr[data-device-id]").first().attr("data-device-id");
				var url = $(e.target).parents("tr[data-device-id]").first().attr("data-device-netconf-url");

				scope.showSpinner = true;
				$http.get(url)
					.success(function(data, status, headers, config) {
						scope.showSpinner = false;
						scope.errorLoadingSettings = null;
						scope.errorSavingSettings = null;

						scope.editedDeviceId = id;
						scope.editedDeviceURL = url;
						scope.editedDeviceProps = {
							name: scope.devices[id].name,
							lan: data.lan || {},
							wifi: data.wifi || {}
						};
						$("#deviceEditDialog").modal('show');
					})
					.error(function(data, status, headers, config) {
						scope.showSpinner = false;
						scope.errorLoadingSettings = data;
					});
			};

			scope.remove = function(e) {
				var url = $(e.target).parents("tr[data-device-id]").first().attr("data-device-remove-url");
				var id = $(e.target).parents("tr[data-device-id]").first().attr("data-device-id");
				scope.showSpinner = true;
				$http.delete(url)
					.success(function(data, status, headers, config) {
						scope.showSpinner = false;
						delete scope.devices[id];
						flash($(e.target).parents(".device-list-").first().find(".device-deleted-"));
					})
					.error(function(data, status, headers, config) {
						scope.showSpinner = false;
						scope.error = data;
					});
			};

			scope.editSensor = function(e) {
				var devId = $(e.target).parents("tr[data-device-id]").first().attr("data-device-id");
				var sensId = $(e.target).parents("tr[data-sensor-id]").first().attr("data-sensor-id");
				var url = $(e.target).parents("tr[data-sensor-conf-url]").first().attr("data-sensor-conf-url");

				scope.errorSavingSensor = null;
				scope.editedSensor = {
					name: scope.devices[devId].sensors[sensId].name,
					confUrl: url,
					devId: devId,
					sensId: sensId,
				};
				$("#sensorEditDialog").modal('show');
			};

			scope.saveSensor = function() {
				var props = {
					name: scope.editedSensor.name
				};

				scope.showSpinner = true;
				$http.post(scope.editedSensor.confUrl, props)
					.success(function(data, status, headers, config) {
						scope.showSpinner = false;
						scope.devices[scope.editedSensor.devId].sensors[scope.editedSensor.sensId].name = props.name;
						scope.editedSensor = null;
						$("#sensorEditDialog").modal('hide');
					})
					.error(function(data, status, headers, config) {
						scope.showSpinner = false;
						scope.errorSavingSensor = data;
					});
			};
		}
	};
}])
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
