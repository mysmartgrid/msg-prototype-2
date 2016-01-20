/// <reference path="../angular.d.ts" />


module Directives.UserInterface {

    export class NumberSpinnerController {
        constructor(private $scope : any) {

            $scope.change = () : void => {
                this._enforceLimits();
                $scope.ngChange();
            };

            $scope.increment = () => {
                $scope.ngModel++;
                $scope.change();
            }

            $scope.decrement = () => {
                $scope.ngModel--;
                $scope.change();
            }
        }

        private _enforceLimits() : void {
            if(this.$scope.ngModel !== undefined && this.$scope.ngModel !== null) {
                console.log("Enforcing limits");
                // Normalize to integer
                if(this.$scope.ngModel !== Math.round(this.$scope.ngModel)) {
                    this.$scope.ngModel = Math.round(this.$scope.ngModel);
                }

                if(this.$scope.ngModel > this.$scope.max) {
                    this.$scope.ngModel = this.$scope.max;
                    console.log("Overflow");
                    this.$scope.overflow();
                }

                if(this.$scope.ngModel < this.$scope.min) {
                    this.$scope.ngModel = this.$scope.min;
                    this.$scope.underflow();
                }

                console.log("Limits enforced");
            }
        }

        private _onMouseWheel(event : any) : void {
            if (event.originalEvent !== undefined) {
                event = event.originalEvent;
            }

            var delta = event.wheelDelta;
            if(delta === undefined) {
                delta = -event.deltaY;
            }

            if(Math.abs(delta) > 10) {
                if(delta > 0) {
                    this.$scope.increment();
                }
                else {
                    this.$scope.decrement();
                }
            }
            event.preventDefault();
        }

        public setupEvents(element : ng.IAugmentedJQuery) : void {
            var input = element.find(".numberSpinner");
            input.bind("mouse wheel", (event : any) : void => this._onMouseWheel(event));
        }
    }


    export function NumberSpinnerFactory() : () => ng.IDirective {
        return () => new NumberSpinnerDirective();
    }

    class NumberSpinnerDirective implements ng.IDirective {
        public require : string = "numberSpinner"
        public restrict : string = "A"
        public templateUrl : string = "/html/number-spinner.html"
        public scope = {
            ngModel: '=',
            ngChange: '&',
            overflow: '&',
            underflow: '&',
            min: '=',
            max: '='
        }

        public controller = ["$scope", NumberSpinnerController];

        // Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
        public link:Function  = ($scope : any,
                                    element : ng.IAugmentedJQuery,
                                    attrs : ng.IAttributes,
                                    numberSpinner : NumberSpinnerController) : void => {

            numberSpinner.setupEvents(element);
        }
    }

}
