package main

import (
	"fmt"
	"ipaddresshelper"
	"net"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"
	"udpbohrerparameters"
)

type Counter struct {
	mu                             sync.Mutex
	GoRunning                      int
	GoFailed                       int
	PacketsInsideToOutsideSent     int
	PacketsInsideToOutsideReceived int
	PacketsOutsideToInsideSent     int
	PacketsOutsideToInsideReceived int
	BytesInsideToOutsideSent       int
	BytesInsideToOutsideReceived   int
	BytesOutsideToInsideSent       int
	BytesOutsideToInsideReceived   int
	FlowsInitiated                 int
	FlowsAcknowledged              int
	TerminatorGoRoutines           bool
	LocalReflectorRunning          int
}

func (c *Counter) IncrGoLocalReflectorRunning(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LocalReflectorRunning += incr
}

func (c *Counter) DecrGoLocalReflectorRunning(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LocalReflectorRunning -= incr
}

func (c *Counter) IsGoLocalReflectorRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.LocalReflectorRunning > 0
}

func (c *Counter) SetTerminatorGoRoutines(b bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.TerminatorGoRoutines = b
}

func (c *Counter) IsTerminatorGoRoutines() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.TerminatorGoRoutines
}

func (c *Counter) IncrGoRunning(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.GoRunning += incr
}

func (c *Counter) DecrGoRunning(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.GoRunning -= incr
}

func (c *Counter) IsGoRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.GoRunning > 0
}

func (c *Counter) IncrGoFailed(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.GoFailed += incr
}

func (c *Counter) IncrPacketsInsideToOutsideSent(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.PacketsInsideToOutsideSent += incr
}

func (c *Counter) IncrPacketsInsideToOutsideReceived(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.PacketsInsideToOutsideReceived += incr
}

func (c *Counter) IncrPacketsOutsideToInsideSent(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.PacketsOutsideToInsideSent += incr
}

func (c *Counter) IncrPacketsOutsideToInsideReceived(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.PacketsOutsideToInsideReceived += incr
}

func (c *Counter) IncrBytesInsideToOutsideSent(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.BytesInsideToOutsideSent += incr
}

func (c *Counter) IncrBytesInsideToOutsideReceived(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.BytesInsideToOutsideReceived += incr
}

func (c *Counter) IncrBytesOutsideToInsideSent(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.BytesOutsideToInsideSent += incr
}

func (c *Counter) IncrBytesOutsideToInsideReceived(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.BytesOutsideToInsideReceived += incr
}

func (c *Counter) IncrFlowsInitiated(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.FlowsInitiated += incr
}

func (c *Counter) IncrFlowsAcknowledged(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.FlowsAcknowledged += incr
}

func ClearConnRec(conn *net.UDPConn) {
	conn.SetReadDeadline(time.Unix(0, 0))
	b := make([]byte, 1024)
	fail := 0

	for i := 0; i < 10000; i++ {
		_, _, err := conn.ReadFromUDP(b)
		if err != nil {
			fail++
			if fail > 100 {
				break
			}
		}
	}
}

func go_local_reflector(id int, par *udpbohrerparameters.UdpproxyConfig, count *Counter) {
	count.IncrGoLocalReflectorRunning(1)
	defer count.DecrGoLocalReflectorRunning(1)
	protLocal := par.Flows[int(id)].InProtocol
	portLocal := par.Flows[int(id)].InPort

	var udpaddr net.UDPAddr
	udpaddr.Port = int(portLocal)
	conn, err := net.ListenUDP(protLocal, &udpaddr)
	if err != nil {
		panic(fmt.Sprintf("udpbohrer: Id %d, could not bind to port %d/%s, exiting!", id, portLocal, protLocal))
	}
	defer conn.Close()

	oob := make([]byte, 0)
	m := make([]byte, 1024*10)
	conn.SetReadBuffer(20 * 1024)
	conn.SetWriteBuffer(25 * 1024)
	ClearConnRec(conn)
	for {
		conn.SetReadDeadline(time.Unix(time.Now().Unix()+1, 0))
		paylen, _, _, addr, errRec := conn.ReadMsgUDPAddrPort(m, oob)

		if count.IsTerminatorGoRoutines() {
			return
		}
		if errRec != nil {
			continue
		}
		conn.WriteMsgUDPAddrPort(m[:paylen], oob, addr)
	}
}

