package tcpoutacceptor

import (
	"connlimiter"
	"fmt"
	"net"
	"os"
	"ratelimitedlogging"
	"strconv"
	"time"
)

type AcceptedConnection struct {
	Id           uint8
	TimeAccepted int64
	Conn         *net.TCPConn
}

func GoAcceptOutside(id uint8, port int, prot string, cha chan AcceptedConnection,
	cl *connlimiter.ConnLimiter, log *ratelimitedlogging.RateLimitedLogger) {

	sAddr := ""
	var tAddr net.TCPAddr
	if prot == "tcp4" {
		sAddr = "127.0.0.1:" + strconv.FormatInt(int64(port), 10)
		tAddr.Port = port
	} else {
		sAddr = "[::1]:" + strconv.FormatInt(int64(port), 10)
		tAddr.Port = port
		prot = "tcp6"
	}
	listener, err := net.ListenTCP(prot, &tAddr)
	if err != nil {
		log.Log(fmt.Sprintf("Outside: Cannot listen to %s/%s", prot, sAddr))
		os.Exit(-1)
	}
	for {
		listener.SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}
		ok := cl.RequestConn()
		if !ok {
			conn.Close()
			continue
		}
		ok = cl.RequestConn()
		if !ok {
			cl.ReleaseConn()
			conn.Close()
			continue
		}
		var a AcceptedConnection
		a.Id = id
		a.TimeAccepted = time.Now().Unix()
		log.Log(fmt.Sprintf("Outside: Id %d, Accepted connection from %s", int(id),
			conn.RemoteAddr().String()))
		a.Id = id
		a.TimeAccepted = time.Now().Unix()
		a.Conn = conn
		cha <- a
		// Connections in cl are released by later stages
	}
}
