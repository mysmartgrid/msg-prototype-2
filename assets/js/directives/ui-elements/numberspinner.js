define(["require", "exports"], function (require, exports) {
    "use strict";
    var NumberSpinnerController = (function () {
        function NumberSpinnerController($scope) {
            var _this = this;
            this.$scope = $scope;
            $scope.change = function () {
                _this._enforceLimits();
                $scope.ngChange();
            };
            $scope.increment = function () {
                $scope.ngModel++;
                $scope.change();
            };
            $scope.decrement = function () {
                $scope.ngModel--;
                $scope.change();
            };
        }
        NumberSpinnerController.prototype._enforceLimits = function () {
            if (this.$scope.ngModel !== undefined && this.$scope.ngModel !== null) {
                console.log("Enforcing limits");
                if (this.$scope.ngModel !== Math.round(this.$scope.ngModel)) {
                    this.$scope.ngModel = Math.round(this.$scope.ngModel);
                }
                if (this.$scope.ngModel > this.$scope.max) {
                    this.$scope.ngModel = this.$scope.max;
                    console.log("Overflow");
                    this.$scope.overflow();
                }
                if (this.$scope.ngModel < this.$scope.min) {
                    this.$scope.ngModel = this.$scope.min;
                    this.$scope.underflow();
                }
                console.log("Limits enforced");
            }
        };
        NumberSpinnerController.prototype._onMouseWheel = function (event) {
            if (event.originalEvent !== undefined) {
                event = event.originalEvent;
            }
            var delta = event.wheelDelta;
            if (delta === undefined) {
                delta = -event.deltaY;
            }
            if (Math.abs(delta) > 10) {
                if (delta > 0) {
                    this.$scope.increment();
                }
                else {
                    this.$scope.decrement();
                }
            }
            event.preventDefault();
        };
        NumberSpinnerController.prototype.setupEvents = function (element) {
            var _this = this;
            var input = element.find(".numberSpinner");
            input.bind("mouse wheel", function (event) { return _this._onMouseWheel(event); });
        };
        return NumberSpinnerController;
    }());
    function NumberSpinnerFactory() {
        return function () { return new NumberSpinnerDirective(); };
    }
    Object.defineProperty(exports, "__esModule", { value: true });
    exports.default = NumberSpinnerFactory;
    var NumberSpinnerDirective = (function () {
        function NumberSpinnerDirective() {
            this.require = "numberSpinner";
            this.restrict = "A";
            this.templateUrl = "/html/number-spinner.html";
            this.scope = {
                ngModel: '=',
                ngChange: '&',
                overflow: '&',
                underflow: '&',
                min: '=',
                max: '='
            };
            this.controller = ["$scope", NumberSpinnerController];
            this.link = function ($scope, element, attrs, numberSpinner) {
                numberSpinner.setupEvents(element);
            };
        }
        return NumberSpinnerDirective;
    }());
});
//# sourceMappingURL=numberspinner.js.map