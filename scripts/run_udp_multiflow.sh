#!/bin/bash
echo "Running Stresstest 1"
./bin/udpbohrer outside udpbohrer.yaml &
sleep 1
./bin/udpbohrer inside udpbohrer.yaml &
./bin/stressudpbohrer udpbohrer.yaml reflector 30000 20
echo "First run is done, now waiting for 100s to reset everything"
sleep 100
./bin/stressudpbohrer udpbohrer.yaml reflector 40000 10000
echo "Second run is done, now waiting for 100s to reset everything"
sleep 100
./bin/stressudpbohrer udpbohrer.yaml reflector 40000 10
killall udpbohrer



