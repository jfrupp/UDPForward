package udpconnin2local

import (
	"fmt"
	"time"
	"udpconnmaplocal"
	"udpmessage"
)

func Go_send_to_local(par *udpconnmaplocal.GoRoutineParametersLocal) {
	par.ManageLocal.IncrRunningGoRoutines(par)
	defer par.ManageLocal.DecrRunningGoRoutines(par)
	par.LocalUDPSocket.SetWriteBuffer(par.PortBufferSize / 6)
	emptySlice := make([]byte, 0)

	par.Logger.Log(fmt.Sprintf("Inside: New Id/Flow %d/%d from port %d to local %s", int(par.Id), int(par.Flow), par.SourcePort, par.TargetAddr))
	for {
		if par.ManageLocal.CheckRequestTermination(par) {
			return
		}
		select {
		case m := <-par.Chlocal:
			par.LocalUDPSocket.WriteMsgUDPAddrPort(m.GetSliceWithoutIDFlow(),
				emptySlice, par.TargetAddr)
			//pay := *m.Payload
			m.IsToRecycle = true
			udpmessage.ReturnMsg(m)

		case <-time.After(1 * time.Second):
			continue
		}
	}
}

func Go_receive_from_local(par *udpconnmaplocal.GoRoutineParametersLocal, sizePayload int) {
	par.ManageLocal.IncrRunningGoRoutines(par)
	defer par.ManageLocal.DecrRunningGoRoutines(par)
	oob := make([]byte, 0)

	par.LocalUDPSocket.SetReadBuffer(par.PortBufferSize / 3)
	for {
		if par.ManageLocal.CheckRequestTermination(par) {
			return
		}
		ti := time.Now()
		ut := ti.Unix()
		// just keep the loop going every second
		par.LocalUDPSocket.SetReadDeadline(time.Unix(ut+1, 0))
		msg := udpmessage.NewUDPMessage(sizePayload)
		paylen, _, _, addr, errRec := par.LocalUDPSocket.ReadMsgUDPAddrPort(msg.GetSliceWithoutIDFlow(), oob)
		// Only accept packets from specified local source
		msg.LenPayload = paylen
		msg.IsControl = false
		if errRec != nil || addr != par.TargetAddr {
			msg.IsToRecycle = true
			udpmessage.ReturnMsg(msg)
			continue
		}
		msg.SetIdFlowIncrLen(par.Id, par.Flow)
		par.Chout <- msg
	}
}
