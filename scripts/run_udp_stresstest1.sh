#!/bin/bash

echo "Running Stresstest 1"

./bin/udpbohrer outside udpbohrer.yaml &
sleep 1
./bin/udpbohrer inside udpbohrer.yaml &
./bin/stressudpbohrer udpbohrer.yaml usual 30000 300000


