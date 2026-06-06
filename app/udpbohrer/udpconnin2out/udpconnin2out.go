package udpconnin2out

import (
	"connlimiter"
	"controlpacker"
	"errors"
	"fmt"
	"ipaddresshelper"
	"net"
	"net/netip"
	"os"
	"ratelimitedlogging"
	"sync"
	"time"
	"udpbohrerparameters"
	"udpconnin2local"
	"udpconnmaplocal"
	"udpmessage"
	"writehelo"
)

type HeloRoundtrip struct {
	mu                sync.Mutex
	timeSent          int64
	bSent             bool
	expectedRoundtrip int64
}

var HeloRound HeloRoundtrip

func (h *HeloRoundtrip) SetExpectedRoundtrip(roundtrip int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.expectedRoundtrip = roundtrip
}

func (h *HeloRoundtrip) HeloSent() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.timeSent = time.Now().Unix()
	h.bSent = true
}

func (h *HeloRoundtrip) HeloReceived() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.bSent = false
}

func (h *HeloRoundtrip) IsToResend() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.bSent {
		return false
	}
	ut := time.Now().Unix()
	if (ut - h.timeSent) > h.expectedRoundtrip {
		return true
	}
	return false
}

var Cl connlimiter.ConnLimiter

func Go_in2out_send(chout chan *udpmessage.UDPMessage,
	conn *net.UDPConn, ipv46 int, saddr string, port uint16, cp *controlpacker.ControlPacker,
	logger *ratelimitedlogging.RateLimitedLogger, helo_repeat_interval float32, sizePayload int,
	portBufferSize int, local *udpconnmaplocal.ManageLocalChannels, roundtrip int64) {

	// var addrPortFromOut AddrPort
	// var bAddrPortFromOutValid bool = false
	var addrOut netip.Addr
	oob := make([]byte, 0)
	var heloEncoded controlpacker.EncodedControlPacket
	var unixTimeSinceLastHelo int64
	unixTimeSinceLastHelo = 0

	var heloDecoded controlpacker.DecodedControlPacket
	heloDecoded.Id = 0
	heloDecoded.Flow = 0

	HeloRound.SetExpectedRoundtrip(roundtrip)
	addrOut, err := ipaddresshelper.CompileAddr(ipv46, saddr)
	if err != nil {
		logger.Log(fmt.Sprintf("Inside: Host or address %s does not resolve or is not requested protocol \"udp%d\". Exiting!", saddr, ipv46))
		os.Exit(-1)
	}
	addrPortToOut := netip.AddrPortFrom(addrOut, port)
	// this sends udp messages
	// Go routine does not terminate
	for {
		ut := time.Now().Unix()
		// Is it time so send out a Helo message?
		conn.SetWriteBuffer(portBufferSize / 2)
		if (ut-unixTimeSinceLastHelo) >= int64(helo_repeat_interval) || HeloRound.IsToResend() {
			// Encoded with current Unix time
			heloEncoded, err = cp.EncodeControlPacket(&heloDecoded, true, ut, false)
			if err == nil {
				logger.Log("Inside: Sent HELO message to outside")
				conn.WriteMsgUDPAddrPort(heloEncoded.AsSlice(), oob, addrPortToOut)
				unixTimeSinceLastHelo = ut
				HeloRound.HeloSent()
			}
		}
		select {
		case msg := <-chout:
			// do not forward control messages from channel!
			if msg.IsControl {
				continue
			}
			if msg.Payload == nil {
				continue
			}
			pay := *msg.Payload
			if len(pay) < 2 {
				continue
			}
			if pay[0] == 0 {
				continue
			}
			local.SetTimePacketToOutside(pay[0], pay[1])
			// simply write out a valid data message, the outside system will decided about processing it
			// do only write the actual length of data
			conn.WriteMsgUDPAddrPort(pay[0:msg.LenPayload], oob, addrPortToOut)
			udpmessage.ReturnMsg(msg)
		// just keep the loop going every second
		case <-time.After(1 * time.Second):
			continue
		}
	}

}

