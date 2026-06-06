package main

import (
	"fmt"
	"udpconnin2out"
	"udpconnout2in"
)

func main() {
	var m udpconnout2in.ManageRemoteId

	fmt.Println(m.GetChForRemoteId(1))
	fmt.Println("Hello, World!")
	a := udpconnin2out.Go_in2out_send
	b := udpconnin2out.Go_in2out_receive
	fmt.Println(a)
	fmt.Println(b)
	a = nil
	b = nil
}
