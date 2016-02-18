define(["require", "exports"], function (require, exports) {
    "use strict";
    var TimeUnits = ["years", "days", "hours", "minutes"];
    var UnitsToMillisecs = {
        "years": 365 * 24 * 60 * 60 * 1000,
        "days": 24 * 60 * 60 * 1000,
        "hours": 60 * 60 * 1000,
        "minutes": 60 * 1000
    };
    var TimeRangeSpinnerController = (function () {
        function TimeRangeSpinnerController($scope) {
            var _this = this;
            this.$scope = $scope;
            $scope.time = {
                years: 0,
                days: 0,
                hours: 0,
                minutes: 0
            };
            if ($scope.ngModel !== undefined) {
                $scope.$watch("ngModel", function () { return _this._setFromMilliseconds($scope.ngModel); });
            }
            console.log($scope);
            $scope.change = function () { return _this._change(); };
            $scope.increment = function (unit) { return _this._increment(unit); };
            $scope.decrement = function (unit) { return _this._decrement(unit); };
        }
        TimeRangeSpinnerController.prototype._increment = function (unit) {
            if (this.$scope.time[unit] !== undefined) {
                this.$scope.time[unit] += 1;
            }
            this.$scope.change();
        };
        TimeRangeSpinnerController.prototype._decrement = function (unit) {
            if (this.$scope.time[unit] !== undefined) {
                this.$scope.time[unit] -= 1;
            }
            this.$scope.change();
        };
        TimeRangeSpinnerController.prototype._change = function () {
            var _this = this;
            var editDone = TimeUnits.every(function (unit) { return (_this.$scope.time[unit] !== null && _this.$scope.time[unit] !== undefined); });
            if (editDone) {
                var milliseconds = 0;
                for (var _i = 0, TimeUnits_1 = TimeUnits; _i < TimeUnits_1.length; _i++) {
                    var unit = TimeUnits_1[_i];
                    milliseconds += this.$scope.time[unit] * UnitsToMillisecs[unit];
                }
                if (this.$scope.min !== undefined) {
                    milliseconds = Math.max(this.$scope.min, milliseconds);
                }
                if (this.$scope.max !== undefined) {
                    milliseconds = Math.min(this.$scope.max, milliseconds);
                }
                this._setFromMilliseconds(milliseconds);
                if (this.$scope.ngModel !== undefined) {
                    this.$scope.ngModel = milliseconds;
                }
                this.$scope.ngChange();
            }
        };
        TimeRangeSpinnerController.prototype._setFromMilliseconds = function (milliseconds) {
            var remainder = milliseconds;
            for (var _i = 0, TimeUnits_2 = TimeUnits; _i < TimeUnits_2.length; _i++) {
                var unit = TimeUnits_2[_i];
                this.$scope.time[unit] = Math.floor(remainder / UnitsToMillisecs[unit]);
                remainder = remainder % UnitsToMillisecs[unit];
            }
        };
        TimeRangeSpinnerController.prototype.setupScrollEvents = function (element) {
            var _this = this;
            element.find("input[type='number']").each(function (index, element) {
                var field = $(element);
                field.bind("mouse wheel", function (jqEvent) {
                    if (jqEvent.originalEvent === undefined) {
                        return;
                    }
                    var event = jqEvent.originalEvent;
                    var delta = event.wheelDelta;
                    if (delta === undefined) {
                        delta = -event.deltaY;
                    }
                    if (delta > 0) {
                        _this.$scope.increment(field.attr('name'));
                    }
                    else {
                        _this.$scope.decrement(field.attr('name'));
                    }
                    jqEvent.preventDefault();
                });
            });
        };
        return TimeRangeSpinnerController;
    }());
    function TimeRangeSpinnerFactory() {
        return function () { return new TimeRangeSpinnerDirective(); };
    }
    Object.defineProperty(exports, "__esModule", { value: true });
    exports.default = TimeRangeSpinnerFactory;
    var TimeRangeSpinnerDirective = (function () {
        function TimeRangeSpinnerDirective() {
            this.restrict = "A";
            this.templateUrl = "/html/time-range-spinner.html";
            this.scope = {
                ngModel: '=?',
                ngChange: '&',
                min: '=?',
                max: '=?'
            };
            this.controller = ["$scope", TimeRangeSpinnerController];
            this.link = function ($scope, element, attrs, controller) {
                controller.setupScrollEvents(element);
            };
        }
        return TimeRangeSpinnerDirective;
    }());
});
//# sourceMappingURL=timerangespinner.js.map