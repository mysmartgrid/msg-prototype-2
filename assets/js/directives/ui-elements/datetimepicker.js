define(["require", "exports"], function (require, exports) {
    "use strict";
    var DateTimePickerController = (function () {
        function DateTimePickerController($scope) {
            var _this = this;
            this.$scope = $scope;
            if ($scope.ngModel !== undefined) {
                $scope.$watch("ngModel", function () {
                    if (_this.$scope.ngModel !== _this._dateToMillisecs()) {
                        _this._millisecsToDate($scope.ngModel);
                    }
                });
            }
            $scope.change = function () { return _this._change(); };
        }
        DateTimePickerController.prototype._millisecsToDate = function (millisecs) {
            this.$scope.date = new Date(millisecs);
        };
        DateTimePickerController.prototype._dateToMillisecs = function () {
            var result = new Date(this.$scope.date);
            return result.getTime();
        };
        DateTimePickerController.prototype._change = function () {
            if (this.$scope.date !== null) {
                var millisecs = this._dateToMillisecs();
                if (this.$scope.min !== undefined) {
                    millisecs = Math.max(millisecs, this.$scope.min);
                }
                if (this.$scope.max !== undefined) {
                    millisecs = Math.min(millisecs, this.$scope.max);
                }
                this.$scope.ngModel = millisecs;
                this.$scope.ngChange();
            }
        };
        return DateTimePickerController;
    }());
    function DateTimePickerFactory() {
        return function () { return new DateTimePickerDirective(); };
    }
    Object.defineProperty(exports, "__esModule", { value: true });
    exports.default = DateTimePickerFactory;
    var DateTimePickerDirective = (function () {
        function DateTimePickerDirective() {
            this.restrict = "A";
            this.templateUrl = "/html/date-time-picker.html";
            this.scope = {
                ngModel: '=?',
                ngChange: '&',
                min: '=?',
                max: '=?'
            };
            this.controller = ["$scope", DateTimePickerController];
            this.link = function ($scope, element, attrs, aateTimePicker) {
            };
        }
        return DateTimePickerDirective;
    }());
});
//# sourceMappingURL=datetimepicker.js.map