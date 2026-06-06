package timehelper

import "time"

func ValidateSystemTime() bool {
	t := time.Now()

	if t.IsZero() || t.Year() < 2025 {
		return false
	}
	return true
}

func UnixTimeToU32(unix uint64) uint32 {
	return uint32(unix & 0xffffffff)
}

// U32ToUnixTime combines the given u32 with the higher 32 bits of currUnix to form a full unix timestamp.
// This will fail around year 2106 when u32 wraps around until both the lowest 32 bit of currUnix and u32
// have wrapped (just for a couple of seconds).
func U32ToUnixTime(u32 uint32, currUnix int64) int64 {
	if currUnix <= 0 {
		currUnix = time.Now().Unix()
	}
	return int64(uint64(u32) | (uint64(currUnix) & 0xffffffff00000000))
}

// / As one time may lag behind the other, always use the absolute difference.
func AbsDiffUnixTime(u1 int64, u2 int64) int64 {
	if u1 > u2 {
		return u1 - u2
	}
	return u2 - u1
}
