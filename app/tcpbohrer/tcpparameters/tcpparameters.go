package tcpparameters

import (
	"controlpacker"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Default parameters are ignored by gopkg.in/yaml.v3!!!
type Flow struct {
	Name           string  `default:"Default Forwarder" yaml:"name"`
	InProtocol     string  `default:"tcp4"              yaml:"inProtocol"`
	OutProtocol    string  `default:"tcp4"              yaml:"outProtocol"`
	OutPort        int     `default:"0"                 yaml:"outPort"`
	InHost         string  `default:"127.0.0.1"         yaml:"inHost"`
	InPort         int     `default:"0"                 yaml:"inPort"`
	SecondsTimeout float32 `default:"130.3"             yaml:"secondsTimeout"`
}

type Header struct {
	Application                string `yaml:"application"`       // no default value, user is required to set this to "tcpbohrer"
	ConfigFileVersion          int    `yaml:"configFileVersion"` // no default value, user is required to provide version between 1 and 255
	MinApplicationMajorVersion int    `yaml:"minApplicationMajorVersion" default:"1"`
	Description                string `yaml:"description" default:"TcpbohrerConfig"`
}
type Log struct {
	IntervalSeconds         float32 `yaml:"intervalSeconds" default:"60.0"`
	LimitBurst              float32 `yaml:"limitBurst" default:"20"`
	LimitHeloLoggingSeconds int     `yaml:"limitHeloLoggingSeconds" default:"20"`
}

type Funnel struct {
	Protocol                         string  `yaml:"protocol" default:"tcp4"`
	OutHost                          string  `yaml:"outHost" default:"127.0.0.1"`
	OutPort                          int     `yaml:"outPort" default:"9999"`
	MaxCurrentConnections            int     `yaml:"maxCurrentConnections" default:"100"`
	HeloRepeatInterval               float32 `yaml:"heloRepeatInterval" default:"10.0"`
	HeloTimeout                      float32 `yaml:"heloTimeout" default:"10.0"`
	MaxControlTimeDifferenceSeconds  int     `yaml:"maxControlTimeDifferenceSeconds" default:"600"`
	MaxWaitForValidSystemTimeSeconds int     `yaml:"maxWaitForValidSystemTimeSeconds" default:"120"`
	PortBufferSize                   int     `yaml:"portBufferSize" default:"50000"`
	ConnectTimeout                   int64   `yaml:"connectTimeout" default:"4.5"`
	HeloFile                         string  `yaml:"heloFile" default:""`
}

type TcpbohrerConfig struct {
	Header Header       `yaml:"header"`
	Secret string       `yaml:"secret"`
	Log    Log          `yaml:"log"`
	Funnel Funnel       `yaml:"funnel"`
	Flows  map[int]Flow `yaml:"flows"`
}

func LoadConfiguration(ConfigFile string, ProgramVersionMajor uint8) (*TcpbohrerConfig, *controlpacker.ControlPacker, error) {
	f, err := os.Open(ConfigFile)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}
	var c TcpbohrerConfig
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, nil, err
	}
	// Check the parameters
	if c.Header.Application != "tcpbohrer" {
		return nil, nil, errors.New("Configuration Error: header:application: Has to be set to \"tcpbohrer\"")
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
	if c.Funnel.Protocol != "tcp4" && c.Funnel.Protocol != "tcp6" {
		return nil, nil, errors.New("Configuration Error: funnel:protocol: Protocol has to be \"tcp4\" or \"tcp6\"")
	}
	if len(c.Funnel.OutHost) < 1 {
		return nil, nil, errors.New("Configuration Error: funnel:outHost: Outside host has to be specified")
	}
	if c.Funnel.OutPort <= 0 || c.Funnel.OutPort > 65535 {
		return nil, nil, errors.New("Configuration Error: funnel:outPort: Must be between 0 and 65535")
	}
	if c.Funnel.MaxCurrentConnections < 2 || c.Funnel.MaxCurrentConnections > 254 {
		return nil, nil, errors.New("Configuration Error: funnel:poolmaxCurrentConnections: must be between 2 and 254")
	}
	if c.Funnel.HeloRepeatInterval < 2 {
		return nil, nil, errors.New("Configuration Error: funnel:heloRepeatInterval: must be between at least 1")
	}
	if c.Funnel.HeloTimeout < 4 {
		return nil, nil, errors.New("Configuration Error: funnel:timeoutHelo: must be between at least 4")
	}
	if c.Funnel.MaxControlTimeDifferenceSeconds < 10 {
		return nil, nil, errors.New("Configuration Error: funnel:maxControlTimeDifference: must be between at least 10")
	}
	//heloBetweenDataPackets is optional, it is not checked here
	if c.Funnel.MaxWaitForValidSystemTimeSeconds < 1 {
		return nil, nil, errors.New("Configuration Error: funnel:maxWaitForValidSystemTimeSeconds: must be between at least 1")
	}
	if c.Funnel.PortBufferSize < 1000 {
		return nil, nil, errors.New("Configuration Error: funnel:portBufferSize: must be at least 1000")
	}
	if c.Funnel.ConnectTimeout < 1 {
		return nil, nil, errors.New("Configuration Error: connectTimeout: Must be at least 1")
	}
	if len(c.Flows) < 1 {
		return nil, nil, errors.New("Configuration Error: flows: Define at least one flow")
	}
	for id, flow := range c.Flows {
		if id < 1 || id > 255 {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d: Must be between 1 and 255", id))
		}
		if flow.InProtocol != "tcp4" && flow.InProtocol != "tcp6" {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:inProtocol: Must be \"tcp4\" or \"tcp6\"", id))
		}
		if flow.OutProtocol != "tcp4" && flow.OutProtocol != "tcp6" {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:outProtocol: Must be \"tcp4\" or \"tcp6\"", id))
		}
		if flow.InPort < 0 || flow.InPort > 65535 {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:inPort: Must be between 1 and 65535", id))
		}
		if flow.OutPort < 0 || flow.OutPort > 65535 {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:outPort: Must be between 1 and 65535", id))
		}
		if flow.SecondsTimeout < 1 {
			return nil, nil, errors.New(fmt.Sprintf("Configuration Error: flows:%d:secondsTimeout: Must be at least 1", id))
		}
	}
	return &c, cp, nil
}
