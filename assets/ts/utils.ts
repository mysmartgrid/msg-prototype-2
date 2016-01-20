module Utils {


    export class ExtArray<U> extends Array<U> {

        public contains(element : U) : boolean {
            var i = this.indexOf(element);
            return i !== -1;
        }

        public remove(element : U) : void {
            var i = this.indexOf(element);
            if(i !== -1) {
                this.splice(i,1);
            }
        }

        public removeWhere(pred: (element : U) => boolean) {
            var i = this.findIndex(pred);
            while(i !== -1) {
                this.splice(i,1);
                var i = this.findIndex(pred);
            }
        }
    }

    export function deepCopyJSON<T>(src : T) : T {
        var dst : any = {};

        for(var key in src) {
            if(src.hasOwnProperty(key)) {
                if(typeof(src[key]) === "object") {
                    dst[key] = deepCopyJSON(src[key]);
                }
                else {
                    dst[key] = src[key];
                }
            }
        }

        return dst;
    }
}
