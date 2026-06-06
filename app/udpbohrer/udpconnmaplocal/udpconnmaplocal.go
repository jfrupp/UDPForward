package udpconnmaplocal

import (
	"crypto/rand"
	"errors"
	"fmt"
	"ipaddresshelper"
	"math/big"
	"net"
	"net/netip"
	"ratelimitedlogging"
	"sync"
	"time"
	"udpmessage"
	"udpbohrerparameters"
)

type UDPConnToLocal struct {
	TimePacketToOutside int64
	TimePacketToLocal   int64
	RunningGoRoutines   int
	RequestTermination  bool
	ChToLocal           chan *udpmessage.UDPMessage
	LocalUDPSocket      *net.UDPConn
}

type FlowTimeouts struct {
	Single        int64
	Bidirectional int64
}

type ManageLocalChannels struct {
	mu              sync.Mutex
	mapLocalChannel map[uint16]*UDPConnToLocal
	mapTimemouts    map[uint8]FlowTimeouts
}

// Key as single uint16
// Do not lock in this function
func (c *ManageLocalChannels) getMapKey(id uint8, flow uint8) uint16 {
	if c.mapLocalChannel == nil {
		c.mapLocalChannel = make(map[uint16]*UDPConnToLocal)
	}
	return uint16(id)<<8 + uint16(flow)
}

func getIdFromMapKey(flowid uint16) uint8 {
	return uint8(flowid >> 8)
}

func (c *ManageLocalChannels) SetTimePacketToOutside(id uint8, flow uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.getMapKey(id, flow)
	m, ok := c.mapLocalChannel[k]
	if !ok {
		return
	}
	m.TimePacketToOutside = time.Now().Unix()
}

func (c *ManageLocalChannels) SetTimePacketToLocal(id uint8, flow uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.getMapKey(id, flow)
	m, ok := c.mapLocalChannel[k]
	if !ok {
		return
	}
	m.TimePacketToLocal = time.Now().Unix()
}

func (c *ManageLocalChannels) IncrRunningGoRoutines(par *GoRoutineParametersLocal) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.getMapKey(par.Id, par.Flow)
	m, ok := c.mapLocalChannel[k]
	if !ok {
		return
	}
	m.RunningGoRoutines++
}

func (c *ManageLocalChannels) DecrRunningGoRoutines(par *GoRoutineParametersLocal) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.getMapKey(par.Id, par.Flow)
	m, ok := c.mapLocalChannel[k]
	if !ok {
		return
	}
	m.RunningGoRoutines--
}

func (c *ManageLocalChannels) checkRunningGoRoutines(id uint8, flow uint8) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.getMapKey(id, flow)
	m, ok := c.mapLocalChannel[k]
	if !ok {
		return false
	}
	return m.RunningGoRoutines > 0
}

// By default terminate, if local channel is not found
func (c *ManageLocalChannels) CheckRequestTermination(par *GoRoutineParametersLocal) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.getMapKey(par.Id, par.Flow)
	m, ok := c.mapLocalChannel[k]
	if !ok {
		return true
	}
	return m.RequestTermination
}

func (c *ManageLocalChannels) getLocalChannel(id uint8, flow uint8) chan *udpmessage.UDPMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.getMapKey(id, flow)
	m, ok := c.mapLocalChannel[k]
	if !ok {
		return nil
	}
	if m.RequestTermination {
		return nil
	}
	return m.ChToLocal
}

func (c *ManageLocalChannels) CheckIdFlow(id uint8, flow uint8) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.getMapKey(id, flow)
	_, ok := c.mapLocalChannel[k]
	return ok
}

// do not lock here!
func (c *ManageLocalChannels) getFlowTimeouts(id uint8) (singleDirection int64,
	bidirection int64) {

	if c.mapTimemouts == nil {
		c.mapTimemouts = make(map[uint8]FlowTimeouts)
	}
	ft, ok := c.mapTimemouts[id]
	if ok {
		return ft.Single, ft.Bidirectional
	}
	return 0, 0
}

func (c *ManageLocalChannels) setFlowTimeouts(id uint8,
	single int64, bidirectional int64) {

	//do not lock here, this is only called when a new local channel is allocated, so no concurrency possible
	if c.mapTimemouts == nil {
		c.mapTimemouts = make(map[uint8]FlowTimeouts)
	}
	var ft FlowTimeouts
	ft.Single = single
	ft.Bidirectional = bidirectional
	c.mapTimemouts[id] = ft
}

