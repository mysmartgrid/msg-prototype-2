/// <reference path="angular.d.ts" />

/// <reference path="msg2socket.ts" />
/// <reference path="sensorvaluestore.ts" />
/// <reference path="sensorcollectiongraph.ts" />

"use strict";

module Directives {
	export interface Sensor extends Msg2Socket.SensorMetadata {
		deviceID : string;
		sensorID : string;
		deviceName : string;
	}

	interface SensorUnitMap {
		[unit : string] : {[sensorKey : string] : Sensor};
	}

	interface GraphViewScope extends ng.IScope {
		sensors : SensorUnitMap
	}



	export function sensorKey(deviceID : string, sensorID : string) : string {
			return deviceID + ':' + sensorID;
	}


	export class GraphViewController {
		public graphs : {[unit : string] : SensorCollectionGraphController} = {};

		private realtimeUpdateTimeout : ng.IPromise<any>;

		constructor(private $scope : GraphViewScope, private $timeout : ng.ITimeoutService, private wsclient : Msg2Socket.Socket, updateDispatcher : any) {
			this.$scope.sensors = {};

			this.realtimeUpdateTimeout = null;

			this.wsclient.onMetadata((meta : Msg2Socket.MetadataUpdate) => {
				for(var deviceID in meta.devices) {
					var device = meta.devices[deviceID];
					for(var sensorID in device.sensors) {
						var sensorMetadata = device.sensors[sensorID];
						this.updateSensors(deviceID, sensorID, device.name, sensorMetadata);
					}
					for(var deletedID in device.deletedSensors) {
						//TODO: Implement
					}
				}

				this.requestRealtimeUpdates();
			});


			this.wsclient.onUpdate((update : Msg2Socket.UpdateData) => {
				var values = update.values;
				for(var deviceID in values) {
					for(var sensorID in values[deviceID]) {
						var unit = this.findUnit(deviceID, sensorID);
						values[deviceID][sensorID].forEach((point : [number, number]) => {
							// We ignore updates we don't have metadata for
							if(this.graphs[unit] !== undefined) {
								this.graphs[unit].updateValues(deviceID, sensorID, point[0], point[1])
							}
						});
					}
				}
			});

			this.wsclient.onOpen((err : Msg2Socket.OpenError) => {
				if (err) {
					return;
				}

				var now = (new Date()).getTime();
				this.wsclient.requestValues(now - 120 * 1000, now, "second", true); //Results in Metadata update
			});

		}

		private updateSensors(deviceID : string, sensorID : string, deviceName : string, meta : Msg2Socket.SensorMetadata) : void {
			var unit = this.findUnit(deviceID, sensorID);

			if(unit === undefined) {
				var sensor : Sensor = {
					deviceID : deviceID,
					sensorID : sensorID,
					deviceName : deviceName,
					name : meta.name,
					port : meta.port,
					unit : meta.unit
				};

				if(this.$scope.sensors[meta.unit] === undefined) {
						this.$scope.sensors[meta.unit] = {};
				}
				this.$scope.sensors[meta.unit][sensorKey(deviceID, sensorID)] = sensor;
			}
			else {
				var sensor : Sensor = this.$scope.sensors[unit][sensorKey(deviceID, sensorID)];
				sensor.deviceName = deviceName || sensor.deviceName;
				sensor.name = meta.name || sensor.name;
				sensor.port = meta.port || sensor.port;
				sensor.unit = meta.unit || sensor.unit;
			}
		}

		private requestRealtimeUpdates() : void {
			if(this.realtimeUpdateTimeout !== null) {
				this.$timeout.cancel(this.realtimeUpdateTimeout);
			}

			var sensors : Msg2Socket.RequestRealtimeUpdateArgs = {};

			for(var unit in this.$scope.sensors) {
				for(var key in this.$scope.sensors[unit]) {
					var sensor = this.$scope.sensors[unit][key];
					if(sensors[sensor.deviceID] === undefined) {
						sensors[sensor.deviceID] = {raw : []};
					}
					sensors[sensor.deviceID]['raw'].push(sensor.sensorID);
				}
			}

			this.wsclient.requestRealtimeUpdates(sensors);

			this.realtimeUpdateTimeout = this.$timeout(() => this.requestRealtimeUpdates(), 30 * 1000);
		}

		private findUnit(deviceID : string, sensorID : string) : string {
			var units : string[] = Object.keys(this.$scope.sensors);
			var unit : string[] = units.filter((unit : string) => this.$scope.sensors[unit][sensorKey(deviceID, sensorID)] !== undefined);

			if(unit.length > 1) {
				throw new Error("Multiple units for sensor " + sensorKey(deviceID, sensorID));
			}
			else if(unit.length === 0) {
				return undefined;
			}

			return unit[0];
		}

		public registerGraph(unit : string, graph : SensorCollectionGraphController) : void {
			this.graphs[unit] = graph;
			//console.log(this.graphs);
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

		public controller =	["$scope", "$timeout", "WSUserClient", "UpdateDispatcher", GraphViewController];
	}


	export function GraphViewFactory() : () => ng.IDirective {
		return () => new GraphViewDirective();
	}

}
