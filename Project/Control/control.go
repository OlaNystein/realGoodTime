package control

import (
	. "../Config"
	"../elevio"
	//"../fsm"
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
func calculateCost(myID int, elevList [NumElevators]Elev, newOrder ButtonEvent, onlineElevators [NumElevators]bool) int {
	if newOrder.Button == BT_Cab {
		return myID
	}
	bestCost := 1000
	theChosenOne := myID
	for elev := 0; elev < NumElevators; elev++ {
		if !onlineElevators[elev] {
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

// func controlCostClearOrders(e_old Elev) Elev{
// 	e := e_old
// 	for btn := 0; btn < NumButtonTypes; btn++ {
// 		if(e.Queue[e.Floor][btn]){
// 			e.Queue[e.Floor][btn] = false
// 		}
// 	}
// 	return e
// }

// func controlTimeToIdle(e Elev) int {
// 	duration := 0

// 	switch(e.State){
// 	case IDLE:
// 		if e.Dir == MD_Stop{
// 			return duration
// 		}
// 		break
// 	case RUNNING:
// 		duration += 2
// 		e.Floor += int(e.Dir)
// 		break
// 	case DOOR_OPEN:
// 		duration += -2
// 		break
// 	}

// 	for{
// 		if fsm.FsmShouldIStop(e){
// 			e = controlCostClearOrders(e)
// 			duration += 3
// 			if !(fsm.FsmShouldIContinue(e)){
// 				return duration
// 			}
// 		}
// 		e.Floor += int(e.Dir)
// 		duration += 4
// 	}

// }

// func controlCostFunction(elevList [NumElevators]Elev, onlineList [NumElevators]bool) int {

// 	bestCost := 1000
// 	theChosenOne := -1
// 	var cost int

// 	for i := 0; i < NumElevators; i++ {

// 		if !onlineList[i] {
// 			continue
// 		}

// 		cost = controlTimeToIdle(elevList[i])

// 		if cost < bestCost {
// 			bestCost = cost
// 			theChosenOne = i
// 		}

// 	}
// 	return theChosenOne
// }

//A routine to handle the order lamps. Ignores cab-lights
//CHECK THIS LATER! SHOULD WORK OK, BUT HAVE TO SEE IN LIGHT OF FSMUPDATECHANNEL
func SetOrderLightsRoutine(updateLightChannel <-chan [NumElevators]Elev) {
	for {
		select {
		case tempList := <-updateLightChannel:
			elevList := tempList
			for btn := ButtonType(0); btn < (3); btn++ { //Used 3 instead of NumButtonTypes as it yields an error
				for floor := 0; floor < NumFloors; floor++ {
					isThereAnOrderHere := false
					for elev := 0; elev < NumElevators; elev++ {
						if !isThereAnOrderHere && (elevList[elev].Queue[floor][btn]) {
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

func ControlRoutine(myID int, ControlToSyncChannel chan<- [NumElevators]Elev,
	SyncToControlChannel <-chan [NumElevators]Elev,
	OrderToFSMChannel chan<- Elev, newOrderChannel <-chan ButtonEvent,
	fsmUpdateChannel <-chan Elev, updateLightChannel chan<- [NumElevators]Elev,
	OnlineElevChannel <-chan [NumElevators]bool, FSMCompleteOrderChannel <-chan Order,
	SyncOrderChannel chan<- Order, reassignChannel <-chan int) {

	var (
		elevatorList    [NumElevators]Elev
		onlineElevators [NumElevators]bool
		online          bool
		order           Order
	)
	onlineElevators = <-OnlineElevChannel

	elevatorList[myID] = <-fsmUpdateChannel
	online = onlineElevators[myID]
	for {
		select {
		//Control recieved an update to the current online elevators
		case OnlineListUpdate := <-OnlineElevChannel:
			onlineElevators = OnlineListUpdate
			online = onlineElevators[myID]

			//Control receieves a new order from the hardware
		case newOrder := <-newOrderChannel:
			println("\nRecieved new order for button ", newOrder.Button, " at floor ", newOrder.Floor)
			if online {
				if (newOrder.Floor == elevatorList[myID].Floor && elevatorList[myID].State != RUNNING) {

					elevatorList[myID].Queue[newOrder.Floor][newOrder.Button] = true
					elevio.SetButtonLamp(newOrder.Button, newOrder.Floor, true)
					println("Sending the order to FSM without bcasting")
					go func() { OrderToFSMChannel <- elevatorList[myID] }()
					println("Order sent to FSM withour bcasting")

				} else if !orderAlreadyRecorded(elevatorList, newOrder) {

					optElev := calculateCost(myID, elevatorList, newOrder, onlineElevators)
					//optElev := controlCostFunction(elevatorList, onlineElevators)
					println("The optimal elevator for this order is: ", optElev)
					order.ID = optElev
					order.Floor = newOrder.Floor
					order.Button = newOrder.Button

					go func() { SyncOrderChannel <- order }()
					println("Order sent to synchronise from control for bcasting")
				}

			} else {
				if !orderAlreadyRecorded(elevatorList, newOrder) && newOrder.Button == BT_Cab {
					//not accepting orders outside the elevator if we're not online
					println("I'm not online")
					elevatorList[myID].Queue[newOrder.Floor][newOrder.Button] = true
					elevio.SetButtonLamp(newOrder.Button, newOrder.Floor, true)
					go func() { OrderToFSMChannel <- elevatorList[myID] }()
				}
			}

		//Control recieves an updated elevatorlist from the Syncronizer
		case tempElevList := <-SyncToControlChannel:
			
			
			for elev := 0; elev < NumElevators; elev++ {
				if elev == myID {
					continue
				}
				elevatorList[elev] = tempElevList[elev]
			}

			for floor := 0; floor < NumFloors; floor++ {
				for button := 0; button < NumButtonTypes; button++ {
					 if !elevatorList[myID].Queue[floor][button] && tempElevList[myID].Queue[floor][button]{
						 println("setting order to fsm")
						 elevatorList[myID].Queue[floor][button] = true
					 } else if elevatorList[myID].Queue[floor][button] && !tempElevList[myID].Queue[floor][button]{
						 println("clearing order to fsm")
						elevatorList[myID].Queue[floor][button] = false
					}
				}
			}
			updateLightChannel <- elevatorList

			go func() { OrderToFSMChannel <- elevatorList[myID]
				println("order sent to fsm") }()

		//Control recieves an update from the local fsm
		case updatedElevator := <-fsmUpdateChannel:
			tempQ := elevatorList[myID].Queue //preserve the queue
			elevatorList[myID] = updatedElevator
			elevatorList[myID].Queue = tempQ
			if online {
				ControlToSyncChannel <- elevatorList
			}

		case finished := <-FSMCompleteOrderChannel:
			if online {
				finished.ID = myID
				finished.Complete = true
				SyncOrderChannel <- finished
			} else {
				for i := ButtonType(0); i < 3; i++ {
					elevatorList[myID].Queue[finished.Floor][i] = false
				}
			}
		case lostID := <-reassignChannel:
			if online {
				for btn := ButtonType(0); btn < 3; btn++ {
					for floor := 0; floor < NumFloors; floor++ {
						if elevatorList[lostID].Queue[floor][btn] {
							var disconnectedOrder ButtonEvent
							disconnectedOrder.Floor = floor
							disconnectedOrder.Button = btn

							optElev := calculateCost(myID, elevatorList, disconnectedOrder, onlineElevators)
							elevatorList[lostID].Queue[floor][btn] = false
							elevatorList[optElev].Queue[floor][btn] = true

							println("\nSWAPPED ", btn, " AT FLOOR ", floor, " FROM ", lostID, " TO ", optElev)

						}
					}

				}
			}
			ControlToSyncChannel <- elevatorList
		}
	}
}
