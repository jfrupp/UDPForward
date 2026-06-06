package timehelper

import (
	"testing"
)

func TestU32ToUnixTime(t *testing.T) {
	currUnix := int64(1700000000) // Example current Unix time
	u32 := uint32(1234567890)

	combined := U32ToUnixTime(u32, currUnix)
	expected := int64(uint64(u32) | (uint64(currUnix) & 0xffffffff00000000))

	if combined != expected {
		t.Errorf("U32ToUnixTime failed: got %d, expected %d", combined, expected)
	}
}

func TestUnixTimeToU32(t *testing.T) {
	unix := uint64(1700000000) // Example Unix time
	u32 := UnixTimeToU32(unix)
	expected := uint32(unix & 0xffffffff)

	if u32 != expected {
		t.Errorf("UnixTimeToU32 failed: got %d, expected %d", u32, expected)
	}
}

func TestAbsDiffUnixTime(t *testing.T) {
	u1 := int64(2000000000)
	u2 := int64(1500000000)

	diff := AbsDiffUnixTime(u1, u2)
	expected := int64(500000000)
	if diff != expected {
		t.Errorf("AbsDiffUnixTime failed: got %d, expected %d", diff, expected)
	}

	// Test the reverse order
	diff = AbsDiffUnixTime(u2, u1)
	if diff != expected {
		t.Errorf("AbsDiffUnixTime failed for reverse order: got %d, expected %d", diff, expected)
	}
}
func TestValidateSystemTime(t *testing.T) {
	if !ValidateSystemTime() {
		t.Errorf("Expected system time to be valid")
	}
}
