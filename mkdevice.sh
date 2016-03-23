#!/bin/bash
./msgpc newRandom heartbeat print save randomdevice.conf
echo "Register device to user, then press enter"
read
./msgpc load randomdevice.conf heartbeat genSensors 9 registerSensors print save randomdevice.conf