func go_twowayexchange_with_local_reflector(id int, runs int, remotePort int, par *udpbohrerparameters.UdpproxyConfig, count *Counter, size int) {
	count.IncrGoRunning(1)
	defer count.DecrGoRunning(1)

	protOut := par.Flows[int(id)].OutProtocol
	portOut := par.Flows[int(id)].OutPort

	var udpaddrRemote net.UDPAddr
	udpaddrRemote.Port = remotePort
	connRemote, errOut := net.ListenUDP(protOut, &udpaddrRemote)
	if errOut != nil {
		panic(fmt.Sprintf("udpbohrer: Could not bind to remote port %s/%d, exiting!",
			par.Funnel.Protocol, udpaddrRemote.Port))
	}
	defer connRemote.Close()
	connRemote.SetReadBuffer(20 * 1024)
	connRemote.SetWriteBuffer(20 * 1024)
	var ip netip.Addr
	ip = netip.MustParseAddr("127.0.0.1")
	sendToAddr := netip.AddrPortFrom(ip, uint16(portOut))
	if protOut == "udp6" {
		ip = netip.MustParseAddr("::1")
		sendToAddr = netip.AddrPortFrom(ip, uint16(portOut))
	}
	var addrRec netip.AddrPort
	for i := 0; i < runs; i++ {
		if i%(runs/10) == 0 {
			fmt.Printf("Id %d, Packet Size (Mtu), Remote Port %d: %d, Run %d/%d\n", id, remotePort, size, i, runs)
		}
		ClearConnRec(connRemote)
		SendPacketWithoutIdFlow(uint8(id+remotePort), uint8(id*3+remotePort), i&0xFFFF, size, sendToAddr, connRemote)
		count.IncrPacketsOutsideToInsideSent(1)
		count.IncrBytesOutsideToInsideSent(128)
		bValidateAddrRec := true
		// learn on first run
		if i == 0 {
			bValidateAddrRec = false
		}
		bIsValid, bIsTimeout, addr := ReceiveAndValidatePacket(uint8(id*3+remotePort), uint8(id+remotePort), i&0xFFFF, size, 1,
			addrRec, bValidateAddrRec, connRemote, false)
		if i == 0 {
			addrRec = addr
		}
		if bIsValid {
			count.IncrPacketsInsideToOutsideReceived(1)
			count.IncrBytesInsideToOutsideReceived(128)
		} else if bIsTimeout {
			fmt.Printf("Id %d, Remote Port %d, Run %d: Timeout receiving packet by local side\n", id, remotePort, i)
		} else {
			fmt.Printf("Id %d, Remote Port %d, Run %d: Invalid packet received by local side\n", id, remotePort, i)
		}
	}
}

