/// <reference path="angular.d.ts" />

/// <reference path="msg2socket.ts" />
/// <reference path="sensorvaluestore.ts" />
/// <reference path="graphview.ts" />

"use strict";

declare var Flotr : any;

module Directives {

	interface SensorCollectionGraphScope  extends ng.IScope {
		unit : string,
		maxAgeMs : number;
		assumeMissingAfterMs : number;
		sensors : Sensor[];
		sensorColors : {[device : string] : {[sensor : string] : string}};
	}

	export class SensorCollectionGraphController implements UpdateDispatcher.Subscriber{
		private store : Store.SensorValueStore
		private graphOptions : any
		private graphNode : HTMLElement

		constructor(private $scope : SensorCollectionGraphScope,
					private $interval : ng.IIntervalService,
					private $timeout: ng.ITimeoutService,
					private _dispatcher : UpdateDispatcher.UpdateDispatcher) {

			this.store = new Store.SensorValueStore();
			this.store.setSlidingWindowMode(true);
			this.store.setEnd(0);

			$scope.$watch('maxAgeMs', (start : number) => this.store.setStart(start));

			$scope.$watch('assumeMissingAfterMs', (timeout : number) => this.store.setTimeout(timeout));

			$scope.$watch('sensors', () => this.updateSensors());

			$interval(() => this.store.clampData(), 1000);
		}


		public updateValue(deviceID : string, sensorID : string, resolution : string, timestamp : number, value : number) : void {
			this.store.addValue(deviceID, sensorID, timestamp, value);
		}

		public updateDeviceMetadata(deviceID : string) : void {};

		public updateSensorMetadata(deviceID : string, sensorID : string) {};

		public removeDevice(deviceID : string) {};

		public removeSensor(deviceID : string, sensorID : string) {};


		private updateSensors() : void {
			var labels = this.store.getLabels();
			for(var sensor of this.$scope.sensors) {
				var metadata = this._dispatcher.devices[sensor.deviceID].sensors[sensor.sensorID];

				if(!this.store.hasSensor(sensor.deviceID, sensor.sensorID)) {
					this.store.addSensor(sensor.deviceID, sensor.sensorID, metadata.name);
					this._dispatcher.subscribeRealtimeSlidingWindow(sensor.deviceID, sensor.sensorID, "raw", this.$scope.maxAgeMs, this);
				}
				else if(labels[sensor.deviceID][sensor.sensorID] !== metadata.name) {
					this.store.setLabel(sensor.deviceID, sensor.sensorID, metadata.name);
				}
				else {
					delete labels[sensor.deviceID][sensor.sensorID];
				}
			}

			for(var deviceID in labels) {
				for(var sensorID in labels[deviceID]) {
					this.store.removeSensor(deviceID, sensorID);
					this._dispatcher.unsubscribeSensor(deviceID, sensorID, "raw", this);
				}
			}

			this.$scope.sensorColors = this.store.getColors();
		}


		public createGraph(element: ng.IAugmentedJQuery) {
			this.graphOptions = {
				xaxis: {
            		mode: 'time',
					timeMode : 'local',
					title: 'Uhrzeit'
				},
        		HtmlText: false,
				preventDefault : false,
        		title: 'Messwerte [' + this.$scope.unit + ']',
				shadowSize: 0,
				lines: {
					lineWidth: 2,
				}
			}

			this.graphNode = element.find(".sensor-graph").get(0);

			this.redrawGraph();
		}

		private redrawGraph() {
			var time = (new Date()).getTime();
			this.graphOptions.xaxis.max = time - 1000;
			this.graphOptions.xaxis.min = time - this.$scope.maxAgeMs + 1000;

			var graph = Flotr.draw(this.graphNode, this.store.getData(), this.graphOptions);

			var delay = (this.$scope.maxAgeMs - 2000) / graph.plotWidth;
			this.$timeout(() => this.redrawGraph(), delay);
		}
	}

	class SensorCollectionGraphDirective implements ng.IDirective {
		public require : string[] = ["^graphView", "sensorCollectionGraph"]
		public restrict : string = "A"
		public templateUrl : string = "/html/sensor-collection-graph.html"
		public scope = {
			unit: "=",
			sensors: "=",
			maxAgeMs: "=",
			assumeMissingAfterMs: "=",
		}

		public controller = ["$scope", "$interval", "$timeout", "UpdateDispatcher", SensorCollectionGraphController];

		// Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
		public link:Function  = ($scope : SensorCollectionGraphScope,
									element : ng.IAugmentedJQuery,
									attrs : ng.IAttributes,
									controllers : [GraphViewController, SensorCollectionGraphController]) : void => {

			var graphView : GraphViewController = controllers[0];
			var sensorCollectionGraph : SensorCollectionGraphController = controllers[1];

			sensorCollectionGraph.createGraph(element);
		}
	}



	export function SensorCollectionGraphFactory() : () => ng.IDirective {
		return () => new SensorCollectionGraphDirective();
	}


}
