#!/bin/bash
echo "Running Invalid Data"
./bin/udpbohrer outside udpbohrer.yaml &
sleep 1
./bin/udpbohrer inside udpbohrer.yaml &
./bin/stressudpbohrer udpbohrer.yaml reflector 30000 20
echo "First run is done, now waiting for 100s to reset everything"
echo "Now sending invalid data to Outside and Inside"
./bin/stressudpbohrer udpbohrer.yaml invalid 3000 20
echo "Wait 100s to reset everything"
sleep 100
echo "Now sending invalid data to Outside and Inside"
./bin/stressudpbohrer udpbohrer.yaml invalid 3000 20
echo "Second run is done, now waiting for 100s to reset everything"
sleep 100
./bin/stressudpbohrer udpbohrer.yaml reflector 40000 1000
killall udpbohrer



