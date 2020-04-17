package order

import (
	. "../Config"
	"../elevio"
)

func printLocalOrders(elevatorList [NumElevators]Elev, myID int) {
	println("\nCURRENT LOCAL QUEUE:")
	println("Floor 4: BT_UP = ", elevatorList[myID].Queue[3][BT_HallUp], ", BT_DOWN = ", elevatorList[myID].Queue[3][BT_HallDown], ", BT_Cab = ", elevatorList[myID].Queue[3][BT_Cab])
	println("Floor 3: BT_UP = ", elevatorList[myID].Queue[2][BT_HallUp], ", BT_DOWN = ", elevatorList[myID].Queue[2][BT_HallDown], ", BT_Cab = ", elevatorList[myID].Queue[2][BT_Cab])
	println("Floor 2: BT_UP = ", elevatorList[myID].Queue[1][BT_HallUp], ", BT_DOWN = ", elevatorList[myID].Queue[1][BT_HallDown], ", BT_Cab = ", elevatorList[myID].Queue[1][BT_Cab])
	println("Floor 1: BT_UP = ", elevatorList[myID].Queue[0][BT_HallUp], ", BT_DOWN = ", elevatorList[myID].Queue[0][BT_HallDown], ", BT_Cab = ", elevatorList[myID].Queue[0][BT_Cab])

}

func orderAlreadyRecorded(elevList [NumElevators]Elev, order ButtonEvent) bool {
	floor := order.Floor
	btn := order.Button
	for elev := 0; elev < NumElevators; elev++ {
		if elevList[elev].Queue[floor][btn] == true {
			return true
		}
	}
	return false
}

//Assigns an order to an elevator
func calculateCost(myID int, lostID int, elevList [NumElevators]Elev, newOrder ButtonEvent,
	onlineElevators [NumElevators]bool) int {
	if newOrder.Button == BT_Cab {
		return myID
	}
	bestCost := 1000
	theChosenOne := myID
	for elev := 0; elev < NumElevators; elev++ {
		if !onlineElevators[elev] || elev == lostID {
			println("Elevator ", elev, " is not online")
			continue
		}
		cost := newOrder.Floor - elevList[elev].Floor

		if cost == 0 && (elevList[elev].State == IDLE || elevList[elev].State == DOOR_OPEN) {
			theChosenOne = elev
			return theChosenOne
		}
		if cost < 0 {
			cost = -cost
			if elevList[elev].Dir == MD_Up {
				cost += (NumFloors - 1) - elevList[elev].Floor
				//cost += 3 //arbitrary number
			}
		} else if cost > 0 && elevList[elev].Dir == MD_Down {
			cost += elevList[elev].Floor
			//cost += 3 //arbitrary number
		}
		if cost == 0 && elevList[elev].State == RUNNING {
			if elevList[elev].Dir == MD_Up {
				cost = (NumFloors - 1 - newOrder.Floor) * 2
			} else if elevList[elev].Dir == MD_Down {
				cost = elevList[elev].Floor * 2
			}
			//cost += 4
		}
		if cost < bestCost {
			bestCost = cost
			theChosenOne = elev
		}
	}
	return theChosenOne

}


func SetOrderLightsRoutine(updateLightChannel <-chan [NumElevators]Elev, myID int) {
	for {
		select {
		case elevList := <-updateLightChannel:
			for btn := ButtonType(0); btn < (3); btn++ {
				for floor := 0; floor < NumFloors; floor++ {
					isThereAnOrderHere := false
					for elev := 0; elev < NumElevators; elev++ {
						if elev == myID && btn == BT_Cab && elevList[elev].Queue[floor][btn] {
							isThereAnOrderHere = true
							elevio.SetButtonLamp(btn, floor, true)
							continue
						} else if btn != BT_Cab && !isThereAnOrderHere && elevList[elev].Queue[floor][btn] {
							isThereAnOrderHere = true
							elevio.SetButtonLamp(btn, floor, true)
						}
					}
					if !isThereAnOrderHere {
						elevio.SetButtonLamp(btn, floor, false)
					}
				}
			}

		}
	}
}

