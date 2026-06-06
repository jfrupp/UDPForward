package udpmessage

import (
	"controlpacker"
	"errors"
	"net/netip"
	"sync"
)

// IDs for message flows are 1..255
const CONTROL_ID int8 = 0
const CONTROL_FLOW_HELO int8 = 0
const CONTROL_FLOW_REPLY int8 = 1
const CONTROL_FLOW_DELETE int8 = 2

type UDPMessage struct {
	IsToRecycle    bool
	IsControl      bool
	IsMSG          bool
	IsValidAddress bool
	AddrPort       netip.AddrPort
	Control        controlpacker.DecodedControlPacket
	Payload        *[]byte
	LenPayload     int
}

const MAX_MSG_CACHE int =2048
const LENPAYLOADCACHE = 2048
const LENMSGCACHE = 2048 + 100

var indexPayloadInCache = 0
var indexMsgInCache = 0
var mutex sync.Mutex
var payloadCache [LENPAYLOADCACHE]*[]byte
var msgCache [LENMSGCACHE]*UDPMessage

var CachePayloadSize int = 0

func SetCachePayloadSize(size int) {
	mutex.Lock()
	defer mutex.Unlock()

	CachePayloadSize = size
	indexPayloadInCache = 0
	indexMsgInCache = 0
	for i := 0; i < LENPAYLOADCACHE; i++ {
		payloadCache[i] = nil
	}
	for i := 0; i < LENMSGCACHE; i++ {
		msgCache[i] = nil
	}
}

func getPayload(size int) *[]byte {

	// no Mutex, only called from getMsg
	if indexPayloadInCache <= 0 || size != CachePayloadSize || payloadCache[indexPayloadInCache-1] == nil {
		if size >= 2 {
			p := make([]byte, size)
			return &p
		} else {
			return nil
		}
	}
	pp := payloadCache[indexPayloadInCache-1]
	payloadCache[indexPayloadInCache-1] = nil
	if indexPayloadInCache > 0 {
		indexPayloadInCache--
	}
	return pp
}

func returnPayload(pay *[]byte) {

	// no Mutex, only called from ReturnMsg

	if pay != nil && len(*pay) == CachePayloadSize && indexPayloadInCache < LENPAYLOADCACHE {
		payloadCache[indexPayloadInCache] = pay
		indexPayloadInCache++
	}
}

func getMsg(sizePayload int) *UDPMessage {
	mutex.Lock()
	defer mutex.Unlock()

	if sizePayload < 2 {
		sizePayload = 0
	}
	var p *UDPMessage
	if indexMsgInCache <= 0 || msgCache[indexMsgInCache-1] == nil {
		var m UDPMessage
		p = &m
	} else {
		p = msgCache[indexMsgInCache-1]
		msgCache[indexMsgInCache-1] = nil
		if indexMsgInCache > 0 {
			indexMsgInCache--
		}
	}
	p.Payload = getPayload(sizePayload)
	return p
}

func ReturnMsg(p *UDPMessage) {
	mutex.Lock()
	defer mutex.Unlock()

	if p == nil || !p.IsToRecycle {
		return
	}
	returnPayload(p.Payload)
	p.Payload = nil
	if indexMsgInCache < LENMSGCACHE {
		msgCache[indexMsgInCache] = p
		indexMsgInCache++
	}
}

func ResetPool() {
	mutex.Lock()
	defer mutex.Unlock()

	for i := 0; i < len(msgCache); i++ {
		msgCache[i] = nil
	}
	for i := 0; i < len(payloadCache); i++ {
		payloadCache[i] = nil
	}
	indexMsgInCache = 0
	indexPayloadInCache = 0
	CachePayloadSize = 0
}

func NewUDPMessage(sizePayload int) *UDPMessage {

	m := getMsg(sizePayload)
	m.IsControl = false
	m.IsMSG = false
	m.IsValidAddress = false
	m.IsToRecycle = false
	m.LenPayload = sizePayload
	return m
}

func (m *UDPMessage) GetIDFlowPayload() (int8, int8, bool) {
	if m.Payload == nil {
		return 0, 0, false
	}
	msg := *m.Payload
	return int8(msg[0]), int8(msg[1]), true
}

func (m *UDPMessage) CloneUDPMessage(withPayload bool) *UDPMessage {

	c := getMsg(0)
	c.IsControl = m.IsControl
	c.IsMSG = m.IsMSG
	c.IsValidAddress = m.IsValidAddress
	c.Control = m.Control
	c.IsToRecycle = false
	if withPayload {
		c.Payload = m.Payload
		c.LenPayload = m.LenPayload
		m.IsToRecycle = false
	} else {
		c.Payload = nil
		c.LenPayload = 0
	}
	return c
}

func (m *UDPMessage) GetSliceWithoutIDFlow() []byte {
	if m.Payload == nil || m.LenPayload <= 2 {
		return make([]byte, 0)
	} else {
		pay := *m.Payload
		return pay[2:m.LenPayload]
	}
}

func (m *UDPMessage) SetIdFlowIncrLen(id uint8, flow uint8) {
	if m.Payload == nil {
		return
	}
	if len(*m.Payload) < 2 {
		return
	}
	(*m.Payload)[0] = id
	(*m.Payload)[1] = flow
	m.LenPayload += 2
}

func UDPControlMessageFromByteSlice(msgblock []byte, source netip.AddrPort,
	bIsInsideToOutside bool, unixTime int64, cp *controlpacker.ControlPacker) *UDPMessage {

	if len(msgblock) != 10 {
		return nil
	}
	if msgblock[0] != 0 {
		return nil
	}
	var e controlpacker.EncodedControlPacket
	copy(e.Data[:], msgblock)
	e.Len = 10
	d, err := cp.DecodeControlPacket(&e, bIsInsideToOutside, unixTime)
	if err != nil {
		return nil
	}
	m := NewUDPMessage(0)
	m.IsControl = true
	m.IsMSG = false
	m.IsValidAddress = true
	m.AddrPort = source
	m.Control = d
	m.Payload = &msgblock
	m.LenPayload = 10
	return m
}

func ByteSliceFromUDPControlMessage(m *UDPMessage, bIsInsideToOutside bool,
	unixTime int64, cp *controlpacker.ControlPacker) ([]byte, *netip.AddrPort, error) {
	if m.IsControl == false || m.IsValidAddress == false || m.Control.Id != 0 {
		return make([]byte, 0), nil, errors.New("not a control message")
	}
	e, err := cp.EncodeControlPacket(&m.Control, bIsInsideToOutside, unixTime, false)
	if err != nil {
		return make([]byte, 0), nil, err
	}
	s := make([]byte, 10)
	copy(s, e.Data[0:10])
	return s, &m.AddrPort, nil
}
