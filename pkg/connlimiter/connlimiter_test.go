package connlimiter

import (
	"testing"
)

func TestRequWithoutLimit(t *testing.T) {
	var cl ConnLimiter

	cl.SetConnLimit(0)
	if cl.RequestConn() == false {
		t.Error("No limit, still connection cannot be requested")
	}
}

func TestReq2Conns(t *testing.T) {
	var cl ConnLimiter

	cl.SetConnLimit(2)
	if cl.RequestConn() == false || cl.RequestConn() == false {
		t.Error("Cannot request two connections")
	}
}

func TestReq3Conn(t *testing.T) {
	var cl ConnLimiter

	cl.SetConnLimit(2)
	a := cl.RequestConn()
	b := cl.RequestConn()
	d := cl.RequestConn()

	if !a || !b || d {
		t.Error("An issue with connections")
	}
}

func TestReq3ConnsWithRelease(t *testing.T) {
	var cl ConnLimiter

	cl.SetConnLimit(2)
	a := cl.RequestConn()
	b := cl.RequestConn()
	cl.ReleaseConn()
	d := cl.RequestConn()

	if !a || !b || !d || cl.GetConnCount() != 2 {
		t.Error("An issue with connections")
	}
}
