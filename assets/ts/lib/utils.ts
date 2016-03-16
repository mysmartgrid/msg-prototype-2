

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

/*
 * Recursively compare two json objects.
 *
 * Return values:
 * [] = No difference (just contains the same values in the same structure; doesn't imply a und b are the same object)
 * [''] = Root objects have different type or at least one root object is undefined
 * ['foo'] = Property foo exists in one object but not the other,
 *              or foo has a different value in the other object.
 *              Arrays are assumed different if length, element values or the order differ.
 * ['foo.bar'] = Property foo exists on both objects, is an object in differs in property bar.
 */
export function differentProperties<T>(a : T, b : T, prefix? : string) : string[] {
    // Root objects have prefix ''
    if(prefix === undefined) {
        prefix = '';
    }

    // One at leats of both is undefined
    if(a === undefined || b === undefined) {
        return [prefix];
    }
    // Objects of different type
    else if(typeof(a) !== typeof(b)) {
        return [prefix];
    }
    // Both are arrays
    else if(Array.isArray(a)) {

        // Lenghts are different
        if((<any>a).length !== (<any>b).length) {
            return [prefix];
        }
        else {
            // Look for different elements
            for(var i = 0; i < (<any>a).length; i++) {
                // Recursive call just to check equaltity on complex values
                if(differentProperties(a[i], b[i]).length !== 0) {
                    return [prefix];
                }
            }
        }
    }
    // Both are objects
    else if(typeof(a) === 'object') {
        var differences = [];

        // Generate the union of both objects property sets
        var keys = Object.keys(a);
        for(var key of Object.keys(b)) {
            addOnce(keys, key);
        }

        // Check each key
        for(var key of keys) {
            // Extend the prefix for recursive call
            var extendedPrefix = prefix !== '' ? prefix + '.' + key : key;
            // Recurse. Returns [extencedPrefix] in case a or b does not have a property key
            differences = differences.concat(differentProperties(a[key], b[key], extendedPrefix));
        }

        return differences;
    }
    // Primitive values, just compare.
    else if(a !== b) {
        return [prefix];
    }

    // If we did not return until here, there are no differences.
    return [];
}
