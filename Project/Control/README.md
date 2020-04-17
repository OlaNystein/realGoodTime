The "Control"-module
====================
Control is responsible for governing all orders and exchange them between relevant modules, namely `fsm` which executes orders on the elevator and `synchronizer` which sends update of the elevator and orders to the network. It includes various functions including one that checks if an order is already recorded and one that finds the optimal elevator for an order, given the current state of the peer-system. Using these, the control module of each elevator is equally influential to all other elevators on the network and can assign orders to any online peer. It also includes a separate routine to the control routine, which is responsible for toggling the appropriate order-lights.

Design choices
--------------
Other than the `fsm`-module, we wanted control to be the only module calling functions from `elevio` and making direct changes to the elevator hardware. It is however not responsible for executing orders, rather taking on the role of assigning and organizing them to lighten the load on other routines. This is why control, contrary to the name, controls very little of the actual elevator. It only governs the lights, which we considered more related to the orders rather than the movement of the elevator. When a hall order is recorded for one elevator at a floor, we wanted all the respective `BT_Hall[X]` lights to be turned on at the floor in question. `BT_Cab` orders only light up inside one elevator, as is natural. 

Contents
--------
- **orderAlreadyRecorded:** A private function checking if an order is present or not in the current list of elevators. 
- **calculateCost:** A private function calculating the optimal elevator to take an order. It uses the current information about all elevators on the network, their online-status and a new order which is to be assigned. It also takes in the assigning elevators own ID to quickly assign `BT_Cab` orders and can reassign orders to elevators that have disconnected or otherwise dissapeared from the network. When no elevator is lost, `lostID = -1`.
- **SetOrderLightsRoutine:** A public function to be run from main. It starts a routine which will update order ligths in parallel with the other routines of the elevator. It will sense updates to `updateLightChannel` and change the button-lights to match the current state of the elevator-network. All elevators are to have the same hall-lights at their separate floors and individual cab-lights. 
- **ControlRoutine:** A public function to be run from main. It works in parallell with all the other routines of the elevator, and handles orders from `newOrderChannel`, `FsmCompleteOrderChannel`, `SyncToControlChannel` and `reassignChannel`. It also records updates to the elevators state, floor and direction through `fsmUpdateChannel`. \\
\\
testing
    - case 