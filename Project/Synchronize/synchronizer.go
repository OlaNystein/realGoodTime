package synchronize

import (
	"fmt"
	"strconv"
	"time"

	// "../network/bcast"
	"../network/peers"
	// "../network/localip"
	// "../network/conn"
	. "../Config"
)

func ConnectedElevatorsRoutine(PeerUpdateChannel <-chan peers.PeerUpdate, OnlineElevChannel chan<- [NumElevators]bool, reassignChannel chan<- int) {
	var onlineElevators [NumElevators]bool
	for {
		select {
		case p := <-PeerUpdateChannel:

			fmt.Printf("\nPeer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

			if len(p.Peers) == 0 { //we have timed out
				//TimeOut = true

			} else if p.New != "" { //new elevator connected
				println("WE GOT A NEW ONE!")

				newPeerID, _ := strconv.Atoi(p.New)
				onlineElevators[newPeerID] = true

			} else if len(p.Lost) > 0 { // lost boys
				println("We lost one...")
				for lostPeer := 0; lostPeer < len(p.Lost); lostPeer++ {
					lostPeerID, _ := strconv.Atoi(p.Lost[lostPeer])
					onlineElevators[lostPeerID] = false
					println(onlineElevators[0], " ", onlineElevators[1], " ", onlineElevators[2])
					reassignChannel <- lostPeerID
				}
			}
			println(onlineElevators[0], " ", onlineElevators[1], " ", onlineElevators[2])
			OnlineElevChannel <- onlineElevators
		}
	}
}

func SynchronizerRoutine(myID int, PeerUpdateChannel <-chan peers.PeerUpdate,
	PeerTxEnable chan<- bool,
	ControlToSyncChannel <-chan [NumElevators]Elev,
	SyncToControlChannel chan<- [NumElevators]Elev,
	IncomingMessageChannel <-chan Message,
	OutgoingMessageChannel chan<- Message,
	//SynchronizeChannel <-chan bool,
	//ErrorChannel chan bool,
	OnlineElevChannel chan<- [NumElevators]bool,
	SyncOrderChannel <-chan Order) {

	var (
		onlineElevators [NumElevators]bool
		elevatorList    [NumElevators]Elev
		outgoingPackage Message
		update          bool
		timeOut         bool = false
	)

	ITimedOut := make(chan bool)
	go func() { time.Sleep(time.Second); ITimedOut <- true }()

	select { //gets msg from bcast.recieve if we have successfully connected
	case initSuccess := <-IncomingMessageChannel:
		elevatorList = initSuccess.ElevList
		update = true

	case <-ITimedOut:
		timeOut = true
	}

	println("Timed out: ", timeOut)

	onlineElevators[myID] = true

	for {
		if timeOut {
			if onlineElevators[myID] {
				onlineElevators[myID] = false

			}
		}
		select {
		case ctrlUpdate := <-ControlToSyncChannel:

			tempQ := elevatorList[myID].Queue
			elevatorList[myID] = ctrlUpdate[myID]
			elevatorList[myID].Queue = tempQ
			update = true

		case order := <-SyncOrderChannel:
			println("hello")
			if order.ID == myID{
				if order.Complete {
					for btn := 0; btn < 3; btn++{
						elevatorList[myID].Queue[order.Floor][btn] = false
					}
					println("local order completed")
				} else {
					elevatorList[myID].Queue[order.Floor][order.Button] = true
					println("local order added")
				}
				go func() {SyncToControlChannel <- elevatorList}()
			} else {
				if onlineElevators[order.ID] {
					outgoingPackage.NewOrder = order
				}
			}
			outgoingPackage.NewOrder = order
			outgoingPackage.ID = myID
			outgoingPackage.ElevList = elevatorList
			go func() {for i:= 0; i < 3; i++{
				OutgoingMessageChannel <- outgoingPackage}
			 }()
			

		case msg := <-IncomingMessageChannel:

		
			if msg.ElevList != elevatorList {
				tempElevator := elevatorList[myID]
				elevatorList = msg.ElevList
				elevatorList[myID] = tempElevator
				update = true
			}
			if msg.NewOrder.ID == myID && msg.ID != myID{
				elevatorList[myID].Queue[msg.NewOrder.Floor][msg.NewOrder.Button] = true
				update = true
			}
			

			if update {
				println("Updating from sync to control")
				update = false
				go func(){SyncToControlChannel <- elevatorList}()
				println("Control updated from sync")
			}
		}
	}
}
