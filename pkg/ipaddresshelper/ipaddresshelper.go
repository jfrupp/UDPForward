package ipaddresshelper

import (
	"errors"
	"net"
	"net/netip"
)

var ErrorBadAddrFormat = errors.New("bad address format")
var ErrorNoHostname = errors.New("hostname does not match IP address")

func CompileAddr(ipv46 int, saddr string) (netip.Addr, error) {
	addr, err := netip.ParseAddr(saddr)
	// This is either an IPV4 or IPV6 address
	if err == nil {
		if ipv46 == 4 && !addr.Is4() && !addr.Is4In6() {
			return addr, ErrorBadAddrFormat
		}
		if ipv46 == 6 && !addr.Is6() {
			return addr, ErrorBadAddrFormat
		}
		return addr, nil
	}
	// here we check for resolved hostnames
	saddrs, err := net.LookupHost(saddr)
	if err != nil {
		return addr, err
	}
	if len(saddrs) == 0 {
		return addr, ErrorNoHostname
	}
	// parse all addresses checking for IPV4 and IPV6
	for _, saddr := range saddrs {
		addr, err = netip.ParseAddr(saddr)
		if err != nil {
			continue
		}
		// identified requested IPV4
		if ipv46 == 4 && (addr.Is4() || addr.Is4In6()) {
			return addr, nil
		}
		if ipv46 == 6 && addr.Is6() {
			return addr, nil
		}
	}
	return addr, ErrorBadAddrFormat
}
