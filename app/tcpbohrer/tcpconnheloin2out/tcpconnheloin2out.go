package tcpconnheloin2out

import (
	"controlpacker"
	"fmt"
	"net"
	"os"
	"ratelimitedlogging"
	"tcpdial"
	"tcpforwarddata"
	"tcpparameters"
	"time"
	"writehelo"
)

func RunInside(par *tcpparameters.TcpbohrerConfig, cp *controlpacker.ControlPacker, log *ratelimitedlogging.RateLimitedLogger) {
	conn, _ := tcpdial.DialToOutsideOrLocal(0, par, log, false)
	if conn == nil {
		log.Log("Inside: Could not connect to Outside, exiting!")
		os.Exit(-1)
	}
	go goSendHelo(par, cp, conn, log)
	timeout := int64(par.Funnel.ConnectTimeout)
	for {
		var e controlpacker.EncodedControlPacket
		n, err := tcpforwarddata.ReadFromTCPConnection(conn, e.Data[:14], timeout)
		e.Len = 14
		d, err2 := cp.DecodeControlPacket(&e, false, 0)
		// We only want to receive a valid HELO message
		if n != 14 || err != nil || err2 != nil || d.Flow != 0 {
			log.Log("Inside: Received incorrect control message or no message, exiting!")
			os.Exit(-1)
		}
		if d.Id == 0 {
			log.Log("Inside: Received HELO message from Outside")
			writehelo.WriteHeloFile(par.Funnel.HeloFile)
			timeout = int64(par.Funnel.HeloTimeout)
			continue
		}
		// Here the actual flows are processed, packet needs to be sent on connection to Outside
		e, _ = cp.EncodeControlPacket(&d, true, 0, true)
		go goConnectToOutside(par, d.Id, &e, log)
	}
}

func goSendHelo(par *tcpparameters.TcpbohrerConfig, cp *controlpacker.ControlPacker, conn *net.TCPConn, log *ratelimitedlogging.RateLimitedLogger) {
	for {
		conn.SetWriteBuffer(30)
		conn.SetWriteDeadline(time.Unix(time.Now().Unix()+int64(par.Funnel.ConnectTimeout), 0))
		conn.SetNoDelay(true)
		var d controlpacker.DecodedControlPacket
		d.Id = 0 // HELO
		d.Flow = 0
		d.OptExtension[3] = 1 // pad to len == 14
		e, _ := cp.EncodeControlPacket(&d, true, 0, false)
		n, err := conn.Write(e.Data[:14])
		log.Log("Inside: Sending HELO message to Outside")
		if err != nil || n != 14 {
			log.Log("Inside: Could not send H E L O to Outside, exiting!")
			os.Exit(-1)
		}
		time.Sleep(time.Duration(par.Funnel.HeloTimeout/3) * time.Second)
	}
}

func goConnectToOutside(par *tcpparameters.TcpbohrerConfig, id uint8, e *controlpacker.EncodedControlPacket,
	log *ratelimitedlogging.RateLimitedLogger) {
	// catch bad Id
	_, ok := par.Flows[int(id)]
	if !ok {
		return
	}
	connLocal, err := tcpdial.DialToOutsideOrLocal(id, par, log, false)
	if connLocal == nil || err != nil {
		log.Log(fmt.Sprintf("Inside: Could not connect to local server for Id %d", id))
		return
	}
	log.Log(fmt.Sprintf("Inside: Id %d, connected to Inside server on port %d", int(id), par.Flows[int(id)].InPort))
	connOutside, err := tcpdial.DialToOutsideOrLocal(0, par, log, false)
	if connOutside == nil || err != nil {
		log.Log(fmt.Sprintf("Inside: Could not connect to outside for Id %d", id))
		return
	}
	connOutside.SetWriteDeadline(time.Unix(time.Now().Unix()+int64(par.Funnel.ConnectTimeout), 0))
	connOutside.Write(e.Data[:14])
	var tcptimeout tcpforwarddata.TCPTimeout
	go tcpforwarddata.GoHalfTransferBetweenTcpConnections(connLocal, connOutside, nil,
		int64(par.Flows[int(id)].SecondsTimeout), par.Funnel.PortBufferSize,
		"Inside: Local->Outside", &tcptimeout)
	go tcpforwarddata.GoHalfTransferBetweenTcpConnections(connOutside, connLocal, nil,
		int64(par.Flows[int(id)].SecondsTimeout), par.Funnel.PortBufferSize,
		"Inside: Outside->Local", &tcptimeout)
}
