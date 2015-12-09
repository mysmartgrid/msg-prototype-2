/// <reference path="angular.d.ts" />

/// <reference path="msg2socket.ts" />
/// <reference path="updatedispatcher.ts"/>
/// <reference path="sensorvaluestore.ts" />
/// <reference path="sensorcollectiongraph.ts" />

"use strict";

module Directives {
	export interface Sensor {
		deviceID : string;
		sensorID : string;
	}

	interface SensorUnitMap {
		[unit : string] : Sensor[];
	}

	interface GraphViewScope extends ng.IScope {
		sensorsByUnit : SensorUnitMap;
		devices : UpdateDispatcher.DeviceMap;
	}


	export function sensorKey(deviceID : string, sensorID : string) : string {
			return deviceID + ':' + sensorID;
	}


	export class GraphViewController {



		constructor(private $scope : GraphViewScope, private $timeout : ng.ITimeoutService, private dispatcher: UpdateDispatcher.UpdateDispatcher) {

			this.$scope.sensorsByUnit = {};

			$scope.$watch('devices', () => this._generateSensorByUnit());

			dispatcher.onInitialMetadata(() => this.$scope.devices = dispatcher.devices);
		}

		private _generateSensorByUnit() : void {
			for(var deviceID in this.$scope.devices) {
				for(var sensorID in this.$scope.devices[deviceID].sensors) {
					var sensor = this.$scope.devices[deviceID].sensors[sensorID];
					if(this.$scope.sensorsByUnit[sensor.unit] === undefined) {
						this.$scope.sensorsByUnit[sensor.unit] = [];
					}

					this.$scope.sensorsByUnit[sensor.unit].push({deviceID : deviceID, sensorID : sensorID});
				}
			}
		}
	}

	class GraphViewDirective implements ng.IDirective {

		public restrict : string =  "A";
		public templateUrl : string =  "/html/graph-view.html";
		public scope = {
						title : "@"
					};

		// Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
		public link:Function  = ($scope : GraphViewScope,
									element : ng.IAugmentedJQuery,
									attrs : ng.IAttributes,
									controller : GraphViewController) => {
		};

		constructor() {};

		public controller =	["$scope", "$timeout", "UpdateDispatcher", GraphViewController];
	}


	export function GraphViewFactory() : () => ng.IDirective {
		return () => new GraphViewDirective();
	}

}
