/// <reference path="../angular.d.ts" />
/// <reference path="../common.ts"/>


module Directives.UserInterface {
    const TimeUnits : string[] = ["years", "days", "hours", "minutes"]

    const UnitsToMillisecs : {[unit : string] : number} = {
        "years" : 365 * 24 * 60 * 60 * 1000,
        "days" : 24 * 60 * 60 * 1000,
        "hours" : 60 * 60 * 1000,
        "minutes" : 60 * 1000
    }

    interface TimeRangeSpinnerScope extends ng.IScope {
        time : {
            years : number;
            days : number;
            hours : number;
            minutes : number;
        };

        ngModel : number;
        min : number;
        max : number;
        ngChange : () => void;

        change : () => void;
        increment : (unit : string) => void;
        decrement : (unit : string) => void;
    }

    export class TimeRangeSpinnerController {
        constructor(private $scope : TimeRangeSpinnerScope) {
            $scope.time = {
                years : 0,
                days : 0,
                hours : 0,
                minutes: 0
            }

            if($scope.ngModel !== undefined) {
                $scope.$watch("ngModel", () : void => this._setFromMilliseconds($scope.ngModel));
            }

            console.log($scope);

            $scope.change = () : void => this._change();
            $scope.increment = (unit) : void => this._increment(unit);
            $scope.decrement = (unit) : void => this._decrement(unit);
        }

        private _increment(unit : string) : void {
            if(this.$scope.time[unit] !== undefined) {
                this.$scope.time[unit] += 1;
            }
            this.$scope.change();
        }

        private _decrement(unit : string) : void {
            if(this.$scope.time[unit] !== undefined) {
                this.$scope.time[unit] -= 1;
            }
            this.$scope.change();
        }

        private _change() : void {
            // because otherwise empty field become 0 during edit, which is a real pain
            var editDone = TimeUnits.every((unit) => (this.$scope.time[unit] !== null && this.$scope.time[unit] !== undefined));

            if(editDone) {
                var milliseconds = 0;
                for(var unit of TimeUnits) {
                    milliseconds += this.$scope.time[unit] * UnitsToMillisecs[unit];
                }

                if(this.$scope.min !== undefined) {
                    milliseconds = Math.max(this.$scope.min, milliseconds);
                }

                if(this.$scope.max !== undefined) {
                    milliseconds = Math.min(this.$scope.max, milliseconds);
                }

                this._setFromMilliseconds(milliseconds);

                if(this.$scope.ngModel !== undefined) {
                    this.$scope.ngModel = milliseconds;
                }

                this.$scope.ngChange();
            }
        }

        private _setFromMilliseconds(milliseconds : number) : void {
            var remainder = milliseconds;
            for(var unit of TimeUnits) {
                this.$scope.time[unit] = Math.floor(remainder / UnitsToMillisecs[unit]);
                remainder = remainder %  UnitsToMillisecs[unit];
            }
        }


        public setupScrollEvents(element : ng.IAugmentedJQuery) : void {
            element.find("input[type='number']").each((index : number, element : Element) : void => {
                var field = $(element);
                field.bind("mouse wheel", (jqEvent : JQueryEventObject) : void => {
                    if (jqEvent.originalEvent === undefined) {
                        return;
                    }

                    var event : any  = jqEvent.originalEvent;

                    var delta = event.wheelDelta;
                    if(delta === undefined) {
                        delta = -event.deltaY;
                    }

                    if(delta > 0) {
                        this.$scope.increment(field.attr('name'));
                    }
                    else {
                        this.$scope.decrement(field.attr('name'));
                    }

                    jqEvent.preventDefault();
                });
            });
        }


    }


    export function TimeRangeSpinnerFactory() : () => ng.IDirective {
        return () => new TimeRangeSpinnerDirective();
    }


    class TimeRangeSpinnerDirective implements ng.IDirective {
        public restrict : string = "A"
        public templateUrl : string = "/html/time-range-spinner.html"
        public scope = {
            ngModel: '=',
            ngChange: '&',
            min: '=',
            max: '='
        }

        public controller = ["$scope", TimeRangeSpinnerController];

        // Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
        public link:Function  = ($scope : TimeRangeSpinnerScope,
                                    element : ng.IAugmentedJQuery,
                                    attrs : ng.IAttributes,
                                    controller : TimeRangeSpinnerController) : void => {

            controller.setupScrollEvents(element);

        }
    }
}
