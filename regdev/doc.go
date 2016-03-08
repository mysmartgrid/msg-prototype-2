// Package regdev contains the device database.
// Every physical (or logical) devices that wants to talk to the MSGp service will require an entry in the device database,
// which stores management and health information of the devices.
//
// In particular, network configurations are stored in the database to allow configuration of devices through a web interface.
// Health information is stored in the form of heartbeats, which the device may send at any time to retrieve configuration information
// and inform the service about its current status (including statistics like current memory usage, uptime and so on).
//
//The device database also stores to which user a device has been connected as part of the device configuration information.
package regdev
