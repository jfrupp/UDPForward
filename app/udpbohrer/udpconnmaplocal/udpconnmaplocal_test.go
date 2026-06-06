package udpconnmaplocal

import (
	"ratelimitedlogging"
	"testing"
	"udpbohrerparameters"
)

func TestXxx(t *testing.T) {

}
func _TestGetLocalChannel(t *testing.T) {
	var Ml ManageLocalChannels
	var L ratelimitedlogging.RateLimitedLogger

	par, _, _ := udpbohrerparameters.LoadConfiguration("../../../udpconfig.yaml", 2)
	ch1, _, err1 := Ml.GetLocalChannel(par, 255, 0, &L, nil)
	ch2, _, err2 := Ml.GetLocalChannel(par, 255, 0, &L, nil)
	ch3, _, err3 := Ml.GetLocalChannel(par, 101, 1, &L, nil)
	ch4, _, err4 := Ml.GetLocalChannel(par, 102, 1, &L, nil)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil ||
		ch1 != ch2 || ch2 == ch3 || ch3 == ch4 {
		t.Error("Fail")
	}
}

func _TestSetTimeLocalChannel(t *testing.T) {
	var Ml ManageLocalChannels
	var L ratelimitedlogging.RateLimitedLogger

	par, _, _ := udpbohrerparameters.LoadConfiguration("../../../udpconfig.yaml", 2)
	_, _, err1 := Ml.GetLocalChannel(par, 100, 3, &L, nil)

	if err1 != nil {
		t.Error("Fail")
	}
	Ml.SetTimePacketToLocal(1, 1)
	Ml.SetTimePacketToOutside(1, 2)
	// Invalid ID!
	Ml.SetTimePacketToLocal(30, 1)
	Ml.SetTimePacketToOutside(30, 1)
}

func _TestCleanupBidirectional(t *testing.T) {
	var Ml ManageLocalChannels
	var L ratelimitedlogging.RateLimitedLogger

	par, _, _ := udpbohrerparameters.LoadConfiguration("../../../udpconfig.yaml", 2)
	Ml.setFlowTimeouts(30, 10, 1000)
	for flow := 0; flow < 256; flow++ {
		Ml.GetLocalChannel(par, 30, uint8(flow), &L, nil)
		Ml.GetLocalChannel(par, 40, uint8(flow), &L, nil)
		Ml.SetTimePacketToLocal(30, uint8(flow))
		Ml.SetTimePacketToOutside(30, uint8(flow))
	}
	Ml.Cleanup()
	Ml.Cleanup()
	ok1 := Ml.CheckIdFlow(30, 10)
	ok2 := Ml.CheckIdFlow(30, 100)
	ok3 := Ml.CheckIdFlow(30, 101)
	ok4 := Ml.CheckIdFlow(40, 101)
	ok5 := Ml.CheckIdFlow(40, 255)
	if !ok1 && !ok2 && ok3 && !ok4 && !ok5 {
		t.Error("Fail")
	}
}

func _TestCleanupSingle(t *testing.T) {
	var Ml ManageLocalChannels
	var L ratelimitedlogging.RateLimitedLogger

	par, _, _ := udpbohrerparameters.LoadConfiguration("../../../udpconfig.yaml", 2)
	Ml.setFlowTimeouts(30, 1000, 1000)
	for flow := 0; flow < 256; flow++ {
		Ml.GetLocalChannel(par, 30, uint8(flow), &L, nil)
		Ml.GetLocalChannel(par, 40, uint8(flow), &L, nil)
		Ml.SetTimePacketToLocal(30, uint8(flow))
		Ml.SetTimePacketToOutside(30, uint8(flow))
	}
	Ml.Cleanup()
	Ml.Cleanup()
	ok1 := Ml.CheckIdFlow(30, 10)
	ok2 := Ml.CheckIdFlow(30, 100)
	ok3 := Ml.CheckIdFlow(30, 101)
	ok4 := Ml.CheckIdFlow(40, 101)
	ok5 := Ml.CheckIdFlow(40, 255)
	if !ok1 && !ok2 && ok3 && !ok4 && !ok5 {
		t.Error("Fail")
	}
}
