package connlimiter

import (
	"sync"
)

type ConnLimiter struct {
	mu        sync.Mutex
	connCount uint32
	connLimit uint32
}

func (cl *ConnLimiter) RequestConn() bool {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// no limit
	if cl.connLimit == 0 {
		return true
	}
	if cl.connCount+1 > cl.connLimit {
		return false
	}
	cl.connCount++
	return true
}

func (cl *ConnLimiter) ReleaseConn() {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if cl.connCount > 0 {
		cl.connCount--
	}
}

func (cl *ConnLimiter) SetConnLimit(connLimit uint32) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.connLimit = connLimit
}

func (cl *ConnLimiter) GetConnCount() uint32 {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	return cl.connCount
}
