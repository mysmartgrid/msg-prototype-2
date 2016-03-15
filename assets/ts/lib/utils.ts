

export function contains<U>(haystack : U[], needle : U) : boolean {
    var i = haystack.indexOf(needle);
    return i !== -1;
}

export function remove<U>(haystack : U[], needle : U) : void {
    var i = haystack.indexOf(needle);
    if(i !== -1) {
        haystack.splice(i,1);
    }
}

export function removeWhere<U>(haystack : U[], pred: (element : U) => boolean) {
    var i = haystack.findIndex(pred);
    while(i !== -1) {
        haystack.splice(i,1);
        i = haystack.findIndex(pred);
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

export function difference<T>(a : T[], b : T[], equals? : (x : T, y : T) => boolean) : T[] {
    if(equals === undefined) {
        equals = (x,y) => x === y;
    }

    return a.filter((a_element) => b.findIndex((b_element) => equals(a_element , b_element)) === -1);
}


export function addOnce<T>(list : T[], element : T) {
    if(!contains(list, element)) {
        list.push(element);
    }
}

export function differentProperties<T>(a : T, b : T) : string[] {
    if(a === undefined || b === undefined) {
        return undefined;
    }

    var differences = [];

    var keys = Object.keys(a);
    for(var key in b) {
        addOnce(keys, key);
    }

    for(var key of keys) {
        // Property is only present a or b
        if(!(a.hasOwnProperty(key) && b.hasOwnProperty(key))) {
            differences.push(key);
        }
        // Properties with different types
        else if(typeof(a[key]) !== typeof(b[key])) {
            differences.push(key);
        }
        // Both properties are arrays
        else if(Array.isArray(a[key])) {
            // Not the some length -> different
            if(a[key].length !== b[key].length) {
                differences.push(key);
            }
            else {
                // Check if the elements at each position are equal
                for(var i = 0; i < a[key].length; i++) {
                    if(a[key][i] !== b[key][i]) {
                        differences.push(key);
                        break;
                    }
                }
            }
        }
        // Both properties are objects -> recursive descent
        else if(typeof(a[key]) === "object") {
            var result = differentProperties(a[key], b[key]);
            result = result.map((subkey) => key + '.' + subkey);
            differences.concat(result);
        }
        // Both properties are primitive types -> compare them
        else if(a[key] !== b[key]) {
            differences.push(key);
        }
    }

    console.log(differences);

    return differences;
}