func OrderRoutine(myID int, ControlToSyncChannel chan<- [NumElevators]Elev,
	SyncToControlChannel <-chan [NumElevators]Elev,
	OrderToFSMChannel chan<- Elev, newOrderChannel <-chan ButtonEvent,
	fsmUpdateChannel <-chan Elev, updateLightChannel chan<- [NumElevators]Elev,
	OnlineElevChannel <-chan [NumElevators]bool, FSMCompleteOrderChannel <-chan Order,
	SyncOrderChannel chan<- Order, reassignChannel <-chan int,
	OnlineElevChannelControl <-chan [NumElevators]bool, motorStoppedChannel <-chan bool) {

	var (
		elevatorList    [NumElevators]Elev
		onlineElevators [NumElevators]bool
		motorStopped    bool
		online          bool
		order           Order
	)

	//Subroutine to update online elevator list
	go func() {
		for {
			select {
			case OnlineListUpdate := <-OnlineElevChannelControl:
				println("ONLINE ELEVATORS UPDATED IN ORDER!")
				onlineElevators = OnlineListUpdate
				println(onlineElevators[0], " ", onlineElevators[1], " ", onlineElevators[2], "\n")
				online = onlineElevators[myID]
			case motorStatus := <-motorStoppedChannel:
				motorStopped = motorStatus
			}
		}
	}()

	for {
		select {
		//Control receieves a new order from the hardware
		case newOrder := <-newOrderChannel:
			println("\nRecieved new order for button ", newOrder.Button, " at floor ", newOrder.Floor)

			if online && !orderAlreadyRecorded(elevatorList, newOrder) {
				if !motorStopped {

					optElev := calculateCost(myID, -1, elevatorList, newOrder, onlineElevators)
					order.ID = optElev
					order.Floor = newOrder.Floor
					order.Button = newOrder.Button
					order.Complete = false
					go func() { SyncOrderChannel <- order }()

				} else if motorStopped {

					println("\nERROR: Cannot accept orders for stuck elevator")
					continue
				}
			} else if !online && !orderAlreadyRecorded(elevatorList, newOrder) {
				if newOrder.Button == BT_Cab {

					println("\nCaborder registered for offline elevator")
					order.ID = myID
					order.Floor = newOrder.Floor
					order.Button = newOrder.Button
					order.Complete = false

					go func() { SyncOrderChannel <- order }()
					go func() { updateLightChannel <- elevatorList }()
					go func() { OrderToFSMChannel <- elevatorList[myID] }()

				} else {

					println("\nCannot accept hall-orders, elevator offline")
					continue
				}
			}

		//Control recieves an updated elevatorlist from the Syncronizer
		case tempElevList := <-SyncToControlChannel:

			for elev := 0; elev < NumElevators; elev++ {
				if elev != myID {
					elevatorList[elev] = tempElevList[elev]
				}
				elevatorList[elev].Queue = tempElevList[elev].Queue
			}

			go func() { updateLightChannel <- elevatorList }()

			go func() { OrderToFSMChannel <- elevatorList[myID] }()

		//Control recieves an update from the local fsm
		case updatedElevator := <-fsmUpdateChannel:

			tempQ := elevatorList[myID].Queue
			elevatorList[myID] = updatedElevator
			elevatorList[myID].Queue = tempQ

			go func() { ControlToSyncChannel <- elevatorList }()

		case finished := <-FSMCompleteOrderChannel:
			println("\nOrder Finished!")
			finished.ID = myID
			finished.Complete = true
			go func() { SyncOrderChannel <- finished }()

		case lostID := <-reassignChannel:
			if online {
				println("\nREASSIGNING ORDERS FOR ELEVATOR: ", lostID)
				println(onlineElevators[0], " ", onlineElevators[1], " ", onlineElevators[2])
				for btn := ButtonType(0); btn < 2; btn++ {
					for floor := 0; floor < NumFloors; floor++ {
						if elevatorList[lostID].Queue[floor][btn] {
							println("FOUND AN ORDER!")
							var disconnectedOrder ButtonEvent
							disconnectedOrder.Floor = floor
							disconnectedOrder.Button = btn
							optElev := calculateCost(myID, lostID, elevatorList, disconnectedOrder, onlineElevators)

							deleteOrder := Order{Complete: true, Button: btn, Floor: floor, ID: lostID}
							go func() { SyncOrderChannel <- deleteOrder }()

							reassignedOrder := Order{Complete: false, Button: btn, Floor: floor, ID: optElev}
							go func() { SyncOrderChannel <- reassignedOrder }()

							println("\nSWAPPED ", btn, " AT FLOOR ", floor, " FROM ", lostID, " TO ", optElev)

						}
					}

				}
			}
		}
	}
}