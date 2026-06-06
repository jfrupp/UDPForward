# HOWTO Build
The folders *.devcontainer*, *.forgejo* and *.vscode* contain the the configurations for VSCode and Forgejo.

## Development builds
Development builds are triggered with *./scripts/build.sh*. Execute from the root of this repository. Builds for detection of race conditions are triggered by *./scripts/build-race.sh*. The results of these builds are stored in the *bin/* directory. The binaries *stresstcpbohrer* and *stressudpbohrer* are test fixtures for testing *tcpbohrer* and *udpbohrer*. All development builds are for the local architecture. Minimum Golang version is 1.24.4.

## Deployment builds
Deployment builds are triggered with *./scripts/build-linux-x64-arm64-arm.sh*. This builds tcpbohrer and udpbohrer for Linux with the architectures x64, arm64 amd arm (32 bit). Builds are in *./bin/architecture* You may modify this file to build for your architecture.


