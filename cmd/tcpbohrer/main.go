package main

import (
	"fmt"
	"log"
	"os"
	"portknocktls"
	"qtime"
	"ratelimitedlogging"
	"strings"
	"tcpconnheloin2out"
	"tcpconnheloout2in"
	"tcpparameters"
	"time"
)

const (
	VERSIONMAJOR = 1
	VERSIONMINOR = 1
	VERSIONPATCH = 1
)

func main() {
	args := os.Args
	//var args [3]string
	//args[1] = "outside"
	//args[2] = "../tcpbohrer.yaml"
	if len(args) != 3 {
		panic("Usage: tcpbohrer inside|outside tcpbohrer.yaml")
	}
	log.Printf("tcpbohrer %d.%d.%d %s %s starting", VERSIONMAJOR, VERSIONMINOR,
		VERSIONPATCH, args[1], args[2])
	if strings.ToLower(args[1]) != "inside" && strings.ToLower(args[1]) != "outside" {
		panic("Usage: tcpbohrer inside|outside tcpbohrer.yaml")
	}
	bIsInside := false
	if strings.ToLower(args[1]) == "inside" {
		bIsInside = true
	}
	par, controlpack, knock, err := tcpparameters.LoadConfiguration(args[2], VERSIONMAJOR)
	if err != nil {
		panic(fmt.Sprintf("tcpbohrer: Error in configuration file %s: %s, terminating!\nCheck for missing parameters!", args[2], err))
	}
	print("Check for valid system time\n")
	if !qtime.QualifySystemTime(int32(par.Funnel.MaxWaitForValidSystemTimeSeconds)) {
		panic(fmt.Sprintf("System time still not valid after %d seconds", par.Funnel.MaxWaitForValidSystemTimeSeconds))
	}
	log := ratelimitedlogging.NewRateLimitedLogger(par.Log.IntervalSeconds, par.Log.LimitBurst,
		int32(par.Log.LimitHeloLoggingSeconds))

	if bIsInside {
		log.Log(fmt.Sprintf("Inside: Starting, Config Id is: %d\n", controlpack.ConfigID))
		tcpconnheloin2out.RunInside(par, controlpack, log)
	} else {
		log.Log(fmt.Sprintf("Outside: Starting, Config Id is: %d\n", controlpack.ConfigID))
		go portknocktls.GoStartTLSPortKnock(&knock)
		tcpconnheloout2in.RunOutside(par, controlpack, &knock, log)
	}
	for {
		time.Sleep(3600)
	}
}
