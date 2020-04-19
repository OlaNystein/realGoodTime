package main

import (
	"flag"
	"strconv"
	"time"

	. "./Config"
	synchronize "./Synchronize"
	"./elevio"
	"./fsm"
	"./network/bcast"
	"./network/peers"
	order "./order"
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

	//Light channel
	UpdateLightsChannel := make(chan [NumElevators]Elev)

	//****fsm-Order channels****

	//from fsm
	fsmUpdateChannel := make(chan Elev)
	fsmOrderCompleteChannel := make(chan Order)
	motorStoppedChannel := make(chan bool)

	//from Order
	OrderToFSMChannel := make(chan Elev)

	//****Order-sync channels****

	//from Order
	OrderToSyncChannel := make(chan [NumElevators]Elev)
	syncOrderCompleteChannel := make(chan Order)

	//from sync
	SyncToOrderChannel := make(chan [NumElevators]Elev)
	OnlineElevOrderChannel := make(chan [NumElevators]bool)
	reassignChannel := make(chan int)
	timedOutChannel := make(chan bool)

	//****sync-network channels****

	//from sync
	PeerTxEnable := make(chan bool)
	OutMsg := make(chan Message)

	//from network
	OnlineElevSyncChannel := make(chan [NumElevators]bool)
	PeerUpdateChannel := make(chan peers.PeerUpdate)
	InMsg := make(chan Message)

	//communication routines
	go peers.Transmitter(42035, ID, PeerTxEnable)
	go synchronize.ConnectedElevatorsRoutine(PeerUpdateChannel, OnlineElevSyncChannel, reassignChannel, timedOutChannel)
	go peers.Receiver(42035, PeerUpdateChannel)
	go bcast.Transmitter(42034, OutMsg)
	go bcast.Receiver(42034, InMsg)

	//hardware routines
	go elevio.PollButtons(newOrderChannel)
	go elevio.PollFloorSensor(sensorChannel)

	//modules
	go fsm.FsmRoutine(intID, sensorChannel, OrderToFSMChannel, fsmUpdateChannel,
		fsmOrderCompleteChannel, reassignChannel, motorStoppedChannel)

	go order.SetOrderLightsRoutine(UpdateLightsChannel, intID)

	go order.OrderRoutine(intID, OrderToSyncChannel, SyncToOrderChannel, OrderToFSMChannel,
		newOrderChannel, fsmUpdateChannel, UpdateLightsChannel,
		OnlineElevSyncChannel, fsmOrderCompleteChannel, syncOrderCompleteChannel,
		reassignChannel, OnlineElevOrderChannel, motorStoppedChannel)

	go synchronize.SynchronizerRoutine(intID, PeerUpdateChannel, PeerTxEnable, OrderToSyncChannel,
		SyncToOrderChannel, InMsg, OutMsg, OnlineElevSyncChannel,
		OnlineElevOrderChannel, syncOrderCompleteChannel, timedOutChannel)

	for {
		time.Sleep(time.Second)
	}

}
