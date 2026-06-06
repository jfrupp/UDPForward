package udpconnout2remote

import (
	"connlimiter"
	"net/netip"
	"testing"
)

func TestReturnFlow(t *testing.T) {
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl
	f, _ := fm.GetNewFlow()
	fm.ReturnFlow(f)
}

func TestMaxOutFlows(t *testing.T) {
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl
	for i := 0; i < 256; i++ {
		_, ok := fm.GetNewFlow()
		if !ok {
			t.Error("Could not get 256 flows")
		}
	}
	_, ok := fm.GetNewFlow()
	if ok {
		t.Error("Got more than 256 flows")
	}
}

func TestMaxOutLimited(t *testing.T) {
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl
	cl.SetConnLimit(100)
	for i := 0; i < 100; i++ {
		_, ok := fm.GetNewFlow()
		if !ok {
			t.Error("Could not get 100 flows")
		}
	}
	_, ok := fm.GetNewFlow()
	if ok {
		t.Error("Got more than 100 flows")
	}
}

func TestGetReturnFlow10000(t *testing.T) {
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl
	for i := 0; i < 21000; i++ {
		f, ok := fm.GetNewFlow()
		if !ok {
			t.Error("Could not get flow")
		}
		fm.ReturnFlow(f)
	}
}

func TestGetReturnCheckRoutine(t *testing.T) {
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl
	flows := make([]byte, 256)

	for j := 0; j < 1000; j++ {
		m := make(map[byte]bool)

		for i := 0; i < 256; i++ {
			f, ok := fm.GetNewFlow()
			flows[i] = f
			m[f] = true
			if !ok {
				t.Error("Could not get flow")
			}
		}
		for i := 0; i < 256; i++ {
			fm.ReturnFlow(flows[i])
		}
		for i := 0; i < 256; i++ {
			delete(m, byte(i))
		}
		if len(m) > 0 {
			t.Error("Map not empty")
		}
	}
}

func TestGetCountActive(t *testing.T) {
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl
	if fm.GetCountFlowsActive() != 0 {
		t.Error("Error1")
	}
	f, _ := fm.GetNewFlow()
	if fm.GetCountFlowsActive() != 1 {
		t.Error("Error2")
	}
	fm.ReturnFlow(f)
	if fm.GetCountFlowsActive() != 0 {
		t.Error("Error3")
	}
	f, _ = fm.GetNewFlow()
	if fm.GetCountFlowsActive() != 1 {
		t.Error("Error4")
	}
	for i := 1; i < 256; i++ {
		fm.GetNewFlow()
	}
	if fm.GetCountFlowsActive() != 256 {
		t.Error("Error5")
	}
	for i := 0; i < 256; i++ {
		fm.ReturnFlow(byte(i))
	}
	if fm.GetCountFlowsActive() != 0 {
		t.Error("Error6")
	}
}

var g_dummyIPV4a = [...]byte{1, 2, 3, 4}
var g_dummyIPV4b = [...]byte{4, 2, 3, 4}
var g_dummyIPV6a = [...]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}
var g_dummyIPV6b = [...]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 2, 3, 4}

func TestRetrieveAddrFail(t *testing.T) {
	var m MapAddrPortToFlow

	_, ok := m.GetAddressPortFromFlow(100)
	if ok {
		t.Error("Fail")
	}
}

func TestRetrieveAddrSuccess(t *testing.T) {
	var m MapAddrPortToFlow
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl
	m.Flows = &fm

	flow, ok1 := m.GetFlowFromAddressPort(netip.AddrPortFrom(
		netip.AddrFrom4(g_dummyIPV4a), 100))
	flow2, ok2 := m.GetFlowFromAddressPort(netip.AddrPortFrom(
		netip.AddrFrom4(g_dummyIPV4a), 100))
	flow3, ok3 := m.GetFlowFromAddressPort(netip.AddrPortFrom(
		netip.AddrFrom4(g_dummyIPV4b), 100))
	_, ok4 := m.GetAddressPortFromFlow(flow)
	if !ok1 || !ok2 || !ok3 || !ok4 || flow != flow2 || flow3 == flow {
		t.Error("Fail")
	}
	for i := 0; i < 255; i++ {
		f, ok := m.GetFlowFromAddressPort(netip.AddrPortFrom(
			netip.AddrFrom4(g_dummyIPV4a), 100))
		f2, ok2 := m.GetFlowFromAddressPort(netip.AddrPortFrom(
			netip.AddrFrom4(g_dummyIPV4a), 100))
		if !ok || !ok2 || f != flow || f2 != flow2 {
			t.Error("FAIL")
		}
	}
}

func TestAddrConnlimitation(t *testing.T) {
	var m MapAddrPortToFlow
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl
	m.Flows = &fm

	cl.SetConnLimit(10)
	count := 0
	for i := 0; i < 13; i++ {
		_, ok := m.GetFlowFromAddressPort(netip.AddrPortFrom(
			netip.AddrFrom16(g_dummyIPV6a), uint16(i)))
		if !ok {
			count++
		}
	}
	if count != 3 {
		t.Error("Fail")
	}
}

func _TestAddrReturnTimeoutSingle(t *testing.T) {
	var m MapAddrPortToFlow
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl // Connections are allocated in flow management
	m.Connlimit = &cl  // and released during clean up of the map
	m.Flows = &fm
	m.TimeOutSingleDirection = 10
	m.TimeOutBidirectional = 100

	cl.SetConnLimit(10)
	count := 0
	for i := 0; i < 13; i++ {
		_, ok := m.GetFlowFromAddressPort(netip.AddrPortFrom(
			netip.AddrFrom16(g_dummyIPV6a), uint16(i)))
		if ok {
			count++
		}
	}
	if count != 10 {
		t.Error("Fail")
	}

	m.CleanMapAddrPort()

	count = 0
	for i := 0; i < 13; i++ {
		_, ok := m.GetFlowFromAddressPort(netip.AddrPortFrom(
			netip.AddrFrom16(g_dummyIPV6b), uint16(i)))
		if ok {
			count++
		}
	}
	if count != 6 {
		t.Errorf("Fail")
	}
}

func _TestAddrReturnTime(t *testing.T) {
	var m MapAddrPortToFlow
	var fm FlowManagement
	var cl connlimiter.ConnLimiter
	fm.Connlimit = &cl // Connections are allocated in flow management
	m.Connlimit = &cl  // and released during clean up of the map
	m.Flows = &fm
	m.TimeOutSingleDirection = 10
	m.TimeOutBidirectional = 100

	cl.SetConnLimit(10)
	count := 0
	flows := make([]byte, 10)
	for i := 0; i < 13; i++ {
		f, ok := m.GetFlowFromAddressPort(netip.AddrPortFrom(
			netip.AddrFrom16(g_dummyIPV6a), uint16(i)))
		if ok {
			count++
			flows[i] = f
		}
	}
	if count != 10 {
		t.Error("Fail")
	}
	count = 0
	for i := 0; i < 6; i++ {
		_, ok := m.GetAddressPortFromFlow(flows[i])
		if ok {
			count++
		}
	}
	if count != 6 {
		t.Error("Fail")
	}
	m.CleanMapAddrPort()

	count = 0
	for i := 0; i < 13; i++ {
		_, ok := m.GetFlowFromAddressPort(netip.AddrPortFrom(
			netip.AddrFrom4(g_dummyIPV4b), uint16(i)))
		if ok {
			count++
		}
	}
	if count != 4 {
		t.Errorf("Fail")
	}
	m.CleanMapAddrPort()
	if m.Connlimit.GetConnCount() != 0 {
		t.Error("Fail")
	}
}
