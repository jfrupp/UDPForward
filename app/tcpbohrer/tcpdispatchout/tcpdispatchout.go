package tcpdispatchout

import (
	"connlimiter"
	"controlpacker"
	"crypto/rand"
	"math/big"
	"net"
	"ratelimitedlogging"
	"sync"
	"tcpforwarddata"
	"tcpoutacceptor"
	"tcpparameters"
	"time"
)

type ConnectionsToAccept struct {
	mu             sync.Mutex
	C              map[[14]byte]tcpoutacceptor.AcceptedConnection
	TimeoutSeconds int64
	Cl             *connlimiter.ConnLimiter
}

// called from StoreConnection and GetConnection
func (a *ConnectionsToAccept) cleanup() {
	// no separate lock here

	ut := time.Now().Unix()
	for k, v := range a.C {
		val := v
		if val.TimeAccepted+a.TimeoutSeconds < ut {
			if v.Conn != nil {
				v.Conn.Close()
			}
			delete(a.C, k)
			a.Cl.ReleaseConn() // release the counter for two half connections
			a.Cl.ReleaseConn()

		}
	}
}

func (a *ConnectionsToAccept) StoreConnection(c tcpoutacceptor.AcceptedConnection, k [14]byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cleanup()
	a.C[k] = c
}

func (a *ConnectionsToAccept) GetConnection(k [14]byte) (c *tcpoutacceptor.AcceptedConnection) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cleanup()
	conn, ok := a.C[k]
	if !ok {
		return nil
	}
	delete(a.C, k)
	return &conn
}

func RunOutsideDispatcher(
	par *tcpparameters.TcpbohrerConfig, tcplistener *net.TCPListener, chToInside chan controlpacker.EncodedControlPacket,
	cha chan tcpoutacceptor.AcceptedConnection, cl *connlimiter.ConnLimiter,
	log *ratelimitedlogging.RateLimitedLogger, cp *controlpacker.ControlPacker) {

	var ca ConnectionsToAccept
	ca.C = make(map[[14]byte]tcpoutacceptor.AcceptedConnection)
	ca.TimeoutSeconds = par.Funnel.ConnectTimeout
	ca.Cl = cl

	go goManageAcceptedConnectionFromRemote(cha, chToInside, cl, log, &ca, cp)
	go goProcessAcceptedConnectionsFromInside(par, tcplistener, cl, log, &ca)
}

func goManageAcceptedConnectionFromRemote(cha chan tcpoutacceptor.AcceptedConnection,
	chToInside chan controlpacker.EncodedControlPacket,
	cl *connlimiter.ConnLimiter, log *ratelimitedlogging.RateLimitedLogger,
	ca *ConnectionsToAccept, cp *controlpacker.ControlPacker) {

	for {
		a := <-cha
		ut := time.Now().Unix()
		randInt32, _ := rand.Int(rand.Reader, big.NewInt(0xFFFFFFFF))
		var d controlpacker.DecodedControlPacket
		d.Id = a.Id
		d.Flow = 0
		d.OptExtension[0] = uint8(randInt32.Int64())
		d.OptExtension[1] = uint8(randInt32.Int64() >> 8)
		d.OptExtension[2] = uint8(randInt32.Int64() >> 16)
		d.OptExtension[3] = uint8(randInt32.Int64()>>24) | 0x1
		eToInside, _ := cp.EncodeControlPacket(&d, false, ut, false)
		eToOutside, _ := cp.EncodeControlPacket(&d, true, ut, false)
		var data [14]byte
		copy(data[:], eToOutside.Data[:14])
		ca.StoreConnection(a, data)
		// send out message after storage to avoir race condition
		chToInside <- eToInside
	}
}

func goProcessAcceptedConnectionsFromInside(par *tcpparameters.TcpbohrerConfig,
	tcplistener *net.TCPListener, cl *connlimiter.ConnLimiter,
	log *ratelimitedlogging.RateLimitedLogger, ca *ConnectionsToAccept) {
	for {
		conn, err := tcplistener.AcceptTCP()
		if err != nil {
			continue
		}
		go goProcessAcceptedConnection(par, tcplistener, conn, cl, log, ca)
	}
}

func goProcessAcceptedConnection(par *tcpparameters.TcpbohrerConfig,
	tcplistener *net.TCPListener, conn *net.TCPConn, cl *connlimiter.ConnLimiter,
	log *ratelimitedlogging.RateLimitedLogger, ca *ConnectionsToAccept) {

	conn.SetNoDelay(true)
	// get and validate the answer packet
	var packet [14]byte
	buf := make([]byte, 14)

	// size does not match, close connection and return
	n, err := tcpforwarddata.ReadFromTCPConnection(conn, buf[:14], par.Funnel.ConnectTimeout)
	if n != 14 || err != nil {
		conn.Close()
		return
	}
	// no valid connection, close connection and return
	copy(packet[:], buf[:])
	a := ca.GetConnection(packet)
	if a == nil {
		conn.Close()
		return
	}
	var tcptimeout tcpforwarddata.TCPTimeout
	go tcpforwarddata.GoHalfTransferBetweenTcpConnections(a.Conn, conn,
		cl, int64(par.Flows[int(a.Id)].SecondsTimeout), par.Funnel.PortBufferSize,
		"Outside: Remote->Outside", &tcptimeout)
	go tcpforwarddata.GoHalfTransferBetweenTcpConnections(conn, a.Conn,
		cl, int64(par.Flows[int(a.Id)].SecondsTimeout), par.Funnel.PortBufferSize,
		"Outside: Outside->Remote", &tcptimeout)

}
