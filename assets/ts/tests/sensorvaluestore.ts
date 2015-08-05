/// <reference path="qunit/qunit.d.ts" />

/// <reference path="../sensorvaluestore.ts"/>
/// <reference path="../msg2socket.ts"/>

"use strict";

function errorCompare(msg : string) : (error : Error) => boolean {
	return function(error : Error) : boolean {
		return error.message === msg;
	};
}


QUnit.module("Misc setup tests");
QUnit.test("Constructor test", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	var data : Array<Array<any>> = store.getGraphData();

	assert.equal(data.length, 2, "Empty data store should have first and last entries");

	assert.equal(data[0].length, 1, "First should have only one element (timestamp)");
	assert.ok(data[0][0] instanceof Date, "First entry should have a timestamp");

	assert.equal(data[1].length, 1, "Last should have only one element (timestamp)");
	assert.ok(data[1][0] instanceof Date, "Last entry should have a timestamp");
});


QUnit.test("Set intervall", function(assert : QUnitAssert) : void {
	var interval = 5 * 60 * 1000; //5 minuntes

	var store = new Store.SensorValueStore();

	store.setInterval(interval);

	var data : Array<Array<any>> = store.getGraphData();

	assert.ok(data[1][0].getTime() - data[0][0].getTime() == interval, "First an last element should be exactly intervall seconds appart");
});


QUnit.test("Add sensor", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	assert.ok(!store.hasSensor("ADevice", "ASensor"), "Store should not know device and sensor by now");

	store.addSensor("ADevice", "ASensor", "ADummySensor");

	assert.ok(store.hasSensor("ADevice", "ASensor"), "Store should know device and sensor by now");

	var data = store.getGraphData();

	var correctForm = data.every(function(entry : Array<any>) : boolean {
		return entry.length == 2 && entry[0] instanceof Date;
	});

	assert.ok(correctForm, "Each element should have a timestamp a sensor value now");

	assert.ok(isNaN(data[0][1]), "First value for the sensor should be NaN");
	assert.ok(isNaN(data[data.length-1][1]), "Last value for the sensor should be NaN");


	var labels = store.getGraphLabels();

	assert.equal(labels[0], "Time", "First label should be Time");
	assert.equal(labels[1], "ADummySensor", "Second label should be first sensor label");
});


QUnit.test("Duplicate sensor", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor", "ADummySensor");

	assert.throws(function() {
		store.addSensor("ADevice", "ASensor", "ADummySensor");
	},
	errorCompare("Sensor has been added already"),
	"Adding a sensor twice should raise an error");
});

QUnit.test("Set sensor label", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor", "ADummySensor");

	store.setSensorLabel("ADevice", "ASensor", "NotADummySensor");

	assert.equal(store.getGraphLabels()[1], "NotADummySensor", "Label should be updated");

	assert.throws(function() {
		store.setSensorLabel("UnknownDevice", "UnkownSensor", "ADummySensor");
	},
	errorCompare("No such sensor"),
	"Updating the label to an unknown sensor should cause an error");
});

QUnit.test("Getting sensor by index", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor", "ADummySensor");

	assert.deepEqual(store.getSensorByIndex(0), ["ADevice", "ASensor"], "First sensor should be the one we just added");

	assert.throws(function() {
		store.getSensorByIndex(-23);
	},
	errorCompare("Sensor index out of range"),
	"Accessing a sensor index out of range should result in an error");
});

QUnit.test("Getter for labels", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "Sensor1", "First Sensor");
	store.addSensor("ADevice", "Sensor2", "Second Sensor");


	assert.deepEqual(store.getGraphLabels(), ["Time", "First Sensor", "Second Sensor"]);

});




