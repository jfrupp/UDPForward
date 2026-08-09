package manageportknock

import (
	"iter"
	"net/netip"
	"sync"
	"time"
)

type ManagePortKnock struct {
	mu               sync.Mutex
	kmap             map[string]map[netip.Addr]int64
	countportknock   int32
	timeOut          int64
	maxportknock     int32
	tlsport          uint16
	portKnockKeyFile string
	portKnockKeyCert string
	isConfig         bool
}

func (knock *ManagePortKnock) cleanup() {
	if knock.kmap != nil {
		now := time.Now().Unix()
		for k1, v1 := range knock.kmap {
			for k2, v2 := range v1 {
				if v2 < now {
					delete(knock.kmap[k1], k2)
					knock.countportknock--
				}
			}
		}
	}
	if knock.countportknock < 0 {
		knock.countportknock = 0
	}
}

func (knock *ManagePortKnock) Check(knockName string, addr netip.Addr) bool {
	knock.mu.Lock()
	defer knock.mu.Unlock()

	if knock.kmap == nil {
		return false
	}
	v1, ok := knock.kmap[knockName]
	if !ok || v1 == nil {
		return false
	}
	v2, ok := v1[addr]
	if !ok {
		return false
	}
	now := time.Now().Unix()
	if v2 <= now {
		delete(knock.kmap[knockName], addr)
		knock.countportknock--
		if knock.countportknock < 0 {
			knock.countportknock = 0
		}
		return false
	}
	return true
}

func (knock *ManagePortKnock) CheckName(knockName string) bool {
	knock.mu.Lock()
	defer knock.mu.Unlock()

	if knock.kmap == nil {
		return false
	}
	_, ok := knock.kmap[knockName]
	if !ok {
		return false
	}
	return true
}

func (knock *ManagePortKnock) Iterator() iter.Seq[string] {
	return func(yield func(string) bool) {
		knock.mu.Lock()
		names := make([]string, 0, len(knock.kmap))
		for name := range knock.kmap {
			names = append(names, name)
		}
		knock.mu.Unlock()

		for _, name := range names {
			if !yield(name) {
				return
			}
		}
	}
}

func (knock *ManagePortKnock) InsertName(knockName string) {
	knock.mu.Lock()
	defer knock.mu.Unlock()

	if knock.kmap == nil {
		knock.kmap = make(map[string]map[netip.Addr]int64)
	}
	_, ok := knock.kmap[knockName]
	if !ok {
		knock.kmap[knockName] = nil
	}
}

func (knock *ManagePortKnock) Insert(knockName string, addr netip.Addr) {
	knock.mu.Lock()
	defer knock.mu.Unlock()
	now := time.Now().Unix()
	knock.cleanup()

	if knock.maxportknock-knock.countportknock <= 0 {
		return
	}
	if knock.kmap == nil {
		knock.kmap = make(map[string]map[netip.Addr]int64)
	}
	v1, ok := knock.kmap[knockName]
	if !ok || v1 == nil {
		knock.kmap[knockName] = make(map[netip.Addr]int64)
	}
	knock.kmap[knockName][addr] = now + knock.timeOut
	knock.countportknock++
}

func (knock *ManagePortKnock) Config(timeOut int64, maxportknock int32,
	tlsport uint16, portKnockKeyFile string, portKnockCert string) {
	if tlsport == 0 {
		return
	}
	knock.timeOut = timeOut
	knock.maxportknock = maxportknock
	knock.tlsport = tlsport
	knock.isConfig = true
	knock.portKnockKeyFile = portKnockKeyFile
	knock.portKnockKeyCert = portKnockCert
}

func (knock *ManagePortKnock) IsConfigAndContainsKnocks() bool {
	return knock.isConfig && knock.kmap != nil && len(knock.kmap) > 0
}

func (knock *ManagePortKnock) Timeout() int64 {
	knock.mu.Lock()
	defer knock.mu.Unlock()
	return knock.timeOut
}

func (knock *ManagePortKnock) TLSPort() uint16 {
	knock.mu.Lock()
	defer knock.mu.Unlock()
	return knock.tlsport
}

func (knock *ManagePortKnock) TLSKeyFile() string {
	knock.mu.Lock()
	defer knock.mu.Unlock()
	return knock.portKnockKeyFile
}

func (knock *ManagePortKnock) TLSCertFile() string {
	knock.mu.Lock()
	defer knock.mu.Unlock()
	return knock.portKnockKeyCert
}
