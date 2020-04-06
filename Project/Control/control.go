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

	//Subroutine to update online elevators
	go func() {
		for {
			select {
			case OnlineListUpdate := <-OnlineElevChannel:
				println("ONLINE ELEVATORS UPDATED!")
				onlineElevators = OnlineListUpdate
				online = onlineElevators[myID]
			}
		}
	}()

	for {
		select {
		//Control receieves a new order from the hardware
		case newOrder := <-newOrderChannel:
			println("\nRecieved new order for button ", newOrder.Button, " at floor ", newOrder.Floor)

			if online && !orderAlreadyRecorded(elevatorList, newOrder) {
				optElev := calculateCost(myID, elevatorList, newOrder, onlineElevators)
				order.ID = optElev
				order.Floor = newOrder.Floor
				order.Button = newOrder.Button
				order.Complete = false
				go func() { SyncOrderChannel <- order }()
			} else if !online {

			}

		//Control recieves an updated elevatorlist from the Syncronizer
		case tempElevList := <-SyncToControlChannel:
			println("Got an update from sync")

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

			tempQ := elevatorList[myID].Queue //preserve the queue
			elevatorList[myID] = updatedElevator
			elevatorList[myID].Queue = tempQ
			if online {
				go func() { ControlToSyncChannel <- elevatorList }()
			}

		case finished := <-FSMCompleteOrderChannel:

			if online || true {
				finished.ID = myID
				finished.Complete = true
				go func() { SyncOrderChannel <- finished }()
			} else {
				for i := ButtonType(0); i < 3; i++ {
					elevatorList[myID].Queue[finished.Floor][i] = false
				}
			}
		case lostID := <-reassignChannel:
			println("Am I online?: ", online)
			if online {
				println("\nREASSIGNING ORDERS FOR ELEVATOR: ", lostID)
				printLocalOrders(elevatorList, 0)
				println(onlineElevators[0], " ", onlineElevators[1], " ", onlineElevators[2])
				for btn := ButtonType(0); btn < 3; btn++ {
					for floor := 0; floor < NumFloors; floor++ {
						if elevatorList[lostID].Queue[floor][btn] {
							println("FOUND AN ORDER!")
							var disconnectedOrder ButtonEvent
							disconnectedOrder.Floor = floor
							disconnectedOrder.Button = btn
							optElev := calculateCost(myID, elevatorList, disconnectedOrder, onlineElevators)

							deleteOrder := Order{Complete: true, Button: btn, Floor: floor, ID: lostID}
							go func() { SyncOrderChannel <- deleteOrder }()

							reassignedOrder := Order{Complete: false, Button: btn, Floor: floor, ID: optElev}
							go func() { SyncOrderChannel <- reassignedOrder }()
							//elevatorList[lostID].Queue[floor][btn] = false
							//elevatorList[optElev].Queue[floor][btn] = true

							println("\nSWAPPED ", btn, " AT FLOOR ", floor, " FROM ", lostID, " TO ", optElev)

						}
					}

				}
			}
		}
	}
}