func go_twowayexchange(id int, runs int, remotePort int, par *udpbohrerparameters.UdpproxyConfig, count *Counter, size int) {
	count.IncrGoRunning(1)
	defer count.DecrGoRunning(1)
	protLocal := par.Flows[int(id)].InProtocol
	portLocal := par.Flows[int(id)].InPort
	protOut := par.Flows[int(id)].OutProtocol
	portOut := par.Flows[int(id)].OutPort

	var udpaddrLocal net.UDPAddr
	udpaddrLocal.Port = int(portLocal)
	connLocal, errLocal := net.ListenUDP(protLocal, &udpaddrLocal)
	if errLocal != nil {
		panic(fmt.Sprintf("udpbohrer: Could not bind to local port %s/%d, exiting!", par.Funnel.Protocol, udpaddrLocal.Port))
	}
	defer connLocal.Close()

	var udpaddrRemote net.UDPAddr
	udpaddrRemote.Port = remotePort
	connRemote, errOut := net.ListenUDP(protOut, &udpaddrRemote)
	if errOut != nil {
		panic(fmt.Sprintf("udpbohrer: Could not bind to remote port %s/%d, exiting!", par.Funnel.Protocol, udpaddrRemote.Port))
	}
	defer connRemote.Close()
	connLocal.SetReadBuffer(20 * 1024)
	connRemote.SetReadBuffer(20 * 1024)
	connLocal.SetWriteBuffer(20 * 1024)
	connRemote.SetWriteBuffer(20 * 1024)
	var ip netip.Addr
	ip = netip.MustParseAddr("127.0.0.1")
	sendToAddr := netip.AddrPortFrom(ip, uint16(portOut))
	if protOut == "udp6" {
		ip = netip.MustParseAddr("::1")
		sendToAddr = netip.AddrPortFrom(ip, uint16(portOut))
	}
	var addrRec netip.AddrPort
	for i := 0; i < runs; i++ {
		if i%(runs/10) == 0 {
			fmt.Printf("Id %d, Packet Size (Mtu): %d, Run %d/%d\n", id, size, i, runs)
		}
		ClearConnRec(connRemote)
		ClearConnRec(connLocal)
		SendPacketWithoutIdFlow(uint8(id), uint8(33), i&0xFFFF, size, sendToAddr, connRemote)
		count.IncrPacketsOutsideToInsideSent(1)
		count.IncrBytesOutsideToInsideSent(128)
		bValidateAddrRec := true
		// learn on first run
		if i == 0 {
			bValidateAddrRec = false
		}
		bIsValid, bIsTimeout, addr := ReceiveAndValidatePacket(uint8(id), uint8(33), i&0xFFFF, size, 1,
			addrRec, bValidateAddrRec, connLocal, false)
		if i == 0 {
			addrRec = addr
		}
		if bIsValid {
			count.IncrPacketsInsideToOutsideReceived(1)
			count.IncrBytesInsideToOutsideReceived(128)
		} else if bIsTimeout {
			fmt.Printf("Id %d, Run %d: Timeout receiving packet by local side\n", id, i)
		} else {
			fmt.Printf("Id %d, Run %d: Invalid packet received by local side\n", id, i)
		}
		ClearConnRec(connRemote)
		ClearConnRec(connLocal)
		SendPacketWithoutIdFlow(uint8(id), uint8(34), (i&0xFFFF)*3, size, addrRec, connLocal)
		count.IncrPacketsInsideToOutsideSent(1)
		count.IncrBytesInsideToOutsideSent(128)
		bIsValid, bIsTimeout, _ = ReceiveAndValidatePacket(uint8(id), uint8(34), (i&0xFFFF)*3, size, 1,
			sendToAddr, false, connRemote, false)
		if bIsValid {
			count.IncrPacketsOutsideToInsideReceived(1)
			count.IncrBytesOutsideToInsideReceived(128)
		} else if bIsTimeout {
			fmt.Printf("Id %d, Run %d: Timeout receiving packet by remote side\n", id, i)
		} else {
			fmt.Printf("Id %d, Run %d: Invalid packet received by remote side\n", id, i)
		}
	}
}

func SimpleTwoWayExchangeBetweenRemoteAndLocal(runs int, id int, par *udpbohrerparameters.UdpproxyConfig, count *Counter) {
	fmt.Println("Starting SimpleTwoWayExchangeBetweenRemoteAndLocal for id: ", id)
	go go_twowayexchange(id, runs, 40000+id, par, count, 128)
	for {
		time.Sleep(1 * time.Second)
		if !count.IsGoRunning() {
			fmt.Println(count)
			break
		}
	}
}

