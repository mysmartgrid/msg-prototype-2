import * as Utils from '../lib/utils';
import * as UpdateDispatcher from '../lib/updatedispatcher';
import * as Store from '../lib/sensorvaluestore';

import {ResolutionsPerMode, MetadataTree, SensorUnitMap} from '../lib/common';

export interface WidgetScope extends ng.IScope {
    units : string[];
    devices : MetadataTree;
    sensorsByUnit : SensorUnitMap;

    openSettings : () => void;
}


export interface WidgetConfig {};



export abstract class WidtgetController implements UpdateDispatcher.Subscriber{

    protected _config : WidgetConfig;

    protected _settingsTemplate : string;
    //TODO: Better type
    protected _settingsControllerFactory : any[];

    constructor(protected $scope : WidgetScope,
                protected _dispatcher : UpdateDispatcher.UpdateDispatcher,
                protected $uibModal : angular.ui.bootstrap.IModalService) {

        $scope.devices = this._dispatcher.devices;
        $scope.units = _dispatcher.units;
        $scope.sensorsByUnit = _dispatcher.sensorsByUnit;

        $scope.openSettings = () => this._openSettings();
    }

    private _openSettings() : void {
        var modalInstance = this.$uibModal.open({
            controller: this._settingsControllerFactory,
            size: "lg",
            templateUrl: this._settingsTemplate,
            resolve: {
                    config: () => {
                    return Utils.deepCopyJSON(this._config);
                    }
            }
        });

        modalInstance.result.then((config) : void => {
            this._applyConfig(config);
        });
    }

    protected abstract _applyConfig(config : WidgetConfig) : void;

    // Default noop implementation for subscriber
    public updateValue(deviceID : string, sensorID : string, resolution : string, timestamp : number, value : number) : void {};
    public updateDeviceMetadata(deviceID : string) : void {};
    public updateSensorMetadata(deviceID : string, sensorID : string) : void {};
    public removeDevice(deviceID : string) : void {};
    public removeSensor(deviceID : string, sensorID : string) : void {};
}

export interface WidgetSettingsScope extends WidgetScope {
    resolutions : typeof ResolutionsPerMode;
    config : WidgetConfig;

    ok : () => void;
    cancel : () => void;
}

export abstract class WidgetSettingsController {
    constructor(protected $scope : WidgetSettingsScope,
				protected $uibModalInstance : angular.ui.bootstrap.IModalServiceInstance,
				protected _dispatcher : UpdateDispatcher.UpdateDispatcher,
				config : WidgetConfig) {

		$scope.devices = _dispatcher.devices;
		$scope.units = _dispatcher.units;
		$scope.sensorsByUnit = _dispatcher.sensorsByUnit;
		$scope.resolutions = ResolutionsPerMode;

        $scope.config = config;

        $scope.ok = () => this._saveConfig();
        $scope.cancel = () => this._close();
    }

    protected abstract _checkConfig() : boolean;

    private _saveConfig() : void {
        if(this._checkConfig()) {
            this.$uibModalInstance.close(this.$scope.config);
        }
    }

    private _close() : void {
        this.$uibModalInstance.dismiss('cancel');
    }

}
