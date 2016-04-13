import * as Common from './common';
import * as Utils from './utils';

export interface TimeSeries {
	lines? : { color : string },
	data : [number, number][]
}

const ColorScheme : string[] = ['#00A8F0', '#C0D800', '#CB4B4B', '#4DA74D', '#9440ED'];

export class SensorValueStore {
	private _series : TimeSeries[];
	private _sensorMap: {[device : string] : {[sensor : string] : number}};

	private _start : number;
	private _end : number;
	private _slidingWindow : boolean;

	private _timeout : number;


	private _colorIndex : number;

	private _now : {() : number};

	constructor() {
		this._series = [];
		this._sensorMap = {};

		this._now = Utils.now;

		this._timeout = 2.5 * 60 * 1000;
		this._start = 5 * 60 * 1000;
		this._end = 0;
		this._slidingWindow = true;

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

	public setTimeProvider(now : {() : number}) {
		this._now = now;
	}

	public setStart(start : number) : void {
		this._start = start;
	}

	public setEnd(end : number) : void {
		this._end = end;
	}

	public setSlidingWindowMode(mode : boolean) : void {
		this._slidingWindow = mode;
	}

	public setTimeout(timeout : number) : void {
		this._timeout = timeout;
	}

	public clampData() : void {
		var oldest = this._start;
		var newest = this._end;

		if(this._slidingWindow) {
			oldest = this._now() - this._start;
			newest = this._now() - this._end;
		}

		this._series.forEach((series : TimeSeries) : void => {
			series.data = series.data.filter((point : [number, number]) : boolean => {
				return point[0] >= oldest && point[0] <= newest;
			});

			//Series should not start or end with null after clamping
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

	public addSensor(deviceId : string, sensorId : string) : void {
		if(this.hasSensor(deviceId, sensorId)) {
			throw new Error("Sensor has been added already");
		}

		var index : number = this._series.length;

		if(this._sensorMap[deviceId] === undefined) {
			this._sensorMap[deviceId] = {};
		}

		this._sensorMap[deviceId][sensorId] = index;

		this._series.push({
			lines: {
				color : this._pickColor(),
			},
			data: []
		})
	}

	public hasSensor(device : string, sensor : string) : boolean {
		return this._getSensorIndex(device, sensor) !== -1;
	}

	public removeSensor(deviceId : string, sensorId : string) {
		var index = this._getSensorIndex(deviceId, sensorId);

		if(index === -1) {
			throw new Error("No such sensor");
		}

		this._series.splice(index,1);
		delete this._sensorMap[deviceId][sensorId];

		// Update remaining indices
		for(deviceId in this._sensorMap) {
			for(sensorId in this._sensorMap[deviceId]) {
				if(this._sensorMap[deviceId][sensorId] > index) {
					this._sensorMap[deviceId][sensorId] -= 1;
				}
			}
		}
	}


	private _findInsertionPos(data : [number, number][], timestamp: number) : number {
		for(var pos = 0; pos < data.length; pos++) {
			if(data[pos][0] > timestamp) {
				return pos;
			}
		}

		return data.length;
	}


	public addValue(deviceId : string, sensorId : string, timestamp : number, value : number) : void {
		var seriesIndex : number = this._getSensorIndex(deviceId, sensorId);
		if(seriesIndex === -1) {
			throw new Error("No such sensor");
		}

		// Find position for inserting
		var data = this._series[seriesIndex].data;
		var pos = this._findInsertionPos(data, timestamp);

		// Check if the value is an update for an existing timestamp
		if(data.length > 0 && pos === 0 && data[0][0] === timestamp) {
			// Update for the first tuple
			data[0][1] = value;
		}
		else if(data.length > 0 && pos > 0 && pos <= data.length && data[pos - 1][0] === timestamp) {
			//Update any other tuple including the last one
			data[pos - 1][1] = value;
		}
		else {
			// Insert
			data.splice(pos, 0, [timestamp, value]);


			//Check if we need to remove a null in the past
			if(pos > 0 && data[pos - 1][1] === null && timestamp - data[pos - 1][0] < this._timeout) {
				data.splice(pos - 1, 1);
				// We delete something bevor pos, so we should move pos
				pos -= 1;
			}

			//Check if we need to remove a null in the future
			if(pos < data.length - 1 && data[pos + 1][1] === null && data[pos + 1][0] - timestamp < this._timeout) {
				data.splice(pos + 1, 1);
			}

			//Check if a null in the past is needed
			if(pos > 0 && data[pos - 1][1] !== null && timestamp - data[pos - 1][0] >= this._timeout) {
				data.splice(pos, 0, [timestamp - 1, null]);
			}

			//Check if a null in the future is needed
			if(pos < data.length - 1 && data[pos + 1][1] !== null && data[pos + 1][0] - timestamp >= this._timeout) {
				data.splice(pos + 1, 0, [timestamp + 1, null]);
			}
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
				colors[deviceId][sensorId] = this._series[index].lines.color;
			}
		}

		return colors;
	}

}
