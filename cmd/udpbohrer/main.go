package main

import (
	"connlimiter"
	"controlpacker"
	"fmt"
	"log"
	"net"
	"os"
	"qtime"
	"ratelimitedlogging"
	"strings"
	"time"
	"udpbohrerparameters"
	"udpconnin2out"
	"udpconnmaplocal"
	"udpconnout2in"
	"udpconnout2remote"
	"udpmessage"
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
	//args[2] = "../udpconfig.yaml"
	if len(args) != 3 {
		panic("Usage: udpbohrer inside|outside udpconfig.yaml")
	}
	log.Printf("udpbohrer %d.%d.%d %s %s starting", VERSIONMAJOR, VERSIONMINOR,
		VERSIONPATCH, args[1], args[2])
	if strings.ToLower(args[1]) != "inside" && strings.ToLower(args[1]) != "outside" {
		panic("Usage: udpbohrer inside|outside udpconfig.yaml")
	}
	bIsInside := false
	if strings.ToLower(args[1]) == "inside" {
		bIsInside = true
	}
	par, controlpack, err := udpbohrerparameters.LoadConfiguration(args[2], VERSIONMAJOR)
	if err != nil {
		panic(fmt.Sprintf("Udpproxy: Error in configuration file %s: %s, terminating!\nCheck for missing parameters!", args[2], err))
	}
	print("Check for valid system time\n")
	if !qtime.QualifySystemTime(int32(par.Funnel.MaxWaitForValidSystemTimeSeconds)) {
		panic(fmt.Sprintf("System time still not valid after %d seconds", par.Funnel.MaxWaitForValidSystemTimeSeconds))
	}
	udpmessage.SetCachePayloadSize(par.Funnel.Mtu)
	logger := ratelimitedlogging.NewRateLimitedLogger(float32(par.Log.IntervalSeconds),
		float32(par.Log.LimitHeloLoggingSeconds), int32(par.Log.LimitBurst))
	if bIsInside {
		logger.Log(fmt.Sprintf("Inside: Starting, Config Id is: %d\n", controlpack.ConfigID))
		runInside(VERSIONMAJOR, par, controlpack, logger)
	} else {
		logger.Log(fmt.Sprintf("Outside: Starting, Config Id is: %d\n", controlpack.ConfigID))
		runOutside(VERSIONMAJOR, par, controlpack, logger)
	}
	for {
		time.Sleep(3600 * time.Second)
	}
}

// does the actual running by starting up the Go routines
func runInside(versionmajor uint8, par *udpbohrerparameters.UdpproxyConfig,
	controlpack *controlpacker.ControlPacker, logger *ratelimitedlogging.RateLimitedLogger) {

	var local udpconnmaplocal.ManageLocalChannels
	var udpaddr net.UDPAddr

	chout := make(chan *udpmessage.UDPMessage, 2048)
	udpaddr.Port = par.Funnel.InPort
	conn, err := net.ListenUDP(par.Funnel.Protocol, &udpaddr)
	if err != nil {
		panic(fmt.Sprintf("udpbohrer: Could not bind to port %s/%d, exiting!", par.Funnel.Protocol, udpaddr.Port))
	}
	ipv46 := 0
	if par.Funnel.Protocol == "udp4" {
		ipv46 = 4
	} else if par.Funnel.Protocol == "udp6" {
		ipv46 = 6
	} else {
		panic(fmt.Sprintf("udpbohrer: Protocol %s is not supported, exiting!", par.Funnel.Protocol))
	}
	go udpconnin2out.Go_in2out_send(chout, conn, ipv46, par.Funnel.OutHost, uint16(par.Funnel.OutPort), controlpack,
		logger, float32(par.Funnel.HeloRepeatInterval), par.Funnel.Mtu,
		par.Funnel.PortBufferSize, &local, int64(par.Funnel.HeloRoundtripTime))
	go udpconnin2out.Go_in2out_receive(chout, par, conn, ipv46, par.Funnel.OutHost, uint16(par.Funnel.OutPort), controlpack,
		logger, float32(par.Funnel.HeloRepeatInterval), par.Funnel.Mtu,
		&local, par.Funnel.HeloTimeout, par.Funnel.HeloFile)
}

func runOutside(versionmajor uint8, par *udpbohrerparameters.UdpproxyConfig,
	controlpack *controlpacker.ControlPacker, logger *ratelimitedlogging.RateLimitedLogger) {

	var addrPortIn udpconnout2in.ManageAddressPort
	var remoteIds udpconnout2in.ManageRemoteId
	var udpaddr net.UDPAddr

	chout := make(chan *udpmessage.UDPMessage, 2048)
	chin := make(chan *udpmessage.UDPMessage, 1024)
	udpaddr.Port = par.Funnel.OutPort
	conn, err := net.ListenUDP(par.Funnel.Protocol, &udpaddr)
	if err != nil {
		panic(fmt.Sprintf("udpbohrer: Could not bind to port %s/%d, exiting!", par.Funnel.Protocol, udpaddr.Port))
	}
	if len(par.Flows) == 0 {
		panic("No Forwarders have been defined in the configuration file!")
	}

	var c connlimiter.ConnLimiter
	var f udpconnout2remote.FlowManagement
	c.SetConnLimit(uint32(par.Funnel.MaxCurrentFlows))
	f.Connlimit = &c
	for id, flowpars := range par.Flows {
		rempar := new(udpconnout2remote.GoParametersToRemote)
		mapAddr := new(udpconnout2remote.MapAddrPortToFlow)
		mapAddr.Connlimit = &c
		mapAddr.Flows = &f
		mapAddr.TimeOutBidirectional = int64(flowpars.TimeoutSeconds)
		mapAddr.TimeOutSingleDirection = int64(flowpars.InitTimeoutSeconds)
		rempar.Id = uint8(id)
		rempar.Chout = chout
		rempar.Chin = make(chan *udpmessage.UDPMessage, 1024)
		rempar.SizePayload = par.Funnel.Mtu
		rempar.MapAddr = mapAddr
		rempar.PortBufferSize = par.Funnel.PortBufferSize
		var udpaddr net.UDPAddr
		udpaddr.Port = flowpars.OutPort
		conn, err := net.ListenUDP(flowpars.OutProtocol, &udpaddr)
		if err != nil {
			panic(fmt.Sprintf("udpbohrer: Id %d, could not bind to port %d/%s, exiting!", id, flowpars.OutPort, flowpars.OutProtocol))
		}
		rempar.LocalUDPSocket = conn
		rempar.Log = logger
		remoteIds.SetChForRemoteId(uint8(id), rempar.Chin)
		go udpconnout2remote.Go_receive_from_remote(rempar)
		go udpconnout2remote.Go_send_to_remote(rempar)
	}
	go udpconnout2in.Go_out2in_send(chout, conn, controlpack,
		logger, &addrPortIn, par.Funnel.Mtu, par.Funnel.PortBufferSize)

	go udpconnout2in.Go_out2in_receive(chin, conn, controlpack,
		logger, &addrPortIn, int64(par.Funnel.HeloRepeatInterval),
		&remoteIds, par.Funnel.Mtu, par.Funnel.PortBufferSize,
		par.Funnel.HeloTimeout, par.Funnel.HeloFile)
}
