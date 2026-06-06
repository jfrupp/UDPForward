#!/bin/bash
echo "Running Stresstest 1"
./bin/tcpbohrer outside tcpbohrer.yaml &
sleep 1
./bin/tcpbohrer inside tcpbohrer.yaml &
./bin/stresstcpbohrer tcpbohrer.yaml usual 100
sleep 2
./bin/stresstcpbohrer tcpbohrer.yaml usual 500
killall tcpbohrer
echo "Done"




