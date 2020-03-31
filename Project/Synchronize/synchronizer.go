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

func checkAcknowledgements(onlineElevators [NumElevators]bool, ackMsg AcknowledgeMsg, ackType Acknowledgement) bool {
	for elev := 0; elev < NumElevators; elev++ {
		if !onlineElevators[elev] {
			continue
		}
		if ackMsg.AckStatus[elev] != ackType {
			return false
		}
	}
	return true
}

func copyAcknowledgements(msg Message, ackMatrix [NumFloors][NumButtonTypes - 1]AcknowledgeMsg, elevator int, floor int, id int, button int) [NumFloors][NumButtonTypes - 1]AcknowledgeMsg {
	ackMatrix[floor][button].AckStatus[id] = msg.AckMatrix[floor][button].AckStatus[elevator]
	ackMatrix[floor][button].AckStatus[elevator] = msg.AckMatrix[floor][button].AckStatus[elevator]
	ackMatrix[floor][button].ChosenElevator = msg.AckMatrix[floor][button].ChosenElevator
	return ackMatrix
}

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
		ackMatrix       [NumFloors][NumButtonTypes - 1]AcknowledgeMsg
		update          bool
		timeOut         bool = false
	)

	ITimedOut := make(chan bool)
	go func() { time.Sleep(time.Second); ITimedOut <- true }()

	select { //gets msg from bcast.recieve if we have successfully connected
	case initSuccess := <-IncomingMessageChannel:
		elevatorList = initSuccess.ElevList
		ackMatrix = initSuccess.AckMatrix
		update = true

	case <-ITimedOut:
		timeOut = true
	}
	bcastTicker := time.NewTicker(100 * time.Millisecond)
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
			if !order.Complete {
				println("\nOrder is not complete")
				if order.Button == BT_Cab {
					elevatorList[myID].Queue[order.Floor][BT_Cab] = true
					update = true
					println("It's a cab, updating queue")
				} else {
					println("It's a hall, starting Acks")
					ackMatrix[order.Floor][order.Button].ChosenElevator = order.ID
					ackMatrix[order.Floor][order.Button].AckStatus[myID] = Ack
					//Vi starter acknowledgen
				}
			} else {
				println("\nOrder is done")
				elevatorList[order.ID].Queue[order.Floor] = [NumButtonTypes]bool{false, false, false}
				update = true

				if order.Button != BT_Cab {
					println("Done with hall order, starting restarting of Acks")
					ackMatrix[order.Floor][BT_HallUp].AckStatus[myID] = OrderDone
					ackMatrix[order.Floor][BT_HallDown].AckStatus[myID] = OrderDone
				}
			}

		case msg := <-IncomingMessageChannel:

			//Sender ut ACK
			//Set at alle har ACKed ordren i msg.AckOrders
			// if msg.ID == myID || !onlineElevators[msg.ID]{
			// 	continue
			// } else {
			if msg.ElevList != elevatorList {
				tempElevator := elevatorList[myID]
				elevatorList = msg.ElevList
				elevatorList[myID] = tempElevator
				update = true
			}
			for elev := 0; elev < NumElevators; elev++ {
				// if elev == myID || !onlineElevators[elev] {
				// 	//skal bare sammenlikne meg selv med andre heiser, ikke vits å sammenlikne med meg selv eller noen som ikke er online
				// 	continue
				// }
				for floor := 0; floor < NumFloors; floor++ {
					for button := 0; button < NumButtonTypes-1; button++ {

						switch msg.AckMatrix[floor][button].AckStatus[elev] {
						//oppdatere lokal liste først i alle cases

						case Ack:

							if ackMatrix[floor][button].AckStatus[myID] == NotAck {
								//Hvis vi henger et steg bak må vi oppdatere all info i lokal liste før utsending
								ackMatrix = copyAcknowledgements(msg, ackMatrix, elev, floor, myID, button)
								println("\nackMatrix at Elevator: ", elev, ", floor: ", floor, ", button: ", button)
								println("we have acknowledged an order for elevator ", ackMatrix[floor][button].ChosenElevator, " in floor ", floor)

							} else if ackMatrix[floor][button].AckStatus[elev] != Ack {
								//Evt hvis en heis har acket etter oss
								ackMatrix[floor][button].AckStatus[elev] = Ack

							}
							if ackMatrix[floor][button].ChosenElevator == myID &&
								checkAcknowledgements(onlineElevators, ackMatrix[floor][button], Ack) &&
								!elevatorList[myID].Queue[floor][button] {
								println("\norder acknowledged by all, added to queue")
								//Må oppdatere ordrekøen dersom alle har acket og det er vår ordre
								elevatorList[myID].Queue[floor][button] = true
								update = true

							}

						case OrderDone:
							println("ORDER DONE")
							if ackMatrix[floor][button].AckStatus[myID] == Ack {
								//Hvis vi henger et steg bak må vi oppdatere all info i lokal liste før utsending
								ackMatrix = copyAcknowledgements(msg, ackMatrix, elev, floor, myID, button)

							} else if ackMatrix[floor][button].AckStatus[elev] != OrderDone {
								//Evt hvis en heis har OrderDone etter oss
								ackMatrix[floor][button].AckStatus[elev] = OrderDone

							}
							if checkAcknowledgements(onlineElevators, ackMatrix[floor][button], OrderDone) {
								//starter på notAck igjen dersom alle har bekreftet at ordren er fullført
								ackMatrix[floor][button].AckStatus[myID] = NotAck
								println("order finished by all")
								if ackMatrix[floor][button].ChosenElevator == myID {
									println("order removed from queue")
									//fjerner fra ordrekøen vår hvis det var vår og må da oppdatere control
									elevatorList[myID].Queue[floor][button] = false
									update = true

								}
							}

						case NotAck:

							if ackMatrix[floor][button].AckStatus[myID] == OrderDone {
								//Hvis vi henger et steg bak må vi oppdatere all info i lokal liste før utsending
								ackMatrix = copyAcknowledgements(msg, ackMatrix, elev, floor, myID, button)

							} else if ackMatrix[floor][button].AckStatus[elev] != NotAck {
								//Evt hvis en heis har notacket etter oss
								ackMatrix[floor][button].AckStatus[elev] = NotAck
							}
						}
					}
				}
			}

			if update {
				println("Updating from sync to control")
				update = false
				SyncToControlChannel <- elevatorList
				println("Control updated from sync")
			}

		case <-bcastTicker.C:
			if timeOut || true {
				outgoingPackage.ElevList = elevatorList
				outgoingPackage.ID = myID
				outgoingPackage.AckMatrix = ackMatrix
				OutgoingMessageChannel <- outgoingPackage
			}
		}
	}
}
