#!/bin/bash

echo "Killing all stresstest processes..."

killall -KILL udpbohrer || true
killall -KILL stressudpbohrer || true
killall -KILL tcpbohrer || true
killall -KILL stresstcpbohrer || true

echo "Stresstest processes killed!"

