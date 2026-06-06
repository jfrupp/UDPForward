#!/bin/bash
./bin/udpbohrer outside udpbohrer.yaml &
sleep 1
./bin/udpbohrer inside udpbohrer.yaml &
sleep 240
killall udpbohrer


