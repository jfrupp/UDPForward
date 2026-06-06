package qtime

import "testing"

func TestQualifySystemTime(t *testing.T) {
	if !QualifySystemTime(0) {
		t.Error("Qualification of system time has failed!")
	}
}
