package writehelo

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

func WriteHeloFile(path string) {
	if len(path) == 0 {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("%d\nNumber of running Go routines: %d\nCurrent Time: %s\n",
		os.Getpid(), runtime.NumGoroutine(), time.Now().Format("2006-01-02 15:04:05")))
}
