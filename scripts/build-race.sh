echo "Build started with race detection"
echo "Building testcode..."
go build -race -v -o bin/testcode cmd/testcode/main.go || exit $?
echo "Building udpbohrer..."
go build -race -v -o bin/udpbohrer cmd/udpbohrer/main.go || exit $?
echo "Building stressudpbohrer..."
go build -race -v -o bin/stressudpbohrer cmd/stressudpbohrer/main.go || exit $?
echo "Building tcpbohrer..."
go build -race -v -o bin/tcpbohrer cmd/tcpbohrer/main.go || exit $?
echo "Building stresstcpbohrer..."
go build -race -v -o bin/stresstcpbohrer cmd/stresstcpbohrer/main.go || exit $?
echo "Build completed"
