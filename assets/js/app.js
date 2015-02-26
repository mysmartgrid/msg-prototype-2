"use strict";

var msgp = function() {
	var result = {};

	var SensorGraph = function(container, options) {
		var maxAgeMs = options.maxAgeMs;
		var graphData = [[new Date(), null]];
		var g = new Dygraph(container.get(0), graphData, {
			labels: ["Time", "Value"]
		});

		var removeOldValues = function() {
			if (options.maxAgeMs === undefined)
				return;

			var now = new Date();
			while (graphData.length > 0 && (now - graphData[0][0]) > options.maxAgeMs) {
				graphData.shift();
			}
		};

		var updateGraph = function(data) {
			var sort = false;
			data.sort(function(a, b) {
				return a[0] - b[0];
			});
			graphData.pop();
			for (var i = 0; i < data.length; i++) {
				var x = new Date(data[i][0]);
				if (graphData.length > 0) {
					sort |= graphData[graphData.length - 1][0] - x > 0;
				}
				if (options.assumeValuesMissingAfterMs !== undefined && (i != 0 || graphData.length != 0)) {
					var lastTime = i == 0 ? graphData[graphData.length - 1][0] : data[i - 1][0];
					if (data[i][0] - lastTime >= options.assumeValuesMissingAfterMs) {
						graphData.push([new Date(x - 1), null]);
					}
				}
				graphData.push([x, data[i][1]]);
			}
			graphData.push([new Date(), null]);
			if (sort) {
				graphData.sort(function(a, b) {
					return a[0] - b[0];
				});
			}
			removeOldValues();
			if (options.maxAgeMs !== undefined) {
				var now = new Date();
				if (now - graphData[0][0] < options.maxAgeMs)
					graphData.unshift([new Date(now - options.maxAgeMs), null]);
			}
			g.updateOptions({
				file: graphData
			});
		};

		var valueMissingTimeoutId;
		var valueMissingTimeoutFn = function() {
			updateGraph([[new Date(), null]]);

			if (options.assumeValuesMissingAfterMs !== undefined)
				valueMissingTimeoutId = window.setTimeout(valueMissingTimeoutFn, options.assumeValuesMissingAfterMs);
		};

		this.update = function(data) {
			if (valueMissingTimeoutId !== undefined)
				window.clearTimeout(valueMissingTimeoutId);

			updateGraph(data);

			if (options.assumeValuesMissingAfterMs !== undefined)
				valueMissingTimeoutId = window.setTimeout(valueMissingTimeoutFn, options.assumeValuesMissingAfterMs);
		};
	};

	var DeviceGraphs = function(container, options) {
		var sensors = {};
		var sensorGraphTemplate = options.sensorGraphTemplate ? $(options.sensorGraphTemplate) : $("<div></div>");

		this.update = function(data) {
			for (var sensor in data) {
				if (!data.hasOwnProperty(sensor))
					continue;

				if (!(sensor in sensors)) {
					var sensorDiv = sensorGraphTemplate
						.clone()
						.removeAttr("visibility")
						.prop("id", "");
					container.append(sensorDiv);
					sensors[sensor] = new SensorGraph(sensorDiv, options);
				}
				sensors[sensor].update(data[sensor]);
			}
		};
	};

	var GraphCollection = result.GraphCollection = function(containerId, options) {
		options = options || {};

		var container = $(containerId);
		var devices = {};

		this.update = function(data) {
			for (var dev in data) {
				if (!data.hasOwnProperty(dev))
					continue;

				if (!(dev in devices)) {
					var devDiv = $("<div></div>");
					container.append(devDiv);
					devices[dev] = new DeviceGraphs(devDiv, options);
				}
				devices[dev].update(data[dev]);
			}
		};
	};

	var WebsocketClient = result.WebsocketClient = function(url) {
		if (!window["WebSocket"])
			throw "websocket support required";

		var ws = new WebSocket(url, ["msg/1/user"]);
		var _this = this;

		this.close = function() {
			if (ws) {
				ws.close();
				ws = null;
			}
		};

		var eventHandler = function(fnName) {
			return function(e) {
				if (!_this[fnName])
					return;

				_this[fnName](e);
			};
		};

		var _onOpen = eventHandler("onOpen");
		var _onClose = eventHandler("onClose");
		var _onUpdate = eventHandler("onUpdate");
		var _onError = eventHandler("onError");

		ws.onopen = function(e) {
			if (ws.protocol == "msg/1/user") {
				_onOpen({error: "protocol negotiation failed"});
			} else {
				ws.close();
				_onOpen(null);
			}
		};

		ws.onclose = _onClose;
		ws.onerror = _onError;

		ws.onmessage = function(msg) {
			var data = JSON.parse(msg.data);

			switch (data.cmd) {
			case "update":
				_onUpdate(data.args);
				break;
			}
		};
	};

	return result;
}();
