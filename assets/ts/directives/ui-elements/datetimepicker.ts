/// <reference path="../angular.d.ts" />
/// <reference path="../common.ts" />

module Directives.UserInterface {

    export class DateTimePickerController {
        constructor(private $scope : any) {

            if($scope.ngModel !== undefined) {
                $scope.$watch("ngModel", () : void => {
                    if(this.$scope.ngModel !== this._dateToMillisecs()) {
                        this._millisecsToDate($scope.ngModel)
                    }
                });
            }

            $scope.change = () : void => this._change();
        }

        private _millisecsToDate(millisecs : number) : void {
            this.$scope.date = new Date(millisecs);
        }

        private _dateToMillisecs() : number {
            var result = new Date(this.$scope.date);
            return result.getTime();
        }

        private _change() : void  {
            if(this.$scope.date !== null) {
                var millisecs  = this._dateToMillisecs();

                if(this.$scope.min !== undefined) {
                    millisecs = Math.max(millisecs, this.$scope.min);
                }
                if(this.$scope.max !== undefined) {
                    millisecs = Math.min(millisecs, this.$scope.max);
                }

                this.$scope.ngModel = millisecs;

                this.$scope.ngChange();
            }
        }

    }


    export function DateTimePickerFactory() : () => ng.IDirective {
        return () => new DateTimePickerDirective();
    }

    class DateTimePickerDirective implements ng.IDirective {
        public restrict : string = "A"
        public templateUrl : string = "/html/date-time-picker.html"
        public scope = {
            ngModel: '=?',
            ngChange: '&',
            min: '=?',
            max: '=?'
        }

        public controller = ["$scope", DateTimePickerController];

        // Link function is special ... see http://blog.aaronholmes.net/writing-angularjs-directives-as-typescript-classes/#comment-2206875553
        public link:Function  = ($scope : any,
                                    element : ng.IAugmentedJQuery,
                                    attrs : ng.IAttributes,
                                    aateTimePicker : DateTimePickerController) : void => {
        }
    }

}
