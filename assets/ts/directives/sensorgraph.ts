import * as Utils from '../lib/utils';
import {ServerTime} from '../lib/servertime';
import * as UpdateDispatcher from '../lib/updatedispatcher';
import * as Store from '../lib/sensorvaluestore';

import {SensorSpecifier, MetadataTree, SensorUnitMap, DeviceSensorMap,
		sensorEqual, SupportedResolutions, ResoltuionToMillisecs, ResolutionsPerMode} from '../lib/common';


import * as Widget from './widget';

declare var Flotr : any;


interface SensorGraphConfig extends Widget.WidgetConfig{
	unit : string,
	resolution : string;
	sensors : SensorSpecifier[];
	mode : string,
	intervalStart : number;
	intervalEnd : number;
	windowStart : number;
	windowEnd : number;
}


interface SensorGraphSettingsScope extends Widget.WidgetSettingsScope {
	config : SensorGraphConfig;
}


const SensorGraphSettingsFactory = ["$scope", "$uibModalInstance", "UpdateDispatcher", "config",
									($scope : SensorGraphSettingsScope,
										$uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
										dispatcher : UpdateDispatcher.UpdateDispatcher,
										config : SensorGraphConfig) => new SensorGraphSettingsController($scope, $uibModalInstance, dispatcher, config)];

class SensorGraphSettingsController extends Widget.WidgetSettingsController {
	constructor(protected $scope : SensorGraphSettingsScope,
				protected $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
				protected _dispatcher : UpdateDispatcher.UpdateDispatcher,
				config : SensorGraphConfig) {

		super($scope, $uibModalInstance, _dispatcher, config);

		$scope.$watch("config.mode", () : void => {
			var mode = $scope.config.mode;
			if($scope.resolutions[mode].indexOf($scope.config.resolution) === -1) {
				$scope.config.resolution = $scope.resolutions[mode][0];
			}

			if(mode === 'realtime') {
				$scope.config.resolution = 'raw';
			}
		});
	}

	protected _checkConfig() {
		return true;
	}

}

interface SensorGraphScope  extends Widget.WidgetScope {
	sensors : SensorSpecifier[];
	sensorColors : DeviceSensorMap<string>;
}

export class SensorGraphController extends Widget.WidtgetController {
	private _store : Store.SensorValueStore;
	protected _config : SensorGraphConfig;
	private _graphNode : HTMLElement;
	private _timeout : ng.IPromise<any>;

	public set graphNode(element: ng.IAugmentedJQuery) {
		this._graphNode = element.find(".sensor-graph").get(0);
	}

	constructor(private $interval : ng.IIntervalService,
				private $timeout: ng.ITimeoutService,
				protected $scope : SensorGraphScope,
				protected $uibModal : angular.ui.bootstrap.IModalService,
				protected _dispatcher : UpdateDispatcher.UpdateDispatcher,
				protected _serverTime : ServerTime) {

		super($scope, _dispatcher, $uibModal);

		this._store = new Store.SensorValueStore();
		this._store.setSlidingWindowMode(true);
		this._store.setEnd(0);

		this._dispatcher.onInitialMetadata(() => {
			//TODO: Add on config callback here
			this._setDefaultConfig();
			this._redrawGraph();
		});

		this._settingsTemplate = 'sensor-graph-settings.html';
		this._settingsControllerFactory = SensorGraphSettingsFactory;

		$interval(() => this._store.clampData(), 60 * 1000);
	}


	public updateValue(deviceID : string, sensorID : string, resolution : string, timestamp : number, value : number) : void {
		this._store.addValue(deviceID, sensorID, timestamp, value);
	}

	private _setDefaultConfig() {
		this._applyConfig({
			unit : this._dispatcher.units[0],
			resolution : SupportedResolutions[0],
			sensors : [],
			mode: 'realtime',
			intervalStart : this._serverTime.now() - 24 * 60 * 1000,
			intervalEnd : this._serverTime.now(),
			windowStart : 5 * 60 * 1000,
			windowEnd : 0
		});
	}

