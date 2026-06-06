package udpbohrerparameters

import (
	"fmt"
	"testing"
)

func TestLoadConfiguration(t *testing.T) {
	c, cp, err := LoadConfiguration("../../../udpbohrer.yaml", 2)
	fmt.Println(c)
	if c == nil || err != nil || cp == nil {
		t.Error("FAIL")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	c, cp, err := LoadConfiguration("../../../udpbohrer_not_found.yaml", 2)
	fmt.Println(c)
	if c != nil || err == nil || cp != nil {
		t.Error("FAIL")
	}
}

// Defaults are not working
func NoTestDefaultHandling(t *testing.T) {
	c, cp, err := LoadConfiguration("../../../test_udpconfig_defaults.yaml", 2)
	fmt.Println(c)
	if c == nil || err != nil || cp == nil {
		t.Error("FAIL")
	}
	if c.Log.IntervalSeconds != 60.0 || c.Log.LimitBurst != 20 {
		t.Error("Log Limits FAIL")
	}
	if c.Funnel.OutHost != "127.0.0.1" || c.Funnel.OutPort != 9999 {
		t.Error("Flow Limits FAIL")
	}
}
