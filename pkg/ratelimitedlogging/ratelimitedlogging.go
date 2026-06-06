package ratelimitedlogging

import (
	"log"
	"strings"
	"sync"
	"time"
)

type RateLimitedLogger struct {
	mu                  sync.Mutex
	interval            float32
	lastLogTime         int64
	lastHeloLogTime     int64
	heloBurst           int32
	burst               int32
	maxBurst            int32
	intervalHeloLogging float32
}

func NewRateLimitedLogger(interval float32, intervalHeloLogging float32, maxBurst int32) *RateLimitedLogger {
	if interval <= 0.00001 {
		interval = 0.00001
	}
	if maxBurst <= 1 {
		maxBurst = 1
	}
	r := RateLimitedLogger{
		interval:            interval,
		lastLogTime:         0,
		burst:               maxBurst,
		maxBurst:            maxBurst,
		intervalHeloLogging: intervalHeloLogging,
	}
	r.lastLogTime = time.Now().Unix()
	return &r
}

func (r *RateLimitedLogger) Log(message string) {
	var now int64
	var incr int32

	r.mu.Lock()
	defer r.mu.Unlock()

	now = time.Now().Unix()
	timeDiff := now - r.lastLogTime
	isHeloToBeLogged := false
	if strings.Contains(message, "HELO") && r.intervalHeloLogging > 0.0 {
		// start new HELO burst
		if now > r.lastHeloLogTime+int64(r.intervalHeloLogging) {
			r.heloBurst += 2
		}
		if r.heloBurst > 3 {
			r.heloBurst = 3
		}
		if r.heloBurst == 0 {
			return
		}
		isHeloToBeLogged = true
	}
	incr = int32((1.0 / r.interval) * float32(timeDiff))
	if incr >= 1 {
		r.burst = r.burst + incr
		r.lastLogTime = int64(float32(incr)*r.interval) + r.lastLogTime
	}
	if r.burst > r.maxBurst {
		r.burst = r.maxBurst
	}
	if r.burst <= 0 {
		return
	}
	r.burst = r.burst - 1
	if isHeloToBeLogged {
		r.heloBurst--
		r.lastHeloLogTime = now
	}
	log.Println(message)
}
