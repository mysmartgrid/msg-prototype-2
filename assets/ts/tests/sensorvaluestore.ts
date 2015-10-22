/// <reference path="qunit/qunit.d.ts" />

/// <reference path="../sensorvaluestore.ts"/>
/// <reference path="../msg2socket.ts"/>

"use strict";

function errorCompare(msg : string) : (error : Error) => boolean {
	return function(error : Error) : boolean {
		return error.message === msg;
	};
}

function now() : number {
	return (new Date()).getTime();
}


QUnit.module("Misc setup tests");
QUnit.test("Constructor test", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	var data = store.getData();

	assert.ok(data.length === 0, "The new store should be empty");
});



QUnit.module("Sensor management tests");
QUnit.test("Add sensor", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();
	assert.ok(!store.hasSensor("ADevice", "ASensor"), "Store should not know device and sensor by now");

	store.addSensor("ADevice", "ASensor", "ADummySensor");
	assert.ok(store.hasSensor("ADevice", "ASensor"), "Store should know device and sensor by now");

	var data = store.getData();
	var labels = store.getLabels();
	assert.ok(data.length === 1, "There should be one timeseries");
	assert.ok(data[0].line !== undefined, "The series should have a line property");
	assert.ok(data[0].line.color !== undefined, "The series should have a line.color property");
	assert.ok(data[0].data !== undefined, "The series should have a data array");
	assert.ok(data[0].data.length === 0, "The data array should be empty");
	assert.ok(store.getLabels()["ADevice"]["ASensor"] === "ADummySensor", "The sensor should have a label");
})


QUnit.test("Duplicate sensor", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();
	store.addSensor("ADevice", "ASensor", "ADummySensor");

	assert.throws(() : void => {
			store.addSensor("ADevice", "ASensor", "ADummySensor");
		},
		errorCompare("Sensor has been added already"),
		"Adding a sensor twice should raise an error");
});


QUnit.test("Remove sensor", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");
	store.addSensor("ADevice", "ASensor2", "ADummySensor2");
	store.addSensor("ADevice", "ASensor3", "ADummySensor3");

	store.removeSensor("ADevice", "ASensor2");

	assert.ok(!store.hasSensor("ADevice", "ASensor2"), "The store should not have the sensor that has been removed");

	var data = store.getData();
	assert.ok(data.length === 2, "There should still be 2 timeseries left.");

	var expectedLabels = {
		"ADevice": {
			"ASensor1": "ADummySensor1",
			"ASensor3": "ADummySensor3",
		}
	}

	var labels = store.getLabels();
	assert.deepEqual(labels, expectedLabels, "There should be two lables left");
});


QUnit.test("Remove nonexistent sensor", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	assert.throws(() : void => {
			store.removeSensor("ADevice", "ASensor");
		},
		errorCompare("No such sensor"),
		"Removing a sensor that does not exist should raise an error");
});

QUnit.test("Change a sensor label", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();
	store.addSensor("ADevice", "ASensor", "ADummySensor");

	store.setLabel("ADevice", "ASensor", "AnOtherDummySensor");

	var expectedLabels = {
		"ADevice": {
			"ASensor": "AnOtherDummySensor",
		}
	};

	var labels = store.getLabels();
	assert.deepEqual(labels, expectedLabels, "Label should be changed");
});


QUnit.module("Value management tests");
QUnit.test("Clamp empty store", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.clampData();

	assert.ok(true, "Nothing should go wrong here");
});


QUnit.test("Add single value", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");

	var timestamp = now();
	store.addValue("ADevice", "ASensor1", timestamp, 42);

	var data = store.getData();
	assert.ok(data[0].data.length === 1, "There should be one value in the data array");
	assert.deepEqual(data[0].data, [[timestamp, 42]], "The value and timestamp should match");
});

QUnit.test("Clamp data", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");

	var timestamp = now();
	var oldTimestamp = timestamp - 6 * 60 * 1000;
	store.addValue("ADevice", "ASensor1", oldTimestamp , 23);
	store.addValue("ADevice", "ASensor1", timestamp, 42);

	var data = store.getData();
	assert.deepEqual(data[0].data,
						[[oldTimestamp, 23], [timestamp - 1, null], [timestamp, 42]],
						"Both values should be in the array");

	store.clampData();

	assert.deepEqual(data[0].data,
						[[timestamp, 42]],
						"The older value should no longer be in the array");

});

QUnit.test("Test timeout", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");

	var timestamp = now();
	var oldTimestamp = timestamp - 3 * 60 * 1000;
	var middleTimestamp = timestamp - 1 * 60 * 1000;
	store.addValue("ADevice", "ASensor1", oldTimestamp , 23);
	store.addValue("ADevice", "ASensor1", timestamp, 42);

	var data = store.getData();
	assert.deepEqual(data[0].data,
						[[oldTimestamp, 23], [timestamp - 1, null], [timestamp, 42]],
						"Both values should be in the array");

	store.addValue("ADevice", "ASensor1", middleTimestamp, 39);

	data = store.getData();
	assert.deepEqual(data[0].data,
						[[oldTimestamp, 23], [middleTimestamp, 39], [timestamp, 42]],
						"Both values should be in the array");
});

