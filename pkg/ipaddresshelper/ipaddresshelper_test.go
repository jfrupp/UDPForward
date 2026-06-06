package ipaddresshelper

import (
	"fmt"
	"testing"
)

func TestParsingIPV4(t *testing.T) {
	addr, err := CompileAddr(4, "1.2.3.4")
	fmt.Println(addr)
	if err != nil {
		t.Errorf("Address is not IPV4")
	}
}

func TestParsingIPV6(t *testing.T) {
	addr, err := CompileAddr(6, "2001:db8:85a3:0:0:8a2e:370:7334")
	fmt.Println(addr)
	if err != nil {
		t.Errorf("Address is not IPV6")
	}
}

func TestParsingHostname4(t *testing.T) {
	addr, err := CompileAddr(4, "localhost")
	fmt.Println(addr)
	if err != nil {
		t.Errorf("Hostname resolution failed")
	}
}

func TestParsingHostname6(t *testing.T) {
	addr, err := CompileAddr(6, "ip6-localhost")
	fmt.Println(addr)
	if err != nil {
		t.Errorf("Hostname resolution failed")
	}
}

func TestParsingIPV6AsIPV6AsIPV4(t *testing.T) {
	addr, err := CompileAddr(4, "2001:db8:85a3:0:0:8a2e:370:7334")
	fmt.Println(addr)
	if err == nil {
		t.Errorf("Mix up of IPV6 and IPV4")
	}
}

func TestParsingIPV4AsIPV6(t *testing.T) {
	addr, err := CompileAddr(6, "1.1.1.1")
	fmt.Println(addr)
	if err == nil {
		t.Errorf("Mix up of IPV6 and IPV4")
	}
}

func TestParsingInvalid(t *testing.T) {
	addr, err := CompileAddr(0, "256.1.1.1")
	fmt.Println(addr)
	if err == nil {
		t.Errorf("Invalid address parsed as valid")
	}
}
