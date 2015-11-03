/// <reference path="es6-shim.d.ts" />

/// <reference path="msg2socket.ts" />

"use strict";

module Store {
	export interface TimeSeries {
		line? : { color : string },
		data : [number, number][]
	}

	const ColorScheme : string[] = ['#00A8F0', '#C0D800', '#CB4B4B', '#4DA74D', '#9440ED'];

	export class SensorValueStore {
		private _series : TimeSeries[];
		private _sensorMap: {[device : string] : {[sensor : string] : number}};
		private _sensorLabels: {[device : string] : {[sensor : string] : string}};

		private _interval : number;
		private _timeout : number;

		private _colorIndex : number;

		constructor() {
			this._series = [];
			this._sensorMap = {};
			this._sensorLabels = {};

			this._timeout = 2.5 * 60 * 1000;
			this._interval = 5 * 60 * 1000;

			this._colorIndex = 0;
		};

		private _pickColor() : string {
			var color : string = ColorScheme[this._colorIndex];
			this._colorIndex = (this._colorIndex + 1) % ColorScheme.length;
			return color;
		}


		private _getSensorIndex(deviceId : string, sensorId : string) : number {
			if(this._sensorMap[deviceId] !== undefined && this._sensorMap[deviceId][sensorId] !== undefined) {
				return this._sensorMap[deviceId][sensorId];
			}

			return -1;
		}

		public setInterval(interval : number) : void {
			this._interval = interval;
		}

		public setTimeout(timeout : number) : void {
			this._timeout = timeout;
		}

		public clampData() : void {
			var oldest : number = (new Date()).getTime() - this._interval;

			this._series.forEach((series : TimeSeries) : void => {
				series.data = series.data.filter((point : [number, number]) : boolean => {
					return point[0] >= oldest;
				});

				if(series.data.length > 0) {
					if(series.data[0][1] === null) {
						series.data.splice(0,1);
					}
					if(series.data[series.data.length - 1][1] === null) {
						series.data.splice(series.data.length - 1,1);
					}
				}
			});
		}

		public addSensor(deviceId : string, sensorId : string, label : string) : void {
			if(this.hasSensor(deviceId, sensorId)) {
				throw new Error("Sensor has been added already");
			}

			var index : number = this._series.length;

			if(this._sensorMap[deviceId] === undefined) {
				this._sensorMap[deviceId] = {};
				this._sensorLabels[deviceId] = {};
			}

			this._sensorMap[deviceId][sensorId] = index;
			this._sensorLabels[deviceId][sensorId] = label;

			this._series.push({
				line: {
					color : this._pickColor(),
				},
				data: []
			})
		}

		public hasSensor(device : string, sensor : string) : boolean {
			return this._getSensorIndex(device, sensor) !== -1;
		}

		public removeSensor(deviceId : string, sensorId : string) {
			var index : number = this._getSensorIndex(deviceId, sensorId);

			if(index === -1) {
				throw new Error("No such sensor");
			}

			this._series.splice(index,1);
			delete this._sensorMap[deviceId][sensorId];
			delete this._sensorLabels[deviceId][sensorId];
		}

		public setLabel(deviceId : string, sensorId : string, label : string) {

			if(!this.hasSensor(deviceId, sensorId)) {
				throw new Error("No such sensor");
			}

			this._sensorLabels[deviceId][sensorId] = label;
		}

		public addValue(deviceId : string, sensorId : string, timestamp : number, value : number) : void {
			var seriesIndex : number = this._getSensorIndex(deviceId, sensorId);
			if(seriesIndex === -1) {
				throw new Error("No such sensor");
			}

			// Find position for inserting
			var data = this._series[seriesIndex].data;
			var pos = data.findIndex((point : [number, number]) : boolean => {
				return point[0] > timestamp;
			});
			if(pos === -1) {
				pos = data.length;
			}

			// Insert
			data.splice(pos, 0, [timestamp, value]);

			//Check if we need to remove a timeout in the past
			if(pos > 0 && data[pos - 1][1] === null && timestamp - data[pos - 1][0] < this._timeout) {
				data.splice(pos - 1, 1);
			}

			//Check if we need to remove a timeout in the future
			if(pos < data.length - 1 && data[pos + 1][1] === null && data[pos + 1][0] - timestamp < this._timeout) {
				data.splice(pos + 1, 1);
			}

			//Check if a null in the past is needed
			if(pos > 0 && data[pos - 1][1] !== null && timestamp - data[pos - 1][0] >= this._timeout) {
				data.splice(pos, 0, [timestamp - 1, null]);
				//console.log(JSON.stringify(data));
			}

			//Check if a null in the future is needed
			if(pos < data.length - 1 && data[pos + 1][1] !== null && data[pos + 1][0] - timestamp >= this._timeout) {
				data.splice(pos + 1, 0, [timestamp + 1, null]);
			}

		}


		public getData() : TimeSeries[] {
			return this._series;
		}


		public getColors() :  {[device: string]: {[sensor: string]: string}} {
			var colors : {[device: string]: {[sensor: string]: string}} = {};

			for(var deviceId in this._sensorMap) {
				colors[deviceId] = {};
				for(var sensorId in this._sensorMap[deviceId]) {
					var index = this._sensorMap[deviceId][sensorId];
					colors[deviceId][sensorId] = this._series[index].line.color;
				}
			}

			return colors;
		}


		public getLabels() : {[device: string]: {[sensor: string]: string}} {
			var labels : {[device: string]: {[sensor: string]: string}} = {};

			for(var deviceId in this._sensorLabels) {
				labels[deviceId] = {};
				for(var sensorId in this._sensorLabels[deviceId]) {
					labels[deviceId][sensorId] = this._sensorLabels[deviceId][sensorId];
				}
			}

			return labels;
		}
	}

}
