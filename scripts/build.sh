echo "Build started"
echo "Building testcode..."
go build -v -o bin/testcode cmd/testcode/main.go || exit $?
echo "Building udpbohrer..."
go build -v -o bin/udpbohrer cmd/udpbohrer/main.go || exit $?
echo "Building stressudpbohrer..."
go build -v -o bin/stressudpbohrer cmd/stressudpbohrer/main.go || exit $?
echo "Building tcpbohrer..."
go build -v -o bin/tcpbohrer cmd/tcpbohrer/main.go || exit $?
echo "Building stresstcpbohrer..."
go build -v -o bin/stresstcpbohrer cmd/stresstcpbohrer/main.go || exit $?
echo "Build completed"
