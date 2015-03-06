"use strict";

angular.module("msgp", [])
.factory("WSUserClient", ["$rootScope", function($rootScope) {
	if (!window["WebSocket"])
		throw "websocket support required";

	var socket = {};
	var socketData = {
		ws: null
	};

	var eventHandler = function(fnName) {
		return function(e) {
			var fn = function() {
				if (!socket[fnName])
					return;

				socket[fnName](e);
			};

			if ($rootScope.$$phase == "apply") {
				fn();
			} else {
				$rootScope.$apply(fn);
			}
		};
	};

	var _onOpen = eventHandler("onOpen");
	var _onClose = eventHandler("onClose");
	var _onUpdate = eventHandler("onUpdate");
	var _onMetadata = eventHandler("onMetadata");
	var _onError = eventHandler("onError");

	var onmessage = function(msg) {
		var data = JSON.parse(msg.data);

		switch (data.cmd) {
		case "update":
			_onUpdate(data.args);
			break;

		case "metadata":
			_onMetadata(data.args);
			break;

		default:
			console.log("bad packet from server", data);
			socket.close();
			break;
		}
	};

	socket.connect = function(url) {
		var ws = socketData.ws = new WebSocket(url, ["msg/1/user"]);

		ws.onerror = _onError;
		ws.onclose = _onClose;

		ws.onopen = function(e) {
			if (ws.protocol != "msg/1/user") {
				_onOpen({error: "protocol negotiation failed"});
				ws.close();
				socketData.ws = null;
				return;
			}

			ws.onmessage = onmessage;
			_onOpen(null);
		};
	};

	socket.close = function() {
		if (socketData.ws) {
			socketData.ws.close();
			socketData.ws = null;
		}
	};

	socket.requestValues = function(since, withMetadata) {
		var cmd = {
			"cmd": "getValues",
			"args": {
				"since": +since,
				"withMetadata": !!withMetadata
			}
		};
		socketData.ws.send(JSON.stringify(cmd));
	};

	return socket;
}])
.controller("GraphPage", ["$scope", "$interval", "WSUserClient", "wsurl", function($scope, $interval, wsclient, wsurl) {
	$scope.devices = {};

	var getDevice = function(id) {
		if (!(id in $scope.devices)) {
			var dev = $scope.devices[id] = {
				sensors: {}
			};

			dev.getSensor = function(id) {
				if (!(id in dev.sensors)) {
					var sens = dev.sensors[id] = {};

					sens.update = function(data) {
						if (sens.graph !== undefined) {
							sens.graph.update(data);
						}
					};
				}
				return dev.sensors[id];
			};

			dev.removeSensor = function(id) {
				if (!(id in dev.sensors))
					throw "no sensor " + id;

				delete dev.sensors[id];
			};
		}
		return $scope.devices[id];
	};

	wsclient.onMetadata = function(md) {
		Object.getOwnPropertyNames(md.devices).forEach(function(did) {
			var dev = md.devices[did];
			var mdev = getDevice(did);

			if ("name" in dev)
				mdev.name = dev.name;

			Object.getOwnPropertyNames(dev.deletedSensors || {}).forEach(function(sid) {
				mdev.removeSensor(sid);
			});

			Object.getOwnPropertyNames(dev.sensors || {}).forEach(function(sid) {
				var sens = dev.sensors[sid];
				var msens = mdev.getSensor(sid);

				msens.name = sens;
			});
		});
	};
	wsclient.onUpdate = function(data) {
		Object.getOwnPropertyNames(data).forEach(function(did) {
			Object.getOwnPropertyNames(data[did]).forEach(function(sid) {
				getDevice(did).getSensor(sid).update(data[did][sid]);
			});
		});
	};
	wsclient.onOpen = function(err) {
		if (err)
			return;

		wsclient.requestValues(new Date() - 120 * 1000, true);
	};
	wsclient.onClose = wsclient.onError = function(e) {
		if (e.wasClean)
			return;

		$scope.wsConnectionFailed = true;
	};

	wsclient.connect(wsurl);
}])
.directive("sensorGraph", ["$interval", function($interval) {
	return {
		scope: {
			title: "=",
			maxAgeMs: "=",
			assumeMissingAfterMs: "=",
			api: "=?"
		},
		restrict: "A",
		template:
			'<h3>{{title}}</h3>' +
			'<div style="width: 100%" class="sensor-graph"></div>',
		link: function(scope, element, attrs) {
			var maxAgeMs = undefined;
			var assumeMissingAfterMs = undefined;
			var graphData = [[new Date(), null]];
			var valueMissingInterval = undefined;

			var g = new Dygraph(element.children("div.sensor-graph").get(0), graphData, {
				labels: ["Time", "Value"],
				connectSeparatedPoints: true
			});

			var api = scope.api = {};

			var clampToMaxAge = function() {
				if (maxAgeMs === undefined)
					return;

				var now = new Date();
				while (graphData.length > 0 && (now - graphData[0][0]) > maxAgeMs) {
					graphData.shift();
				}

				graphData.unshift([new Date(new Date() - maxAgeMs), NaN]);
			};

			var mergeGraphData = function(data) {
				var needsSorting = false;

				for (var i = 0; i < data.length; i++) {
					var at = new Date(Math.floor(data[i][0]));

					if (graphData.length > 0) {
						needsSorting |= graphData[graphData.length - 1][0] - at > 0;
					}

					if (!needsSorting &&
							assumeMissingAfterMs !== undefined &&
							graphData.length > 0 &&
							at - graphData[graphData.length - 1][0] >= assumeMissingAfterMs) {
						graphData.push([new Date(at - assumeMissingAfterMs / 2), NaN]);
					}
					graphData.push([at, data[i][1]]);
				}

				if (needsSorting) {
					graphData.sort(function(a, b) { return a[0] - b[0]; });
					var data = graphData;
					graphData = [];
					mergeGraphData(data);
				}
			};

			var restartValueMissingTimeout;

			api.update = function(data) {
				graphData.pop();

				mergeGraphData(data);
				clampToMaxAge();

				graphData.push([new Date(), NaN]);

				g.updateOptions({
					file: graphData
				});
				restartValueMissingTimeout();
			};

			var valuesMissing = function() {
				api.update([[+new Date(), NaN]]);
			};

			scope.$watch(attrs.maxAgeMs, function (val) {
				maxAgeMs = val === undefined ? undefined : +val;
			});

			scope.$watch(attrs.assumeMissingAfterMs, function(val) {
				assumeMissingAfterMs = val === undefined ? undefined : +val;
				restartValueMissingTimeout();
			});

			restartValueMissingTimeout = function() {
				if (valueMissingInterval !== undefined) {
					$interval.cancel(valueMissingInterval);
				}
				if (assumeMissingAfterMs !== undefined) {
					valueMissingInterval = $interval(valuesMissing, assumeMissingAfterMs);
				}
			};
		}
	};
}]);
