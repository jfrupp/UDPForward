package udpbohrerparameters

import (
	"controlpacker"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"

	"gopkg.in/yaml.v3"
)

// Default parameters are ignored by gopkg.in/yaml.v3!!!
type Flow struct {
	Name               string  `default:"Default Forwarder" yaml:"name"`
	InProtocol         string  `default:"udp4"              yaml:"inProtocol"`
	OutProtocol        string  `default:"udp4"              yaml:"outProtocol"`
	OutPort            int     `default:"0"                 yaml:"outPort"`
	InHost             string  `default:"127.0.0.1"         yaml:"inHost"`
	InPort             int     `default:"0"                 yaml:"inPort"`
	TimeoutSeconds     float32 `default:"187.5"             yaml:"timeoutSeconds"`
	InitTimeoutSeconds float32 `default:"7.7"               yaml:"initTimeoutSeconds"`
}

type Header struct {
	Application                string `yaml:"application"`       // no default value, user is required to set this to "udpbohrer"
	ConfigFileVersion          int    `yaml:"configFileVersion"` // no default value, user is required to provide version between 1 and 255
	MinApplicationMajorVersion int    `yaml:"minApplicationMajorVersion" default:"1"`
	Description                string `yaml:"description" default:"Udpproxy Config"`
}
type Log struct {
	IntervalSeconds         float32 `yaml:"intervalSeconds" default:"60.0"`
	LimitBurst              int     `yaml:"limitBurst" default:"20"`
	LimitHeloLoggingSeconds int     `yaml:"limitHeloLoggingSeconds" default:"20"`
}

type Funnel struct {
	Protocol                         string  `yaml:"protocol" default:"udp4"`
	OutHost                          string  `yaml:"outHost" default:"127.0.0.1"`
	Mtu                              int     `yaml:"mtu"	default:"1422"`
	OutPort                          int     `yaml:"outPort" default:"9999"`
	InPort                           int     `yaml:"inPort" default:"9999"`
	PoolMinInPortToHost              int     `yaml:"poolMinInPortToHost" default:"10000"`
	PoolMaxInPortToHost              int     `yaml:"poolMaxInPortToHost" default:"20000"`
	MaxCurrentFlows                  int     `yaml:"maxCurrentFlows" default:"100"`
	HeloRepeatInterval               float32 `yaml:"heloRepeatInterval" default:"10.0"`
	HeloTimeout                      float32 `yaml:"heloTimeout" default:"10.0"`
	HeloRoundtripTime                float32 `yaml:"heloRoundtripTime" default:"2.0"`
	MaxControlTimeDifferenceSeconds  int     `yaml:"maxControlTimeDifferenceSeconds" default:"600"`
	MaxWaitForValidSystemTimeSeconds int     `yaml:"maxWaitForValidSystemTimeSeconds" default:"120"`
	NumReservedMessageBlocks         int     `yaml:"numReservedMessageBlocks" default:"2000"`
	PortBufferSize                   int     `yaml:"portBufferSize" default:"50000"`
	HeloFile                         string  `yaml:"heloFile" default:""`
}

type UdpproxyConfig struct {
	Header Header       `yaml:"header"`
	Secret string       `yaml:"secret"`
	Log    Log          `yaml:"log"`
	Funnel Funnel       `yaml:"funnel"`
	Flows  map[int]Flow `yaml:"flows"`
}

