/// <reference path="es6-shim.d.ts" />

/// <reference path="msg2socket.ts" />

"use strict";

module Store {
	function tsToDate(ts : number) : Date {
		return new Date(ts);
	}

	function dateToTs(date : Date) : number {
		return date.getTime();
	}

	interface Sensor {
		deviceId : string;
		sensorId : string;
		label : string;
	}

	export class SensorValueStore {
		private _data : any[][];
		private _sensors : Sensor[];
		private _interval : number;
		private _timeout : number;

		private _firstEntry() : any[] {
			return this._data[0];
		}

		private _lastEntry() : any[] {
			return this._data[this._data.length - 1];
		}


		constructor() {
			this._data = [];
			this._sensors = [];

			this._data.push([new Date()]);
			this._data.push([new Date()]);

			this._timeout = 2.5 * 60 * 1000;
			this._interval = 5 * 60 * 1000;
		};

		public setInterval(interval : number) : void {
			this._interval = interval;
			this._clampData();
		}

		public setTimeout(timeout : number) : void {
			this._timeout = timeout;
			this._compactData();
		}

		private _clampData() : void {
			var now = new Date();
			var oldest = tsToDate(dateToTs(now) - this._interval);

			this._firstEntry()[0] = oldest;
			this._lastEntry()[0] = now;


			this._data = this._data.filter(function(entry: any[]) : boolean {
				return entry[0] >= oldest && entry[0] <= now;
			});
		}

		public addSensor(deviceId : string, sensorId : string, label : string) : void {
			if(this.hasSensor(deviceId, sensorId)) {
				throw new Error("Sensor has been added already");
			}

			var sensor : Sensor = {
				deviceId : deviceId,
				sensorId : sensorId,
				label : label
			};

			this._sensors.push(sensor);

			this._firstEntry().push(NaN);
			this._lastEntry().push(NaN);
			for(var i = 1; i < this._data.length - 1; i++) {
					this._data[i].push(null);
			}
		}

		public setSensorLabel(deviceId : string, sensorId : string, label : string) : void {
			var index = this._getSensorIndex(deviceId, sensorId);

			if(index === -1) {
				throw new Error("No such sensor");
			}

			this._sensors[index - 1].label = label;
		}

		public getSensorByIndex(index : number) : [string, string] {
			if(index < 0 || index >= this._sensors.length) {
				throw new Error("Sensor index out of range");
			}

			return [this._sensors[index].deviceId, this._sensors[index].sensorId];
		}

		private _getSensorIndex(deviceId : string, sensorId : string) : number {
			var index = this._sensors.findIndex(function(sensor : Sensor) : boolean {
				return sensor.deviceId === deviceId && sensor.sensorId === sensorId;
			});

			if(index >= 0) {
				return index + 1;
			}

			return -1;
		}

		public hasSensor(device : string, sensor : string) : boolean {
			return this._getSensorIndex(device, sensor) !== -1
		}

		private _makeEntry(device : string, sensor : string, timestamp : number, value : number) : any[] {
			var sensorIndex = this._getSensorIndex(device, sensor);

			if(sensorIndex === -1) {
				throw new Error("Sensor " + device + "." + sensor + "does not exist");
			}

			var entry = new Array<any>(this._sensors.length + 1);

			entry[0] = tsToDate(timestamp);
			entry.fill(null,1);
			entry[sensorIndex] = value;

			return entry;
		}

