package udpconnout2remote

import (
	"connlimiter"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"ratelimitedlogging"
	"sync"
	"time"
	"udpmessage"
)

type GoParametersToRemote struct {
	mu             sync.Mutex
	gorunning      int
	Id             uint8
	Chout          chan *udpmessage.UDPMessage
	Chin           chan *udpmessage.UDPMessage
	MapAddr        *MapAddrPortToFlow
	LocalUDPSocket *net.UDPConn
	SizePayload    int
	Log            *ratelimitedlogging.RateLimitedLogger
	PortBufferSize int
}

func (par *GoParametersToRemote) IncrRunningGoRoutines() {
	par.mu.Lock()
	defer par.mu.Unlock()

	par.gorunning++
}

func (par *GoParametersToRemote) DecrCleanup() {
	par.mu.Lock()
	defer par.mu.Unlock()

	par.gorunning--
	if par.gorunning <= 0 {
		(*par.LocalUDPSocket).Close()
	}
}

func Go_send_to_remote(par *GoParametersToRemote) {
	par.IncrRunningGoRoutines()
	defer par.DecrCleanup()
	emptySlice := make([]byte, 0)
	par.LocalUDPSocket.SetWriteBuffer(par.PortBufferSize / 6)

	// This Go routine runs forever
	for {
		select {
		case m := <-par.Chin:
			if m.Payload != nil && m.LenPayload >= 2 {
				pay := *m.Payload
				targetAddr, ok := par.MapAddr.GetAddressPortFromFlow(pay[1])
				if ok {
					par.LocalUDPSocket.WriteMsgUDPAddrPort(m.GetSliceWithoutIDFlow(),
						emptySlice, targetAddr)
				}
			}
			m.IsToRecycle = true
			udpmessage.ReturnMsg(m)
		case <-time.After(time.Duration(1 * time.Second)):
			par.MapAddr.CleanMapAddrPort()
			continue
		}
	}
}

func Go_receive_from_remote(par *GoParametersToRemote) {
	par.IncrRunningGoRoutines()
	defer par.DecrCleanup()
	oob := make([]byte, 0)
	par.LocalUDPSocket.SetReadBuffer(par.PortBufferSize / 6)
	flowLog := -1
	var addrLog netip.AddrPort

	// This Go routine runs forever
	for {
		ti := time.Now()
		ut := ti.Unix()
		// just keep the loop going every second
		par.LocalUDPSocket.SetReadDeadline(time.Unix(ut+1, 0))
		msg := udpmessage.NewUDPMessage(par.SizePayload)
		msg.LenPayload = par.SizePayload
		paylen, _, _, addr, errRec := par.LocalUDPSocket.ReadMsgUDPAddrPort(msg.GetSliceWithoutIDFlow(), oob)
		if errRec != nil {
			msg.IsToRecycle = true
			udpmessage.ReturnMsg(msg)
			continue
		}
		(*msg.Payload)[0] = par.Id
		flow, bValid := par.MapAddr.GetFlowFromAddressPort(addr)
		if bValid == false {
			par.Log.Log("Outside: Could not allocate flow, packet is ignored!")
		}
		(*msg.Payload)[0] = par.Id
		(*msg.Payload)[1] = flow
		msg.LenPayload = paylen + 2
		msg.AddrPort = addr
		if bValid == false {
			msg.IsToRecycle = true
			udpmessage.ReturnMsg(msg)
			continue
		}
		if addrLog != addr || int(flow) != flowLog {
			par.Log.Log(fmt.Sprintf("Outside: New Id/Flow %d/%d from remote %s", int(par.Id), int(flow), addr))
		}
		addrLog = addr
		flowLog = int(flow)
		par.Chout <- msg
	}
}

type FlowManagement struct {
	mu             sync.Mutex
	flowsAvailable *([]byte)
	flowsReturned  map[byte]bool
	iFlowAvailable int
	nFlowsActive   int
	Connlimit      *connlimiter.ConnLimiter // must be set before first call to GetNewFlow
}