func TwoWayExchangesBetweenRemoteAndLocal(runs int, par *udpbohrerparameters.UdpproxyConfig, count *Counter, size int) {
	for id := range par.Flows {
		fmt.Println("Starting TwoWayExchangeBetweenRemoteAndLocal for id: ", id)
		go go_twowayexchange(id, runs, 40000+id, par, count, size)
	}
	for {
		time.Sleep(1 * time.Second)
		if !count.IsGoRunning() {
			fmt.Println(count)
			break
		}
	}
}

func TwoWayExchangeViaReflector(runs int, remotePort int, size int, par *udpbohrerparameters.UdpproxyConfig, count *Counter) {

	maxFlows := par.Funnel.MaxCurrentFlows
	fmt.Println("Starting TwoWayExchangeViaReflector for maxFlows: ", maxFlows)
	fmt.Println("Starting Local Reflectors")
	for id := range par.Flows {
		go go_local_reflector(id, par, count)
	}
	fmt.Println("Starting Flows with remote reflector")
	for flow := 0; flow < maxFlows; {
		for id := range par.Flows {
			time.Sleep(20 * time.Millisecond) // too much parallelism
			go go_twowayexchange_with_local_reflector(id, runs, remotePort, par, count, size)
			remotePort++
			flow++
			if flow >= maxFlows {
				break
			}
		}
	}
	fmt.Println("Terminating TwoWayExchangeViaReflector")
	for {
		time.Sleep(1 * time.Second)
		if !count.IsGoRunning() {
			fmt.Println(count)
			break
		}
	}
	count.SetTerminatorGoRoutines(true)
	// wait for all Go routines to terminate
	for {
		time.Sleep(1 * time.Second)
		if !count.IsGoLocalReflectorRunning() {
			fmt.Println("All local reflector Go routines terminated, exiting!")
			break
		}
	}
	count.SetTerminatorGoRoutines(false)
}

func buildMessage(id uint8, bIncludeId bool, flow uint8, bIncludeFlow bool, variant int, sizePayload int) (m *[]byte) {
	var accu uint64
	i := 0
	msg := make([]byte, sizePayload)

	if bIncludeId {
		msg[0] = id
		i++
	}
	if bIncludeFlow {
		msg[1] = flow
		i++
	}
	for j := 0; i < sizePayload; i++ {
		// Extra counter needed here to avoid miscalculation
		j++
		accu++
		accu *= uint64(id)*79517 + uint64(flow)*79999 + uint64(j)*31 + uint64(variant)*97
		accu &= 0xFFFFFFFF
		msg[i] = uint8(accu % 251)
	}
	return &msg
}

func SendPacket(id uint8, flow uint8, variant int, sizePayload int, addrPort netip.AddrPort, conn *net.UDPConn) {
	m := buildMessage(id, true, flow, true, variant, sizePayload)
	emptySlice := make([]byte, 0)
	conn.WriteMsgUDPAddrPort(*m, emptySlice, addrPort)
}

func SendPacketWithoutIdFlow(id uint8, flow uint8, variant int, sizePayload int, addrPort netip.AddrPort, conn *net.UDPConn) {
	m := buildMessage(id, false, flow, false, variant, sizePayload)
	emptySlice := make([]byte, 0)
	conn.WriteMsgUDPAddrPort(*m, emptySlice, addrPort)
}

func SendInvalidPacket(id uint8, bWithId bool, addrPort netip.AddrPort, conn *net.UDPConn) {
	var m []byte
	emptySlice := make([]byte, 0)
	if bWithId {
		m = make([]byte, 1)
		m[0] = id
	} else {
		m = make([]byte, 0)
	}
	conn.WriteMsgUDPAddrPort(m, emptySlice, addrPort)
}

