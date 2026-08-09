package tcpconnheloout2in

import (
	"connlimiter"
	"controlpacker"
	"manageportknock"
	"net"
	"os"
	"ratelimitedlogging"
	"tcpdispatchout"
	"tcpforwarddata"
	"tcpoutacceptor"
	"tcpparameters"
	"time"
	"writehelo"
)

func RunOutside(par *tcpparameters.TcpbohrerConfig, cp *controlpacker.ControlPacker,
	knock *manageportknock.ManagePortKnock, log *ratelimitedlogging.RateLimitedLogger) {

	ach := make(chan tcpoutacceptor.AcceptedConnection, 128)
	var cl connlimiter.ConnLimiter
	cl.SetConnLimit(uint32(par.Funnel.MaxCurrentConnections) * 2) // every TCP conection is counted twice

	for id, flow := range par.Flows {
		go tcpoutacceptor.GoAcceptOutside(uint8(id), flow.OutPort,
			flow.OutProtocol, ach, &cl, flow.PortKnockName, knock, log)
	}
	prot := par.Funnel.Protocol
	port := par.Funnel.OutPort
	var tAddr net.TCPAddr

	if prot == "tcp4" {
		tAddr.Port = port
	} else {
		prot = "tcp6"
		tAddr.Port = port
	}
	listener, err := net.ListenTCP(prot, &tAddr)
	if err != nil {
		log.Log("Outside: Could not listen on Funnel:OutPort")
		os.Exit(-1)
	}
	heloConn, err := listener.AcceptTCP()
	heloConn.SetReadBuffer(10 * 14)
	if heloConn == nil || err != nil {
		log.Log("Outside: Did not get connection from Inside within Funnel:HeloTimeout:")
		os.Exit(-1)
	}
	var bReceivedHelo bool
	chToInside := make(chan controlpacker.EncodedControlPacket, 20)
	heloTimeout := int64(par.Funnel.ConnectTimeout)
	for {
		if bReceivedHelo {
			heloTimeout = int64(par.Funnel.HeloTimeout)
		}
		var e controlpacker.EncodedControlPacket
		n, err := tcpforwarddata.ReadFromTCPConnection(heloConn, e.Data[:14], heloTimeout)
		e.Len = 14
		d, err2 := cp.DecodeControlPacket(&e, true, 0)
		// We only want to receive a valid HELO message
		if n != 14 || err != nil || err2 != nil || d.Id != 0 || d.Flow != 0 {
			log.Log("Outside: Received incorrect control message or no message, exiting!")
			os.Exit(-1)
		}
		// prepare and send the answer
		log.Log("Outside: Received HELO message from Inside, sending back answer")
		d.OptExtension[3] = 1 // pad to len == 14
		e2, _ := cp.EncodeControlPacket(&d, false, 0, true)
		chToInside <- e2
		writehelo.WriteHeloFile(par.Funnel.HeloFile)
		if !bReceivedHelo {
			// start the dispatcher & sending to Inside after reception of frist HELO
			tcpdispatchout.RunOutsideDispatcher(par, listener, chToInside, ach, &cl, log, cp)
			go goSendToInside(par, heloConn, chToInside, log)
			bReceivedHelo = true
		}
	}
}

func goSendToInside(par *tcpparameters.TcpbohrerConfig, heloConn *net.TCPConn, chToInside chan controlpacker.EncodedControlPacket, log *ratelimitedlogging.RateLimitedLogger) {
	for {
		e := <-chToInside
		if e.Len != 14 {
			continue
		}
		heloConn.SetWriteDeadline(time.Unix(time.Now().Unix()+int64(par.Funnel.ConnectTimeout), 0))
		n, err := heloConn.Write(e.Data[:14])
		if n != 14 || err != nil {
			log.Log("Outside: Cannot send control packet over control connection, exiting")
			os.Exit(-1)
		}
	}
}
