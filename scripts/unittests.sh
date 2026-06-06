echo "Unittests started"
go clean -testcache
go test -v -race ./pkg/boolstorage ./pkg/timehelper ./pkg/controlpacker ./pkg/ipaddresshelper \
./app/udpbohrer/udpmessage  ./pkg/connlimiter ./pkg/qtime ./app/udpbohrer/udpconnout2remote \
./app/udpbohrer/udpconnmaplocal ./app/udpbohrer/udpbohrerparameters || exit $?
echo "Unittests completed"