func LoadConfiguration(ConfigFile string, ProgramVersionMajor uint8) (*UdpproxyConfig, *controlpacker.ControlPacker, error) {
	f, err := os.Open(ConfigFile)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}
	var c UdpproxyConfig
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, nil, err
	}
	// Check the parameters
	if c.Header.Application != "udpbohrer" {
		return nil, nil, errors.New("Configuration Error: header:application: Has to be set to \"udpbohrer\"")
	}
	if c.Header.ConfigFileVersion < 1 {
		return nil, nil, errors.New("Configuration Error: header:configFileVersion: Version has to be at least 1")
	}
	if c.Header.MinApplicationMajorVersion > int(ProgramVersionMajor) {
		return nil, nil, fmt.Errorf("Configuration Error: minApplicationMajorVersion: Program version major is %d, this configuration expects at least %d",
			ProgramVersionMajor, c.Header.MinApplicationMajorVersion)
	}
	if len(c.Secret) < 1 {
		return nil, nil, errors.New("Configuration Error: secret: No secret has been defined!")
	}
	// Secret is not needed in this data structure, ControPacker hashes the whole configuration file
	c.Secret = ""
	cp := controlpacker.NewControlPacker(ConfigFile, ProgramVersionMajor, int64(c.Funnel.MaxControlTimeDifferenceSeconds))
	if cp == nil {
		return nil, nil, errors.New(fmt.Sprintf("Configuration Error: Reading cofiguration file %s failed", ConfigFile))
	}
	if c.Log.IntervalSeconds < 0.001 {
		return nil, nil, errors.New("Configuration Error: log:intervalSeconds: Logging interval has to be at least 1ms")
	}
	if c.Log.LimitBurst < 2 {
		return nil, nil, errors.New("Configuration Error: log:limitBurst: Logging burst length has to be at least 2")
	}
	if c.Funnel.Protocol != "udp4" && c.Funnel.Protocol != "udp6" {
		return nil, nil, errors.New("Configuration Error: funnel:protocol: Protocol has to be \"⅛udp4\" or \"udp6\"")
	}
	if len(c.Funnel.OutHost) < 1 {
		return nil, nil, errors.New("Configuration Error: funnel:outHost: Outside host has to be specified")
	}
	if c.Funnel.OutPort <= 0 || c.Funnel.OutPort > 65535 {
		return nil, nil, errors.New("Configuration Error: funnel:outPort: Must be between 0 and 65535")
	}
	if c.Funnel.InPort <= 0 || c.Funnel.InPort > 65535 {
		return nil, nil, errors.New("Configuration Error: funnel:inPort: Must be between 0 and 65535")
	}
	if c.Funnel.Mtu < 256 {
		return nil, nil, errors.New("Configuration Error: funnel:mtu: Must be greater than 255")
		if c.Funnel.Protocol != "udp4" { // remove the IPV4 + UDP overhead, it will be added during sending
			c.Funnel.Mtu -= 28
		}
		if c.Funnel.Protocol != "udp6" { // remove the IPV6 + UDP overhead, it will be added during sending
			c.Funnel.Mtu -= 48
		}
	}
	if c.Funnel.PoolMinInPortToHost > c.Funnel.PoolMaxInPortToHost {
		p := c.Funnel.PoolMaxInPortToHost
		c.Funnel.PoolMaxInPortToHost = c.Funnel.PoolMinInPortToHost
		c.Funnel.PoolMinInPortToHost = p
	}
	if c.Funnel.PoolMinInPortToHost < 0 || c.Funnel.PoolMinInPortToHost > 65535 ||
		c.Funnel.PoolMaxInPortToHost < 0 || c.Funnel.PoolMaxInPortToHost > 65535 ||
		c.Funnel.PoolMaxInPortToHost-c.Funnel.PoolMinInPortToHost < 100 {
		return nil, nil, errors.New("Configuration Error: funnel:poolMinInPortToHost: and funnel:poolMaxInPortToHost: Must be between 0 and 65535, difference at least 100")
	}
	if c.Funnel.MaxCurrentFlows < 2 || c.Funnel.MaxCurrentFlows > 254 {
		return nil, nil, errors.New("Configuration Error: funnel:poolmaxCurrentFlows: must be between 2 and 254")
	}
	if c.Funnel.HeloRepeatInterval < 2 {
		return nil, nil, errors.New("Configuration Error: funnel:heloRepeatInterval: must be between at least 1")
	}
	if c.Funnel.HeloTimeout < 20 {
		return nil, nil, errors.New("Configuration Error: funnel:timeoutHelo: must be between at least 20")
	}
	if c.Funnel.HeloRoundtripTime < 1 {
		return nil, nil, errors.New("Configuration Error: funnel:roundtripHelo: must be between at least 1")
	}
	if c.Funnel.MaxControlTimeDifferenceSeconds < 10 {
		return nil, nil, errors.New("Configuration Error: funnel:maxControlTimeDifference: must be between at least 10")
	}
	//heloBetweenDataPackets is optional, it is not checked here
	if c.Funnel.MaxWaitForValidSystemTimeSeconds < 1 {
		return nil, nil, errors.New("Configuration Error: funnel:maxWaitForValidSystemTimeSeconds: must be between at least 1")
	}
	if c.Funnel.PortBufferSize < 10*c.Funnel.Mtu {
		return nil, nil, errors.New("Configuration Error: funnel:portBufferSize: must be between at least 10*funnel:mtu:")
	}
	if len(c.Flows) < 1 {
		return nil, nil, errors.New("Configuration Error: flows: Define at least one flow")
	}
	for id, flow := range c.Flows {
		if id < 1 || id > 255 {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d: Must be between 1 and 255", id))
		}
		if flow.InProtocol != "udp4" && flow.InProtocol != "udp6" {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:inProtocol: Must be \"udp4\" or \"udp6\"", id))
		}
		if flow.OutProtocol != "udp4" && flow.OutProtocol != "udp6" {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:outProtocol: Must be \"udp4\" or \"udp6\"", id))
		}
		if flow.InPort < 0 || flow.InPort > 65535 {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:inPort: Must be between 1 and 65535", id))
		}
		if flow.OutPort < 0 || flow.OutPort > 65535 {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:outPort: Must be between 1 and 65535", id))
		}
		if flow.TimeoutSeconds < 1 {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:timeoutSeconds: Must be at least 1", id))
		}
		if flow.InitTimeoutSeconds < 1 {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:initTimeoutSeconds: Must be at least 1", id))
		}
	}
	return &c, cp, nil
}

var g_dummyIPV4a = [...]byte{1, 2, 3, 4}
var g_dummyIPV4b = [...]byte{4, 2, 3, 4}
var g_dummyIPV6a = [...]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}
var g_dummyIPV6b = [...]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 2, 3, 4}

// Dummy implementation, we need it for testing
func GetLocalAddrPort(id uint8) (*netip.AddrPort, error) {
	var ap netip.AddrPort

	ap = netip.AddrPortFrom(netip.AddrFrom4(g_dummyIPV4b), uint16(id)*5)
	return &ap, nil
}

func GetLocalPortHighLow() (high int, low int) {
	return 3000, 2000
}
