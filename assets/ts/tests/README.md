How to run the tests
====================
To run the tests you'll need to have the typescript compiler tsc and browserify installed.

1. Run `tsc` in the tests directory to generate the `js_modules` directory.
2. Run `browserify -o sensorvaluestore.js js_modules/tests/sensorvaluestore.js`.
2. Open runtests.html in Firefox/Chrome/Chormium/Opera ...
