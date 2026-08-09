package tcpparameters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigurationReadsFlowPortKnockName(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "tcpbohrer.yaml")

	config := `header:
  application: "tcpbohrer"
  configFileVersion: 1
  minApplicationMajorVersion: 1
secret: "secret"
log:
  intervalSeconds: 1.0
  limitBurst: 20
  limitHeloLoggingSeconds: 20
funnel:
  protocol: "tcp4"
  outHost: "127.0.0.1"
  outPort: 9999
  maxCurrentConnections: 100
  heloRepeatInterval: 10.0
  heloTimeout: 10.0
  maxControlTimeDifferenceSeconds: 60
  maxWaitForValidSystemTimeSeconds: 120
  portBufferSize: 4000
  connectTimeout: 4
  heloFile: ""
portknock:
  portknockport: 443
  portknockkey: "ssl-cert-snakeoil.key"
  portknockcert: "ssl-cert-snakeoil.crt"
flows:
  1:
    name: "Flow"
    inProtocol: "tcp4"
    outProtocol: "tcp4"
    outPort: 1234
    inHost: "127.0.0.1"
    inPort: 80
    secondsTimeout: 10.0
    portknockname: "1234"
`

	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, _, err := LoadConfiguration(configPath, 1)
	if err != nil {
		t.Fatalf("expected config to load, got: %v", err)
	}
	if got := cfg.Flows[1].PortKnockName; got != "1234" {
		t.Fatalf("flow portknockname = %q, want %q", got, "1234")
	}
}

func TestLoadConfigurationRejectsInvalidPortKnockConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "tcpbohrer.yaml")

	config := `header:
  application: "tcpbohrer"
  configFileVersion: 1
  minApplicationMajorVersion: 1
secret: "secret"
log:
  intervalSeconds: 1.0
  limitBurst: 20
  limitHeloLoggingSeconds: 20
funnel:
  protocol: "tcp4"
  outHost: "127.0.0.1"
  outPort: 9999
  maxCurrentConnections: 100
  heloRepeatInterval: 10.0
  heloTimeout: 10.0
  maxControlTimeDifferenceSeconds: 60
  maxWaitForValidSystemTimeSeconds: 120
  portBufferSize: 4000
  connectTimeout: 4
  heloFile: ""
portknock:
  portknockport: 0
  portknockkey: "ssl-cert-snakeoil.key"
  portknockcert: "ssl-cert-snakeoil.crt"
flows:
  1:
    name: "Flow"
    inProtocol: "tcp4"
    outProtocol: "tcp4"
    outPort: 1234
    inHost: "127.0.0.1"
    inPort: 80
    secondsTimeout: 10.0
    portknockname: "1234"
`

	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, _, _, err := LoadConfiguration(configPath, 1)
	if err == nil {
		t.Fatal("expected an error for invalid port-knock configuration")
	}
	if !strings.Contains(err.Error(), "portknock") {
		t.Fatalf("expected an error mentioning portknock, got: %v", err)
	}
}

func TestLoadConfigurationInitializesPortKnockFromFlows(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "tcpbohrer.yaml")

	config := `header:
  application: "tcpbohrer"
  configFileVersion: 1
  minApplicationMajorVersion: 1
secret: "secret"
log:
  intervalSeconds: 1.0
  limitBurst: 20
  limitHeloLoggingSeconds: 20
funnel:
  protocol: "tcp4"
  outHost: "127.0.0.1"
  outPort: 9999
  maxCurrentConnections: 100
  heloRepeatInterval: 10.0
  heloTimeout: 10.0
  maxControlTimeDifferenceSeconds: 60
  maxWaitForValidSystemTimeSeconds: 120
  portBufferSize: 4000
  connectTimeout: 4
  heloFile: ""
portknock:
  portknockport: 443
  portknockkey: "ssl-cert-snakeoil.key"
  portknockcert: "ssl-cert-snakeoil.crt"
flows:
  1:
    name: "Flow 1"
    inProtocol: "tcp4"
    outProtocol: "tcp4"
    outPort: 1234
    inHost: "127.0.0.1"
    inPort: 80
    secondsTimeout: 10.0
    portknockname: "1234"
  2:
    name: "Flow 2"
    inProtocol: "tcp4"
    outProtocol: "tcp4"
    outPort: 1235
    inHost: "127.0.0.1"
    inPort: 81
    secondsTimeout: 10.0
    portknockname: "5678"
  3:
    name: "Flow 3"
    inProtocol: "tcp4"
    outProtocol: "tcp4"
    outPort: 1236
    inHost: "127.0.0.1"
    inPort: 82
    secondsTimeout: 10.0
    # portknockname: "1234"
`

	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, _, knock, err := LoadConfiguration(configPath, 1)
	if err != nil {
		t.Fatalf("expected flow-driven port-knock config to load, got: %v", err)
	}
	if !knock.CheckName("1234") {
		t.Fatal("expected port-knock name 1234 to be initialized from flow config")
	}
	if !knock.CheckName("5678") {
		t.Fatal("expected port-knock name 5678 to be initialized from flow config")
	}
}
