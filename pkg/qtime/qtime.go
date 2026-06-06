package qtime

import "time"

func QualifySystemTime(maxTimeWait int32) bool {
	for ; maxTimeWait >= 0; maxTimeWait-- {
		// 2025-12-31 23:00 UTC+0
		if time.Now().Unix() > 1767222000 {
			return true
		}
		time.Sleep(1)
	}
	return false
}
