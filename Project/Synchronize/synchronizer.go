package synchronize

import (
	"fmt"
	"strconv"
	"time"

	//"time"

	// "../network/bcast"
	"../network/peers"
	// "../network/localip"
	// "../network/conn"
	. "../Config"
)

func syncClearNonLocalOrders(elevatorList [NumElevators]Elev, myID int) [NumElevators]Elev {
	for e := 0; e < NumElevators; e++ {
		if e != myID {
			for floor := 0; floor < NumFloors; floor++ {
				for btn := 0; btn < NumButtonTypes; btn++ {
					elevatorList[e].Queue[floor][btn] = false
				}
			}
		}
	}
	return elevatorList
}

func checkOnlineElevators(onlineElevators [NumElevators]bool) int {
	onlinecount := 0
	for elev := 0; elev < len(onlineElevators); elev++ {
		if onlineElevators[elev] {
			onlinecount++
		}
	}
	return onlinecount
}

func ConnectedElevatorsRoutine(PeerUpdateChannel <-chan peers.PeerUpdate, OnlineElevChannel chan<- [NumElevators]bool, reassignChannel chan<- int, timedOutChannel chan<- bool) {

	var (
		onlineElevators [NumElevators]bool
		timeOut         bool = true
	)
	for {
		select {
		case p := <-PeerUpdateChannel:

			fmt.Printf("\nPeer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

			if len(p.Peers) == 0 { //we have timed out
				timeOut = true
				go func() { timedOutChannel <- timeOut }()

			}
			if p.New != "" { //new elevator connected
				newPeerID, _ := strconv.Atoi(p.New)
				println(newPeerID, "Just connected!")
				onlineElevators[newPeerID] = true
				timeOut = false
				go func() { timedOutChannel <- timeOut }()

			}
			if len(p.Lost) > 0 { // lost boys
				for lostPeer := 0; lostPeer < len(p.Lost); lostPeer++ {
					lostPeerID, _ := strconv.Atoi(p.Lost[lostPeer])
					println(lostPeerID, " just disconnected")
					onlineElevators[lostPeerID] = false
					go func() { reassignChannel <- lostPeerID }()
				}
			}
			go func() { OnlineElevChannel <- onlineElevators }()
		}
	}
}

func SynchronizerRoutine(myID int, PeerUpdateChannel <-chan peers.PeerUpdate,
	PeerTxEnable chan<- bool,
	ControlToSyncChannel <-chan [NumElevators]Elev,
	SyncToControlChannel chan<- [NumElevators]Elev,
	IncomingMessageChannel <-chan Message,
	OutgoingMessageChannel chan<- Message,
	OnlineElevChannel <-chan [NumElevators]bool,
	OnlineElevChannelControl chan<- [NumElevators]bool,
	SyncOrderChannel <-chan Order,
	timedOutChannel <-chan bool) {

	var (
		onlineElevators [NumElevators]bool
		elevatorList    [NumElevators]Elev
		outgoingPackage Message
		update          bool
		timeOut         bool = false
	)

	go func() {
		for {
			select {
			case OnlineListUpdate := <-OnlineElevChannel:
				println("ONLINE ELEVATORS UPDATED IN SYNC!")
				onlineElevators = OnlineListUpdate
				println(onlineElevators[0], " ", onlineElevators[1], " ", onlineElevators[2], "\n")
				if !onlineElevators[myID] {
					timeOut = true
				}
				if checkOnlineElevators(onlineElevators) > 1 {
					outgoingPackage.ID = myID
					outgoingPackage.ElevList = elevatorList
					outgoingPackage.NewOrder.ID = -1
					go func() {
						if !timeOut {
							for i := 0; i < 3; i++ {
								OutgoingMessageChannel <- outgoingPackage
							}
						}
					}()
				}
				go func() { OnlineElevChannelControl <- onlineElevators }()
			case timeOut = <-timedOutChannel:

				if timeOut {
					println("I disconnected from the network!")
					elevatorList = syncClearNonLocalOrders(elevatorList, myID)
				} else {
					println("I connected to the network")
				}
			}
		}
	}()

	//Initialize the elevatorlist
	initTimer := time.NewTimer(3 * time.Second)
	select {
	case initMsg := <-IncomingMessageChannel:
		println("Current length of online elevators: ", checkOnlineElevators(onlineElevators))
		if checkOnlineElevators(onlineElevators) > 1 {
			elevatorList = initMsg.ElevList
			go func() { SyncToControlChannel <- elevatorList }()
		}
	case <-initTimer.C:
		println("Warning: Not able to initialize peer elevatorlist")
	}

	for {
		select {

		//Records update to local elevator
		case ctrlUpdate := <-ControlToSyncChannel:

			tempQ := elevatorList[myID].Queue
			elevatorList[myID] = ctrlUpdate[myID]
			elevatorList[myID].Queue = tempQ
			update = true

		//Distributes an order recorded in control
		case order := <-SyncOrderChannel:
			if order.ID == myID {
				if order.Complete {
					for btn := 0; btn < 3; btn++ {
						elevatorList[myID].Queue[order.Floor][btn] = false
					}
					println("local order completed")
				} else {
					elevatorList[myID].Queue[order.Floor][order.Button] = true
					println("local order added")

				}
			} else {
				if order.Complete { //order reassigned
					elevatorList[order.ID].Queue[order.Floor][order.Button] = false
				} else if onlineElevators[order.ID] {
					println("Order registered for elevator ", order.ID)
					elevatorList[order.ID].Queue[order.Floor][order.Button] = true
				} else {
					println("ERROR: Could not assign order to offline-elevator", order.ID)
					println(onlineElevators[0], " ", onlineElevators[1], " ", onlineElevators[2], "\n")
				}
			}
			outgoingPackage.NewOrder = order
			outgoingPackage.ID = myID
			outgoingPackage.ElevList = elevatorList
			go func() {
				if !timeOut {
					for i := 0; i < 3; i++ {
						OutgoingMessageChannel <- outgoingPackage
					}
				}
			}()
			go func() { SyncToControlChannel <- elevatorList }()

		//Recieves an update from one of the other elevators
		case msg := <-IncomingMessageChannel:

			if msg.ElevList != elevatorList {
				println("\nUPDATED ELEVATORLIST")
				tempElevator := elevatorList[myID]
				elevatorList = msg.ElevList
				elevatorList[myID] = tempElevator
				update = true
			}
			if msg.NewOrder.ID == myID && msg.ID != myID {
				println("UPDATED ORDER AT SELF")
				elevatorList[myID].Queue[msg.NewOrder.Floor][msg.NewOrder.Button] = true
				update = true
			}
		}
		if update {
			println("Updating from sync to control")
			update = false
			go func() { SyncToControlChannel <- elevatorList }()
			println("Control updated from sync")
		}
	}
}
