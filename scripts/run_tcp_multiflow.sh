#!/bin/bash
echo "Running Stresstest 1"
./bin/tcpbohrer outside tcpbohrer.yaml &
sleep 1
./bin/tcpbohrer inside tcpbohrer.yaml &
sleep 2
./bin/stresstcpbohrer tcpbohrer.yaml usual 2
killall tcpbohrer