func (par *FlowManagement) GetNewFlow() (flow uint8, ok bool) {
	par.mu.Lock()
	defer par.mu.Unlock()

	if par.flowsReturned == nil {
		par.flowsReturned = make(map[byte]bool)
	}
	if par.flowsAvailable == nil {
		f := make([]byte, 256)
		par.flowsAvailable = &f
		for i := 0; i < 256; i++ {
			(*par.flowsAvailable)[i] = byte(i)
		}
		par.iFlowAvailable = 0
		rand.Shuffle(len(*par.flowsAvailable), func(i, j int) {
			(*par.flowsAvailable)[i], (*par.flowsAvailable)[j] = (*par.flowsAvailable)[j], (*par.flowsAvailable)[i]
		})
	}
	if par.iFlowAvailable >= len(*par.flowsAvailable) && len(par.flowsReturned) > 0 {
		fs := make([]byte, len(par.flowsReturned))
		par.flowsAvailable = &fs
		i := 0
		for k := range par.flowsReturned {
			fs[i] = k
			i++
		}
		par.flowsReturned = make(map[byte]bool)
		par.iFlowAvailable = 0
		if len(fs) > 2 {
			rand.Shuffle(len(fs), func(i, j int) {
				fs[i], fs[j] = fs[j], fs[i]
			})
		}
	}
	// no flows are available
	if par.iFlowAvailable >= len(*par.flowsAvailable) {
		return 0, false
	}
	if !par.Connlimit.RequestConn() {
		return 0, false
	}
	par.nFlowsActive++
	fs2 := *par.flowsAvailable
	f := fs2[par.iFlowAvailable]
	par.iFlowAvailable++
	return f, true
}

func (par *FlowManagement) ReturnFlow(flow uint8) {
	par.mu.Lock()
	defer par.mu.Unlock()

	if par.flowsReturned == nil {
		par.flowsReturned = make(map[byte]bool)
	}
	_, ok := par.flowsReturned[flow]
	if !ok {
		par.flowsReturned[flow] = true
		par.nFlowsActive--
	}
}

func (par *FlowManagement) GetCountFlowsActive() int {
	par.mu.Lock()
	defer par.mu.Unlock()

	return par.nFlowsActive
}

type AddrPortToFlow struct {
	addrPort     netip.AddrPort
	flow         byte
	timeToRemote int64
	timeToInside int64
}

type MapAddrPortToFlow struct {
	mu                     sync.Mutex
	mToI                   map[netip.AddrPort]*AddrPortToFlow // to inside
	mToR                   map[byte]*AddrPortToFlow           // to remote
	Connlimit              *connlimiter.ConnLimiter           // must be set before first call to GetNewFlow
	Flows                  *FlowManagement                    // must be set before first call to GetNewFlow
	TimeOutBidirectional   int64
	TimeOutSingleDirection int64
}

// Init the map. Do not lock here!
func (m *MapAddrPortToFlow) initMap() {
	if m.mToI == nil {
		m.mToI = make(map[netip.AddrPort]*AddrPortToFlow)
	}
	if m.mToR == nil {
		m.mToR = make(map[byte]*AddrPortToFlow)
	}
}

func (m *MapAddrPortToFlow) GetFlowFromAddressPort(ap netip.AddrPort) (flow byte, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initMap()

	mf, ok := m.mToI[ap]
	// flow does not exist, try to get a new one and init it
	if !ok {
		flow, ok2 := m.Flows.GetNewFlow()
		if !ok2 {
			return 0, false
		}
		var f AddrPortToFlow
		f.addrPort = ap
		f.flow = flow
		f.timeToRemote = 0
		mf = &f
		m.mToI[ap] = &f
		m.mToR[flow] = &f
	}
	mf.timeToInside = time.Now().Unix()
	return mf.flow, true
}

func (m *MapAddrPortToFlow) GetAddressPortFromFlow(flow byte) (addrPort netip.AddrPort, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initMap()

	ap, ok := m.mToR[flow]
	// found
	if ok {
		ap.timeToRemote = time.Now().Unix()
		return ap.addrPort, true
	}
	return addrPort, false
}

func (m *MapAddrPortToFlow) CleanMapAddrPort() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initMap()

	ut := time.Now().Unix()
	for addrP, mapFlow := range m.mToI {
		bDelete := false
		if mapFlow.timeToRemote == 0 && mapFlow.timeToInside+m.TimeOutSingleDirection < ut {
			bDelete = true
		}
		if mapFlow.timeToInside == 0 && mapFlow.timeToRemote+m.TimeOutSingleDirection < ut {
			bDelete = true
		}
		if mapFlow.timeToInside != 0 && mapFlow.timeToRemote != 0 &&
			mapFlow.timeToInside+m.TimeOutBidirectional < ut && mapFlow.timeToRemote+m.TimeOutBidirectional < ut {
			bDelete = true
		}
		if bDelete {
			m.Flows.ReturnFlow(mapFlow.flow)
			delete(m.mToR, mapFlow.flow)
			delete(m.mToI, addrP)
			m.Connlimit.ReleaseConn()
		}
	}
}