	private _subscribeSensor(config : SensorGraphConfig, deviceID: string, sensorID : string) {
		if(config.mode === 'realtime') {
			this._dispatcher.subscribeRealtimeSlidingWindow(deviceID,
															sensorID,
															config.windowStart,
															this);
		}
		else if(config.mode === 'slidingWindow'){
			this._dispatcher.subscribeSlidingWindow(deviceID,
														sensorID,
														config.resolution,
														config.windowStart,
														config.windowEnd,
														this);
		}
		else if(config.mode === 'interval') {
			this._dispatcher.subscribeInterval(deviceID,
												sensorID,
												config.resolution,
												config.intervalStart,
												config.intervalEnd,
												this);
		}
		else {
			throw new Error("Unknown mode:" + config.mode);
		}
	}


	protected _applyConfig(config : SensorGraphConfig) {
		var differences = Utils.differentProperties(this._config, config);
		// Only sensors or unit changed so no need to redo everything
		if(differences !== undefined && Utils.difference(differences, ["sensors", "unit"]).length === 0) {

			var addedSensors = Utils.difference(config.sensors, this._config.sensors, sensorEqual);
			var	removedSensors = Utils.difference(this._config.sensors, config.sensors, sensorEqual);

			for(var {deviceID: deviceID, sensorID: sensorID} of addedSensors) {
				this._subscribeSensor(config, deviceID, sensorID);
				this._store.addSensor(deviceID,
									sensorID);
			}

			for(var {deviceID: deviceID, sensorID: sensorID} of removedSensors) {
				this._dispatcher.unsubscribeSensor(deviceID, sensorID, config.resolution, this);
				this._store.removeSensor(deviceID, sensorID);
			}

			console.log("Sensor/Unit change");
		} //Redo all the things !
		else {
			this._dispatcher.unsubscribeAll(this);
			this._store = new Store.SensorValueStore();

			if(config.mode === 'realtime') {
				this._store.setSlidingWindowMode(true);
				this._store.setStart(config.windowStart);
				this._store.setEnd(0);
			}
			else if(config.mode === 'slidingWindow') {
				this._store.setSlidingWindowMode(true);
				this._store.setStart(config.windowStart);
				this._store.setEnd(config.windowEnd);
			}
			else if(config.mode === 'interval') {
				this._store.setSlidingWindowMode(false);
				this._store.setStart(config.intervalStart);
				this._store.setEnd(config.intervalEnd);
			}

			for(var {deviceID: deviceID, sensorID: sensorID} of config.sensors) {
				this._subscribeSensor(config, deviceID, sensorID);

				this._store.addSensor(deviceID, sensorID);
			}

			console.log("Redo all");

			this._store.setTimeout(ResoltuionToMillisecs[config.resolution] * 60);
			this._store.setTimeProvider(() => this._serverTime.now());
		}


		this._config = config;
		this.$scope.sensorColors = this._store.getColors();
		this.$scope.sensors = config.sensors;

		this._redrawGraph();
	}

	private _redrawGraph() {
		this.$timeout.cancel(this._timeout);

		var time = this._serverTime.now();

		var graphOptions : any = {
			xaxis: {
				mode: 'time',
				timeMode : 'local',
				title: 'Time',
				noTicks: 15,
				minorTickFreq: 1
			},
			HtmlText: false,
			preventDefault : false,
			title: 'Messwerte',
			shadowSize: 0,
			lines: {
				lineWidth: 2,
			}
		}

		graphOptions.title = 'Values [' + this._config.unit + ']';

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

		delay = delay / graph.plotWidth;
		delay = Math.min(10000, delay);

		this._timeout = this.$timeout(() => this._redrawGraph(), delay);
	}
}

class SensorGraphDirective implements ng.IDirective {
	public require : string = "sensorGraph"
	public restrict : string = "A"
	public templateUrl : string = "/html/sensor-graph.html"
	public scope = {}

	public controller = ["$interval", "$timeout", "$scope", "$uibModal", "UpdateDispatcher", "ServerTime", SensorGraphController];

	// Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
	public link:Function  = ($scope : SensorGraphScope,
								element : ng.IAugmentedJQuery,
								attrs : ng.IAttributes,
								sensorGraph : SensorGraphController) : void => {



		sensorGraph.graphNode = element;
	}
}

export default function SensorGraphFactory() : () => ng.IDirective {
	return () => new SensorGraphDirective();
}
