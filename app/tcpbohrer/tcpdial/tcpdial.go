package tcpdial

import (
	"errors"
	"fmt"
	"net"
	"ratelimitedlogging"
	"strings"
	"tcpparameters"
	"time"
)

func DialToOutsideOrLocal(id uint8, par *tcpparameters.TcpbohrerConfig,
	log *ratelimitedlogging.RateLimitedLogger, bConnectToOutPort bool) (*net.TCPConn, error) {
	err := errors.New("No Error")
	var tcpConn *net.TCPConn
	var ok bool

	for i := 0; i < 3; i++ {
		var network string
		var sAddr string

		var host string
		var protocol string
		var port int
		if id == 0 {
			host = par.Funnel.OutHost
			port = par.Funnel.OutPort
			protocol = par.Funnel.Protocol
		} else {
			flow, ok := par.Flows[int(id)]
			if !ok {
				return nil, errors.New("Id does not exist!")
			}

			if bConnectToOutPort {
				// only for debugging
				port = flow.OutPort
				host = flow.InHost
				protocol = flow.OutProtocol
			} else {
				port = flow.InPort
				host = flow.InHost
				protocol = flow.InProtocol
			}
		}
		if protocol == "tcp6" {
			if strings.Contains(host, ":") {
				sAddr = fmt.Sprintf("[%s]:%d", host, port)
			} else {
				sAddr = fmt.Sprintf("%s:%d", host, port)
			}
			network = "tcp6"
		} else {
			sAddr = fmt.Sprintf("%s:%d", host, port)
			network = "tcp4"
		}
		conn, err := net.DialTimeout(network, sAddr,
			time.Second*time.Duration(par.Funnel.ConnectTimeout))
		if err != nil || conn == nil {
			continue
		}
		tcpConn, ok = conn.(*net.TCPConn)
		if ok == true {
			break
		}
	}
	if tcpConn != nil {
		err = nil
		tcpConn.SetKeepAlive(false)
	}
	return tcpConn, err
}
