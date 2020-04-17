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
		for j := ButtonType(0); j < 3; j++ {
			if elevator.Queue[i][j] {
				return true
			}
		}
	}
	return false
}

func fsmOrdersAtMe(elevator Elev) bool {
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
	fmt.Println("2")

	elevator.Dir = MD_Down
	elevio.SetMotorDirection(elevator.Dir)
	for elevator.Floor == -1 {
		elevator.Floor = <-sensorChannel
	}
	elevio.SetFloorIndicator(elevator.Floor)
	elevator.State = IDLE

	fmt.Println("3")
	return elevator
}

func FsmRoutine(myID int,
	sensorChannel <-chan int,
	orderToFsmChannel <-chan Elev,
	fsmUpdateChannel chan<- Elev,
	FSMCompleteOrderChannel chan<- Order,
	reassignChannel chan<- int,
	motorStoppedChannel chan<- bool) {
	var (
		elevator      Elev
		lastElevator  Elev
		motorProblems bool
		update        bool
	)

	elevator = fsmInit(sensorChannel, elevator)
	fsmUpdateChannel <- elevator

	go func() {
		for {
			select {
			case tempElev := <-orderToFsmChannel:
				lastElevator = elevator
				elevator.Queue = tempElev.Queue
			}
		}
	}()

	for {
		switch elevator.State {
		case IDLE:

			if motorProblems {
				elevator.Dir = MD_Down
				elevio.SetMotorDirection(elevator.Dir)

				go func() {
					for elevator.Floor == -1 {
						resetDirTimer := time.NewTimer(2 * time.Second)
						select {
						case <-resetDirTimer.C:
							if elevator.Floor == -1 {
								elevio.SetMotorDirection(MD_Down)
							} else {
								break
							}
						}
					}
				}()

				elevator.Floor = <-sensorChannel
				println("We re-initialized at floor: ", elevator.Floor)
				elevio.SetMotorDirection(MD_Stop)
				elevio.SetFloorIndicator(elevator.Floor)
				motorProblems = false
				motorStoppedChannel <- motorProblems
			}

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
				update = true

			} else if fsmOrdersAbove(elevator) {
				elevator.State = RUNNING
				elevator.Dir = MD_Up
				update = true
			}

			break

		case RUNNING:
			elevio.SetMotorDirection(elevator.Dir)
			motorErrorTimer := time.NewTimer(4 * time.Second)

			select {

			case tempFloor := <-sensorChannel:
				println("at floor", tempFloor)
				if tempFloor != -1 {

					motorErrorTimer.Stop()
					elevator.Floor = tempFloor
					elevio.SetFloorIndicator(elevator.Floor)
					if FsmShouldIStop(elevator) {

						elevator.State = DOOR_OPEN
					}
					update = true
				}
			case <-motorErrorTimer.C:
				println("Motor has stopped! Trying to restart")
				retryCounter := 0
				for retryCounter < 3 {

					motorErrorTimer.Reset(2 * time.Second)
					elevio.SetMotorDirection(elevator.Dir)
					select {

					case tempFloor := <-sensorChannel:

						if tempFloor != -1 {

							motorErrorTimer.Stop()
							elevator.Floor = tempFloor
							elevio.SetFloorIndicator(elevator.Floor)
							if FsmShouldIStop(elevator) {

								elevator.State = DOOR_OPEN
							}
							retryCounter = 3
							update = true

						}
					case <-motorErrorTimer.C:
						retryCounter++
						if retryCounter == 3 {
							println("ERROR: Was not able to restart engine. Reassigning orders.")
							reassignChannel <- myID
							elevator.Floor = -1
							motorProblems = true
							motorStoppedChannel <- motorProblems
							elevator.State = IDLE
						}
					}
				}
			}
			break

		case DOOR_OPEN:

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

			update = false
			go func() { fsmUpdateChannel <- elevator }()
		}

	}
}