		public addValues(update : Msg2Socket.UpdateData) : void {
			this._clampData();

			var oldestTs = dateToTs(this._firstEntry()[0]);
			var newestTs = dateToTs(this._lastEntry()[0]);

			for(var deviceID in update) {
				for(var sensorID in update[deviceID]) {
					if(this.hasSensor(deviceID, sensorID)) {
						var sensorIndex = this._getSensorIndex(deviceID, sensorID);
						for(var i = 0; i < update[deviceID][sensorID].length; i++) {
							var tuple : [number, number] = update[deviceID][sensorID][i];
							//We do not accept data beyond our terminators
							if(tuple[0] > oldestTs && tuple[0] < newestTs) {
								this._data.push(this._makeEntry(deviceID, sensorID, tuple[0], tuple[1]));
							}
						}
					}
				}
			}

			this._data.sort(function(a : any[], b : any[]) : number {
				if(a[0] < b[0]) {
					return -1;
				}
				else if(a[0] > b[0]) {
					return 1;
				}
				return 0;
			});

			this._compactData();
		}

		private _makeNaNEntry(timestamp: number, index : number) : any[] {
			var nanEntry = new Array(this._sensors.length + 1);
			nanEntry[0] = tsToDate(timestamp);
			nanEntry.fill(null,1);
			nanEntry[index] = NaN;
			return nanEntry;
		}

		private _compactData() : void {
			//First pass: Drop all old NaN entries and merge entries with same timestamp
			var entryIndex = 1;
			while(entryIndex < this._data.length - 1) {
				//Drop NaNs
				for(var valueIndex = 1; valueIndex <= this._sensors.length; valueIndex++) {
					if(isNaN(this._data[entryIndex][valueIndex])) {
						this._data[entryIndex][valueIndex] = null;
					}
				}


				//Remove all entries only containing null values
				var allNull = this._data[entryIndex].every((value : any, index : number) => {
					return index < 1 || value === null;
				});
				if(allNull) {
					this._data.splice(entryIndex,1);
				}


				//Merge entries with same timestamp
				if(entryIndex > 1 && dateToTs(this._data[entryIndex][0]) === dateToTs(this._data[entryIndex-1][0])) {
					var entry = this._data[entryIndex];
					var prevEntry = this._data[entryIndex-1];

					var mergeAble = entry.every((value : any, index : number) => {
						// index > 1: we don't care about the dates
						// value === null: we don't care about nulls in the current set
						// value !== null && prevEntry[index] === null: there is null in the previous entry, for the not-null value
						return index < 1 || value === null || value !== null && prevEntry[index] === null;
					});

					if(mergeAble) {
						for(var valueIndex = 1; valueIndex <= this._sensors.length; valueIndex++) {
							if(prevEntry[valueIndex] === null && entry[valueIndex] !== null) {
								prevEntry[valueIndex] = entry[valueIndex];
							}
						}
						this._data.splice(entryIndex, 1);
						entryIndex--;
					}
				}

				entryIndex++;
			}


			//Second pass: Reinsert NaNs if gap is big enough
			var timedout = new Array<boolean>(this._sensors.length);
			timedout.fill(false, 0);
			var lastUpdate = new Array<number>(this._sensors.length);
			lastUpdate.fill(dateToTs(this._firstEntry()[0]), 0);
			entryIndex = 1;
			while(entryIndex < this._data.length - 1) {
				var entry = this._data[entryIndex];
				var timestamp = dateToTs(entry[0]);
				for(var valueIndex = 1; valueIndex <= this._sensors.length; valueIndex++) {
					if(timestamp - lastUpdate[valueIndex - 1] > this._timeout && !timedout[valueIndex - 1]) {
						this._data.splice(entryIndex, 0, this._makeNaNEntry(timestamp, valueIndex));
						entryIndex++;

						timedout[valueIndex - 1] = true;
					}

					if(entry[valueIndex] !== null) {
						lastUpdate[valueIndex - 1] = timestamp;
						timedout[valueIndex - 1] = false;
					}
				}
				entryIndex++;
			}

		}

		public getGraphData() : any[][] {
			return this._data;
		}


		public getGraphLabels() : string[] {
			var labels : string[] = ["Time"];

			for(var index = 0; index < this._sensors.length; index++) {
				labels.push(this._sensors[index].label);
			}

			return labels;
		}
	}

}
