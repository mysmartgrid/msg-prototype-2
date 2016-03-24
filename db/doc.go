// Package db is the user/value database.
// db/interfaces.go contains the list of interfaces to the database: there are Users, which have Devices, which have Sensors, which have Values.
// Sensors and Users can be furter organized in Groups.
// Everything is stored using a postrgesql database.
//
// All access to the user database and value storage is handled through the Tx object, which behaves a lot like a PostgreSQL transaction.
//
// Since Users, Devices and Sensors form a hierarchy, removing an instance of any object automatically removes all instances of nested objects and attached measurement data.
package db
