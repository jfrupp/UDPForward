package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"tcpdial"
	"tcpparameters"
	"time"
)

type Counter struct {
	mu                      sync.Mutex
	GoRunning               int
	GoFailed                int
	ConnectionsInitiated    int
	ConnectionsAcknowledged int
	BytesSent               int
	BytesReceived           int
	TerminatorGoRoutines    bool
	LocalReflectorRunning   int
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

func (c *Counter) IncrConnectionsInitiated(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ConnectionsInitiated += incr
}

func (c *Counter) IncrConnectionsAcknowledged(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ConnectionsAcknowledged += incr
}

func (c *Counter) IncrBytesSent(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BytesSent += incr
}

func (c *Counter) IncrBytesReceived(incr int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BytesReceived += incr
}

func go_local_reflector(id int, par *tcpparameters.TcpbohrerConfig, count *Counter) {
	count.IncrGoLocalReflectorRunning(1)
	defer count.DecrGoLocalReflectorRunning(1)

	flow := par.Flows[id]
	prot := flow.InProtocol
	port := flow.InPort

	var tcpAddr net.TCPAddr
	if prot == "tcp6" {
		tcpAddr.IP = net.ParseIP("::1")
	} else {
		tcpAddr.IP = net.ParseIP("127.0.0.1")
	}
	tcpAddr.Port = port

	listener, err := net.ListenTCP(prot, &tcpAddr)
	if err != nil {
		panic(fmt.Sprintf("stresstcpbohrer: Id %d, could not bind to port %d/%s, exiting!", id, port, prot))
	}
	defer listener.Close()
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}
		go handleConnection(conn, count)
	}
}

func handleConnection(conn *net.TCPConn, count *Counter) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		n, _ := conn.Read(buf)
		n2, _ := conn.Write(buf[:n])
		if n != n2 || n <= 0 {
			return
		}
	}
}

func go_exchange(id int, runs int, par *tcpparameters.TcpbohrerConfig, count *Counter, size int) {
	count.IncrGoRunning(1)
	defer count.DecrGoRunning(1)

	for i := 0; i < runs; i++ {

		fmt.Printf("Id %d, Packet Size: %d, Run %d/%d\n", id, size, i, runs)
		conn, err := tcpdial.DialToOutsideOrLocal(uint8(id), par, nil, true)
		if err != nil || conn == nil {
			count.IncrGoFailed(1)
			continue
		}
		count.IncrConnectionsInitiated(1)

		data := make([]byte, size)
		for j := range data {
			data[j] = byte(j % 256)
		}
		conn.Write(data)

		count.IncrBytesSent(size)

		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		recvBuf := make([]byte, size)
		n, err := conn.Read(recvBuf)
		conn.Close()
		if err != nil || n != size {
			count.IncrGoFailed(1)
			continue
		}
		for j := 0; j < size; j++ {
			if recvBuf[j] != data[j] {
				count.IncrGoFailed(1)
				break
			}
		}
		count.IncrBytesReceived(size)
		count.IncrConnectionsAcknowledged(1)
	}
}

func SimpleTwoWayExchange(runs int, par *tcpparameters.TcpbohrerConfig, count *Counter) {
	for id := range par.Flows {
		go go_exchange(id, runs, par, count, 128)
	}
	time.Sleep(1 * time.Second)
	for count.IsGoRunning() {
		time.Sleep(1 * time.Second)
	}
}

func TwoWayExchanges(runs int, par *tcpparameters.TcpbohrerConfig, count *Counter, size int) {
	for id := range par.Flows {
		go go_exchange(id, runs, par, count, size)
	}
	time.Sleep(1 * time.Second)
	for count.IsGoRunning() {
		time.Sleep(1 * time.Second)
	}
}

func main() {
	fmt.Println("Stresstest TCPBohrer", time.Now())
	if len(os.Args) < 3 {
		fmt.Println("Usage: stresstcpbohrer <config.yaml> <mode> [runs]")
		os.Exit(1)
	}
	par, _, err := tcpparameters.LoadConfiguration(os.Args[1], 1)
	if err != nil {
		panic(fmt.Sprintf("Error loading config: %s", err))
	}
	fmt.Println(par)
	count := &Counter{}

	// Start local reflectors
	for id := range par.Flows {
		go go_local_reflector(id, par, count)
	}
	time.Sleep(1 * time.Second) // Wait for reflectors to start

	if os.Args[2] == "usual" {
		runs := 100
		if len(os.Args) > 3 {
			runs, _ = strconv.Atoi(os.Args[3])
		}
		SimpleTwoWayExchange(runs, par, count)
		TwoWayExchanges(runs, par, count, 10)
		TwoWayExchanges(runs, par, count, 0)
		TwoWayExchanges(runs, par, count, 1024)
	}
	fmt.Printf("Connections Initiated: %d\n", count.ConnectionsInitiated)
	fmt.Printf("Connections Acknowledged: %d\n", count.ConnectionsAcknowledged)
	fmt.Printf("Bytes Sent: %d\n", count.BytesSent)
	fmt.Printf("Bytes Received: %d\n", count.BytesReceived)
	fmt.Printf("Failed: %d\n", count.GoFailed)
}
