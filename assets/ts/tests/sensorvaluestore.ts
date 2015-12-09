/// <reference path="qunit/qunit.d.ts" />

/// <reference path="../sensorvaluestore.ts"/>
/// <reference path="../msg2socket.ts"/>
/// <reference path="../common.ts"/>

"use strict";

function errorCompare(msg : string) : (error : Error) => boolean {
	return function(error : Error) : boolean {
		return error.message === msg;
	};
}

function rand(min : number, max : number) : number {
	return Math.floor(min + Math.random() * (max -min));
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

	var timestamp = Common.now();
	store.addValue("ADevice", "ASensor1", timestamp, 42);

	var data = store.getData();
	assert.ok(data[0].data.length === 1, "There should be one value in the data array");
	assert.deepEqual(data[0].data, [[timestamp, 42]], "The value and timestamp should match");
});


QUnit.test("Add two values with different timestamps", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");

	var timestamp = Common.now();
	store.addValue("ADevice", "ASensor1", timestamp, 42);
	store.addValue("ADevice", "ASensor1", timestamp - 1000, 84);

	var data = store.getData();
	assert.ok(data[0].data.length === 2, "There be two values in the data array");
	assert.deepEqual(data[0].data, [ [timestamp - 1000, 84], [timestamp, 42]], "The values and timestamps should match");
});

QUnit.test("Add values with same imestamps", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");

	var timestamp = Common.now();

	store.addValue("ADevice", "ASensor1", timestamp - 3000, 23);
	store.addValue("ADevice", "ASensor1", timestamp - 1000, 666);
	store.addValue("ADevice", "ASensor1", timestamp - 2000, 1337);
	store.addValue("ADevice", "ASensor1", timestamp, 42);

	//Update first tuple
	store.addValue("ADevice", "ASensor1", timestamp - 3000, 46);

	//Update tuple in between
	store.addValue("ADevice", "ASensor1", timestamp - 1000, 0);

	//Update last tuple
	store.addValue("ADevice", "ASensor1", timestamp, 84);


	var data = store.getData();
	assert.ok(data[0].data.length === 4, "There should be four values in the data array");
	assert.deepEqual(data[0].data, [[timestamp - 3000, 46],
									[timestamp - 2000, 1337],
									[timestamp - 1000, 0],
									[timestamp, 84]],
					"The values and timestamps should match");
});

QUnit.test("Clamp data - slinding window", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");

	var timestamp = Common.now();
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

QUnit.test("Clamp data - fixed interval", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");

	store.setStart(630 * 1000);
	store.setEnd(840 * 1000);
	store.setSlidingWindowMode(false);

	var timestamp = 840 * 1000;
	var oldTimestamp = 420 * 1000;
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

QUnit.test("Test timeout past", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");

	var timestamp = Common.now();
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


QUnit.test("Test timeout future", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");

	var timestamp = Common.now();
	var oldTimestamp = timestamp - 3 * 60 * 1000;
	var middleTimestamp = timestamp - 1 * 60 * 1000;
	store.addValue("ADevice", "ASensor1", timestamp, 42);
	store.addValue("ADevice", "ASensor1", oldTimestamp , 23);


	var data = store.getData();
	assert.deepEqual(data[0].data,
						[[oldTimestamp, 23], [oldTimestamp + 1, null], [timestamp, 42]],
						"Both values should be in the array");

	store.addValue("ADevice", "ASensor1", middleTimestamp, 39);

	data = store.getData();
	assert.deepEqual(data[0].data,
						[[oldTimestamp, 23], [middleTimestamp, 39], [timestamp, 42]],
						"Both values should be in the array");
});

QUnit.test("Remove past timeout, reinsert in future", function(assert : QUnitAssert) : void {
	var store = new Store.SensorValueStore();

	const Timeout = 10 * 1000;

	store.setTimeout(Timeout);

	store.addSensor("ADevice", "ASensor1", "ADummySensor1");


	var lastTimestamp = Common.now();
	var firstTimestamp = Common.now() - 60 * 1000;
	var middleTimestamp = firstTimestamp + 9 * 1000;

	store.addValue("ADevice", "ASensor1", lastTimestamp, 0);
	store.addValue("ADevice", "ASensor1", firstTimestamp, 1);

	var result1 = [[firstTimestamp, 1],
					[firstTimestamp + 1, null],
					[lastTimestamp, 0]];

	assert.deepEqual(store.getData()[0].data, result1, "There should be a timeout.");

	store.addValue("ADevice", "ASensor1", middleTimestamp, 2);

	var result2 = [[firstTimestamp, 1],
					[middleTimestamp, 2],
					[middleTimestamp + 1, null],
					[lastTimestamp, 0]];

	assert.deepEqual(store.getData()[0].data, result2, "Timeout should have shifted.");
});
