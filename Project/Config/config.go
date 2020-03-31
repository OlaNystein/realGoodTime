package config

import (
	"net"
	"sync"
	"time"
)

const NumFloors int = 4
const NumElevators int = 3
const NumButtonTypes int = 3

var _initialized bool = false
var _mtx sync.Mutex
var _conn net.Conn

//HARDWARE configs
const PollRate = 20 * time.Millisecond

type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown            = 1
	BT_Cab                 = 2
)

type MotorDirection int

const (
	MD_Up   MotorDirection = 1
	MD_Down                = -1
	MD_Stop                = 0
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

//CONFIG-types:
type Elev struct {
	ElevatorID int
	State      ElevState
	Dir        MotorDirection
	Floor      int
	Queue      [NumFloors][NumButtonTypes]bool
}

type ElevState int

const (
	UNDEFINED ElevState = -1
	IDLE                = 0
	RUNNING             = 1
	DOOR_OPEN           = 2
)


type Order struct {
	Complete bool
	Button      ButtonType
	Floor    int
	ID       int
}

type Message struct {
	ElevList  [NumElevators]Elev
	NewOrder  Order
	ID        int
}