QUnit.module("Tests for adding values");
QUnit.test("Add values without needing to compact", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();
	store.addSensor("ADevice", "Sensor1", "A Dummy Sensor");
	var interval = 5 * 60 * 1000; //5 minuntes
	store.setInterval(interval);

	var update : Msg2Socket.UpdateData = {
		"ADevice" : {
			"Sensor1": []
		}
	};

	var now = (new Date()).getTime();
	var expectedResult = [];
	for(var i = 0; i < 10; i++) {
		var time = now - interval + 10 + i * 30 * 1000;
		update["ADevice"]["Sensor1"].push([time, i]);
		expectedResult.push([new Date(time), i]);
	}

	store.addValues(update);

	var data = store.getGraphData();

	assert.ok(isNaN(data[0][1]), "First value for the sensor should be NaN");
	assert.ok(isNaN(data[data.length-1][1]), "Last value for the sensor should be NaN");

	assert.deepEqual(data.slice(1,-1), expectedResult, "Data without terminators should match excepted result");
});


QUnit.test("Add values and compact", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();
	store.addSensor("ADevice", "Sensor1", "First Sensor");
	store.addSensor("ADevice", "Sensor2", "Second Sensor");
	var interval = 5 * 60 * 1000; //5 minuntes
	store.setInterval(interval);


	var update : Msg2Socket.UpdateData = {
		"ADevice" : {
			"Sensor1": [],
			"Sensor2": []
		}
	};

	var now = (new Date()).getTime();
	var expectedResult = [];
	for(var i = 0; i < 10; i++) {
		var time = now - interval + 10 + i * 30 * 1000;
		update["ADevice"]["Sensor1"].push([time, i]);
		update["ADevice"]["Sensor2"].push([time, i * i]);
		expectedResult.push([new Date(time), i, i * i]);
	}

	store.addValues(update);

	var data = store.getGraphData();

	assert.ok(isNaN(data[0][1]) && isNaN(data[0][2]), "First values for the sensors should be NaN");
	assert.ok(isNaN(data[data.length-1][1]) && isNaN(data[data.length-1][2]), "Last values for the sensors should be NaN");

	assert.deepEqual(data.slice(1,-1), expectedResult, "Data without terminators should match excepted result");
});

QUnit.test("Add values with timeout", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();
	store.addSensor("ADevice", "Sensor1", "First Sensor");
	store.addSensor("ADevice", "Sensor2", "Second Sensor");
	var interval = 5 * 60 * 1000; //5 minuntes
	var timeout = 1 * 60 * 1000; //1 minute
	store.setInterval(interval);
	store.setTimeout(timeout);

	var update : Msg2Socket.UpdateData = {
		"ADevice" : {
			"Sensor1": [],
			"Sensor2": []
		}
	};

	var now = (new Date()).getTime();
	for(var i = 0; i < 10; i++) {
		var time = now - interval + 10 + i * 30 * 1000;
		update["ADevice"]["Sensor1"].push([time, i]);
	}

	update["ADevice"]["Sensor2"].push([now - interval + 10 + 30 * 1000, 23]);
	update["ADevice"]["Sensor2"].push([now - interval + 10 + 150 * 1000, 42]);


	function makeDate(i : number) : Date {
		return new Date(now - interval + 10 + i * 30 * 1000);
	}

	var expectedResult = [
		[makeDate(0), 0, null],
		[makeDate(1), 1, 23],
		[makeDate(2), 2, null],
		[makeDate(3), 3, null],
		[makeDate(4), null, NaN],
		[makeDate(4), 4, null],
		[makeDate(5), 5, 42],
		[makeDate(6), 6, null],
		[makeDate(7), 7, null],
		[makeDate(8), null, NaN],
		[makeDate(8), 8, null],
		[makeDate(9), 9, null],
	];

	store.addValues(update);

	var data = store.getGraphData();

	assert.ok(isNaN(data[0][1]) && isNaN(data[0][2]), "First values for the sensors should be NaN");
	assert.ok(isNaN(data[data.length-1][1]) && isNaN(data[data.length-1][2]), "Last values for the sensors should be NaN");

	assert.deepEqual(data.slice(1,-1), expectedResult, "Data without terminators should match excepted result");
});
