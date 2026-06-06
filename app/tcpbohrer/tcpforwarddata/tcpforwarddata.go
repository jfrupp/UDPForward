package tcpforwarddata

import (
	"connlimiter"
	"io"
	"net"
	"sync"
	"time"
)

type TCPTimeout struct {
	mu               sync.Mutex
	Timeout          int64
	lastTransmission int64
}

func (t *TCPTimeout) DataReceived() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastTransmission = time.Now().Unix()
}

func (t *TCPTimeout) IsTimeout() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.lastTransmission+t.Timeout < time.Now().Unix() {
		return true
	} else {
		return false
	}
}

func clReleaseConn(cl *connlimiter.ConnLimiter) {
	if cl != nil {
		cl.ReleaseConn()
	}
}

// Out of Band data is not  handled!
func GoHalfTransferBetweenTcpConnections(cread *net.TCPConn, cwrite *net.TCPConn,
	cl *connlimiter.ConnLimiter, timeOut int64, sizeBuffer int, desc string, tcptimeout *TCPTimeout) {

	if cl != nil {
		defer cl.ReleaseConn()
	}
	if tcptimeout != nil {
		tcptimeout.Timeout = timeOut
		tcptimeout.DataReceived()
	}
	defer cread.CloseRead()
	defer cwrite.CloseWrite()
	cread.SetNoDelay(true)
	cwrite.SetNoDelay(true)
	cread.SetReadBuffer(sizeBuffer)
	cwrite.SetWriteBuffer(sizeBuffer)
	buf := make([]byte, 1600) // one Ethernet frame plus some bytes
	var err error
	var err3 error
	var n int
	var n2 int
	var n3 int

	for {
		cread.SetReadDeadline(time.Unix(time.Now().Unix()+int64(timeOut), 0))
		n, err = cread.Read(buf[0:1])
		n2 = 0
		n3 = 0
		err3 = nil
		if n > 0 {
			cread.SetReadDeadline(time.Time{})
			n2, _ = cread.Read(buf[n:])
			if n2 < 0 {
				n2 = 0
			}
		}
		if n+n2 > 0 {
			cwrite.SetWriteDeadline(time.Unix(time.Now().Unix()+int64(timeOut), 0))
			n3, err3 = cwrite.Write(buf[:n+n2])
		} else {
			// no data has been read, so check timeout
			if tcptimeout != nil {
				if tcptimeout.IsTimeout() {
					return
				}
			}
		}
		if err != nil || err3 != nil || n3 != n+n2 {
			return
		}
		// if data has been sent, retrigger timeout
		if tcptimeout != nil && n3 > 0 {
			tcptimeout.DataReceived()
		}
	}
}

func ReadFromTCPConnection(c *net.TCPConn, d []byte, timeOut int64) (int, error) {

	var n int
	c.SetReadDeadline(time.Unix(time.Now().Unix(), 0))
	n, err := c.Read(d)
	if n >= len(d) {
		return n, nil
	}
	if err == io.EOF {
		return n, err
	}
	utstart := time.Now().Unix()
	ut := utstart
	for ; ut < utstart+timeOut; ut = time.Now().Unix() {
		c.SetReadDeadline(time.Unix(ut, 1000000000))
		nr2, _ := c.Read(d[n:])

		n += nr2
		if n >= len(d) {
			return n, nil
		}
		if err == io.EOF {
			return n, err
		}
	}
	return n, err
}
