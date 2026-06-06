package udpconnout2in

import (
	"connlimiter"
	"controlpacker"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"ratelimitedlogging"
	"sync"
	"time"
	"udpmessage"
	"writehelo"
)

var Cl connlimiter.ConnLimiter

type ManageRemoteId struct {
	mu sync.Mutex
	// default vaue is nil
	aCh [256]chan *udpmessage.UDPMessage
}

func (m *ManageRemoteId) SetChForRemoteId(id uint8, ch chan *udpmessage.UDPMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.aCh[id] = ch
}

func (m *ManageRemoteId) GetChForRemoteId(id uint8) (ch chan *udpmessage.UDPMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.aCh[id]
}

type ManageAddressPort struct {
	mu         sync.Mutex
	addrPort   netip.AddrPort
	bAddrValid bool
}

func (a *ManageAddressPort) Setinvalid() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.bAddrValid = false
}

func (a *ManageAddressPort) SetAddress(addrPort netip.AddrPort) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.addrPort = addrPort
	a.bAddrValid = true
}

func (a *ManageAddressPort) GetAddress() (addrPort netip.AddrPort, bValid bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.addrPort, a.bAddrValid
}

func Go_out2in_send(chin chan *udpmessage.UDPMessage, conn *net.UDPConn, cp *controlpacker.ControlPacker,
	logger *ratelimitedlogging.RateLimitedLogger, addrPortIn *ManageAddressPort,
	sizePayload int, portBufferSize int) {

	oob := make([]byte, 0)
	var msg *udpmessage.UDPMessage

	conn.SetWriteBuffer(portBufferSize / 2)
	for {
		msg = <-chin
		if msg == nil || msg.Payload == nil {
			continue
		}
		pay := *msg.Payload
		if len(pay) < 2 {
			continue
		}
		if pay[0] == 0 {
			continue
		}
		addrPort, bValid := addrPortIn.GetAddress()
		if bValid {
			conn.WriteMsgUDPAddrPort(pay[0:msg.LenPayload], oob, addrPort)
		}
		msg.IsToRecycle = true
		udpmessage.ReturnMsg(msg)
	}
}

func Go_out2in_receive(chout chan *udpmessage.UDPMessage, conn *net.UDPConn, cp *controlpacker.ControlPacker,
	logger *ratelimitedlogging.RateLimitedLogger, addrPortIn *ManageAddressPort, helo_repeat_interval int64,
	r *ManageRemoteId, sizePayload int, portBufferSize int, pHeloTimeout float32, heloFile string) {

	heloTimeOut := time.Now().Unix() + int64(pHeloTimeout)
	oob := make([]byte, 0)
	for {
		conn.SetReadBuffer(portBufferSize)
		ut := time.Now().Unix()
		if ut > heloTimeOut {
			logger.Log("Outside: Timeout waiting for HELO messages from the Inside, Exiting!")
			os.Exit(-1)
		}
		// just keep the loop going every second
		conn.SetReadDeadline(time.Unix(ut+1, 0))
		msg := udpmessage.NewUDPMessage(sizePayload)
		pay := msg.Payload
		paylen, _, _, addr, errRec := conn.ReadMsgUDPAddrPort(*pay, oob)
		msg.LenPayload = paylen
		msg.AddrPort = addr
		ut = time.Now().Unix()
		if errRec != nil {
			msg.IsToRecycle = true
			udpmessage.ReturnMsg(msg)
			continue
		}

		// we have a message to process, must be control or a data message

		if msg.LenPayload == 10 && msg.Payload != nil {
			pay := *msg.Payload
			// Check for HELO message
			if pay[0] == 0 && pay[1] == 0 {
				e := controlpacker.NewEncodedControlPacket()
				e.Data = [16]byte(pay)
				e.Len = 10
				errDec := errors.New("decoding error")
				d, errDec := cp.DecodeControlPacket(e, true, ut)
				// Received HELO packet is valid, extend timeout and store address
				if errDec != nil {
					logger.Log(fmt.Sprintf("Outside: HELO decoding error %s for message from %s", errDec, msg.AddrPort))
					continue
				}
				if errDec == nil && d.Id == 0 && d.Flow == 0 {
					logger.Log(fmt.Sprintf("Outside: Received a HELO message from %s", msg.AddrPort))
					addrPortIn.SetAddress(addr)
					ut = time.Now().Unix()
					heloTimeOut = ut +
						int64(pHeloTimeout)
					e, errEnc := cp.EncodeControlPacket(&d, false, ut, false)
					addrPortToIn, bValid := addrPortIn.GetAddress()
					if errEnc == nil && bValid {
						conn.WriteMsgUDPAddrPort(e.AsSlice(), oob, addrPortToIn)
					}
					msg.IsToRecycle = true
					udpmessage.ReturnMsg(msg)
					writehelo.WriteHeloFile(heloFile)
					continue
				}
			}
		}
		// 	Only accept messages from addresses already validated by HELO from Inside
		addrPortToIn, bValid := addrPortIn.GetAddress()
		if !bValid || addrPortToIn != addr {
			logger.Log(fmt.Sprintf("Outside: Received packet without receiving HELO first from %s", msg.AddrPort))
			msg.IsToRecycle = true
			udpmessage.ReturnMsg(msg)
			continue
		}
		// This is a valid data message
		if msg.LenPayload >= 2 && msg.Payload != nil {
			pay := *msg.Payload
			ch := r.GetChForRemoteId(pay[0])
			if ch != nil && pay[0] != 0 {
				ch <- msg
			}
		}
	}
}