// To be periodically called
// Two stage cleanup:
// First remove those with termination requested and no running Go routines
// Then check timeouts of those which have not communicated in both directions yet, request termination
// Finally check the timeout for two way communications, request termintaion
// Do not call functions which look as well!
func (c *ManageLocalChannels) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	unixtime := time.Now().Unix()
	for flowid, l := range c.mapLocalChannel {
		single, bidirectional := c.getFlowTimeouts(getIdFromMapKey(flowid))
		if l.RequestTermination && l.RunningGoRoutines == 0 {
			l.LocalUDPSocket.Close()
			close(l.ChToLocal)
			delete(c.mapLocalChannel, flowid)
			continue
		}
		if l.TimePacketToLocal == 0 && unixtime-l.TimePacketToOutside > single {
			l.RequestTermination = true
			continue
		}
		if l.TimePacketToOutside == 0 && unixtime-l.TimePacketToLocal > single {
			l.RequestTermination = true
			continue
		}
		if l.TimePacketToOutside != 0 && l.TimePacketToLocal != 0 &&
			unixtime-l.TimePacketToOutside > bidirectional && unixtime-l.TimePacketToLocal > bidirectional {
			l.RequestTermination = true
			continue
		}
	}
}

type GoRoutineParametersLocal struct {
	Id             uint8
	Flow           uint8
	Chlocal        chan *udpmessage.UDPMessage
	Chout          chan *udpmessage.UDPMessage
	TargetAddr     netip.AddrPort
	LocalUDPSocket *net.UDPConn
	ManageLocal    *ManageLocalChannels
	PortBufferSize int
	SourcePort     int
	Logger         *ratelimitedlogging.RateLimitedLogger
}

func (c *ManageLocalChannels) GetLocalChannel(par *udpbohrerparameters.UdpproxyConfig, id uint8, flow uint8,
	logger *ratelimitedlogging.RateLimitedLogger,
	chout chan *udpmessage.UDPMessage) (chan *udpmessage.UDPMessage,
	*GoRoutineParametersLocal, error) {

	// c.getLocalChannel locks, so no need for locking here
	ch := c.getLocalChannel(id, flow)
	if ch != nil {
		return ch, nil, nil
	}
	// here we need to create the channel, allocate ports, start the Go routines
	// we also need to check availability of flows and connection limit
	var conn *net.UDPConn
	erru := errors.New("")
	var bConnValid = false
	f, ok1 := par.Flows[int(id)]
	if ok1 == false {
		logger.Log(fmt.Sprintf("Inside: Id %d does not exist!", id))
		return nil, nil, errors.New("Id does not exist!")
	}
	sAddressTarget := f.InHost
	portTarget := f.InPort
	protTarget := f.InProtocol
	ipv46 := 4
	if protTarget == "udp6" {
		ipv46 = 6
	}
	addressTarget, err := ipaddresshelper.CompileAddr(ipv46, sAddressTarget)
	porthigh := par.Funnel.PoolMaxInPortToHost
	portlow := par.Funnel.PoolMinInPortToHost
	if err != nil || porthigh == 0 || portlow == 0 || porthigh <= portlow {
		return nil, nil, errors.New("Inside: Local address or port range is wrong!")
	}
	targetAddr := netip.AddrPortFrom(addressTarget, uint16(portTarget))

	var sourceport int
	for i := 0; i < 32; i++ {
		max := big.NewInt(int64(porthigh - portlow))
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			continue
		}
		sourceport = int(n.Int64()) + portlow
		var sockaddr net.UDPAddr
		sockaddr.Port = int(sourceport)
		if targetAddr.Addr().Is6() {
			sockaddr.IP = net.IPv6zero
			conn, erru = net.ListenUDP("udp6", &sockaddr)
		} else {
			sockaddr.IP = net.IPv4zero
			conn, erru = net.ListenUDP("udp4", &sockaddr)
		}
		if erru == nil {
			bConnValid = true
			break
		}
	}
	if bConnValid {
		var m UDPConnToLocal
		m.LocalUDPSocket = conn
		ch = make(chan *udpmessage.UDPMessage, 128)
		m.ChToLocal = ch
		m.RequestTermination = false
		m.RunningGoRoutines = 0
		m.TimePacketToLocal = time.Now().Unix()
		m.TimePacketToOutside = 0
		m.LocalUDPSocket = conn

		// Now change the map
		c.mu.Lock()
		defer c.mu.Unlock()

		k := c.getMapKey(id, flow)
		c.setFlowTimeouts(id, int64(par.Flows[int(id)].InitTimeoutSeconds), int64(par.Flows[int(id)].TimeoutSeconds))
		var goparlocal GoRoutineParametersLocal
		goparlocal.Id = id
		goparlocal.Flow = flow
		goparlocal.Chlocal = ch
		goparlocal.Chout = chout
		goparlocal.SourcePort = sourceport
		goparlocal.TargetAddr = targetAddr
		goparlocal.LocalUDPSocket = conn
		c.mapLocalChannel[k] = &m
		goparlocal.ManageLocal = c
		goparlocal.PortBufferSize = par.Funnel.PortBufferSize
		goparlocal.Logger = logger
		return ch, &goparlocal, nil
	}
	logger.Log(fmt.Sprintf("All local ports are occupied, no allocation is possible!"))
	return nil, nil, errors.New("No channel allocation possible!")
}
