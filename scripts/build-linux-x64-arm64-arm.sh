echo "Cross build started"
echo "Building udpbohrer for Linux x64"
env GOOS=linux GOARCH=amd64 go build  -v -o bin/bin_x64/udpbohrer cmd/udpbohrer/main.go || exit $?
echo "Building udpbohrer for Linux arm64"
env GOOS=linux GOARCH=arm64 go build -v -o bin/bin_arm64/udpbohrer cmd/udpbohrer/main.go || exit $?
echo "Building udpbohrer for Linux arm (32bit)"
env GOOS=linux GOARCH=arm go build -v -o bin/bin_arm/udpbohrer cmd/udpbohrer/main.go || exit $?
echo "Building tcpbohrer for Linux x64"
env GOOS=linux GOARCH=amd64 go build  -v -o bin/bin_x64/tcpbohrer cmd/tcpbohrer/main.go || exit $?
echo "Building tcpbohrer for Linux arm64"
env GOOS=linux GOARCH=arm64 go build -v -o bin/bin_arm64/tcpbohrer cmd/tcpbohrer/main.go || exit $?
echo "Building tcpbohrer for Linux arm (32bit)"
env GOOS=linux GOARCH=arm go build -v -o bin/bin_arm/tcpbohrer cmd/tcpbohrer/main.go || exit $?
echo "Build completed"
