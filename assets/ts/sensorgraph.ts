/// <reference path="angular.d.ts" />
/// <reference path="angular-ui-bootstrap.d.ts" />

/// <reference path="common.ts"/>
/// <reference path="msg2socket.ts" />
/// <reference path="sensorvaluestore.ts" />
/// <reference path="graphview.ts" />

"use strict";

declare var Flotr : any;

module Directives {

	interface SensorGraphConfig {
		unit : string,
		resolution : string;
		sensors : Sensor[];
		mode : string,
		intervalStart : number;
		intervalEnd : number;
		windowStart : number;
		windowEnd : number;
	}



	interface SensorGraphSettingsScope extends SensorGraphScope {
		pickerModes : {[resoltuion : string] : string};
		resolutions : string[];
		units : string[];
		config : SensorGraphConfig;
		sensorsByUnit : UpdateDispatcher.UnitSensorMap;
		ok : () => void;
		cancel : () => void;
	}


	const SensorGraphSettingsFactory = ["$scope", "$uibModalInstance", "UpdateDispatcher", "config",
										($scope : SensorGraphSettingsScope,
											$uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
											dispatcher : UpdateDispatcher.UpdateDispatcher,
											config : SensorGraphConfig) => new SensorGraphSettingsController($scope, $uibModalInstance, dispatcher, config)];

	class SensorGraphSettingsController {
		constructor(private $scope : SensorGraphSettingsScope,
					private $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
					private _dispatcher : UpdateDispatcher.UpdateDispatcher,
					config : SensorGraphConfig) {

			this.$scope.devices = _dispatcher.devices;
			this.$scope.resolutions = Array.from(UpdateDispatcher.SupportedResolutions.values());
			this.$scope.units = _dispatcher.units;
			this.$scope.sensorsByUnit = _dispatcher.sensorsByUnit;

			this._configToScope(config);

			$scope.pickerModes = {
				raw : 'day',
				second : 'day',
				minute : 'day',
				hour : 'day',
				day : 'day',
				week : 'day',
				month : 'month',
				year : 'year'
			}


			$scope.ok = () : void => {

				$uibModalInstance.close();
			};

			$scope.cancel = () : void => {
				$uibModalInstance.dismiss('cancel');
			};
		}

		private _configToScope(config : SensorGraphConfig) {



			this.$scope.config = {
				unit : config.unit,
				resolution : config.resolution,
				sensors : config.sensors,
				mode : config.mode,
				windowStart : 0,
				windowEnd : 0,
				intervalStart: 0,
				intervalEnd: 0,
			};

		}
	}

	interface SensorGraphScope  extends ng.IScope {
		openSettings : () => void;

		sensors : Sensor[];
		sensorColors : {[device : string] : {[sensor : string] : string}};
		devices : UpdateDispatcher.DeviceMap;
	}

	export class SensorGraphController implements UpdateDispatcher.Subscriber{
		private _store : Store.SensorValueStore
		private _config : SensorGraphConfig;
		private _graphNode : HTMLElement

		public set graphNode(element: ng.IAugmentedJQuery) {
			this._graphNode = element.find(".sensor-graph").get(0);
		}

		constructor(private $scope : SensorGraphScope,
					private $interval : ng.IIntervalService,
					private $timeout: ng.ITimeoutService,
					private $uibModal : angular.ui.bootstrap.IModalService,
					private _dispatcher : UpdateDispatcher.UpdateDispatcher) {

			this._store = new Store.SensorValueStore();
			this._store.setSlidingWindowMode(true);
			this._store.setEnd(0);

			this.$scope.devices = this._dispatcher.devices;


			this._dispatcher.onInitialMetadata(() => {
				//TODO: Add on config callback here
				this._setDefaultConfig();
				this._redrawGraph();
			});

			this.$scope.openSettings = () => {
				var modalInstance = $uibModal.open({
					controller: SensorGraphSettingsFactory,
					size: "lg",
				    templateUrl: 'sensor-graph-settings.html',
					resolve: {
  						config: () : SensorGraphConfig => {
							return this._config;
  						}
					}
				});

				modalInstance.result.then((config : SensorGraphConfig) : void => {
					console.log(config);

					this.$scope.sensors = config.sensors;
				});
			};

			$interval(() => this._store.clampData(), 1000);
		}


		public updateValue(deviceID : string, sensorID : string, resolution : string, timestamp : number, value : number) : void {
			this._store.addValue(deviceID, sensorID, timestamp, value);
		}

		public updateDeviceMetadata(deviceID : string) : void {};

		public updateSensorMetadata(deviceID : string, sensorID : string) {
		};

		public removeDevice(deviceID : string) {};

