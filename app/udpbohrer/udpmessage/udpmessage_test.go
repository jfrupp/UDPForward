package udpmessage

import (
	"fmt"
	"testing"
)

func TestMessageWithoutID(t *testing.T) {

	m := NewUDPMessage(2048)
	s := m.GetSliceWithoutIDFlow()

	msg := *m.Payload
	s[1] = 3
	if msg[3] != 3 {
		t.Error("Expected different value in msg[3]")
	}
}

func TestMessageGetBackPayload(t *testing.T) {
	SetCachePayloadSize(2048)
	m := NewUDPMessage(2048)
	s := m.GetSliceWithoutIDFlow()
	s[4] = 3
	s[100] = 1
	s[1000] = 4
	m.IsToRecycle = true
	ReturnMsg(m)
	n := NewUDPMessage(2048)
	u := n.GetSliceWithoutIDFlow()
	if u[4] != 3 || u[100] != 1 || u[1000] != 4 {
		fmt.Println(n)
		fmt.Println(u)
		t.Error("Did not get payload again from Pool")
	}
	if n.IsToRecycle {
		t.Error("Got back a message with IsToRecycle==true")
	}
}

func TestReturn1000Messages(t *testing.T) {
	var mv [1000]*UDPMessage

	ResetPool()
	SetCachePayloadSize(2048)
	m := NewUDPMessage(2048)
	s := m.GetSliceWithoutIDFlow()
	s[4] = 3
	s[100] = 1
	s[1000] = 4
	m.IsToRecycle = true
	mv[0] = m
	for i := 1; i < 1000; i++ {
		mv[i] = NewUDPMessage(2048)
		mv[i].IsToRecycle = true
	}
	for i := 999; i >= 0; i-- {
		ReturnMsg(mv[i])
		mv[i] = nil
	}
	n := NewUDPMessage(2048)
	u := n.GetSliceWithoutIDFlow()
	if u[4] != 3 || u[100] != 1 || u[1000] != 4 {
		fmt.Println(n)
		fmt.Println(u)
		t.Error("Did not get payload again from Pool")
	}
	if n.IsToRecycle {
		t.Error("Got back a message with IsToRecycle==true")
	}
	o := NewUDPMessage(2048)
	v := o.GetSliceWithoutIDFlow()
	if v[4] == 3 {
		t.Error("Message recylce twice")
	}
}

func TestReturn5x20000Messages(t *testing.T) {
	var mv [20000]*UDPMessage

	ResetPool()
	SetCachePayloadSize(2048)
	for j := 0; j < 5; j++ {
		for i := 0; i < 20000; i++ {
			mv[i] = NewUDPMessage(2048)
			mv[i].IsToRecycle = true
		}
		for i := 19999; i >= 0; i-- {
			ReturnMsg(mv[i])
			mv[i] = nil
		}
	}
}

func TestReturn1000x1000Messages(t *testing.T) {
	var mv [20000]*UDPMessage

	ResetPool()
	SetCachePayloadSize(2048)
	for j := 0; j < 1000; j++ {
		for i := 0; i < 1000; i++ {
			mv[i] = NewUDPMessage(2048)
			mv[i].IsToRecycle = true
		}
		for i := 999; i >= 0; i-- {
			ReturnMsg(mv[i])
			mv[i] = nil
		}
	}
}

func TestReturnNils(t *testing.T) {

	SetCachePayloadSize(2048)
	m1 := NewUDPMessage(2048)
	m2 := NewUDPMessage(2048)

	m1 = nil
	m2.Payload = nil
	ReturnMsg(m1)
	ReturnMsg(m2)
}

func TestReturnDifferentSize(t *testing.T) {

	ResetPool()
	SetCachePayloadSize(2048)
	m := NewUDPMessage(2047)
	s := m.GetSliceWithoutIDFlow()
	s[4] = 3
	s[100] = 1
	s[1000] = 4
	m.IsToRecycle = true
	ReturnMsg(m)
	n := NewUDPMessage(2048)
	u := n.GetSliceWithoutIDFlow()
	if u[4] == 3 && u[100] == 1 && u[1000] == 4 {
		fmt.Println(n)
		fmt.Println(u)
		t.Error("Did get payload again from the pool")
	}
}

func SmallPayloads(t *testing.T) {

	SetCachePayloadSize(2048)
	m0 := NewUDPMessage(0)
	m1 := NewUDPMessage(1)
	m2 := NewUDPMessage(2)
	m3 := NewUDPMessage(3)

	if len(*m0.Payload) != 0 || m0.LenPayload != 0 {
		t.Error("m1 Payload broken")
	}
	if len(*m1.Payload) != 1 || m1.LenPayload != 1 {
		t.Error("m1 Payload broken")
	}
	if len(*m2.Payload) != 2 || m2.LenPayload != 2 {
		t.Error("m1 Payload broken")
	}
	if len(*m3.Payload) != 3 || m3.LenPayload != 3 {
		t.Error("m2 Payload broken")
	}
}
