package fsm

import (
	"fmt"
	"time"

	. "../Config"
	"../elevio"
)

func printLocalOrders(elevator Elev) {
	println("\nCURRENT LOCAL QUEUE:")
	println("Floor 4: BT_UP = ", elevator.Queue[3][BT_HallUp], ", BT_DOWN = ", elevator.Queue[3][BT_HallDown], ", BT_Cab = ", elevator.Queue[3][BT_Cab])
	println("Floor 3: BT_UP = ", elevator.Queue[2][BT_HallUp], ", BT_DOWN = ", elevator.Queue[2][BT_HallDown], ", BT_Cab = ", elevator.Queue[2][BT_Cab])
	println("Floor 2: BT_UP = ", elevator.Queue[1][BT_HallUp], ", BT_DOWN = ", elevator.Queue[1][BT_HallDown], ", BT_Cab = ", elevator.Queue[1][BT_Cab])
	println("Floor 1: BT_UP = ", elevator.Queue[0][BT_HallUp], ", BT_DOWN = ", elevator.Queue[0][BT_HallDown], ", BT_Cab = ", elevator.Queue[0][BT_Cab])

}

func FsmShouldIStop(elevator Elev) bool {
	switch elevator.Dir {
	case MD_Up:
		return elevator.Queue[elevator.Floor][BT_HallUp] || elevator.Queue[elevator.Floor][BT_Cab] || !fsmOrdersAbove(elevator)
	case MD_Down:
		return elevator.Queue[elevator.Floor][BT_HallDown] || elevator.Queue[elevator.Floor][BT_Cab] || !fsmOrdersBelow(elevator)
	case MD_Stop:
	default:
	}
	return false
}

func FsmShouldIContinue(elevator Elev) bool {
	switch elevator.Dir {
	case MD_Up:
		return (fsmOrdersAbove(elevator) && elevator.Floor != (NumFloors-1))
	case MD_Down:
		return (fsmOrdersBelow(elevator) && elevator.Floor != 0)
	}
	return false
}

func fsmOrdersAbove(elevator Elev) bool {
	for i := elevator.Floor + 1; i < NumFloors; i++ {
		//for j := 0; j < NumButtonTypes; j++ {
		for j := ButtonType(0); j < 3; j++ {
			if elevator.Queue[i][j] {
				return true
			}
		}
	}
	return false
}

func fsmOrdersBelow(elevator Elev) bool {
	for i := 0; i < elevator.Floor; i++ {
		//for j := 0; j < NumButtonTypes; j++ {
		for j := ButtonType(0); j < 3; j++ {
			if elevator.Queue[i][j] {
				return true
			}
		}
	}
	return false
}

func fsmOrdersAtMe(elevator Elev) bool {
	//for j := 0; j < NumButtonTypes; j++ {
	for j := ButtonType(0); j < 3; j++ {
		if elevator.Queue[elevator.Floor][j] {
			return true
		}
	}
	return false
}

func fsmInit(sensorChannel <-chan int, elevator Elev) Elev {

	elevator.Floor = -1
	elevator.Dir = MD_Stop
	fmt.Println("1")
	for i := 0; i < NumFloors; i++ {
		for j := 0; j < NumButtonTypes; j++ {
			elevator.Queue[i][j] = false
			elevio.SetButtonLamp(ButtonType(j), i, false)
		}
	}
	fmt.Println("2")
	if elevator.Floor != -1 {
		elevator.State = IDLE
		elevio.SetFloorIndicator(elevator.Floor)
	} else {
		elevator.Dir = MD_Down
		elevio.SetMotorDirection(elevator.Dir)
		for elevator.Floor == -1 {
			elevator.Floor = <-sensorChannel
		}
		elevio.SetFloorIndicator(elevator.Floor)
		elevator.State = IDLE
	}
	fmt.Println("3")
	return elevator
}

func FsmRoutine(sensorChannel <-chan int, orderToFsmChannel <-chan Elev, fsmUpdateChannel chan<- Elev, FSMCompleteOrderChannel chan<- Order) {
	var (
		elevator     Elev
		lastElevator Elev
		update       bool
	)

	// doorTimer := time.NewTimer(3 * time.Second)
	printTicker := time.NewTicker(2 * time.Second)
	//Remember to add a try/catch here later on
	elevator = fsmInit(sensorChannel, elevator)
	fsmUpdateChannel <- elevator

	go func() {
		for {
			select {
			case tempElev := <-orderToFsmChannel:
				lastElevator = elevator
				elevator.Queue = tempElev.Queue
				println("\nLocal FSM recieved elevator-update!")
			case <-printTicker.C:
				//printLocalOrders(elevator)
			}
		}
	}()

	for {
		switch elevator.State {
		case IDLE:
			//println("I am IDLE")
			if elevator.Dir != MD_Stop {
				elevio.SetMotorDirection(MD_Stop)
				elevator.Dir = MD_Stop
			}

			if fsmOrdersAtMe(elevator) {
				elevator.State = DOOR_OPEN
				update = true
			} else if fsmOrdersBelow(elevator) {
				elevator.State = RUNNING
				elevator.Dir = MD_Down
				println("There are orders below")

				update = true
			} else if fsmOrdersAbove(elevator) {
				elevator.State = RUNNING
				elevator.Dir = MD_Up
				println("There are orders above")
				update = true
			}

			break

		case RUNNING:
			elevio.SetMotorDirection(elevator.Dir)
			select {
			case tempFloor := <-sensorChannel:
				println("at floor", tempFloor)
				if tempFloor != -1 {
					elevator.Floor = tempFloor
					elevio.SetFloorIndicator(elevator.Floor)
					if FsmShouldIStop(elevator) {
						elevator.State = DOOR_OPEN
					}
					update = true
				}

			}
			break

		case DOOR_OPEN:
			// println("I am in DOOR OPEN")
			elevio.SetMotorDirection(MD_Stop)

			doorTimer := time.NewTimer(3 * time.Second)
			elevio.SetDoorOpenLamp(true)

			var finishedOrder Order
			finishedOrder.Floor = elevator.Floor

			go func() { FSMCompleteOrderChannel <- finishedOrder }()
			go func() { fsmUpdateChannel <- elevator }()

			<-doorTimer.C
			elevio.SetDoorOpenLamp(false)
			if fsmOrdersAtMe(elevator) {
				//Stay in the same state
			} else if FsmShouldIContinue(elevator) {
				elevator.State = RUNNING

			} else {
				elevator.State = IDLE

			}
			update = true
			break

		}
		if update && lastElevator != elevator {
			println("UPDATING CONTROL")
			update = false
			go func() { fsmUpdateChannel <- elevator }()
			println("CONTROL UPDATED!")
		}

	}
}
