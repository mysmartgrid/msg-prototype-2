

export function contains<U>(haystack : U[], needle : U) : boolean {
    var i = haystack.indexOf(needle);
    return i !== -1;
}

export function remove<U>(haystack : U[], needle : U) : void {
    var i = this.indexOf(needle);
    if(i !== -1) {
        this.splice(i,1);
    }
}

export function removeWhere<U>(haystack : U[], pred: (element : U) => boolean) {
    var i = this.findIndex(pred);
    while(i !== -1) {
        this.splice(i,1);
        var i = this.findIndex(pred);
    }
}


export function updateProperties<U>(target : U, source: U) : boolean {
    var wasUpdated = false;
    for(var prop in target) {
        if(target[prop] !== source[prop]) {
            target[prop] = source[prop];
            wasUpdated = true;
        }
    }

    return wasUpdated;
}

export function now() : number {
    return (new Date()).getTime();
}


export function deepCopyJSON<T>(src : T) : T {
    var dst : any = {};

    if(Array.isArray(src)) {
        dst = [];
    }

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

export function difference<T>(a : T[], b : T[], equals : (x : T, y : T) => boolean) : T[] {
    return a.filter((a_element) => b.findIndex((b_element) => equals(a_element , b_element)) === -1);
}