func ReceiveAndValidatePacket(id uint8, flow uint8, variant int, sizePayload int, timeout int,
	sourceAddr netip.AddrPort, bValidateSourceAddr bool,
	conn *net.UDPConn, bWithIdFlow bool) (bIsValid bool, bIsTimeout bool, addrRec netip.AddrPort) {

	emptySlice := make([]byte, 0)
	mgen := buildMessage(id, bWithIdFlow, flow, bWithIdFlow, variant, sizePayload)
	mrec := make([]byte, sizePayload*2)
	ti := time.Now()
	ut := ti.Unix()
	// just keep the loop going every second
	conn.SetReadDeadline(time.Unix(ut+int64(timeout), 0))
	paylen, _, _, addrRec, errRec := conn.ReadMsgUDPAddrPort(mrec, emptySlice)
	if errRec != nil {
		return false, true, addrRec
	}
	// too much data
	if paylen != sizePayload {
		return false, false, addrRec
	}
	if addrRec != sourceAddr && bValidateSourceAddr {
		return false, false, addrRec
	}
	// slice back
	mrec = mrec[:sizePayload]
	return true, slices.Equal(*mgen, mrec), addrRec
}

func MixValidAndInvalidPacketsAndIDs(addrPort netip.AddrPort, injectPort int, injectProtocol string) {

	var udpaddrInject net.UDPAddr
	udpaddrInject.Port = int(injectPort)
	conn, err := net.ListenUDP(injectProtocol, &udpaddrInject)
	if err != nil {
		panic(fmt.Sprintf("udpbohrer: Could not bind to inject port %s/%d, exiting!", injectProtocol, injectPort))
	}
	defer conn.Close()
	for i := 0; i < 255; i++ {
		SendInvalidPacket(uint8(i), true, addrPort, conn)
		SendInvalidPacket(uint8(i), false, addrPort, conn)
		SendPacketWithoutIdFlow(uint8(i), uint8(i), 0, 1024, addrPort, conn)
		SendPacketWithoutIdFlow(uint8(i), uint8(i), 0, 4096, addrPort, conn)
		// Valid packet but a lot of ID/Flow combinations are invalid, so this is also a good test
		SendPacket(uint8(i), uint8(i), 0, 1024, addrPort, conn)
	}

}

func main() {
	fmt.Println("Stresstest", time.Now())
	par, _, _ := udpbohrerparameters.LoadConfiguration(os.Args[1], 2)
	fmt.Println(par)
	count := &Counter{}
	//TwoWayExchangeBetweenRemoteAndLocal(par, count)
	if os.Args[2] == "usual" {
		runs, _ := strconv.Atoi(os.Args[4])
		SimpleTwoWayExchangeBetweenRemoteAndLocal(runs, 255, par, count)
		TwoWayExchangesBetweenRemoteAndLocal(runs, par, count, 128)
		TwoWayExchangesBetweenRemoteAndLocal(runs, par, count, 10)
		TwoWayExchangesBetweenRemoteAndLocal(runs, par, count, 0)
		TwoWayExchangesBetweenRemoteAndLocal(runs, par, count, par.Funnel.Mtu)
	}
	if os.Args[2] == "reflector" {
		remotePort, _ := strconv.Atoi(os.Args[3])
		runs, _ := strconv.Atoi(os.Args[4])
		TwoWayExchangeViaReflector(runs, remotePort, 1400, par, count)
	}
	if os.Args[3] == "invalid" {
		injectPort, _ := strconv.Atoi(os.Args[3])
		ipv46 := 4
		if par.Funnel.Protocol == "udp6" {
			ipv46 = 6
		}
		addr, _ := ipaddresshelper.CompileAddr(ipv46, par.Funnel.OutHost)
		print("Run with invalid data to Outside")
		addrPort := netip.AddrPortFrom(addr, uint16(par.Funnel.OutPort))
		MixValidAndInvalidPacketsAndIDs(addrPort, injectPort, par.Funnel.Protocol)
		print("Run with invalid data to Inside")
		addrPort = netip.AddrPortFrom(addr, uint16(par.Funnel.InPort))
		MixValidAndInvalidPacketsAndIDs(addrPort, injectPort, par.Funnel.Protocol)
	}
}
