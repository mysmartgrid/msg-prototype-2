var __extends = (this && this.__extends) || function (d, b) {
    for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p];
    function __() { this.constructor = d; }
    d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
};
define(["require", "exports"], function (require, exports) {
    "use strict";
    var ExtArray = (function (_super) {
        __extends(ExtArray, _super);
        function ExtArray() {
            _super.apply(this, arguments);
        }
        ExtArray.prototype.contains = function (element) {
            var i = this.indexOf(element);
            return i !== -1;
        };
        ExtArray.prototype.remove = function (element) {
            var i = this.indexOf(element);
            if (i !== -1) {
                this.splice(i, 1);
            }
        };
        ExtArray.prototype.removeWhere = function (pred) {
            var i = this.findIndex(pred);
            while (i !== -1) {
                this.splice(i, 1);
                var i = this.findIndex(pred);
            }
        };
        return ExtArray;
    }(Array));
    exports.ExtArray = ExtArray;
    function deepCopyJSON(src) {
        var dst = {};
        if (Array.isArray(src)) {
            dst = [];
        }
        for (var key in src) {
            if (src.hasOwnProperty(key)) {
                if (typeof (src[key]) === "object") {
                    dst[key] = deepCopyJSON(src[key]);
                }
                else {
                    dst[key] = src[key];
                }
            }
        }
        return dst;
    }
    exports.deepCopyJSON = deepCopyJSON;
    function difference(a, b, equals) {
        return a.filter(function (a_element) { return b.findIndex(function (b_element) { return equals(a_element, b_element); }) === -1; });
    }
    exports.difference = difference;
});
//# sourceMappingURL=utils.js.map