		public removeSensor(deviceID : string, sensorID : string) {};

		private _setDefaultConfig() {
			this._config = {
				unit : this._dispatcher.units[0],
				resolution : UpdateDispatcher.SupportedResolutions.values().next().value,
				sensors : [],
				mode: 'realtime',
				intervalStart : 0,
				intervalEnd : 0,
				windowStart : 5 * 60 * 1000,
				windowEnd : 0
			};
		}


		/*private _updateSettings() {
			var map = {
				raw: 1000,
				second: 1000,
				minute: 60 * 1000,
				hour: 60 * 60 * 1000,
				day: 24 * 60 * 60 * 1000,
				week: 7 * 24 * 60 * 60 * 1000,
				month: 31 * 24 * 60 * 60 * 1000,
				year: 365 * 24 * 60 * 60 * 1000
			};

			this._timeResolution = map[this.$scope.resolution];

			if(this.$scope.slidingWindow) {
				this._intervalEnd = this.$scope.intervalEnd * this._timeResolution;
				this._intervalStart = this.$scope.intervalStart * this._timeResolution;

				this._store.setSlidingWindowMode(true);

			}
			else {
				this._store.setSlidingWindowMode(false);
			}

			this._store.setStart(this._intervalStart);
			this._store.setEnd(this._intervalEnd);

			this._dispatcher.unsubscribeAll(this);

			if(this.$scope.sensorsByUnit !== undefined && this.$scope.sensorsByUnit[this.$scope.unit] !== undefined) {
				for(var sensor of this.$scope.sensorsByUnit[this.$scope.unit]) {
					if(this._store.hasSensor(sensor.deviceID, sensor.sensorID)) {
						this._store.removeSensor(sensor.deviceID, sensor.sensorID);
					}
				}
			}

			if(this.$scope.sensors !== undefined) {
				for(var sensor of this.$scope.sensors) {
					this._store.addSensor(sensor.deviceID, sensor.sensorID, sensor.sensorID);
					if(this.$scope.slidingWindow && this._intervalEnd === 0) {
						this._dispatcher.subscribeRealtimeSlidingWindow(sensor.deviceID, sensor.sensorID, this.$scope.resolution, this._intervalStart, this);
					}
					else if(this.$scope.slidingWindow) {
						this._dispatcher.subscribeSlidingWindow(sensor.deviceID, sensor.sensorID, this.$scope.resolution, this._intervalStart, this._intervalEnd, this);
					}
					else {
						this._dispatcher.subscribeInterval(sensor.deviceID, sensor.sensorID, this.$scope.resolution, this._intervalStart, this._intervalEnd, this);
					}
				}
			}
		}*/


		private _redrawGraph() {

			var time = Common.now();

			var graphOptions : any = {
				xaxis: {
					mode: 'time',
					timeMode : 'local',
					title: 'Uhrzeit'
				},
				HtmlText: false,
				preventDefault : false,
				title: 'Messwerte',
				shadowSize: 0,
				lines: {
					lineWidth: 2,
				}
			}

			graphOptions.title = 'Messwerte [' + this._config.unit + ']';

			var delay;

			if(this._config.mode === "slidingWindow" || this._config.mode === "realtime") {
				graphOptions.xaxis.min = time - this._config.windowStart;
				if(this._config.mode === "realtime") {
					graphOptions.xaxis.max = time;
					delay = this._config.windowStart;
				}
				else {
					graphOptions.xaxis.max = time - this._config.windowEnd;
					delay = this._config.windowStart -  this._config.windowEnd;
				}
			}
			else {
				graphOptions.xaxis.min = this._config.intervalStart;
				graphOptions.xaxis.max = this._config.intervalEnd;
				delay = this._config.intervalStart -  this._config.intervalEnd;
			}

			var graph = Flotr.draw(this._graphNode, this._store.getData(), graphOptions);

			this.$timeout(() => this._redrawGraph(), delay / graph.plotWidth);
		}
	}

	class SensorGraphDirective implements ng.IDirective {
		public require : string = "sensorGraph"
		public restrict : string = "A"
		public templateUrl : string = "/html/sensor-graph.html"
		public scope = {}

		public controller = ["$scope", "$interval", "$timeout", "$uibModal", "UpdateDispatcher", SensorGraphController];

		// Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
		public link:Function  = ($scope : SensorGraphScope,
									element : ng.IAugmentedJQuery,
									attrs : ng.IAttributes,
									sensorGraph : SensorGraphController) : void => {



			sensorGraph.graphNode = element;
		}
	}

	export function SensorGraphFactory() : () => ng.IDirective {
		return () => new SensorGraphDirective();
	}

}
