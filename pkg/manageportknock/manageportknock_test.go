package manageportknock

import (
	"net/netip"
	"testing"
	"time"
)

func TestInsertAndCheck(t *testing.T) {
	mp := &ManagePortKnock{timeOut: 3600, maxportknock: 10}
	a := netip.MustParseAddr("127.0.0.1")

	mp.Insert("knock1", a)
	if !mp.Check("knock1", a) {
		t.Fatalf("expected Check to return true after Insert")
	}

	// simulate expiry by setting timestamp in the past
	mp.mu.Lock()
	if mp.kmap != nil {
		if inner, ok := mp.kmap["knock1"]; ok {
			inner[a] = time.Now().Unix() - 10
		}
	}
	mp.mu.Unlock()

	if mp.Check("knock1", a) {
		t.Fatalf("expected Check to return false for expired entry")
	}
}

func TestMaxPortKnockEnforced(t *testing.T) {
	mp := &ManagePortKnock{timeOut: 3600, maxportknock: 1}
	a1 := netip.MustParseAddr("127.0.0.1")
	a2 := netip.MustParseAddr("127.0.0.2")

	mp.Insert("kn", a1)
	if !mp.Check("kn", a1) {
		t.Fatalf("first insert should succeed")
	}

	mp.Insert("kn", a2)
	// second insert should be rejected because maxportknock==1
	if mp.Check("kn", a2) {
		t.Fatalf("second insert should have been rejected due to max limit")
	}

	// ensure countportknock is 1
	mp.mu.Lock()
	if mp.countportknock != 1 {
		t.Fatalf("expected countportknock==1, got %d", mp.countportknock)
	}
	mp.mu.Unlock()
}

func TestConfigSetsTimeoutAndMax(t *testing.T) {
	mp := &ManagePortKnock{timeOut: 10, maxportknock: 5}

	mp.Config(42, 7, 8443, "key.pem", "cert.pem")

	if mp.timeOut != 42 {
		t.Fatalf("expected timeOut to be 42, got %d", mp.timeOut)
	}
	if mp.maxportknock != 7 {
		t.Fatalf("expected maxportknock to be 7, got %d", mp.maxportknock)
	}
	if mp.TLSPort() != 8443 {
		t.Fatalf("expected TLS port to be 8443, got %d", mp.TLSPort())
	}
	if mp.TLSKeyFile() != "key.pem" {
		t.Fatalf("expected TLS key file to be key.pem, got %s", mp.TLSKeyFile())
	}
	if mp.TLSCertFile() != "cert.pem" {
		t.Fatalf("expected TLS cert file to be cert.pem, got %s", mp.TLSCertFile())
	}
}

func TestCheckNilMap(t *testing.T) {
	mp := &ManagePortKnock{timeOut: 10, maxportknock: 5}
	a := netip.MustParseAddr("127.0.0.1")
	if mp.Check("any", a) {
		t.Fatalf("expected Check to return false when kmap is nil")
	}
}

func TestCheckNamePositiveAndNegative(t *testing.T) {
	mp := &ManagePortKnock{timeOut: 3600, maxportknock: 10}
	if mp.CheckName("knock1") {
		t.Fatalf("expected CheckName to return false when kmap is nil")
	}

	mp.Insert("knock1", netip.MustParseAddr("127.0.0.1"))
	if !mp.CheckName("knock1") {
		t.Fatalf("expected CheckName to return true for an existing name")
	}
	if mp.CheckName("missing") {
		t.Fatalf("expected CheckName to return false for a missing name")
	}
}

func TestIterator(t *testing.T) {
	mp := &ManagePortKnock{}
	seen := make(map[string]bool)
	for name := range mp.Iterator() {
		seen[name] = true
	}
	if len(seen) != 0 {
		t.Fatalf("expected empty iterator, got %v", seen)
	}

	mp.InsertName("knock1")
	mp.InsertName("knock2")
	seen = make(map[string]bool)
	for name := range mp.Iterator() {
		seen[name] = true
	}

	if len(seen) != 2 || !seen["knock1"] || !seen["knock2"] {
		t.Fatalf("iterator returned unexpected names: %v", seen)
	}
}

func TestInsertWhenMaxZero(t *testing.T) {
	mp := &ManagePortKnock{timeOut: 10, maxportknock: 0}
	a := netip.MustParseAddr("127.0.0.1")
	mp.Insert("kn", a)
	if mp.Check("kn", a) {
		t.Fatalf("expected no insert when maxportknock==0")
	}
}

func TestCleanupRemovesExpiredAndResetsCount(t *testing.T) {
	mp := &ManagePortKnock{timeOut: 1, maxportknock: 10}
	a1 := netip.MustParseAddr("127.0.0.1")
	a2 := netip.MustParseAddr("127.0.0.2")

	mp.Insert("kn", a1)
	mp.Insert("kn", a2)

	// expire both entries
	mp.mu.Lock()
	if mp.kmap != nil {
		if inner, ok := mp.kmap["kn"]; ok {
			inner[a1] = time.Now().Unix() - 10
			inner[a2] = time.Now().Unix() - 10
		}
	}
	mp.mu.Unlock()

	mp.cleanup()

	mp.mu.Lock()
	if mp.countportknock != 0 {
		t.Fatalf("expected countportknock==0 after cleanup, got %d", mp.countportknock)
	}
	if mp.kmap != nil {
		if inner, ok := mp.kmap["kn"]; ok && len(inner) != 0 {
			t.Fatalf("expected no entries for 'kn' after cleanup")
		}
	}
	mp.mu.Unlock()
}

func TestCheckNonexistentNameOrAddr(t *testing.T) {
	mp := &ManagePortKnock{timeOut: 3600, maxportknock: 10}
	a := netip.MustParseAddr("127.0.0.1")
	if mp.Check("nope", a) {
		t.Fatalf("expected false for nonexistent name")
	}
	mp.Insert("kn", a)
	if mp.Check("kn", netip.MustParseAddr("127.0.0.2")) {
		t.Fatalf("expected false for nonexistent addr under existing name")
	}
}

func TestCountNotNegative(t *testing.T) {
	mp := &ManagePortKnock{timeOut: 10, maxportknock: 10}
	mp.countportknock = -5
	mp.cleanup()
	if mp.countportknock != 0 {
		t.Fatalf("expected countportknock reset to 0, got %d", mp.countportknock)
	}
}