func Go_in2out_receive(chout chan *udpmessage.UDPMessage,
	par *udpbohrerparameters.UdpproxyConfig, conn *net.UDPConn, ipv46 int, saddr string, port uint16,
	cp *controlpacker.ControlPacker, logger *ratelimitedlogging.RateLimitedLogger,
	helo_repeat_interval float32, sizePayload int,
	local *udpconnmaplocal.ManageLocalChannels, pHeloTimeout float32, heloFile string) {

	heloTimeOut := time.Now().Unix() + int64(float32(pHeloTimeout+2.3*helo_repeat_interval))
	cleanUpTimeout := int64(0)

	var addrFromOutside netip.AddrPort
	var bAddrFromOutsideValid bool = false

	oob := make([]byte, 0)

	for {
		ti := time.Now()
		ut := ti.Unix()
		// just keep the loop going every second
		conn.SetReadDeadline(time.Unix(ut+1, 0))
		msg := udpmessage.NewUDPMessage(sizePayload)
		conn.SetReadBuffer(par.Funnel.PortBufferSize)
		paylen, _, _, addr, errRec := conn.ReadMsgUDPAddrPort(*msg.Payload, oob)
		msg.LenPayload = paylen
		msg.AddrPort = addr
		ti = time.Now()
		ut = ti.Unix()
		// we have a message to process, must be control or a data message
		if errRec == nil && msg.LenPayload == 10 && msg.Payload != nil {
			// check out for helo message
			pay := *msg.Payload
			if pay[0] == 0 && pay[1] == 0 {
				e := controlpacker.NewEncodedControlPacket()
				e.Data = [16]byte(pay)
				e.Len = 10
				errDec := errors.New("decoding error")
				d, errDec := cp.DecodeControlPacket(e, false, ut)
				if errDec != nil {
					logger.Log(fmt.Sprintf("Inside: HELO decoding error %s for message from %s", errDec, msg.AddrPort))
					continue
				}
				// Received HELO packet is valid, extend timeout and store address
				if errDec == nil && d.Id == 0 && d.Flow == 0 {
					logger.Log(fmt.
						Sprint("Inside: Received a HELO message from ", msg.AddrPort))
					addrFromOutside = msg.AddrPort
					bAddrFromOutsideValid = true
					heloTimeOut = time.Now().Unix() +
						int64(pHeloTimeout+2*helo_repeat_interval)
					HeloRound.HeloReceived()
					writehelo.WriteHeloFile(heloFile)
					continue
				}
			}
		}
		if ut > heloTimeOut {
			logger.Log(fmt.Sprintf("Inside: Timeout waiting for HELO messages from the  Outside, Exiting!"))
			os.Exit(-1)
		}
		// Send data flow messages here
		if errRec == nil && msg.LenPayload >= 2 && msg.Payload != nil {
			pay := *msg.Payload
			if pay[0] != 0 {

				if msg.AddrPort != addrFromOutside || !bAddrFromOutsideValid {
					logger.Log(fmt.Sprintf("Inside: Received data packet from wrong Outside address %s", msg.AddrPort))
				} else {
					ch, gopar, errLocal := local.GetLocalChannel(par, pay[0], pay[1], logger, chout)
					if gopar != nil {
						go udpconnin2local.Go_send_to_local(gopar)
						go udpconnin2local.Go_receive_from_local(gopar, sizePayload)
					}

					if errLocal == nil {
						local.SetTimePacketToLocal(pay[0], pay[1])
						ch <- msg
					}
				}
			}
		}
		// clean up the local channel map every 5 seconds
		if ut > cleanUpTimeout+5 {
			cleanUpTimeout = ut + 5
			local.Cleanup()
		}
		// We do not return a message to the pool here - this is done by in2local
		// udpmessage.ReturnMsg(msg)
	}
}
