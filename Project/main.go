package main

import (
	"flag"
	"strconv"
	"time"

	. "./Config"
	control "./Control"
	synchronize "./Synchronize"
	"./elevio"
	"./fsm"
	"./network/bcast"
	"./network/peers"
)

func main() {

	var (
		ID      string
		intID   int
		simPort string
	)

	flag.StringVar(&ID, "id", "0", "local elevator id")
	flag.StringVar(&simPort, "port", "", "local port used by the simulator")
	flag.Parse()
	intID, _ = strconv.Atoi(ID)

	println("My ID is: ", intID)
	println(simPort)

	if simPort != "" {
		elevio.Init("localhost:" + simPort)
	} else {
		elevio.Init("localhost:15657")
	}

	//HardwareChannels
	newOrderChannel := make(chan ButtonEvent)
	sensorChannel := make(chan int)

	//fsm-control channels
	OrderToFSMChannel := make(chan Elev)
	fsmUpdateChannel := make(chan Elev)
	fsmOrderCompleteChannel := make(chan Order)
	motorStoppedChannel := make(chan bool)

	//control-sync channels
	ControlToSyncChannel := make(chan [NumElevators]Elev)
	SyncToControlChannel := make(chan [NumElevators]Elev)
	OnlineElevSyncChannel := make(chan [NumElevators]bool)
	OnlineElevControlChannel := make(chan [NumElevators]bool)
	syncOrderCompleteChannel := make(chan Order)
	reassignChannel := make(chan int)
	timedOutChannel := make(chan bool)

	//Light channel
	UpdateLightsChannel := make(chan [NumElevators]Elev)

	//sync-network channels
	PeerUpdateChannel := make(chan peers.PeerUpdate)
	PeerTxEnable := make(chan bool)
	OutMsg := make(chan Message)
	InMsg := make(chan Message)

	go peers.Transmitter(42035, ID, PeerTxEnable)
	go synchronize.ConnectedElevatorsRoutine(PeerUpdateChannel, OnlineElevSyncChannel, reassignChannel, timedOutChannel)
	go peers.Receiver(42035, PeerUpdateChannel)

	go elevio.PollButtons(newOrderChannel)
	go elevio.PollFloorSensor(sensorChannel)

	go fsm.FsmRoutine(intID, sensorChannel, OrderToFSMChannel, fsmUpdateChannel, fsmOrderCompleteChannel, reassignChannel, motorStoppedChannel)
	go control.SetOrderLightsRoutine(UpdateLightsChannel, intID)
	go control.ControlRoutine(intID, ControlToSyncChannel, SyncToControlChannel, OrderToFSMChannel, newOrderChannel, fsmUpdateChannel, UpdateLightsChannel, OnlineElevSyncChannel, fsmOrderCompleteChannel, syncOrderCompleteChannel, reassignChannel, OnlineElevControlChannel, motorStoppedChannel)
	go synchronize.SynchronizerRoutine(intID, PeerUpdateChannel, PeerTxEnable, ControlToSyncChannel, SyncToControlChannel, InMsg, OutMsg, OnlineElevSyncChannel, OnlineElevControlChannel, syncOrderCompleteChannel, timedOutChannel)

	go bcast.Transmitter(42034, OutMsg)
	go bcast.Receiver(42034, InMsg)

	for {
		time.Sleep(time.Second)
	}

}
